package handler

import (
	"cakecake/internal/model/user"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"cakecake/internal/errcode"
	"cakecake/internal/middleware"
	"cakecake/internal/pkg/iplocate"
	"cakecake/internal/pkg/resp"
	"cakecake/internal/service"
)

type articleCommentItemDTO struct {
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
	IsByAuthor   bool   `json:"is_by_author"`
	IPLocation   string `json:"ip_location"`
}

type articleCommentListResponse struct {
	Items           []articleCommentItemDTO `json:"items"`
	CommentsCurated bool                    `json:"comments_curated"`
	CommentsClosed  bool                    `json:"comments_closed"`
}

type postArticleCommentResponse struct {
	ID        uint64  `json:"id"`
	UserID    uint64  `json:"user_id"`
	Content   string  `json:"content"`
	ParentID  *uint64 `json:"parent_id"`
	Level     int     `json:"level"`
	Approved  bool    `json:"approved"`
	LikeCount uint64  `json:"like_count"`
}

func (a *API) ListArticleComments(c *gin.Context) {
	aid, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || aid == 0 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	uid, _ := middleware.UserID(c)
	result, svcErr := a.CommentSvc.ListArticleComments(c.Request.Context(), aid, uid)
	if svcErr != nil {
		resp.Err(c, httpStatusFromSvc(errCodeFromSvc(svcErr)), errCodeFromSvc(svcErr))
		return
	}
	out := make([]articleCommentItemDTO, 0, len(result.Items))
	for _, item := range result.Items {
		out = append(out, articleCommentItemDTO{
			ID: item.ID, UserID: item.UserID, Username: item.Username,
			AvatarURL: item.AvatarURL, ParentID: item.ParentID,
			Level: item.Level, UserLevel: item.UserLevel, Content: item.Content,
			LikeCount: item.LikeCount, CreatedAt: item.CreatedAt,
			LikedByMe: item.LikedByMe, DislikedByMe: item.DislikedByMe,
			Pinned: item.Pinned, IsByAuthor: item.IsByAuthor,
			IPLocation: iplocate.DisplayLabel(item.IPLocation),
		})
	}
	resp.OK(c, articleCommentListResponse{Items: out, CommentsCurated: result.CommentsCurated, CommentsClosed: result.CommentsClosed})
}

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
	art, err := a.ArticleSvc.GetPublishedArticle(c.Request.Context(), aid)
	if err != nil {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	if art.CommentsClosed {
		resp.Err(c, http.StatusForbidden, errcode.CodeCommentsClosed)
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
	ipLoc := ""
	if a.IPLocate != nil {
		ip := getClientIP(c)
		if ip != "" {
			ipLoc = a.IPLocate.Province(ip)
		}
	}
	cm, svcErr := a.CommentSvc.PostArticleComment(c.Request.Context(), uid, aid,
		service.PostCommentReq{Content: content, ParentID: req.ParentID}, ipLoc)
	if svcErr != nil {
		resp.Err(c, httpStatusFromSvc(errCodeFromSvc(svcErr)), errCodeFromSvc(svcErr))
		return
	}
	resp.JSON(c, http.StatusCreated, errcode.CodeSuccess, postArticleCommentResponse{ID: cm.ID, UserID: cm.UserID, Content: cm.Content, Level: cm.Level, Approved: !art.CommentsCurated})
}

func getClientIP(c *gin.Context) string {
	if c == nil || c.Request == nil {
		return ""
	}
	ip := c.GetHeader("X-Forwarded-For")
	if ip == "" {
		ip = c.GetHeader("X-Real-IP")
	}
	if ip == "" {
		ip = c.Request.RemoteAddr
	}
	return ip
}

func (a *API) DeleteArticleComment(c *gin.Context) {
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
	cm, findErr := a.CommentSvc.GetArticleComment(c.Request.Context(), cid)
	if findErr != nil {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	isAuthor := false
	if uid != cm.UserID {
		art, artErr := a.ArticleSvc.GetArticleByID(c.Request.Context(), cm.ArticleID)
		if artErr == nil && art.UserID == uid {
			isAuthor = true
		}
	}
	err = a.CommentSvc.DeleteArticleComment(c.Request.Context(), uid, cid, isAuthor)
	if err != nil {
		resp.Err(c, httpStatusFromSvc(errCodeFromSvc(err)), errCodeFromSvc(err))
		return
	}
	resp.OK(c, okResponse{OK: true})
}

func (a *API) PinArticleComment(c *gin.Context) {
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
	cm, findErr := a.CommentSvc.GetArticleComment(c.Request.Context(), cid)
	if findErr != nil {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	art, artErr := a.ArticleSvc.GetArticleByID(c.Request.Context(), cm.ArticleID)
	if artErr != nil || art.UserID != uid {
		resp.Err(c, http.StatusForbidden, errcode.CodeForbidden)
		return
	}
	pinned, svcErr := a.CommentSvc.PinArticleComment(c.Request.Context(), cm.ArticleID, cid)
	if svcErr != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	resp.OK(c, pinnedResponse{Pinned: pinned})
}

func (a *API) ToggleArticleCommentLike(c *gin.Context) {
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
	liked, total, svcErr := a.CommentSvc.ToggleArticleCommentLike(c.Request.Context(), uid, cid)
	if svcErr != nil {
		resp.Err(c, httpStatusFromSvc(errCodeFromSvc(svcErr)), errCodeFromSvc(svcErr))
		return
	}
	resp.OK(c, likeToggleResponse{Liked: liked, LikeCount: total})
}

func (a *API) ToggleArticleCommentDislike(c *gin.Context) {
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
	disliked, svcErr := a.CommentSvc.ToggleArticleCommentDislike(c.Request.Context(), uid, cid)
	if svcErr != nil {
		resp.Err(c, httpStatusFromSvc(errCodeFromSvc(svcErr)), errCodeFromSvc(svcErr))
		return
	}
	resp.OK(c, dislikeToggleResponse{Disliked: disliked})
}

func (a *API) ApproveArticleComment(c *gin.Context) {
	cid, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	if err := a.CommentSvc.ApproveArticleComment(c.Request.Context(), cid); err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	resp.OK(c, okResponse{OK: true})
}

func (a *API) IgnoreCuratedArticleComment(c *gin.Context) {
	cid, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	if err := a.CommentSvc.IgnoreArticleComment(c.Request.Context(), cid); err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	resp.OK(c, okResponse{OK: true})
}

// GetMyArticle returns a user's own article.
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
	art, err := a.ArticleSvc.GetOwnedArticle(c.Request.Context(), id, uid)
	if err != nil {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	var author user.User
	userPub, _ := a.UserSvc.GetUserPublic(c.Request.Context(), uid)
	if userPub != nil {
		author = user.User{ID: userPub.ID, Username: userPub.Username, AvatarURL: userPub.AvatarURL}
	}
	eng := toArticleEngagement(a.ArticleSvc.BatchArticleEngagementByViewer(c.Request.Context(), uid, []uint64{id})[id])
	resp.OK(c, articleDetailPayload(a, art, &author, eng, uid))
}
