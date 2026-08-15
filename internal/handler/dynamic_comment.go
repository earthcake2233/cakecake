package handler

import (
	"cakecake/internal/model/dynamic"
	"context"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"cakecake/internal/errcode"
	"cakecake/internal/middleware"
	"cakecake/internal/pkg/iplocate"
	"cakecake/internal/pkg/resp"
	"cakecake/internal/service/comment"
)

type dynamicCommentItemDTO struct {
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
	IPLocation   string `json:"ip_location"`
	IsByUploader bool   `json:"is_by_uploader"`
}

type dynamicCommentListResponse struct {
	Items           []dynamicCommentItemDTO `json:"items"`
	CommentsCurated bool                    `json:"comments_curated"`
	CommentsClosed  bool                    `json:"comments_closed"`
	Page            int                     `json:"page"`
	PageSize        int                     `json:"page_size"`
	Total           int64                   `json:"total"`
	TotalPages      int                     `json:"total_pages"`
}

type postDynamicCommentResponse struct {
	ID        uint64  `json:"id"`
	UserID    uint64  `json:"user_id"`
	Content   string  `json:"content"`
	ParentID  *uint64 `json:"parent_id"`
	LikeCount uint64  `json:"like_count"`
}

// ListDynamicComments lists comments on a dynamic.
func (a *API) ListDynamicComments(c *gin.Context) {
	did, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || did == 0 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	uid, _ := middleware.UserID(c)
	result, svcErr := a.CommentSvc.ListDynamicComments(c.Request.Context(), did, uid, parseCommentListQuery(c))
	if svcErr != nil {
		resp.Err(c, httpStatusFromSvc(errCodeFromSvc(svcErr)), errCodeFromSvc(svcErr))
		return
	}
	out := make([]dynamicCommentItemDTO, 0, len(result.Items))
	for _, item := range result.Items {
		out = append(out, dynamicCommentItemDTO{
			ID: item.ID, UserID: item.UserID, Username: item.Username,
			AvatarURL: item.AvatarURL, ParentID: item.ParentID,
			Level: item.Level, UserLevel: item.UserLevel, Content: item.Content,
			LikeCount: item.LikeCount, CreatedAt: item.CreatedAt,
			LikedByMe: item.LikedByMe, DislikedByMe: item.DislikedByMe,
			IPLocation: iplocate.DisplayLabel(item.IPLocation), IsByUploader: item.IsByUploader,
		})
	}
	resp.OK(c, dynamicCommentListResponse{
		Items: out, CommentsCurated: result.CommentsCurated, CommentsClosed: result.CommentsClosed,
		Page: result.Page, PageSize: result.PageSize, Total: result.Total, TotalPages: result.TotalPages,
	})
}

// PostDynamicComment creates a comment on a dynamic.
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
	var req struct {
		Content  string `json:"content"`
		ParentID uint64 `json:"parent_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	content := strings.TrimSpace(req.Content)
	ipLoc := a.resolveCommentIPLocation(c)
	cm, svcErr := a.CommentSvc.PostDynamicComment(c.Request.Context(), uid, did,
		comment.PostCommentReq{Content: content, ParentID: req.ParentID}, ipLoc)
	if svcErr != nil {
		resp.Err(c, httpStatusFromSvc(errCodeFromSvc(svcErr)), errCodeFromSvc(svcErr))
		return
	}
	resp.OK(c, postDynamicCommentResponse{ID: cm.ID, UserID: cm.UserID, Content: cm.Content})
}

// DeleteDynamicComment deletes a comment on a dynamic.
func (a *API) DeleteDynamicComment(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	cid, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	cm, err := a.CommentSvc.GetDynamicCommentByID(c.Request.Context(), cid)
	if err != nil {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	isUploader := false
	if uid != cm.UserID {
		d, err := a.DynamicSvc.GetDynamicByID(c.Request.Context(), cm.DynamicID)
		if err == nil && d.UserID == uid {
			isUploader = true
		}
	}
	err = a.CommentSvc.DeleteDynamicComment(c.Request.Context(), uid, cid, isUploader)
	if err != nil {
		resp.Err(c, httpStatusFromSvc(errCodeFromSvc(err)), errCodeFromSvc(err))
		return
	}
	resp.OK(c, okResponse{OK: true})
}

// ToggleDynamicCommentLike toggles the caller's like on a dynamic comment.
func (a *API) ToggleDynamicCommentLike(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	cid, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	liked, total, svcErr := a.CommentSvc.ToggleDynamicCommentReaction(c.Request.Context(), uid, cid, true)
	if svcErr != nil {
		resp.Err(c, httpStatusFromSvc(errCodeFromSvc(svcErr)), errCodeFromSvc(svcErr))
		return
	}
	resp.OK(c, likeToggleResponse{Liked: liked, LikeCount: total})
}

// ToggleDynamicCommentDislike toggles the caller's dislike on a dynamic comment.
func (a *API) ToggleDynamicCommentDislike(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	cid, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	liked, _, svcErr := a.CommentSvc.ToggleDynamicCommentReaction(c.Request.Context(), uid, cid, false)
	if svcErr != nil {
		resp.Err(c, httpStatusFromSvc(errCodeFromSvc(svcErr)), errCodeFromSvc(svcErr))
		return
	}
	disliked := liked
	resp.OK(c, dislikeToggleResponse{Disliked: disliked})
}

// ApproveDynamicComment approves a dynamic comment for public display.
func (a *API) ApproveDynamicComment(c *gin.Context) {
	cid, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	if err := a.CommentSvc.ApproveDynComment(c.Request.Context(), cid); err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	resp.OK(c, okResponse{OK: true})
}

// IgnoreCuratedDynamicComment marks a dynamic comment as curated-ignored.
func (a *API) IgnoreCuratedDynamicComment(c *gin.Context) {
	cid, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	if err := a.CommentSvc.IgnoreDynComment(c.Request.Context(), cid); err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	resp.OK(c, okResponse{OK: true})
}
func loadUserDynamic(ctx context.Context, a *API, id uint64) (*dynamic.UserDynamic, bool) {
	dyn, err := a.DynamicSvc.GetDynamicByID(ctx, id)
	if err != nil {
		return nil, false
	}
	return dyn, true
}
