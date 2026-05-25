package handler

import (
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"minibili/internal/errcode"
	"minibili/internal/middleware"
	"minibili/internal/model"
	"minibili/internal/pkg/iplocate"
	"minibili/internal/pkg/resp"
	"minibili/internal/pkg/userlevel"
)

func loadUserDynamic(a *API, id uint64) (*model.UserDynamic, bool) {
	var dyn model.UserDynamic
	if err := a.DB.First(&dyn, id).Error; err != nil {
		return nil, false
	}
	return &dyn, true
}

func (a *API) dynamicCommentsToJSON(list []model.DynamicComment, authorID, viewer uint64) []gin.H {
	var uids []uint64
	for _, cm := range list {
		uids = append(uids, cm.UserID)
	}
	names := map[uint64]string{}
	avatars := map[uint64]string{}
	userLevels := map[uint64]int{}
	if len(uids) > 0 {
		var users []model.User
		_ = a.DB.Where("id IN ?", uids).Find(&users).Error
		for i := range users {
			usr := &users[i]
			names[usr.ID] = model.DisplayUsername(usr)
			avatars[usr.ID] = uploaderAvatarForAPI(usr)
		}
		userLevels = userlevel.BatchCurrentLevels(a.DB, uids)
	}
	likedByViewer := map[uint64]bool{}
	dislikedByViewer := map[uint64]bool{}
	if viewer > 0 && len(list) > 0 {
		ids := make([]uint64, len(list))
		for i := range list {
			ids[i] = list[i].ID
		}
		var likes []model.DynamicCommentLike
		_ = a.DB.Where("user_id = ? AND comment_id IN ?", viewer, ids).Find(&likes).Error
		for _, lk := range likes {
			likedByViewer[lk.CommentID] = true
		}
		var dislikes []model.DynamicCommentDislike
		_ = a.DB.Where("user_id = ? AND comment_id IN ?", viewer, ids).Find(&dislikes).Error
		for _, dk := range dislikes {
			dislikedByViewer[dk.CommentID] = true
		}
	}
	out := make([]gin.H, 0, len(list))
	for _, cm := range list {
		ulv := userLevels[cm.UserID]
		if ulv < 1 {
			ulv = 1
		}
		out = append(out, gin.H{
			"id":             cm.ID,
			"user_id":        cm.UserID,
			"username":       names[cm.UserID],
			"avatar_url":     avatars[cm.UserID],
			"parent_id":      cm.ParentID,
			"level":          cm.Level,
			"user_level":     ulv,
			"content":        cm.Content,
			"like_count":     cm.LikeCount,
			"created_at":     cm.CreatedAt.Format("2006-01-02 15:04:05"),
			"liked_by_me":    likedByViewer[cm.ID],
			"disliked_by_me": dislikedByViewer[cm.ID],
			"pinned":         cm.Pinned,
			"is_by_uploader": cm.UserID == authorID,
			"ip_location":    iplocate.DisplayLabel(cm.IpLocation),
		})
	}
	return out
}

// ListDynamicComments returns comments for a user dynamic.
func (a *API) ListDynamicComments(c *gin.Context) {
	did, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || did == 0 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	dyn, ok := loadUserDynamic(a, did)
	if !ok {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	if dyn.CommentsClosed {
		resp.OK(c, gin.H{
			"items":            []gin.H{},
			"comments_closed":  true,
			"comments_curated": dyn.CommentsCurated,
		})
		return
	}
	commentQ := a.DB.Where("dynamic_id = ?", did)
	if dyn.CommentsCurated {
		commentQ = commentQ.Where("approved = ?", true)
	}
	var list []model.DynamicComment
	if err := commentQ.Order("id ASC").Find(&list).Error; err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	var viewer uint64
	if uid, ok := middleware.UserID(c); ok {
		viewer = uid
	}
	out := a.dynamicCommentsToJSON(list, dyn.UserID, viewer)
	resp.OK(c, gin.H{
		"items":            out,
		"comments_closed":  dyn.CommentsClosed,
		"comments_curated": dyn.CommentsCurated,
	})
}

// PostDynamicComment creates a comment on a user dynamic.
func (a *API) PostDynamicComment(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	did, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || did == 0 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	dyn, ok := loadUserDynamic(a, did)
	if !ok {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	if dyn.CommentsClosed {
		resp.Err(c, http.StatusForbidden, errcode.CodeCommentsClosed)
		return
	}
	var req commentPost
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	content := strings.TrimSpace(req.Content)
	if n := utf8.RuneCountInString(content); n < 1 || n > 1000 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	if a.rejectIfCommentSensitive(c, content) {
		return
	}
	level := 1
	parentID := req.ParentID
	if parentID != 0 {
		var parent model.DynamicComment
		if err := a.DB.First(&parent, parentID).Error; err != nil || parent.DynamicID != did {
			resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
			return
		}
		if dyn.CommentsCurated && !parent.Approved && uid != dyn.UserID {
			resp.Err(c, http.StatusForbidden, errcode.CodeParamError)
			return
		}
		if parent.Level >= 3 {
			level = 3
		} else {
			level = parent.Level + 1
		}
	}
	approved := !dyn.CommentsCurated || uid == dyn.UserID
	cm := model.DynamicComment{
		DynamicID:  did,
		UserID:     uid,
		ParentID:   parentID,
		Level:      level,
		Content:    content,
		Approved:   approved,
		IpLocation: a.resolveCommentIPLocation(c),
	}
	if err := a.DB.Create(&cm).Error; err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	if !approved {
		if err := a.DB.Model(&cm).UpdateColumn("approved", false).Error; err != nil {
			resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
			return
		}
		cm.Approved = false
	}
	if approved {
		_ = a.DB.Model(&model.UserDynamic{}).Where("id = ?", did).
			UpdateColumn("comment_count", gorm.Expr("comment_count + ?", 1)).Error
	}
	resp.JSON(c, http.StatusCreated, errcode.CodeSuccess, gin.H{
		"id":          cm.ID,
		"approved":    cm.Approved,
		"ip_location": iplocate.DisplayLabel(cm.IpLocation),
	})
}

// ApproveDynamicComment marks a comment visible under curated mode (dynamic owner only).
func (a *API) ApproveDynamicComment(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	cid, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || cid == 0 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	var cm model.DynamicComment
	if err := a.DB.First(&cm, cid).Error; err != nil {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	var dyn model.UserDynamic
	if err := a.DB.First(&dyn, cm.DynamicID).Error; err != nil {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	if dyn.UserID != uid {
		resp.Err(c, http.StatusForbidden, errcode.CodeForbidden)
		return
	}
	wasApproved := cm.Approved
	if !wasApproved && !dyn.CommentsCurated && !cm.CuratedIgnored {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	if err := a.DB.Model(&cm).Updates(map[string]interface{}{
		"approved":        true,
		"curated_ignored": false,
	}).Error; err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	if !wasApproved {
		_ = a.DB.Model(&model.UserDynamic{}).Where("id = ?", dyn.ID).
			UpdateColumn("comment_count", gorm.Expr("comment_count + ?", 1)).Error
	}
	resp.OK(c, gin.H{"id": cm.ID, "approved": true})
}

// IgnoreCuratedDynamicComment marks a pending curated dynamic comment as ignored by the owner.
func (a *API) IgnoreCuratedDynamicComment(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	cid, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || cid == 0 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	var cm model.DynamicComment
	if err := a.DB.First(&cm, cid).Error; err != nil {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	var dyn model.UserDynamic
	if err := a.DB.First(&dyn, cm.DynamicID).Error; err != nil {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	if dyn.UserID != uid {
		resp.Err(c, http.StatusForbidden, errcode.CodeForbidden)
		return
	}
	if cm.Approved {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	if cm.CuratedIgnored {
		resp.OK(c, gin.H{"id": cm.ID, "curated_ignored": true})
		return
	}
	if !dyn.CommentsCurated {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	if err := a.DB.Model(&cm).Update("curated_ignored", true).Error; err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	resp.OK(c, gin.H{"id": cm.ID, "curated_ignored": true})
}

// DeleteDynamicComment removes a dynamic comment subtree.
func (a *API) DeleteDynamicComment(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	cid, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || cid == 0 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	var cm model.DynamicComment
	if err := a.DB.First(&cm, cid).Error; err != nil {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	var dyn model.UserDynamic
	if err := a.DB.First(&dyn, cm.DynamicID).Error; err != nil {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	if cm.UserID != uid && dyn.UserID != uid {
		resp.Err(c, http.StatusForbidden, errcode.CodeForbidden)
		return
	}
	var all []model.DynamicComment
	if err := a.DB.Where("dynamic_id = ?", cm.DynamicID).Find(&all).Error; err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	children := map[uint64][]uint64{}
	for _, row := range all {
		children[row.ParentID] = append(children[row.ParentID], row.ID)
	}
	remove := map[uint64]struct{}{}
	var stack []uint64
	stack = append(stack, cm.ID)
	for len(stack) > 0 {
		id := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if _, ok := remove[id]; ok {
			continue
		}
		remove[id] = struct{}{}
		for _, ch := range children[id] {
			stack = append(stack, ch)
		}
	}
	ids := make([]uint64, 0, len(remove))
	for id := range remove {
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		resp.OK(c, gin.H{"deleted": 0})
		return
	}
	var approvedN int64
	if err := a.DB.Model(&model.DynamicComment{}).Where("id IN ? AND approved = ?", ids, true).Count(&approvedN).Error; err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	if err := a.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("comment_id IN ?", ids).Delete(&model.DynamicCommentLike{}).Error; err != nil {
			return err
		}
		if err := tx.Where("comment_id IN ?", ids).Delete(&model.DynamicCommentDislike{}).Error; err != nil {
			return err
		}
		if err := tx.Where("id IN ?", ids).Delete(&model.DynamicComment{}).Error; err != nil {
			return err
		}
		if approvedN > 0 {
			return tx.Model(&model.UserDynamic{}).Where("id = ?", cm.DynamicID).
				UpdateColumn("comment_count", gorm.Expr("GREATEST(comment_count - ?, 0)", approvedN)).Error
		}
		return nil
	}); err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	resp.OK(c, gin.H{"deleted": len(ids)})
}

// ToggleDynamicCommentLike toggles like on a dynamic comment.
func (a *API) ToggleDynamicCommentLike(c *gin.Context) {
	a.toggleDynamicCommentReaction(c, true)
}

// ToggleDynamicCommentDislike toggles dislike on a dynamic comment.
func (a *API) ToggleDynamicCommentDislike(c *gin.Context) {
	a.toggleDynamicCommentReaction(c, false)
}

func (a *API) toggleDynamicCommentReaction(c *gin.Context, like bool) {
	uid, ok := middleware.UserID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	cid, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || cid == 0 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	var cm model.DynamicComment
	if err := a.DB.First(&cm, cid).Error; err != nil {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	if like {
		var existing model.DynamicCommentDislike
		if res := a.DB.Where("user_id = ? AND comment_id = ?", uid, cid).Limit(1).Find(&existing); res.Error == nil && res.RowsAffected > 0 {
			_ = a.DB.Delete(&existing).Error
		}
		var lk model.DynamicCommentLike
		res := a.DB.Where("user_id = ? AND comment_id = ?", uid, cid).Limit(1).Find(&lk)
		if res.Error != nil {
			resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
			return
		}
		if res.RowsAffected == 0 {
			if err := a.DB.Create(&model.DynamicCommentLike{UserID: uid, CommentID: cid}).Error; err != nil {
				resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
				return
			}
			_ = a.DB.Model(&model.DynamicComment{}).Where("id = ?", cid).
				UpdateColumn("like_count", gorm.Expr("like_count + ?", 1)).Error
			resp.OK(c, gin.H{"liked": true})
			return
		}
		if err := a.DB.Delete(&lk).Error; err != nil {
			resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
			return
		}
		_ = a.DB.Model(&model.DynamicComment{}).Where("id = ?", cid).
			UpdateColumn("like_count", gorm.Expr("GREATEST(like_count - ?, 0)", 1)).Error
		resp.OK(c, gin.H{"liked": false})
		return
	}
	var existing model.DynamicCommentLike
	if res := a.DB.Where("user_id = ? AND comment_id = ?", uid, cid).Limit(1).Find(&existing); res.Error == nil && res.RowsAffected > 0 {
		_ = a.DB.Delete(&existing).Error
		_ = a.DB.Model(&model.DynamicComment{}).Where("id = ?", cid).
			UpdateColumn("like_count", gorm.Expr("GREATEST(like_count - ?, 0)", 1)).Error
	}
	var dk model.DynamicCommentDislike
	res := a.DB.Where("user_id = ? AND comment_id = ?", uid, cid).Limit(1).Find(&dk)
	if res.Error != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	if res.RowsAffected == 0 {
		if err := a.DB.Create(&model.DynamicCommentDislike{UserID: uid, CommentID: cid}).Error; err != nil {
			resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
			return
		}
		resp.OK(c, gin.H{"disliked": true})
		return
	}
	if err := a.DB.Delete(&dk).Error; err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	resp.OK(c, gin.H{"disliked": false})
}
