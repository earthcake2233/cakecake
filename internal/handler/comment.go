package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"cakecake/internal/errcode"
	"cakecake/internal/middleware"
	"cakecake/internal/model/comment"
	"cakecake/internal/pkg/iplocate"
	"cakecake/internal/pkg/netutil"
	"cakecake/internal/pkg/resp"
	"cakecake/internal/service"
)

type commentPost struct {
	Content  string `json:"content"`
	ParentID uint64 `json:"parent_id"`
}

func errCodeFromSvc(err error) int {
	var se *service.SvcError
	if errors.As(err, &se) {
		return se.Code
	}
	return errcode.CodeInternalError
}

func httpStatusFromSvc(code int) int {
	switch code {
	case errcode.CodeNotFound:
		return http.StatusNotFound
	case errcode.CodeForbidden:
		return http.StatusForbidden
	case errcode.CodeUnauthorized:
		return http.StatusUnauthorized
	case errcode.CodeParamError, errcode.CodeCommentSensitive:
		return http.StatusBadRequest
	case errcode.CodeCommentsClosed:
		return http.StatusForbidden
	default:
		return http.StatusInternalServerError
	}
}

func (a *API) resolveCommentIPLocation(c *gin.Context) string {
	if a.IPLocate == nil {
		return ""
	}
	ip := netutil.ClientIP(c)
	if ip == "" {
		return ""
	}
	return a.IPLocate.Province(ip)
}

// ListComments returns flat comments for a video.
func (a *API) ListComments(c *gin.Context) {
	vid, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || vid == 0 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	uid, _ := middleware.UserID(c)
	result, svcErr := a.CommentSvc.ListComments(c.Request.Context(), vid, uid)
	if svcErr != nil {
		resp.Err(c, httpStatusFromSvc(errCodeFromSvc(svcErr)), errCodeFromSvc(svcErr))
		return
	}
	out := make([]gin.H, 0, len(result.Items))
	for _, item := range result.Items {
		out = append(out, gin.H{
			"id": item.ID, "user_id": item.UserID, "username": item.Username,
			"avatar_url": item.AvatarURL, "parent_id": item.ParentID,
			"level": item.Level, "user_level": item.UserLevel, "content": item.Content,
			"like_count": item.LikeCount, "created_at": item.CreatedAt,
			"liked_by_me": item.LikedByMe, "disliked_by_me": item.DislikedByMe,
			"pinned": item.Pinned, "is_by_uploader": item.IsByUploader,
			"ip_location": iplocate.DisplayLabel(item.IPLocation),
		})
	}
	resp.OK(c, gin.H{"items": out, "comments_curated": result.CommentsCurated, "comments_closed": result.CommentsClosed})
}

// PostComment creates a comment or reply.
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
	var req commentPost
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	content := strings.TrimSpace(req.Content)
	ipLoc := a.resolveCommentIPLocation(c)
	cm, svcErr := a.CommentSvc.PostComment(c.Request.Context(), uid, vid,
		service.PostCommentReq{Content: content, ParentID: req.ParentID}, ipLoc)
	if svcErr != nil {
		resp.Err(c, httpStatusFromSvc(errCodeFromSvc(svcErr)), errCodeFromSvc(svcErr))
		return
	}
	uploadedAt := time.Now().Format("2006-01-02 15:04:05")
	if !cm.CreatedAt.IsZero() {
		uploadedAt = cm.CreatedAt.Format("2006-01-02 15:04:05")
	}
	r := gin.H{"id": cm.ID, "user_id": cm.UserID, "content": cm.Content, "like_count": 0,
		"created_at": uploadedAt, "level": cm.Level, "liked_by_me": false, "pinned": false, "approved": cm.Approved}
	if req.ParentID != 0 {
		r["parent_id"] = req.ParentID
	} else {
		r["parent_id"] = nil
	}
	resp.JSON(c, http.StatusCreated, errcode.CodeSuccess, r)
}

// DeleteComment deletes a comment and its descendants.
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
	if err := a.CommentSvc.DeleteComment(c.Request.Context(), uid, cid, false); err != nil {
		resp.Err(c, httpStatusFromSvc(errCodeFromSvc(err)), errCodeFromSvc(err))
		return
	}
	resp.OK(c, gin.H{"ok": true})
}

// PinComment toggles pin status.
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
	// look up the comment to find the video ID, then verify caller owns the video
	var cm comment.Comment
	if err := a.DB.First(&cm, cid).Error; err != nil {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	v, err := a.VideoSvc.GetVideoByID(c.Request.Context(), cm.VideoID)
	if err != nil || v.UserID != uid {
		resp.Err(c, http.StatusForbidden, errcode.CodeForbidden)
		return
	}
	pinned, svcErr := a.CommentSvc.PinComment(c.Request.Context(), cm.VideoID, cid)
	if svcErr != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	resp.OK(c, gin.H{"pinned": pinned})
}

// ToggleLike toggles like on a comment.
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
	liked, total, svcErr := a.CommentSvc.ToggleCommentLike(c.Request.Context(), uid, cid)
	if svcErr != nil {
		resp.Err(c, httpStatusFromSvc(errCodeFromSvc(svcErr)), errCodeFromSvc(svcErr))
		return
	}
	resp.OK(c, gin.H{"liked": liked, "like_count": total})
}

// ToggleDislike toggles dislike on a comment.
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
	disliked, svcErr := a.CommentSvc.ToggleCommentDislike(c.Request.Context(), uid, cid)
	if svcErr != nil {
		resp.Err(c, httpStatusFromSvc(errCodeFromSvc(svcErr)), errCodeFromSvc(svcErr))
		return
	}
	resp.OK(c, gin.H{"disliked": disliked})
}

// ApproveComment approves a curated comment.
func (a *API) ApproveComment(c *gin.Context) {
	cid, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || cid == 0 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	if err := a.CommentSvc.ApproveComment(c.Request.Context(), cid); err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	resp.OK(c, gin.H{"ok": true})
}

// IgnoreCuratedComment ignores (deletes) a curated comment.
func (a *API) IgnoreCuratedComment(c *gin.Context) {
	cid, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || cid == 0 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	if err := a.CommentSvc.IgnoreCuratedComment(c.Request.Context(), cid); err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	resp.OK(c, gin.H{"curated_ignored": true})
}

// UnreadSummary returns unread notification counts.
func (a *API) UnreadSummary(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	r := a.NotifSvc.UnreadSummary(c.Request.Context(), uid)
	resp.JSON(c, http.StatusCreated, errcode.CodeSuccess, r)
}

// ListNotifications lists notifications for the user.
func (a *API) ListNotifications(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	cat := c.Query("category")
	page, pageSize := parsePagination(c, 20)
	list, total, err := a.NotifSvc.ListNotifications(c.Request.Context(), uid, cat, page, pageSize)
	if err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	out := make([]gin.H, 0, len(list))
	for _, n := range list {
		out = append(out, gin.H{
			"id": n.ID, "type": n.Type, "related_id": n.RelatedID,
			"comment_preview": n.CommentPreview, "payload_json": n.PayloadJSON,
			"sender_names_json": n.SenderNamesJSON, "total_likes": n.TotalLikes,
			"is_read": n.IsRead, "created_at": n.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}
	resp.OK(c, gin.H{"items": out, "total": total})
}

// MarkNotificationCategoryRead marks all notifications in a category as read.
func (a *API) MarkNotificationCategoryRead(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	cat := c.Query("category")
	if err := a.NotifSvc.MarkCategoryRead(c.Request.Context(), uid, cat); err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	resp.OK(c, gin.H{"ok": true})
}

// MarkNotificationsReadBatch marks specific notifications as read.
func (a *API) MarkNotificationsReadBatch(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	var ids []uint64
	if err := c.ShouldBindJSON(&ids); err != nil {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	if err := a.NotifSvc.MarkNotificationsRead(c.Request.Context(), uid, ids); err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	resp.OK(c, gin.H{"ok": true})
}

// MarkNotificationRead marks a single notification as read.
func (a *API) MarkNotificationRead(c *gin.Context) {
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
	if err := a.NotifSvc.MarkNotificationsRead(c.Request.Context(), uid, []uint64{nid}); err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	resp.OK(c, gin.H{"ok": true})
}

// DeleteNotification deletes a notification.
func (a *API) DeleteNotification(c *gin.Context) {
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
	if err := a.NotifSvc.DeleteNotification(c.Request.Context(), uid, nid); err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	resp.OK(c, gin.H{"ok": true})
}

// MuteLikeNotification mutes like notifications for a comment.
func (a *API) MuteLikeNotification(c *gin.Context) {
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
	if err := a.NotifSvc.MuteLikeNotification(c.Request.Context(), uid, nid); err != nil {
		if errors.Is(err, service.ErrNotFound) {
			resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
			return
		}
		if errors.Is(err, service.ErrParamError) {
			resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
			return
		}
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	resp.OK(c, gin.H{"likes_muted": true})
}

// ToggleNotificationCommentLike toggles a like on the comment referenced by a notification.
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
	n, err := a.NotifSvc.GetNotification(c.Request.Context(), nid, uid)
	if err != nil {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	c.Params = gin.Params{{Key: "id", Value: strconv.FormatUint(n.RelatedID, 10)}}
	a.ToggleLike(c)
}

// PostNotificationCommentReply posts a reply via a notification.
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
	n, err := a.NotifSvc.GetNotification(c.Request.Context(), nid, uid)
	if err != nil {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	var body struct {
		Content string `json:"content"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	if n.RelatedID == 0 {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	cm, err := a.CommentSvc.GetCommentByID(c.Request.Context(), n.RelatedID)
	if err != nil {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	payload, _ := json.Marshal(commentPost{Content: body.Content, ParentID: cm.ID})
	c.Request.Body = io.NopCloser(bytes.NewReader(payload))
	c.Request.ContentLength = int64(len(payload))
	c.Params = gin.Params{{Key: "id", Value: strconv.FormatUint(cm.VideoID, 10)}}
	a.PostComment(c)
}

// ListNotificationLikeLikers lists users who liked a comment via a notification.
func (a *API) ListNotificationLikeLikers(c *gin.Context) {
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
	users, err := a.NotifSvc.ListNotificationLikers(c.Request.Context(), uid, nid)
	if err != nil {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	out := make([]gin.H, 0, len(users))
	for _, u := range users {
		out = append(out, gin.H{"user_id": u.ID, "username": u.Nickname})
	}
	resp.OK(c, gin.H{"items": out})
}
