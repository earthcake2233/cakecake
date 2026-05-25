package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"minibili/internal/errcode"
	"minibili/internal/middleware"
	"minibili/internal/model"
	"minibili/internal/pkg/iplocate"
	"minibili/internal/pkg/netutil"
	"minibili/internal/pkg/resp"
	"minibili/internal/pkg/userlevel"
)

type commentPost struct {
	Content  string `json:"content"`
	ParentID uint64 `json:"parent_id"`
}

// ListComments returns flat comments for a video (F7, F8).
func (a *API) ListComments(c *gin.Context) {
	vid, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || vid == 0 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	var v model.Video
	if err := a.DB.First(&v, vid).Error; err != nil || v.Status != "published" {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	if v.CommentsClosed {
		resp.OK(c, gin.H{"items": []gin.H{}, "comments_closed": true, "comments_curated": v.CommentsCurated})
		return
	}
	commentQ := a.DB.Where("video_id = ?", vid)
	if v.CommentsCurated {
		commentQ = commentQ.Where("approved = ?", true)
	}
	var list []model.Comment
	if err := commentQ.Order("id ASC").Find(&list).Error; err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
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
	if uid, ok := middleware.UserID(c); ok && uid > 0 && len(list) > 0 {
		ids := make([]uint64, len(list))
		for i := range list {
			ids[i] = list[i].ID
		}
		var likes []model.CommentLike
		_ = a.DB.Where("user_id = ? AND comment_id IN ?", uid, ids).Find(&likes).Error
		for _, lk := range likes {
			likedByViewer[lk.CommentID] = true
		}
		var dislikes []model.CommentDislike
		_ = a.DB.Where("user_id = ? AND comment_id IN ?", uid, ids).Find(&dislikes).Error
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
			"is_by_uploader": cm.UserID == v.UserID,
			"ip_location":    iplocate.DisplayLabel(cm.IpLocation),
		})
	}
	resp.OK(c, gin.H{
		"items":            out,
		"comments_curated": v.CommentsCurated,
	})
}

// PostComment creates a comment or reply (F7, F8).
func (a *API) PostComment(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	vid, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || vid == 0 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	var v model.Video
	if err := a.DB.First(&v, vid).Error; err != nil || v.Status != "published" {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	if v.CommentsClosed {
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
		var parent model.Comment
		if err := a.DB.First(&parent, parentID).Error; err != nil || parent.VideoID != vid {
			resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
			return
		}
		if v.CommentsCurated && !parent.Approved && uid != v.UserID {
			resp.Err(c, http.StatusForbidden, errcode.CodeParamError)
			return
		}
		if parent.Level >= 3 {
			level = 3
		} else {
			level = parent.Level + 1
		}
	}
	approved := !v.CommentsCurated || uid == v.UserID
	cm := model.Comment{
		VideoID:    vid,
		UserID:     uid,
		ParentID:   parentID,
		Level:      level,
		Content:    content,
		LikeCount:  0,
		Approved:   approved,
		IpLocation: a.resolveCommentIPLocation(c),
	}
	if err := a.DB.Create(&cm).Error; err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	// GORM/DB default 可能把 false 写成 true，强制落库待精选状态。
	if !approved {
		if err := a.DB.Model(&cm).UpdateColumn("approved", false).Error; err != nil {
			resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
			return
		}
		cm.Approved = false
	}
	if approved {
		_ = a.DB.Model(&model.Video{}).Where("id = ?", vid).UpdateColumn("comment_count", gorm.Expr("comment_count + ?", 1)).Error
	}
	if parentID == 0 && uid != v.UserID {
		a.notifyUploaderOnVideoComment(&v, uid, &cm)
	}
	if parentID != 0 {
		a.notifyParentOnReply(vid, uid, &cm, parentID)
	}
	resp.JSON(c, http.StatusCreated, errcode.CodeSuccess, gin.H{
		"id":          cm.ID,
		"approved":    cm.Approved,
		"ip_location": iplocate.DisplayLabel(cm.IpLocation),
	})
}

// ApproveComment marks a comment visible under curated mode (video owner only).
func (a *API) ApproveComment(c *gin.Context) {
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
	var cm model.Comment
	if err := a.DB.First(&cm, cid).Error; err != nil {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	var v model.Video
	if err := a.DB.First(&v, cm.VideoID).Error; err != nil {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	if v.UserID != uid {
		resp.Err(c, http.StatusForbidden, errcode.CodeForbidden)
		return
	}
	wasApproved := cm.Approved
	// 待精选或已忽略均可再次精选（不要求稿件仍开启精选）
	if !wasApproved && !v.CommentsCurated && !cm.CuratedIgnored {
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
		_ = a.DB.Model(&model.Video{}).Where("id = ?", v.ID).
			UpdateColumn("comment_count", gorm.Expr("comment_count + ?", 1)).Error
	}
	resp.OK(c, gin.H{"id": cm.ID, "approved": true})
}

// IgnoreCuratedComment marks a pending curated comment as ignored by the video owner.
func (a *API) IgnoreCuratedComment(c *gin.Context) {
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
	var cm model.Comment
	if err := a.DB.First(&cm, cid).Error; err != nil {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	var v model.Video
	if err := a.DB.First(&v, cm.VideoID).Error; err != nil {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	if v.UserID != uid {
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
	if !v.CommentsCurated {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	if err := a.DB.Model(&cm).Update("curated_ignored", true).Error; err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	resp.OK(c, gin.H{"id": cm.ID, "curated_ignored": true})
}

func (a *API) resolveCommentIPLocation(c *gin.Context) string {
	if a.IPLocate == nil {
		return ""
	}
	ip := netutil.ClientIP(c)
	if ip == "" {
		return ""
	}
	lookupIP := ip
	if netutil.IsLoopbackOrPrivate(ip) {
		if a.Cfg == nil || a.Cfg.AppEnv != "development" {
			return ""
		}
		fallback := strings.TrimSpace(a.Cfg.IP2RegionDevClientIP)
		if fallback == "" || netutil.IsLoopbackOrPrivate(fallback) {
			return ""
		}
		lookupIP = fallback
	}
	return strings.TrimSpace(a.IPLocate.Province(lookupIP))
}

// DeleteComment removes a subtree (F7, S-008).
func (a *API) DeleteComment(c *gin.Context) {
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
	var cm model.Comment
	if err := a.DB.First(&cm, cid).Error; err != nil {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	var v model.Video
	if err := a.DB.First(&v, cm.VideoID).Error; err != nil {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	if cm.UserID != uid && v.UserID != uid {
		resp.Err(c, http.StatusForbidden, errcode.CodeForbidden)
		return
	}
	ids, err := collectDescendantIDs(a.DB, cid)
	if err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	var approvedN int64
	if err := a.DB.Model(&model.Comment{}).Where("id IN ? AND approved = ?", ids, true).Count(&approvedN).Error; err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	tx := a.DB.Begin()
	if err := tx.Where("id IN ?", ids).Delete(&model.Comment{}).Error; err != nil {
		tx.Rollback()
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	if err := purgeReplyInboxNotifications(tx, ids); err != nil {
		tx.Rollback()
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	if approvedN > 0 {
		if err := tx.Model(&model.Video{}).Where("id = ?", v.ID).UpdateColumn("comment_count", gorm.Expr("GREATEST(comment_count - ?, 0)", approvedN)).Error; err != nil {
			tx.Rollback()
			resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
			return
		}
	}
	if err := tx.Commit().Error; err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	a.Hub.BroadcastJSON(v.ID, gin.H{"type": "comment_deleted", "comment_id": strconv.FormatUint(cid, 10)})
	resp.OK(c, nil)
}

func collectDescendantIDs(db *gorm.DB, root uint64) ([]uint64, error) {
	var all []uint64
	queue := []uint64{root}
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		all = append(all, id)
		var children []uint64
		if err := db.Model(&model.Comment{}).Where("parent_id = ?", id).Pluck("id", &children).Error; err != nil {
			return nil, err
		}
		queue = append(queue, children...)
	}
	return all, nil
}

// PinComment toggles the pinned root comment for a video (owner only, root comments only).
func (a *API) PinComment(c *gin.Context) {
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
	var cm model.Comment
	if err := a.DB.First(&cm, cid).Error; err != nil {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	var v model.Video
	if err := a.DB.First(&v, cm.VideoID).Error; err != nil || v.Status != "published" {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	if v.UserID != uid {
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
		if err := tx.Model(&model.Comment{}).Where("video_id = ? AND parent_id = ?", cm.VideoID, 0).Update("pinned", false).Error; err != nil {
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

// ToggleLike toggles like on a comment (F9, S-010).
func (a *API) ToggleLike(c *gin.Context) {
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
	var cm model.Comment
	if err := a.DB.First(&cm, cid).Error; err != nil {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	var like model.CommentLike
	q := a.DB.Where("user_id = ? AND comment_id = ?", uid, cid).Limit(1)
	res := q.Find(&like)
	if res.Error != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	if res.RowsAffected == 0 {
		if _, err := a.clearCommentDislike(uid, cid); err != nil {
			resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
			return
		}
		like = model.CommentLike{UserID: uid, CommentID: cid}
		if err := a.DB.Create(&like).Error; err != nil {
			resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
			return
		}
		_ = a.DB.Model(&cm).UpdateColumn("like_count", gorm.Expr("like_count + ?", 1)).Error
		var liker model.User
		_ = a.DB.First(&liker, uid).Error
		if cm.UserID != uid {
			a.upsertLikeNotification(cm, model.DisplayUsername(&liker))
		}
		resp.OK(c, gin.H{"liked": true})
		return
	}
	if err := a.DB.Delete(&like).Error; err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	_ = a.DB.Model(&cm).UpdateColumn("like_count", gorm.Expr("GREATEST(like_count - ?, 0)", 1)).Error
	resp.OK(c, gin.H{"liked": false})
}

// ToggleDislike toggles dislike on a comment (no public count; mutually exclusive with like).
func (a *API) ToggleDislike(c *gin.Context) {
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
	var cm model.Comment
	if err := a.DB.First(&cm, cid).Error; err != nil {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	var dislike model.CommentDislike
	q := a.DB.Where("user_id = ? AND comment_id = ?", uid, cid).Limit(1)
	res := q.Find(&dislike)
	if res.Error != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	if res.RowsAffected == 0 {
		if _, err := a.clearCommentLike(uid, cid, &cm); err != nil {
			resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
			return
		}
		dislike = model.CommentDislike{UserID: uid, CommentID: cid}
		if err := a.DB.Create(&dislike).Error; err != nil {
			resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
			return
		}
		resp.OK(c, gin.H{"disliked": true})
		return
	}
	if err := a.DB.Delete(&dislike).Error; err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	resp.OK(c, gin.H{"disliked": false})
}

func (a *API) clearCommentDislike(uid, cid uint64) (bool, error) {
	var dk model.CommentDislike
	res := a.DB.Where("user_id = ? AND comment_id = ?", uid, cid).Limit(1).Find(&dk)
	if res.Error != nil {
		return false, res.Error
	}
	if res.RowsAffected == 0 {
		return false, nil
	}
	if err := a.DB.Delete(&dk).Error; err != nil {
		return false, err
	}
	return true, nil
}

func (a *API) clearCommentLike(uid, cid uint64, cm *model.Comment) (bool, error) {
	var like model.CommentLike
	res := a.DB.Where("user_id = ? AND comment_id = ?", uid, cid).Limit(1).Find(&like)
	if res.Error != nil {
		return false, res.Error
	}
	if res.RowsAffected == 0 {
		return false, nil
	}
	if err := a.DB.Delete(&like).Error; err != nil {
		return false, err
	}
	if cm != nil {
		_ = a.DB.Model(cm).UpdateColumn("like_count", gorm.Expr("GREATEST(like_count - ?, 0)", 1)).Error
	}
	return true, nil
}

func (a *API) upsertLikeNotification(cm model.Comment, likerName string) {
	ownerID := cm.UserID
	var muted int64
	_ = a.DB.Model(&model.LikeNotifMute{}).
		Where("recipient_id = ? AND comment_id = ?", ownerID, cm.ID).
		Count(&muted).Error
	if muted > 0 {
		return
	}
	var n model.Notification
	nq := a.DB.Where("recipient_id = ? AND type = ? AND related_id = ? AND is_read = ?",
		ownerID, "like_aggregation", cm.ID, false).Limit(1)
	nres := nq.Find(&n)
	preview := cm.Content
	r := []rune(preview)
	if len(r) > 15 {
		preview = string(r[:15])
	}
	if nres.Error != nil {
		return
	}
	if nres.RowsAffected == 0 {
		names, _ := json.Marshal([]string{likerName})
		n = model.Notification{
			RecipientID:     ownerID,
			Type:            "like_aggregation",
			RelatedID:       cm.ID,
			SenderNamesJSON: string(names),
			TotalLikes:      1,
			CommentPreview:  preview,
			IsRead:          false,
		}
		_ = a.DB.Create(&n).Error
		return
	}
	var names []string
	_ = json.Unmarshal([]byte(n.SenderNamesJSON), &names)
	for _, x := range names {
		if x == likerName {
			return
		}
	}
	names = append(names, likerName)
	b, _ := json.Marshal(names)
	n.SenderNamesJSON = string(b)
	n.TotalLikes++
	n.CommentPreview = preview
	_ = a.DB.Save(&n).Error
}

type replyNotifPayload struct {
	SenderID             uint64 `json:"sender_id"`
	SenderUsername       string `json:"sender_username"`
	SenderAvatarURL      string `json:"sender_avatar_url"`
	ReplyCommentID       uint64 `json:"reply_comment_id"`
	ReplyContent         string `json:"reply_content"`
	ParentCommentID      uint64 `json:"parent_comment_id"`
	ParentContentPreview string `json:"parent_content_preview"`
	VideoID              uint64 `json:"video_id"`
}

type videoCommentNotifPayload struct {
	SenderID        uint64 `json:"sender_id"`
	SenderUsername  string `json:"sender_username"`
	SenderAvatarURL string `json:"sender_avatar_url"`
	CommentID       uint64 `json:"comment_id"`
	CommentContent  string `json:"comment_content"`
	VideoID         uint64 `json:"video_id"`
	VideoTitle      string `json:"video_title"`
	CoverURL        string `json:"cover_url"`
}

type articleCommentNotifPayload struct {
	SenderID        uint64 `json:"sender_id"`
	SenderUsername  string `json:"sender_username"`
	SenderAvatarURL string `json:"sender_avatar_url"`
	CommentID       uint64 `json:"comment_id"`
	CommentContent  string `json:"comment_content"`
	ArticleID       uint64 `json:"article_id"`
	ArticleTitle    string `json:"article_title"`
	CoverURL        string `json:"cover_url"`
}

type articleReplyNotifPayload struct {
	SenderID             uint64 `json:"sender_id"`
	SenderUsername       string `json:"sender_username"`
	SenderAvatarURL      string `json:"sender_avatar_url"`
	ReplyCommentID       uint64 `json:"reply_comment_id"`
	ReplyContent         string `json:"reply_content"`
	ParentCommentID      uint64 `json:"parent_comment_id"`
	ParentContentPreview string `json:"parent_content_preview"`
	ArticleID            uint64 `json:"article_id"`
}

// replyInboxNotificationTypes are shown under the「回复我的」inbox tab.
var replyInboxNotificationTypes = []string{
	"reply_received",
	"video_comment_received",
	"article_comment_received",
	"article_reply_received",
}

type replyInboxTarget struct {
	CommentID uint64
	VideoID   uint64
	ArticleID uint64
	IsArticle bool
}

func isReplyInboxType(t string) bool {
	for _, x := range replyInboxNotificationTypes {
		if x == t {
			return true
		}
	}
	return false
}

func purgeReplyInboxNotifications(tx *gorm.DB, relatedIDs []uint64) error {
	if len(relatedIDs) == 0 {
		return nil
	}
	types := append([]string(nil), replyInboxNotificationTypes...)
	types = append(types, "like_aggregation")
	return tx.Where("type IN ? AND related_id IN ?", types, relatedIDs).
		Delete(&model.Notification{}).Error
}

// resolveReplyInboxTarget maps an inbox row to the live comment + video/article for like/reply actions.
func (a *API) resolveReplyInboxTarget(n *model.Notification) (replyInboxTarget, bool) {
	var t replyInboxTarget
	cid := n.RelatedID

	switch n.Type {
	case "video_comment_received":
		var pl videoCommentNotifPayload
		_ = json.Unmarshal([]byte(n.PayloadJSON), &pl)
		if pl.CommentID != 0 {
			cid = pl.CommentID
		}
		if cid == 0 {
			return t, false
		}
		var cm model.Comment
		if err := a.DB.First(&cm, cid).Error; err != nil {
			return t, false
		}
		t.CommentID = cm.ID
		t.VideoID = cm.VideoID
		if pl.VideoID != 0 {
			t.VideoID = pl.VideoID
		}
		return t, true

	case "article_comment_received":
		var pl articleCommentNotifPayload
		_ = json.Unmarshal([]byte(n.PayloadJSON), &pl)
		if pl.CommentID != 0 {
			cid = pl.CommentID
		}
		if cid == 0 {
			return t, false
		}
		var acm model.ArticleComment
		if err := a.DB.First(&acm, cid).Error; err != nil {
			return t, false
		}
		t.CommentID = acm.ID
		t.ArticleID = acm.ArticleID
		t.IsArticle = true
		if pl.ArticleID != 0 {
			t.ArticleID = pl.ArticleID
		}
		return t, true

	case "article_reply_received":
		var pl articleReplyNotifPayload
		_ = json.Unmarshal([]byte(n.PayloadJSON), &pl)
		if pl.ReplyCommentID != 0 {
			cid = pl.ReplyCommentID
		}
		if cid == 0 {
			return t, false
		}
		var acm model.ArticleComment
		if err := a.DB.First(&acm, cid).Error; err != nil {
			return t, false
		}
		t.CommentID = acm.ID
		t.ArticleID = acm.ArticleID
		t.IsArticle = true
		if pl.ArticleID != 0 {
			t.ArticleID = pl.ArticleID
		}
		return t, true

	case "reply_received":
		var pl replyNotifPayload
		_ = json.Unmarshal([]byte(n.PayloadJSON), &pl)
		if pl.ReplyCommentID != 0 {
			cid = pl.ReplyCommentID
		}
		if cid == 0 {
			return t, false
		}
		var cm model.Comment
		if err := a.DB.First(&cm, cid).Error; err == nil {
			t.CommentID = cm.ID
			t.VideoID = cm.VideoID
			if pl.VideoID != 0 {
				t.VideoID = pl.VideoID
			}
			return t, true
		}
		var acm model.ArticleComment
		if err := a.DB.First(&acm, cid).Error; err == nil {
			t.CommentID = acm.ID
			t.ArticleID = acm.ArticleID
			t.IsArticle = true
			return t, true
		}
		return t, false
	}
	return t, false
}

func loadRecipientReplyInboxNotification(a *API, c *gin.Context, uid, nid uint64) (*model.Notification, bool) {
	if nid == 0 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return nil, false
	}
	var n model.Notification
	if err := a.DB.Where("id = ? AND recipient_id = ?", nid, uid).First(&n).Error; err != nil {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return nil, false
	}
	if !isReplyInboxType(n.Type) {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return nil, false
	}
	return &n, true
}

// notifyUploaderOnVideoComment notifies the video owner when someone posts a top-level comment (「对我的视频发表了评论」).
func (a *API) notifyUploaderOnVideoComment(v *model.Video, commenterUID uint64, cm *model.Comment) {
	if v == nil || v.UserID == 0 || v.UserID == commenterUID {
		return
	}
	var u model.User
	if err := a.DB.First(&u, commenterUID).Error; err != nil {
		return
	}
	title := strings.TrimSpace(v.Title)
	tr := []rune(title)
	if len(tr) > 80 {
		title = string(tr[:80])
	}
	pl := videoCommentNotifPayload{
		SenderID:        commenterUID,
		SenderUsername:  model.DisplayUsername(&u),
		SenderAvatarURL: uploaderAvatarForAPI(&u),
		CommentID:       cm.ID,
		CommentContent:  cm.Content,
		VideoID:         v.ID,
		VideoTitle:      title,
		CoverURL:        strings.TrimSpace(v.CoverURL),
	}
	pb, err := json.Marshal(pl)
	if err != nil {
		return
	}
	prevShort := pl.CommentContent
	sr := []rune(prevShort)
	if len(sr) > 32 {
		prevShort = string(sr[:32])
	}
	nm, _ := json.Marshal([]string{pl.SenderUsername})
	n := model.Notification{
		RecipientID:     v.UserID,
		Type:            "video_comment_received",
		RelatedID:       cm.ID,
		SenderNamesJSON: string(nm),
		TotalLikes:      0,
		CommentPreview:  prevShort,
		PayloadJSON:     string(pb),
		IsRead:          false,
	}
	if err := a.DB.Create(&n).Error; err != nil && a.Log != nil {
		a.Log.Warn("create video_comment_received failed", zap.Error(err), zap.Uint64("comment_id", cm.ID))
	}
}

// notifyUploaderOnArticleComment notifies the article owner on a top-level comment (「对我的文章发表了评论」).
func (a *API) notifyUploaderOnArticleComment(art *model.Article, commenterUID uint64, cm *model.ArticleComment) {
	if art == nil || art.UserID == 0 || art.UserID == commenterUID {
		return
	}
	var u model.User
	if err := a.DB.First(&u, commenterUID).Error; err != nil {
		return
	}
	title := strings.TrimSpace(art.Title)
	tr := []rune(title)
	if len(tr) > 80 {
		title = string(tr[:80])
	}
	pl := articleCommentNotifPayload{
		SenderID:        commenterUID,
		SenderUsername:  model.DisplayUsername(&u),
		SenderAvatarURL: uploaderAvatarForAPI(&u),
		CommentID:       cm.ID,
		CommentContent:  cm.Content,
		ArticleID:       art.ID,
		ArticleTitle:    title,
		CoverURL:        strings.TrimSpace(art.CoverURL),
	}
	pb, err := json.Marshal(pl)
	if err != nil {
		return
	}
	prevShort := pl.CommentContent
	sr := []rune(prevShort)
	if len(sr) > 32 {
		prevShort = string(sr[:32])
	}
	nm, _ := json.Marshal([]string{pl.SenderUsername})
	n := model.Notification{
		RecipientID:     art.UserID,
		Type:            "article_comment_received",
		RelatedID:       cm.ID,
		SenderNamesJSON: string(nm),
		TotalLikes:      0,
		CommentPreview:  prevShort,
		PayloadJSON:     string(pb),
		IsRead:          false,
	}
	if err := a.DB.Create(&n).Error; err != nil && a.Log != nil {
		a.Log.Warn("create article_comment_received failed", zap.Error(err), zap.Uint64("comment_id", cm.ID))
	}
}

// notifyParentOnArticleReply notifies the parent comment author on an article comment reply.
func (a *API) notifyParentOnArticleReply(articleID, replierUID uint64, reply *model.ArticleComment, parentID uint64) {
	var parent model.ArticleComment
	if err := a.DB.First(&parent, parentID).Error; err != nil {
		return
	}
	if parent.UserID == replierUID {
		return
	}
	var u model.User
	if err := a.DB.First(&u, replierUID).Error; err != nil {
		return
	}
	preview := strings.TrimSpace(parent.Content)
	runes := []rune(preview)
	if len(runes) > 120 {
		preview = string(runes[:120])
	}
	pl := articleReplyNotifPayload{
		SenderID:             replierUID,
		SenderUsername:       model.DisplayUsername(&u),
		SenderAvatarURL:      uploaderAvatarForAPI(&u),
		ReplyCommentID:       reply.ID,
		ReplyContent:         reply.Content,
		ParentCommentID:      parentID,
		ParentContentPreview: preview,
		ArticleID:            articleID,
	}
	pb, err := json.Marshal(pl)
	if err != nil {
		return
	}
	prevShort := preview
	sr := []rune(prevShort)
	if len(sr) > 32 {
		prevShort = string(sr[:32])
	}
	nm, _ := json.Marshal([]string{pl.SenderUsername})
	n := model.Notification{
		RecipientID:     parent.UserID,
		Type:            "article_reply_received",
		RelatedID:       reply.ID,
		SenderNamesJSON: string(nm),
		TotalLikes:      0,
		CommentPreview:  prevShort,
		PayloadJSON:     string(pb),
		IsRead:          false,
	}
	if err := a.DB.Create(&n).Error; err != nil && a.Log != nil {
		a.Log.Warn("create article_reply_received failed", zap.Error(err), zap.Uint64("reply_id", reply.ID))
	}
}

func (a *API) upsertArticleCommentLikeNotification(cm model.ArticleComment, likerName string) {
	ownerID := cm.UserID
	var muted int64
	_ = a.DB.Model(&model.LikeNotifMute{}).
		Where("recipient_id = ? AND comment_id = ?", ownerID, cm.ID).
		Count(&muted).Error
	if muted > 0 {
		return
	}
	var n model.Notification
	nq := a.DB.Where("recipient_id = ? AND type = ? AND related_id = ? AND is_read = ?",
		ownerID, "like_aggregation", cm.ID, false).Limit(1)
	nres := nq.Find(&n)
	preview := cm.Content
	r := []rune(preview)
	if len(r) > 15 {
		preview = string(r[:15])
	}
	if nres.Error != nil {
		return
	}
	artTitle := ""
	artCover := ""
	var art model.Article
	if err := a.DB.First(&art, cm.ArticleID).Error; err == nil {
		artTitle = strings.TrimSpace(art.Title)
		tr := []rune(artTitle)
		if len(tr) > 80 {
			artTitle = string(tr[:80])
		}
		artCover = strings.TrimSpace(art.CoverURL)
	}
	likePl, _ := json.Marshal(gin.H{
		"like_subject":  "article_comment",
		"article_id":    cm.ArticleID,
		"article_title": artTitle,
		"cover_url":     artCover,
	})
	payload := string(likePl)
	if nres.RowsAffected == 0 {
		names, _ := json.Marshal([]string{likerName})
		n = model.Notification{
			RecipientID:     ownerID,
			Type:            "like_aggregation",
			RelatedID:       cm.ID,
			SenderNamesJSON: string(names),
			TotalLikes:      1,
			CommentPreview:  preview,
			PayloadJSON:     payload,
			IsRead:          false,
		}
		_ = a.DB.Create(&n).Error
		return
	}
	var names []string
	_ = json.Unmarshal([]byte(n.SenderNamesJSON), &names)
	for _, x := range names {
		if x == likerName {
			return
		}
	}
	names = append(names, likerName)
	b, _ := json.Marshal(names)
	n.SenderNamesJSON = string(b)
	n.TotalLikes++
	n.CommentPreview = preview
	if strings.TrimSpace(n.PayloadJSON) == "" {
		n.PayloadJSON = payload
	}
	_ = a.DB.Save(&n).Error
}

// notifyParentOnReply inserts reply_received for the parent comment author (SPEC F9 / 消息中心).
func (a *API) notifyParentOnReply(videoID, replierUID uint64, reply *model.Comment, parentID uint64) {
	var parent model.Comment
	if err := a.DB.First(&parent, parentID).Error; err != nil {
		return
	}
	if parent.UserID == replierUID {
		return
	}
	var u model.User
	if err := a.DB.First(&u, replierUID).Error; err != nil {
		return
	}
	preview := strings.TrimSpace(parent.Content)
	runes := []rune(preview)
	if len(runes) > 120 {
		preview = string(runes[:120])
	}
	pl := replyNotifPayload{
		SenderID:             replierUID,
		SenderUsername:       model.DisplayUsername(&u),
		SenderAvatarURL:      strings.TrimSpace(u.AvatarURL),
		ReplyCommentID:       reply.ID,
		ReplyContent:         reply.Content,
		ParentCommentID:      parentID,
		ParentContentPreview: preview,
		VideoID:              videoID,
	}
	pb, err := json.Marshal(pl)
	if err != nil {
		return
	}
	prevShort := preview
	sr := []rune(prevShort)
	if len(sr) > 32 {
		prevShort = string(sr[:32])
	}
	nm, _ := json.Marshal([]string{pl.SenderUsername})
	n := model.Notification{
		RecipientID:     parent.UserID,
		Type:            "reply_received",
		RelatedID:       reply.ID,
		SenderNamesJSON: string(nm),
		TotalLikes:      0,
		CommentPreview:  prevShort,
		PayloadJSON:     string(pb),
		IsRead:          false,
	}
	if err := a.DB.Create(&n).Error; err != nil && a.Log != nil {
		a.Log.Warn("create reply_received failed", zap.Error(err), zap.Uint64("reply_id", reply.ID))
	}
}

// UnreadSummary returns per-category unread counts (AC-12).
func (a *API) UnreadSummary(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	cats := []string{"reply_received", "at_me", "like_aggregation", "system_notice", "my_message"}
	out := gin.H{}
	for _, t := range cats {
		var cnt int64
		if t == "reply_received" {
			_ = a.DB.Model(&model.Notification{}).Where("recipient_id = ? AND type IN ? AND is_read = ?", uid, replyInboxNotificationTypes, false).Count(&cnt).Error
		} else if t == "my_message" {
			var sum struct{ Total int64 }
			_ = a.DB.Model(&model.DmParticipant{}).
				Select("COALESCE(SUM(unread_count), 0) as total").
				Where("user_id = ?", uid).
				Scan(&sum).Error
			cnt = sum.Total
		} else {
			_ = a.DB.Model(&model.Notification{}).Where("recipient_id = ? AND type = ? AND is_read = ?", uid, t, false).Count(&cnt).Error
		}
		out[t] = cnt
	}
	resp.OK(c, out)
}

// ListNotifications lists notifications for a category.
func (a *API) ListNotifications(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	cat := c.Query("category")
	if cat == "" {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	limit := 30
	curID, _ := strconv.ParseUint(c.Query("cursor"), 10, 64)
	q := a.DB.Model(&model.Notification{}).Where("recipient_id = ? AND type = ?", uid, cat)
	if cat == "reply_received" {
		q = a.DB.Model(&model.Notification{}).Where("recipient_id = ? AND type IN ?", uid, replyInboxNotificationTypes)
	}
	if curID > 0 {
		q = q.Where("id < ?", curID)
	}
	var list []model.Notification
	if err := q.Order("id DESC").Limit(limit + 1).Find(&list).Error; err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	hasMore := len(list) > limit
	if hasMore {
		list = list[:limit]
	}
	mutedComments := map[uint64]struct{}{}
	if cat == "like_aggregation" && len(list) > 0 {
		ids := make([]uint64, 0, len(list))
		for _, n := range list {
			if n.Type == "like_aggregation" && n.RelatedID != 0 {
				ids = append(ids, n.RelatedID)
			}
		}
		if len(ids) > 0 {
			var mutes []model.LikeNotifMute
			_ = a.DB.Where("recipient_id = ? AND comment_id IN ?", uid, ids).Find(&mutes).Error
			for _, m := range mutes {
				mutedComments[m.CommentID] = struct{}{}
			}
		}
	}
	items := make([]gin.H, 0, len(list))
	for _, n := range list {
		item := a.formatNotification(n)
		if cat == "like_aggregation" && n.Type == "like_aggregation" && n.RelatedID != 0 {
			_, ok := mutedComments[n.RelatedID]
			item["likes_muted"] = ok
		}
		items = append(items, item)
	}
	if cat == "reply_received" && len(items) > 0 {
		attachReplyInboxLikedByMe(a.DB, uid, items)
	}
	next := ""
	if hasMore && len(list) > 0 {
		next = strconv.FormatUint(list[len(list)-1].ID, 10)
	}
	resp.OK(c, gin.H{"items": items, "next_cursor": next})
}

// attachReplyInboxLikedByMe sets liked_by_me on each inbox row for the target comment (reply_comment_id).
func attachReplyInboxLikedByMe(db *gorm.DB, viewerID uint64, items []gin.H) {
	if viewerID == 0 || len(items) == 0 {
		return
	}
	videoCids := make([]uint64, 0)
	articleCids := make([]uint64, 0)
	for _, it := range items {
		rc := notifUint64(it["reply_comment_id"])
		if rc == 0 {
			continue
		}
		t := strings.TrimSpace(fmt.Sprint(it["type"]))
		if t == "article_comment_received" || t == "article_reply_received" {
			articleCids = append(articleCids, rc)
		} else {
			videoCids = append(videoCids, rc)
		}
	}
	liked := make(map[uint64]bool)
	if len(videoCids) > 0 {
		var likes []model.CommentLike
		_ = db.Where("user_id = ? AND comment_id IN ?", viewerID, videoCids).Find(&likes).Error
		for _, lk := range likes {
			liked[lk.CommentID] = true
		}
	}
	if len(articleCids) > 0 {
		var likes []model.ArticleCommentLike
		_ = db.Where("user_id = ? AND comment_id IN ?", viewerID, articleCids).Find(&likes).Error
		for _, lk := range likes {
			liked[lk.CommentID] = true
		}
	}
	for i := range items {
		rc := notifUint64(items[i]["reply_comment_id"])
		if rc == 0 {
			continue
		}
		items[i]["liked_by_me"] = liked[rc]
	}
}

func notifUint64(v interface{}) uint64 {
	switch x := v.(type) {
	case uint64:
		return x
	case uint:
		return uint64(x)
	case int64:
		if x < 0 {
			return 0
		}
		return uint64(x)
	case int:
		if x < 0 {
			return 0
		}
		return uint64(x)
	case float64:
		if x < 0 || x > float64(^uint64(0)>>1) {
			return 0
		}
		return uint64(x)
	case json.Number:
		u, err := strconv.ParseUint(string(x), 10, 64)
		if err != nil {
			return 0
		}
		return u
	default:
		return 0
	}
}

// likeNotifTopSenders resolves up to max users (by username order) for like aggregation rows:
// avatar URLs and user IDs for the first two likers (same DB lookup per name).
func (a *API) likeNotifTopSenders(usernames []string, max int) (urls []string, userIDs []uint64) {
	urls = make([]string, 0, max)
	userIDs = make([]uint64, 0, max)
	for _, raw := range usernames {
		if len(urls) >= max {
			break
		}
		name := strings.TrimSpace(raw)
		if name == "" {
			continue
		}
		var u model.User
		if err := a.DB.Where("username = ?", name).First(&u).Error; err != nil {
			urls = append(urls, "")
			userIDs = append(userIDs, 0)
			continue
		}
		urls = append(urls, strings.TrimSpace(u.AvatarURL))
		userIDs = append(userIDs, u.ID)
	}
	return urls, userIDs
}

func (a *API) formatNotification(n model.Notification) gin.H {
	var names []string
	_ = json.Unmarshal([]byte(n.SenderNamesJSON), &names)
	msg := ""
	out := gin.H{
		"id":              n.ID,
		"type":            n.Type,
		"message":         msg,
		"comment_preview": n.CommentPreview,
		"is_read":         n.IsRead,
		"total_likes":     n.TotalLikes,
		"created_at":      n.CreatedAt.Format("2006-01-02 15:04:05"),
	}
	switch n.Type {
	case "like_aggregation":
		targetZh := "评论"
		var likePl struct {
			LikeSubject string `json:"like_subject"`
		}
		_ = json.Unmarshal([]byte(n.PayloadJSON), &likePl)
		switch strings.TrimSpace(likePl.LikeSubject) {
		case "danmaku":
			targetZh = "弹幕"
		case "article_comment":
			targetZh = "专栏评论"
		}
		out["like_target"] = targetZh
		out["sender_names"] = names
		avatars, sids := a.likeNotifTopSenders(names, 2)
		out["sender_avatar_urls"] = avatars
		out["sender_user_ids"] = sids
		if n.RelatedID != 0 {
			var cm model.Comment
			if err := a.DB.First(&cm, n.RelatedID).Error; err == nil {
				out["video_id"] = cm.VideoID
				out["liked_comment_id"] = cm.ID
				out["comment_full_text"] = strings.TrimSpace(cm.Content)
			} else {
				var acm model.ArticleComment
				if err := a.DB.First(&acm, n.RelatedID).Error; err == nil {
					out["article_id"] = acm.ArticleID
					out["liked_comment_id"] = acm.ID
					out["comment_full_text"] = strings.TrimSpace(acm.Content)
					var art model.Article
					if err := a.DB.First(&art, acm.ArticleID).Error; err == nil {
						out["article_title"] = strings.TrimSpace(art.Title)
						out["article_cover_url"] = strings.TrimSpace(art.CoverURL)
					}
				}
			}
		}
		var likeMeta struct {
			LikeSubject  string `json:"like_subject"`
			ArticleID    uint64 `json:"article_id"`
			ArticleTitle string `json:"article_title"`
			CoverURL     string `json:"cover_url"`
		}
		if json.Unmarshal([]byte(n.PayloadJSON), &likeMeta) == nil {
			if likeMeta.ArticleID > 0 {
				out["article_id"] = likeMeta.ArticleID
			}
			if strings.TrimSpace(likeMeta.ArticleTitle) != "" {
				out["article_title"] = strings.TrimSpace(likeMeta.ArticleTitle)
			}
			if strings.TrimSpace(likeMeta.CoverURL) != "" {
				out["article_cover_url"] = strings.TrimSpace(likeMeta.CoverURL)
			}
		}
		suffix := "赞了我的" + targetZh
		switch n.TotalLikes {
		case 1:
			if len(names) > 0 {
				msg = names[0] + " " + suffix
			}
		case 2:
			if len(names) >= 2 {
				msg = names[0] + "、" + names[1] + " " + suffix
			}
		default:
			if len(names) >= 2 {
				msg = names[0] + "、" + names[1] + " 等总计" + strconv.Itoa(n.TotalLikes) + "人" + suffix
			} else if len(names) == 1 {
				msg = names[0] + " 等总计" + strconv.Itoa(n.TotalLikes) + "人" + suffix
			}
		}
		out["message"] = msg
	case "reply_received":
		out["inbox_kind"] = "reply_to_comment"
		var pl replyNotifPayload
		if json.Unmarshal([]byte(n.PayloadJSON), &pl) == nil && pl.SenderUsername != "" {
			out["sender_username"] = pl.SenderUsername
			out["sender_avatar_url"] = pl.SenderAvatarURL
			out["reply_content"] = pl.ReplyContent
			out["parent_content_preview"] = pl.ParentContentPreview
			out["video_id"] = pl.VideoID
			out["reply_comment_id"] = pl.ReplyCommentID
			out["parent_comment_id"] = pl.ParentCommentID
		} else if len(names) > 0 {
			out["sender_username"] = names[0]
		}
	case "video_comment_received":
		out["inbox_kind"] = "video_comment"
		var pl videoCommentNotifPayload
		if json.Unmarshal([]byte(n.PayloadJSON), &pl) == nil {
			if pl.SenderUsername != "" {
				out["sender_username"] = pl.SenderUsername
				out["sender_avatar_url"] = pl.SenderAvatarURL
			}
			if strings.TrimSpace(pl.CommentContent) != "" {
				out["reply_content"] = pl.CommentContent
			}
			if pl.CommentID != 0 {
				out["reply_comment_id"] = pl.CommentID
			}
			if pl.VideoID != 0 {
				out["video_id"] = pl.VideoID
			}
			if strings.TrimSpace(pl.VideoTitle) != "" {
				out["video_title"] = pl.VideoTitle
			}
			if strings.TrimSpace(pl.CoverURL) != "" {
				out["video_cover_url"] = pl.CoverURL
			}
		}
		if len(names) > 0 && fmt.Sprint(out["sender_username"]) == "" {
			out["sender_username"] = names[0]
		}
		if notifUint64(out["reply_comment_id"]) == 0 && n.RelatedID != 0 {
			out["reply_comment_id"] = n.RelatedID
		}
		if notifUint64(out["video_id"]) == 0 && n.RelatedID != 0 {
			var cm model.Comment
			if err := a.DB.First(&cm, n.RelatedID).Error; err == nil {
				out["video_id"] = cm.VideoID
				if fmt.Sprint(out["reply_content"]) == "" {
					out["reply_content"] = strings.TrimSpace(cm.Content)
				}
			}
		}
	case "article_comment_received":
		out["inbox_kind"] = "article_comment"
		var pl articleCommentNotifPayload
		if json.Unmarshal([]byte(n.PayloadJSON), &pl) == nil {
			if pl.SenderUsername != "" {
				out["sender_username"] = pl.SenderUsername
				out["sender_avatar_url"] = pl.SenderAvatarURL
			}
			if strings.TrimSpace(pl.CommentContent) != "" {
				out["reply_content"] = pl.CommentContent
			}
			if pl.CommentID != 0 {
				out["reply_comment_id"] = pl.CommentID
			}
			if pl.ArticleID != 0 {
				out["article_id"] = pl.ArticleID
			}
			if strings.TrimSpace(pl.ArticleTitle) != "" {
				out["article_title"] = pl.ArticleTitle
			}
			if strings.TrimSpace(pl.CoverURL) != "" {
				out["article_cover_url"] = pl.CoverURL
			}
		}
		if len(names) > 0 && fmt.Sprint(out["sender_username"]) == "" {
			out["sender_username"] = names[0]
		}
		if notifUint64(out["reply_comment_id"]) == 0 && n.RelatedID != 0 {
			out["reply_comment_id"] = n.RelatedID
		}
		if notifUint64(out["article_id"]) == 0 && n.RelatedID != 0 {
			var acm model.ArticleComment
			if err := a.DB.First(&acm, n.RelatedID).Error; err == nil {
				out["article_id"] = acm.ArticleID
				if fmt.Sprint(out["reply_content"]) == "" {
					out["reply_content"] = strings.TrimSpace(acm.Content)
				}
				var art model.Article
				if err := a.DB.First(&art, acm.ArticleID).Error; err == nil {
					if fmt.Sprint(out["article_title"]) == "" {
						out["article_title"] = strings.TrimSpace(art.Title)
					}
					if fmt.Sprint(out["article_cover_url"]) == "" {
						out["article_cover_url"] = strings.TrimSpace(art.CoverURL)
					}
				}
			}
		}
	case "article_reply_received":
		out["inbox_kind"] = "article_reply"
		var pl articleReplyNotifPayload
		if json.Unmarshal([]byte(n.PayloadJSON), &pl) == nil && pl.SenderUsername != "" {
			out["sender_username"] = pl.SenderUsername
			out["sender_avatar_url"] = pl.SenderAvatarURL
			out["reply_content"] = pl.ReplyContent
			out["parent_content_preview"] = pl.ParentContentPreview
			out["reply_comment_id"] = pl.ReplyCommentID
			out["parent_comment_id"] = pl.ParentCommentID
			out["article_id"] = pl.ArticleID
		} else if len(names) > 0 {
			out["sender_username"] = names[0]
		}
	}
	return out
}

// ListNotificationLikeLikers lists users who liked the comment for a like_aggregation inbox row (newest first).
func (a *API) ListNotificationLikeLikers(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	nid, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || nid == 0 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	var n model.Notification
	if err := a.DB.Where("id = ? AND recipient_id = ?", nid, uid).First(&n).Error; err != nil {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	if n.Type != "like_aggregation" || n.RelatedID == 0 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	isArticleLike := false
	var likeMeta struct {
		LikeSubject string `json:"like_subject"`
	}
	_ = json.Unmarshal([]byte(n.PayloadJSON), &likeMeta)
	if strings.TrimSpace(likeMeta.LikeSubject) == "article_comment" {
		isArticleLike = true
	} else {
		var probe model.ArticleComment
		if err := a.DB.First(&probe, n.RelatedID).Error; err == nil {
			isArticleLike = true
		}
	}
	limit := 30
	curID, _ := strconv.ParseUint(c.Query("cursor"), 10, 64)
	items := make([]gin.H, 0)
	var next string
	if isArticleLike {
		q := a.DB.Model(&model.ArticleCommentLike{}).Where("comment_id = ?", n.RelatedID)
		if curID > 0 {
			q = q.Where("id < ?", curID)
		}
		var likes []model.ArticleCommentLike
		if err := q.Order("id DESC").Limit(limit + 1).Find(&likes).Error; err != nil {
			resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
			return
		}
		hasMore := len(likes) > limit
		if hasMore {
			likes = likes[:limit]
		}
		for _, lk := range likes {
			var u model.User
			if err := a.DB.First(&u, lk.UserID).Error; err != nil {
				continue
			}
			items = append(items, gin.H{
				"id":         lk.ID,
				"user_id":    u.ID,
				"username":   model.DisplayUsername(&u),
				"avatar_url": uploaderAvatarForAPI(&u),
				"created_at": lk.CreatedAt.Format("2006-01-02 15:04:05"),
			})
		}
		if hasMore && len(likes) > 0 {
			next = strconv.FormatUint(likes[len(likes)-1].ID, 10)
		}
		enrichNotificationLikeLikerItems(a.DB, uid, items)
	} else {
		q := a.DB.Model(&model.CommentLike{}).Where("comment_id = ?", n.RelatedID)
		if curID > 0 {
			q = q.Where("id < ?", curID)
		}
		var likes []model.CommentLike
		if err := q.Order("id DESC").Limit(limit + 1).Find(&likes).Error; err != nil {
			resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
			return
		}
		hasMore := len(likes) > limit
		if hasMore {
			likes = likes[:limit]
		}
		for _, lk := range likes {
			var u model.User
			if err := a.DB.First(&u, lk.UserID).Error; err != nil {
				continue
			}
			items = append(items, gin.H{
				"id":         lk.ID,
				"user_id":    u.ID,
				"username":   model.DisplayUsername(&u),
				"avatar_url": uploaderAvatarForAPI(&u),
				"created_at": lk.CreatedAt.Format("2006-01-02 15:04:05"),
			})
		}
		if hasMore && len(likes) > 0 {
			next = strconv.FormatUint(likes[len(likes)-1].ID, 10)
		}
		enrichNotificationLikeLikerItems(a.DB, uid, items)
	}
	resp.OK(c, gin.H{"items": items, "next_cursor": next})
}

func enrichNotificationLikeLikerItems(db *gorm.DB, viewerID uint64, items []gin.H) {
	if len(items) == 0 {
		return
	}
	ids := make([]uint64, 0, len(items))
	for _, it := range items {
		if uid, ok := it["user_id"].(uint64); ok && uid > 0 {
			ids = append(ids, uid)
		}
	}
	followed := userFolloweeIDsSet(db, viewerID, ids)
	for i := range items {
		uid, _ := items[i]["user_id"].(uint64)
		items[i]["followed_by_me"] = followed[uid]
	}
}

// MarkNotificationCategoryRead marks all unread rows in one inbox category (F9).
func (a *API) MarkNotificationCategoryRead(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	cat := strings.TrimSpace(c.Query("category"))
	if cat == "" || cat == "my_message" {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	q := a.DB.Model(&model.Notification{}).Where("recipient_id = ? AND is_read = ?", uid, false)
	switch cat {
	case "reply_received":
		q = q.Where("type IN ?", replyInboxNotificationTypes)
	case "at_me", "like_aggregation", "system_notice":
		q = q.Where("type = ?", cat)
	default:
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	if err := q.Update("is_read", true).Error; err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	resp.OK(c, nil)
}

// MarkNotificationsReadBatch marks multiple inbox rows read (F9: viewed in message center).
func (a *API) MarkNotificationsReadBatch(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	var body struct {
		IDs []uint64 `json:"ids"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || len(body.IDs) == 0 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	ids := make([]uint64, 0, len(body.IDs))
	seen := make(map[uint64]struct{}, len(body.IDs))
	for _, id := range body.IDs {
		if id == 0 {
			continue
		}
		if _, dup := seen[id]; dup {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	if err := a.DB.Model(&model.Notification{}).
		Where("recipient_id = ? AND id IN ? AND is_read = ?", uid, ids, false).
		Update("is_read", true).Error; err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	resp.OK(c, nil)
}

// MarkNotificationRead marks one notification read (F9).
// Idempotent: already-read rows still return OK (PATCH may affect 0 rows).
func (a *API) MarkNotificationRead(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	nid, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || nid == 0 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	var n model.Notification
	if err := a.DB.Where("id = ? AND recipient_id = ?", nid, uid).First(&n).Error; err != nil {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	if err := a.DB.Model(&n).Update("is_read", true).Error; err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	resp.OK(c, nil)
}

// ToggleNotificationCommentLike toggles like on the comment behind a「回复我的」inbox row.
func (a *API) ToggleNotificationCommentLike(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	nid, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	n, ok := loadRecipientReplyInboxNotification(a, c, uid, nid)
	if !ok {
		return
	}
	target, ok := a.resolveReplyInboxTarget(n)
	if !ok {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	c.Params = gin.Params{{Key: "id", Value: strconv.FormatUint(target.CommentID, 10)}}
	if target.IsArticle {
		a.ToggleArticleCommentLike(c)
		return
	}
	a.ToggleLike(c)
}

// PostNotificationCommentReply posts a reply to the comment behind a「回复我的」inbox row.
func (a *API) PostNotificationCommentReply(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	nid, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	n, ok := loadRecipientReplyInboxNotification(a, c, uid, nid)
	if !ok {
		return
	}
	var body struct {
		Content string `json:"content"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	content := strings.TrimSpace(body.Content)
	if n := utf8.RuneCountInString(content); n < 1 || n > 1000 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	target, ok := a.resolveReplyInboxTarget(n)
	if !ok {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	payload, err := json.Marshal(commentPost{Content: content, ParentID: target.CommentID})
	if err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	c.Request.Body = io.NopCloser(bytes.NewReader(payload))
	c.Request.ContentLength = int64(len(payload))
	if target.IsArticle {
		c.Params = gin.Params{{Key: "id", Value: strconv.FormatUint(target.ArticleID, 10)}}
		a.PostArticleComment(c)
		return
	}
	c.Params = gin.Params{{Key: "id", Value: strconv.FormatUint(target.VideoID, 10)}}
	a.PostComment(c)
}

// DeleteNotification removes one inbox row for the recipient (创作中心「删除该通知」).
func (a *API) DeleteNotification(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	nid, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || nid == 0 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	var n model.Notification
	if err := a.DB.Where("id = ? AND recipient_id = ?", nid, uid).First(&n).Error; err != nil {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	tx := a.DB.Begin()
	if n.Type == "like_aggregation" && n.RelatedID != 0 {
		if err := tx.Where("recipient_id = ? AND comment_id = ?", uid, n.RelatedID).
			Delete(&model.LikeNotifMute{}).Error; err != nil {
			tx.Rollback()
			resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
			return
		}
	}
	res := tx.Where("id = ? AND recipient_id = ?", nid, uid).Delete(&model.Notification{})
	if res.Error != nil || res.RowsAffected == 0 {
		tx.Rollback()
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	if err := tx.Commit().Error; err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	resp.OK(c, gin.H{"ok": true})
}

// MuteLikeNotification stops new like aggregation updates for the related comment (不再通知).
func (a *API) MuteLikeNotification(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	nid, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || nid == 0 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	var n model.Notification
	if err := a.DB.Where("id = ? AND recipient_id = ?", nid, uid).First(&n).Error; err != nil {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	if n.Type != "like_aggregation" || n.RelatedID == 0 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	mute := model.LikeNotifMute{RecipientID: uid, CommentID: n.RelatedID}
	if err := a.DB.Where("recipient_id = ? AND comment_id = ?", uid, n.RelatedID).FirstOrCreate(&mute).Error; err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	resp.OK(c, gin.H{"likes_muted": true})
}
