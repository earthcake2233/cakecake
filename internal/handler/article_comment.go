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

// ListArticleComments returns flat comments for an article.
func (a *API) ListArticleComments(c *gin.Context) {
	aid, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || aid == 0 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	art, ok := loadPublishedArticle(a, aid)
	if !ok {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	if art.CommentsClosed {
		resp.OK(c, gin.H{
			"items":            []gin.H{},
			"comments_closed":  true,
			"comments_curated": art.CommentsCurated,
		})
		return
	}
	commentQ := a.DB.Where("article_id = ?", aid)
	if art.CommentsCurated {
		commentQ = commentQ.Where("approved = ?", true)
	}
	var list []model.ArticleComment
	if err := commentQ.Order("id ASC").Find(&list).Error; err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	var viewer uint64
	if uid, ok := middleware.UserID(c); ok {
		viewer = uid
	}
	out := a.articleCommentsToJSON(list, art.UserID, viewer)
	resp.OK(c, gin.H{
		"items":            out,
		"comments_closed":  art.CommentsClosed,
		"comments_curated": art.CommentsCurated,
	})
}

func (a *API) articleCommentsToJSON(list []model.ArticleComment, authorID, viewer uint64) []gin.H {
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
		var likes []model.ArticleCommentLike
		_ = a.DB.Where("user_id = ? AND comment_id IN ?", viewer, ids).Find(&likes).Error
		for _, lk := range likes {
			likedByViewer[lk.CommentID] = true
		}
		var dislikes []model.ArticleCommentDislike
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

// PostArticleComment creates a comment on an article.
func (a *API) PostArticleComment(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	aid, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || aid == 0 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	var art model.Article
	if err := a.DB.First(&art, aid).Error; err != nil || art.Status != articleStatusPublished {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	if art.CommentsClosed {
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
		var parent model.ArticleComment
		if err := a.DB.First(&parent, parentID).Error; err != nil || parent.ArticleID != aid {
			resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
			return
		}
		if art.CommentsCurated && !parent.Approved && uid != art.UserID {
			resp.Err(c, http.StatusForbidden, errcode.CodeParamError)
			return
		}
		if parent.Level >= 3 {
			level = 3
		} else {
			level = parent.Level + 1
		}
	}
	approved := !art.CommentsCurated || uid == art.UserID
	cm := model.ArticleComment{
		ArticleID:  aid,
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
		_ = a.DB.Model(&model.Article{}).Where("id = ?", aid).
			UpdateColumn("comment_count", gorm.Expr("comment_count + ?", 1)).Error
	}
	if parentID == 0 && uid != art.UserID {
		a.notifyUploaderOnArticleComment(&art, uid, &cm)
	} else if parentID != 0 {
		a.notifyParentOnArticleReply(aid, uid, &cm, parentID)
	}
	resp.JSON(c, http.StatusCreated, errcode.CodeSuccess, gin.H{
		"id":          cm.ID,
		"approved":    cm.Approved,
		"ip_location": iplocate.DisplayLabel(cm.IpLocation),
	})
}

// ApproveArticleComment marks a comment visible under curated mode (article owner only).
func (a *API) ApproveArticleComment(c *gin.Context) {
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
	var cm model.ArticleComment
	if err := a.DB.First(&cm, cid).Error; err != nil {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	var art model.Article
	if err := a.DB.First(&art, cm.ArticleID).Error; err != nil {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	if art.UserID != uid {
		resp.Err(c, http.StatusForbidden, errcode.CodeForbidden)
		return
	}
	wasApproved := cm.Approved
	if !wasApproved && !art.CommentsCurated && !cm.CuratedIgnored {
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
		_ = a.DB.Model(&model.Article{}).Where("id = ?", art.ID).
			UpdateColumn("comment_count", gorm.Expr("comment_count + ?", 1)).Error
	}
	resp.OK(c, gin.H{"id": cm.ID, "approved": true})
}

// IgnoreCuratedArticleComment marks a pending curated article comment as ignored by the owner.
func (a *API) IgnoreCuratedArticleComment(c *gin.Context) {
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
	var cm model.ArticleComment
	if err := a.DB.First(&cm, cid).Error; err != nil {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	var art model.Article
	if err := a.DB.First(&art, cm.ArticleID).Error; err != nil {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	if art.UserID != uid {
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
	if !art.CommentsCurated {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	if err := a.DB.Model(&cm).Update("curated_ignored", true).Error; err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	resp.OK(c, gin.H{"id": cm.ID, "curated_ignored": true})
}

// ToggleArticleCommentLike toggles like on an article comment.
func (a *API) ToggleArticleCommentLike(c *gin.Context) {
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
	var cm model.ArticleComment
	if err := a.DB.First(&cm, cid).Error; err != nil {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	var like model.ArticleCommentLike
	res := a.DB.Where("user_id = ? AND comment_id = ?", uid, cid).Limit(1).Find(&like)
	if res.Error != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	if res.RowsAffected == 0 {
		_ = a.DB.Where("user_id = ? AND comment_id = ?", uid, cid).Delete(&model.ArticleCommentDislike{}).Error
		if err := a.DB.Create(&model.ArticleCommentLike{UserID: uid, CommentID: cid}).Error; err != nil {
			resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
			return
		}
		_ = a.DB.Model(&cm).UpdateColumn("like_count", gorm.Expr("like_count + ?", 1)).Error
		var liker model.User
		if err := a.DB.First(&liker, uid).Error; err == nil && cm.UserID != uid {
			a.upsertArticleCommentLikeNotification(cm, model.DisplayUsername(&liker))
		}
		resp.OK(c, gin.H{"liked": true})
		return
	}
	_ = a.DB.Delete(&like).Error
	_ = a.DB.Model(&cm).UpdateColumn("like_count", gorm.Expr("GREATEST(like_count - ?, 0)", 1)).Error
	resp.OK(c, gin.H{"liked": false})
}

// ToggleArticleCommentDislike toggles dislike on an article comment.
func (a *API) ToggleArticleCommentDislike(c *gin.Context) {
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
	var cm model.ArticleComment
	if err := a.DB.First(&cm, cid).Error; err != nil {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	var dislike model.ArticleCommentDislike
	res := a.DB.Where("user_id = ? AND comment_id = ?", uid, cid).Limit(1).Find(&dislike)
	if res.Error != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	if res.RowsAffected == 0 {
		_ = a.DB.Where("user_id = ? AND comment_id = ?", uid, cid).Delete(&model.ArticleCommentLike{}).Error
		if err := a.DB.Create(&model.ArticleCommentDislike{UserID: uid, CommentID: cid}).Error; err != nil {
			resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
			return
		}
		_ = a.DB.Model(&cm).UpdateColumn("like_count", gorm.Expr("GREATEST(like_count - ?, 0)", 1)).Error
		resp.OK(c, gin.H{"disliked": true})
		return
	}
	_ = a.DB.Delete(&dislike).Error
	resp.OK(c, gin.H{"disliked": false})
}

func collectArticleDescendantIDs(db *gorm.DB, root uint64) ([]uint64, error) {
	var all []uint64
	queue := []uint64{root}
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		all = append(all, id)
		var children []uint64
		if err := db.Model(&model.ArticleComment{}).Where("parent_id = ?", id).Pluck("id", &children).Error; err != nil {
			return nil, err
		}
		queue = append(queue, children...)
	}
	return all, nil
}

// PinArticleComment toggles the pinned root comment for an article (owner only, root comments only).
func (a *API) PinArticleComment(c *gin.Context) {
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
	var cm model.ArticleComment
	if err := a.DB.First(&cm, cid).Error; err != nil {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	var art model.Article
	if err := a.DB.First(&art, cm.ArticleID).Error; err != nil || art.Status != articleStatusPublished {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	if art.UserID != uid {
		resp.Err(c, http.StatusForbidden, errcode.CodeForbidden)
		return
	}
	if cm.ParentID != 0 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	wasPinned := cm.Pinned
	tx := a.DB.Begin()
	if wasPinned {
		if err := tx.Model(&cm).Update("pinned", false).Error; err != nil {
			tx.Rollback()
			resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
			return
		}
	} else {
		if err := tx.Model(&model.ArticleComment{}).
			Where("article_id = ? AND parent_id = ?", cm.ArticleID, 0).
			Update("pinned", false).Error; err != nil {
			tx.Rollback()
			resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
			return
		}
		if err := tx.Model(&cm).Update("pinned", true).Error; err != nil {
			tx.Rollback()
			resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
			return
		}
	}
	if err := tx.Commit().Error; err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	resp.OK(c, gin.H{"pinned": !wasPinned})
}

// DeleteArticleComment removes a comment subtree.
func (a *API) DeleteArticleComment(c *gin.Context) {
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
	var cm model.ArticleComment
	if err := a.DB.First(&cm, cid).Error; err != nil {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	var art model.Article
	if err := a.DB.First(&art, cm.ArticleID).Error; err != nil {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	if cm.UserID != uid && art.UserID != uid {
		resp.Err(c, http.StatusForbidden, errcode.CodeForbidden)
		return
	}
	ids, err := collectArticleDescendantIDs(a.DB, cid)
	if err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	var approvedN int64
	if err := a.DB.Model(&model.ArticleComment{}).Where("id IN ? AND approved = ?", ids, true).Count(&approvedN).Error; err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	if err := a.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("id IN ?", ids).Delete(&model.ArticleComment{}).Error; err != nil {
			return err
		}
		if err := purgeReplyInboxNotifications(tx, ids); err != nil {
			return err
		}
		if approvedN > 0 {
			return tx.Model(&model.Article{}).Where("id = ?", art.ID).
				UpdateColumn("comment_count", gorm.Expr("GREATEST(comment_count - ?, 0)", approvedN)).Error
		}
		return nil
	}); err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	resp.OK(c, nil)
}

// GetMyArticle returns an article for editing (owner only).
func (a *API) GetMyArticle(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	var art model.Article
	if err := a.DB.First(&art, id).Error; err != nil || art.UserID != uid {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	var author model.User
	_ = a.DB.First(&author, uid).Error
	eng := articleEngagementByViewer(a.DB, uid, []uint64{id})[id]
	resp.OK(c, articleDetailPayload(a, &art, &author, eng, uid))
}
