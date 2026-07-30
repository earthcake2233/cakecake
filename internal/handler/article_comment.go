package handler

import (
	"minibili/internal/model/user"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"minibili/internal/errcode"
	"minibili/internal/middleware"
	"minibili/internal/pkg/iplocate"
	"minibili/internal/pkg/resp"
	"minibili/internal/service"
)

func (a *API) ListArticleComments(c *gin.Context) {
	aid, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || aid == 0 { resp.Err(c, http.StatusBadRequest, errcode.CodeParamError); return }
	uid, _ := middleware.UserID(c)
	result, svcErr := a.CommentSvc.ListArticleComments(c.Request.Context(), aid, uid)
	if svcErr != nil { resp.Err(c, httpStatusFromSvc(errCodeFromSvc(svcErr)), errCodeFromSvc(svcErr)); return }
	out := make([]gin.H, 0, len(result.Items))
	for _, item := range result.Items {
		out = append(out, gin.H{
			"id": item.ID, "user_id": item.UserID, "username": item.Username,
			"avatar_url": item.AvatarURL, "parent_id": item.ParentID,
			"level": item.Level, "user_level": item.UserLevel, "content": item.Content,
			"like_count": item.LikeCount, "created_at": item.CreatedAt,
			"liked_by_me": item.LikedByMe, "disliked_by_me": item.DislikedByMe,
			"pinned": item.Pinned, "is_by_author": item.IsByAuthor,
			"ip_location": iplocate.DisplayLabel(item.IPLocation),
		})
	}
	resp.OK(c, gin.H{"items": out, "comments_curated": result.CommentsCurated, "comments_closed": result.CommentsClosed})
}

func (a *API) PostArticleComment(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok { resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized); return }
	aid, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || aid == 0 { resp.Err(c, http.StatusBadRequest, errcode.CodeParamError); return }
	art, err := a.ArticleSvc.GetPublishedArticle(c.Request.Context(), aid)
	if err != nil {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound); return
	}
	if art.CommentsClosed { resp.Err(c, http.StatusForbidden, errcode.CodeCommentsClosed); return }
	var req struct {
		Content  string `json:"content"`
		ParentID uint64 `json:"parent_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil { resp.Err(c, http.StatusBadRequest, errcode.CodeParamError); return }
	content := strings.TrimSpace(req.Content)
	ipLoc := ""
	if a.IPLocate != nil {
		ip := getClientIP(c)
		if ip != "" { ipLoc = a.IPLocate.Province(ip) }
	}
	cm, svcErr := a.CommentSvc.PostArticleComment(c.Request.Context(), uid, aid,
		service.PostCommentReq{Content: content, ParentID: req.ParentID}, ipLoc)
	if svcErr != nil { resp.Err(c, httpStatusFromSvc(errCodeFromSvc(svcErr)), errCodeFromSvc(svcErr)); return }
	resp.JSON(c, http.StatusCreated, errcode.CodeSuccess, gin.H{"id": cm.ID, "user_id": cm.UserID, "content": cm.Content, "parent_id": nil, "level": cm.Level, "approved": !art.CommentsCurated, "like_count": 0})
}

func getClientIP(c *gin.Context) string {
	if c == nil || c.Request == nil { return "" }
	ip := c.GetHeader("X-Forwarded-For")
	if ip == "" { ip = c.GetHeader("X-Real-IP") }
	if ip == "" { ip = c.Request.RemoteAddr }
	return ip
}

func (a *API) DeleteArticleComment(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok { resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized); return }
	cid, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil { resp.Err(c, http.StatusBadRequest, errcode.CodeParamError); return }
	cm, findErr := a.CommentSvc.GetArticleComment(c.Request.Context(), cid)
	if findErr != nil { resp.Err(c, http.StatusNotFound, errcode.CodeNotFound); return }
	isAuthor := false
	if uid != cm.UserID {
		art, artErr := a.ArticleSvc.GetArticleByID(c.Request.Context(), cm.ArticleID)
		if artErr == nil && art.UserID == uid { isAuthor = true }
	}
	err = a.CommentSvc.DeleteArticleComment(c.Request.Context(), uid, cid, isAuthor)
	if err != nil { resp.Err(c, httpStatusFromSvc(errCodeFromSvc(err)), errCodeFromSvc(err)); return }
	resp.OK(c, gin.H{"ok": true})
}

func (a *API) PinArticleComment(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok { resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized); return }
	cid, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil { resp.Err(c, http.StatusBadRequest, errcode.CodeParamError); return }
	cm, findErr := a.CommentSvc.GetArticleComment(c.Request.Context(), cid)
	if findErr != nil { resp.Err(c, http.StatusNotFound, errcode.CodeNotFound); return }
	art, artErr := a.ArticleSvc.GetArticleByID(c.Request.Context(), cm.ArticleID)
	if artErr != nil || art.UserID != uid {
		resp.Err(c, http.StatusForbidden, errcode.CodeForbidden); return
	}
	pinned, svcErr := a.CommentSvc.PinArticleComment(c.Request.Context(), cm.ArticleID, cid)
	if svcErr != nil { resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError); return }
	resp.OK(c, gin.H{"pinned": pinned})
}

func (a *API) ToggleArticleCommentLike(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok { resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized); return }
	cid, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil { resp.Err(c, http.StatusBadRequest, errcode.CodeParamError); return }
	liked, total, svcErr := a.CommentSvc.ToggleArticleCommentLike(c.Request.Context(), uid, cid)
	if svcErr != nil { resp.Err(c, httpStatusFromSvc(errCodeFromSvc(svcErr)), errCodeFromSvc(svcErr)); return }
	resp.OK(c, gin.H{"liked": liked, "like_count": total})
}

func (a *API) ToggleArticleCommentDislike(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok { resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized); return }
	cid, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil { resp.Err(c, http.StatusBadRequest, errcode.CodeParamError); return }
	disliked, svcErr := a.CommentSvc.ToggleArticleCommentDislike(c.Request.Context(), uid, cid)
	if svcErr != nil { resp.Err(c, httpStatusFromSvc(errCodeFromSvc(svcErr)), errCodeFromSvc(svcErr)); return }
	resp.OK(c, gin.H{"disliked": disliked})
}

func (a *API) ApproveArticleComment(c *gin.Context) {
	cid, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil { resp.Err(c, http.StatusBadRequest, errcode.CodeParamError); return }
	if err := a.CommentSvc.ApproveArticleComment(c.Request.Context(), cid); err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError); return
	}
	resp.OK(c, gin.H{"ok": true})
}

func (a *API) IgnoreCuratedArticleComment(c *gin.Context) {
	cid, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil { resp.Err(c, http.StatusBadRequest, errcode.CodeParamError); return }
	if err := a.CommentSvc.IgnoreArticleComment(c.Request.Context(), cid); err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError); return
	}
	resp.OK(c, gin.H{"ok": true})
}
// GetMyArticle returns a user's own article.
func (a *API) GetMyArticle(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok { resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized); return }
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 { resp.Err(c, http.StatusBadRequest, errcode.CodeParamError); return }
	art, err := a.ArticleSvc.GetOwnedArticle(c.Request.Context(), id, uid)
	if err != nil {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound); return
	}
	var author user.User
	userPub, _ := a.UserSvc.GetUserPublic(c.Request.Context(), uid)
	if userPub != nil { author = user.User{ID: userPub.ID, Username: userPub.Username, AvatarURL: userPub.AvatarURL} }
	eng := toArticleEngagement(a.ArticleSvc.BatchArticleEngagementByViewer(c.Request.Context(), uid, []uint64{id})[id])
	resp.OK(c, articleDetailPayload(a, art, &author, eng, uid))
}
