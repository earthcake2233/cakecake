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
	"cakecake/internal/pkg/iplocate"
	"cakecake/internal/pkg/netutil"
	"cakecake/internal/pkg/resp"
	"cakecake/internal/service"
	"cakecake/internal/service/comment"
)

type commentPost struct {
	Content  string `json:"content"`
	ParentID uint64 `json:"parent_id"`
}

type commentItemDTO struct {
	ID           uint64 `json:"id"`
	UserID       uint64 `json:"user_id"`
	Username     string `json:"username"`
	AvatarURL    string `json:"avatar_url"`
	ParentID     uint64 `json:"parent_id"`
	Level        int    `json:"level"`
	UserLevel    int    `json:"user_level"`
	Content      string `json:"content"`
	LikeCount    uint64 `json:"like_count"`
	CreatedAt    string `json:"created_at"`
	LikedByMe    bool   `json:"liked_by_me"`
	DislikedByMe bool   `json:"disliked_by_me"`
	Pinned       bool   `json:"pinned"`
	IsByUploader bool   `json:"is_by_uploader"`
	IPLocation   string `json:"ip_location"`
}

type commentListResponse struct {
	Items           []commentItemDTO `json:"items"`
	CommentsCurated bool             `json:"comments_curated"`
	CommentsClosed  bool             `json:"comments_closed"`
	Page            int              `json:"page"`
	PageSize        int              `json:"page_size"`
	Total           int64            `json:"total"`
	TotalPages      int              `json:"total_pages"`
}

// parseCommentListQuery reads page/page_size/sort with safe defaults.
func parseCommentListQuery(c *gin.Context) comment.CommentListQuery {
	parse := func(raw string, def int) int {
		n, err := strconv.Atoi(raw)
		if err != nil {
			return def
		}
		return n
	}
	q := comment.CommentListQuery{
		Page:     parse(c.DefaultQuery("page", "1"), 1),
		PageSize: parse(c.DefaultQuery("page_size", "20"), 20),
		Sort:     c.DefaultQuery("sort", "hot"),
	}
	page, pageSize, sort := q.Normalized()
	return comment.CommentListQuery{Page: page, PageSize: pageSize, Sort: sort}
}

type postCommentResponse struct {
	ID        uint64  `json:"id"`
	UserID    uint64  `json:"user_id"`
	Content   string  `json:"content"`
	LikeCount uint64  `json:"like_count"`
	CreatedAt string  `json:"created_at"`
	Level     int     `json:"level"`
	LikedByMe bool    `json:"liked_by_me"`
	Pinned    bool    `json:"pinned"`
	Approved  bool    `json:"approved"`
	ParentID  *uint64 `json:"parent_id"`
}

type notificationItemDTO struct {
	ID              uint64 `json:"id"`
	Type            string `json:"type"`
	RelatedID       uint64 `json:"related_id"`
	CommentPreview  string `json:"comment_preview"`
	PayloadJSON     string `json:"payload_json"`
	SenderNamesJSON string `json:"sender_names_json"`
	TotalLikes      int    `json:"total_likes"`
	IsRead          bool   `json:"is_read"`
	CreatedAt       string `json:"created_at"`
}

type notificationListResponse struct {
	Items []notificationItemDTO `json:"items"`
	Total int64                 `json:"total"`
}

type notificationLikerDTO struct {
	UserID   uint64 `json:"user_id"`
	Username string `json:"username"`
}

type notificationLikerListResponse struct {
	Items []notificationLikerDTO `json:"items"`
}

type pinnedResponse struct {
	Pinned bool `json:"pinned"`
}

type likeToggleResponse struct {
	Liked     bool `json:"liked"`
	LikeCount int  `json:"like_count"`
}

type dislikeToggleResponse struct {
	Disliked bool `json:"disliked"`
}

type curatedIgnoredResponse struct {
	CuratedIgnored bool `json:"curated_ignored"`
}

type likesMutedResponse struct {
	LikesMuted bool `json:"likes_muted"`
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
	result, svcErr := a.CommentSvc.ListComments(c.Request.Context(), vid, uid, parseCommentListQuery(c))
	if svcErr != nil {
		resp.Err(c, httpStatusFromSvc(errCodeFromSvc(svcErr)), errCodeFromSvc(svcErr))
		return
	}
	out := make([]commentItemDTO, 0, len(result.Items))
	for _, item := range result.Items {
		out = append(out, commentItemDTO{
			ID: item.ID, UserID: item.UserID, Username: item.Username,
			AvatarURL: item.AvatarURL, ParentID: item.ParentID,
			Level: item.Level, UserLevel: item.UserLevel, Content: item.Content,
			LikeCount: item.LikeCount, CreatedAt: item.CreatedAt,
			LikedByMe: item.LikedByMe, DislikedByMe: item.DislikedByMe,
			Pinned: item.Pinned, IsByUploader: item.IsByUploader,
			IPLocation: iplocate.DisplayLabel(item.IPLocation),
		})
	}
	resp.OK(c, commentListResponse{
		Items: out, CommentsCurated: result.CommentsCurated, CommentsClosed: result.CommentsClosed,
		Page: result.Page, PageSize: result.PageSize, Total: result.Total, TotalPages: result.TotalPages,
	})
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
		comment.PostCommentReq{Content: content, ParentID: req.ParentID}, ipLoc)
	if svcErr != nil {
		resp.Err(c, httpStatusFromSvc(errCodeFromSvc(svcErr)), errCodeFromSvc(svcErr))
		return
	}
	uploadedAt := time.Now().Format("2006-01-02 15:04:05")
	if !cm.CreatedAt.IsZero() {
		uploadedAt = cm.CreatedAt.Format("2006-01-02 15:04:05")
	}
	r := postCommentResponse{ID: cm.ID, UserID: cm.UserID, Content: cm.Content, LikeCount: 0,
		CreatedAt: uploadedAt, Level: cm.Level, LikedByMe: false, Pinned: false, Approved: cm.Approved}
	if req.ParentID != 0 {
		pid := req.ParentID
		r.ParentID = &pid
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
	resp.OK(c, okResponse{OK: true})
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
	cm, err := a.CommentSvc.GetCommentByID(c.Request.Context(), cid)
	if err != nil {
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
	resp.OK(c, pinnedResponse{Pinned: pinned})
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
	resp.OK(c, likeToggleResponse{Liked: liked, LikeCount: total})
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
	resp.OK(c, dislikeToggleResponse{Disliked: disliked})
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
	resp.OK(c, okResponse{OK: true})
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
	resp.OK(c, curatedIgnoredResponse{CuratedIgnored: true})
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
	out := make([]notificationItemDTO, 0, len(list))
	for _, n := range list {
		out = append(out, notificationItemDTO{
			ID: n.ID, Type: n.Type, RelatedID: n.RelatedID,
			CommentPreview: n.CommentPreview, PayloadJSON: n.PayloadJSON,
			SenderNamesJSON: n.SenderNamesJSON, TotalLikes: n.TotalLikes,
			IsRead: n.IsRead, CreatedAt: n.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}
	resp.OK(c, notificationListResponse{Items: out, Total: total})
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
	resp.OK(c, okResponse{OK: true})
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
	resp.OK(c, okResponse{OK: true})
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
	resp.OK(c, okResponse{OK: true})
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
	resp.OK(c, okResponse{OK: true})
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
	resp.OK(c, likesMutedResponse{LikesMuted: true})
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
	out := make([]notificationLikerDTO, 0, len(users))
	for _, u := range users {
		out = append(out, notificationLikerDTO{UserID: u.ID, Username: u.Nickname})
	}
	resp.OK(c, notificationLikerListResponse{Items: out})
}
