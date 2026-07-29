package handler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"minibili/internal/errcode"
	"minibili/internal/middleware"
	"minibili/internal/model"
	"minibili/internal/pkg/iplocate"
	"minibili/internal/pkg/resp"
	"minibili/internal/service"
)

func (a *API) ListDynamicComments(c *gin.Context) {
	did, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || did == 0 { resp.Err(c, http.StatusBadRequest, errcode.CodeParamError); return }
	uid, _ := middleware.UserID(c)
	result, svcErr := a.CommentSvc.ListDynamicComments(c.Request.Context(), did, uid)
	if svcErr != nil { resp.Err(c, httpStatusFromSvc(errCodeFromSvc(svcErr)), errCodeFromSvc(svcErr)); return }
	out := make([]gin.H, 0, len(result.Items))
	for _, item := range result.Items {
		out = append(out, gin.H{
			"id": item.ID, "user_id": item.UserID, "username": item.Username,
			"avatar_url": item.AvatarURL, "parent_id": item.ParentID,
			"level": item.Level, "user_level": item.UserLevel, "content": item.Content,
			"like_count": item.LikeCount, "created_at": item.CreatedAt,
			"liked_by_me": item.LikedByMe, "disliked_by_me": item.DislikedByMe,
			"ip_location": iplocate.DisplayLabel(item.IPLocation), "is_by_uploader": item.IsByUploader,
		})
	}
	resp.OK(c, gin.H{"items": out, "comments_curated": result.CommentsCurated})
}

func (a *API) PostDynamicComment(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok { resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized); return }
	did, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || did == 0 { resp.Err(c, http.StatusBadRequest, errcode.CodeParamError); return }
	var req struct {
		Content  string `json:"content"`
		ParentID uint64 `json:"parent_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil { resp.Err(c, http.StatusBadRequest, errcode.CodeParamError); return }
	content := strings.TrimSpace(req.Content)
	ipLoc := a.resolveCommentIPLocation(c)
	cm, svcErr := a.CommentSvc.PostDynamicComment(c.Request.Context(), uid, did,
		service.PostCommentReq{Content: content, ParentID: req.ParentID}, ipLoc)
	if svcErr != nil { resp.Err(c, httpStatusFromSvc(errCodeFromSvc(svcErr)), errCodeFromSvc(svcErr)); return }
	resp.OK(c, gin.H{"id": cm.ID, "user_id": cm.UserID, "content": cm.Content, "parent_id": nil, "like_count": 0})
}

func (a *API) DeleteDynamicComment(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok { resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized); return }
	cid, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil { resp.Err(c, http.StatusBadRequest, errcode.CodeParamError); return }
	var cm model.DynamicComment
	if err := a.DB.First(&cm, cid).Error; err != nil { resp.Err(c, http.StatusNotFound, errcode.CodeNotFound); return }
	isUploader := false
	if uid != cm.UserID {
		var d model.UserDynamic
		if err := a.DB.First(&d, cm.DynamicID).Error; err == nil && d.UserID == uid { isUploader = true }
	}
	err = a.CommentSvc.DeleteDynamicComment(c.Request.Context(), uid, cid, isUploader)
	if err != nil { resp.Err(c, httpStatusFromSvc(errCodeFromSvc(err)), errCodeFromSvc(err)); return }
	resp.OK(c, gin.H{"ok": true})
}

func (a *API) ToggleDynamicCommentLike(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok { resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized); return }
	cid, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil { resp.Err(c, http.StatusBadRequest, errcode.CodeParamError); return }
	liked, total, svcErr := a.CommentSvc.ToggleDynamicCommentReaction(c.Request.Context(), uid, cid, true)
	if svcErr != nil { resp.Err(c, httpStatusFromSvc(errCodeFromSvc(svcErr)), errCodeFromSvc(svcErr)); return }
	resp.OK(c, gin.H{"liked": liked, "like_count": total})
}

func (a *API) ToggleDynamicCommentDislike(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok { resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized); return }
	cid, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil { resp.Err(c, http.StatusBadRequest, errcode.CodeParamError); return }
	liked, _, svcErr := a.CommentSvc.ToggleDynamicCommentReaction(c.Request.Context(), uid, cid, false)
	if svcErr != nil { resp.Err(c, httpStatusFromSvc(errCodeFromSvc(svcErr)), errCodeFromSvc(svcErr)); return }
	disliked := liked
	resp.OK(c, gin.H{"disliked": disliked})
}

func (a *API) ApproveDynamicComment(c *gin.Context) {
	cid, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil { resp.Err(c, http.StatusBadRequest, errcode.CodeParamError); return }
	if err := a.CommentSvc.ApproveDynComment(c.Request.Context(), cid); err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError); return
	}
	resp.OK(c, gin.H{"ok": true})
}

func (a *API) IgnoreCuratedDynamicComment(c *gin.Context) {
	cid, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil { resp.Err(c, http.StatusBadRequest, errcode.CodeParamError); return }
	if err := a.CommentSvc.IgnoreDynComment(c.Request.Context(), cid); err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError); return
	}
	resp.OK(c, gin.H{"ok": true})
}
func loadUserDynamic(a *API, id uint64) (*model.UserDynamic, bool) {
	var dyn model.UserDynamic
	if err := a.DB.First(&dyn, id).Error; err != nil { return nil, false }
	return &dyn, true
}
