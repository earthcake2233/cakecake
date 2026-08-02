package handler

import (
	"cakecake/internal/model/article"
	"cakecake/internal/model/user"
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"cakecake/internal/errcode"
	"cakecake/internal/middleware"
	"cakecake/internal/pkg/markdown"
	"cakecake/internal/pkg/resp"
)

func adminArticleStatusFilter(q string) []string {
	switch strings.TrimSpace(q) {
	case "", "all":
		return nil
	case article.StatusPendingReview, "pending":
		return []string{article.StatusPendingReview}
	case article.StatusPublished, article.StatusPassed:
		return []string{article.StatusPublished}
	case article.StatusRejected:
		return []string{article.StatusRejected}
	default:
		return []string{strings.TrimSpace(q)}
	}
}

type adminArticleItem struct {
	ID                uint64     `json:"id"`
	Title             string     `json:"title"`
	CoverURL          string     `json:"cover_url"`
	BodyMD            string     `json:"body_md"`
	BodyHTML          string     `json:"body_html"`
	Status            string     `json:"status"`
	FailReason        string     `json:"fail_reason"`
	UserID            uint64     `json:"user_id"`
	UploaderName      string     `json:"uploader_name"`
	ViewCount         uint64     `json:"view_count"`
	CommentCount      uint64     `json:"comment_count"`
	PublishedAt       string     `json:"published_at"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
	ReviewedAt        *time.Time `json:"reviewed_at,omitempty"`
	ReviewedByAdminID *uint64    `json:"reviewed_by_admin_id,omitempty"`
}

type adminArticleListResponse struct {
	Items        []adminArticleItem `json:"items"`
	Page         int                `json:"page"`
	PageSize     int                `json:"page_size"`
	Total        int64              `json:"total"`
	TotalPages   int                `json:"total_pages"`
	PendingCount int64              `json:"pending_count"`
}

func adminArticleToJSON(art *article.Article, uploaderName string) adminArticleItem {
	bodyHTML, _, _ := markdown.Render(art.BodyMD)
	pubAt := ""
	if art.PublishedAt != nil {
		pubAt = art.PublishedAt.Format("2006-01-02 15:04:05")
	}
	item := adminArticleItem{
		ID:           art.ID,
		Title:        art.Title,
		CoverURL:     art.CoverURL,
		BodyMD:       art.BodyMD,
		BodyHTML:     bodyHTML,
		Status:       art.Status,
		FailReason:   strings.TrimSpace(art.FailReason),
		UserID:       art.UserID,
		UploaderName: uploaderName,
		ViewCount:    art.ViewCount,
		CommentCount: art.CommentCount,
		PublishedAt:  pubAt,
		CreatedAt:    art.CreatedAt,
		UpdatedAt:    art.UpdatedAt,
	}
	item.ReviewedAt = art.ReviewedAt
	item.ReviewedByAdminID = art.ReviewedByAdminID
	return item
}

// AdminListArticles GET /api/v1/admin/articles

func (a *API) AdminListArticles(c *gin.Context) {
	page, pageSize := parsePagination(c, 20)
	statusQ := c.DefaultQuery("status", article.StatusPendingReview)
	titleQ := strings.TrimSpace(c.Query("q"))

	statuses := adminArticleStatusFilter(statusQ)
	result, err := a.ArticleSvc.AdminListArticles(c.Request.Context(), statuses, titleQ, page, pageSize)
	if err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}

	totalPages := int((result.Total + int64(pageSize) - 1) / int64(pageSize))
	if totalPages < 1 {
		totalPages = 1
	}
	if page > totalPages {
		page = totalPages
	}

	uids := make([]uint64, 0, len(result.Rows))
	for i := range result.Rows {
		uids = append(uids, result.Rows[i].UserID)
	}
	names := map[uint64]string{}
	if len(uids) > 0 {
		usersMap := a.UserSvc.BatchGetUsers(c.Request.Context(), uids)
		for id, u := range usersMap {
			names[id] = user.DisplayUsername(u)
		}
	}
	items := make([]adminArticleItem, 0, len(result.Rows))
	for i := range result.Rows {
		items = append(items, adminArticleToJSON(&result.Rows[i], names[result.Rows[i].UserID]))
	}

	resp.OK(c, adminArticleListResponse{
		Items:        items,
		Page:         page,
		PageSize:     pageSize,
		Total:        result.Total,
		TotalPages:   totalPages,
		PendingCount: result.PendingCount,
	})
}

// AdminGetArticle GET /api/v1/admin/articles/:id
func (a *API) AdminGetArticle(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	art, err := a.ArticleSvc.GetArticleByID(c.Request.Context(), id)
	if err != nil {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	u, _ := a.UserSvc.GetUserByID(c.Request.Context(), art.UserID)
	resp.OK(c, adminArticleToJSON(art, user.DisplayUsername(u)))
}

type adminArticleRejectReq struct {
	Reason string `json:"reason"`
}

// AdminApproveArticle POST /api/v1/admin/articles/:id/approve
func (a *API) AdminApproveArticle(c *gin.Context) {
	adminID, ok := middleware.AdminID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	art, err := a.ArticleSvc.GetArticleByID(c.Request.Context(), id)
	if err != nil {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	if art.Status != article.StatusPendingReview {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 20*time.Second)
	defer cancel()
	aid := adminID
	if err := a.ArticleSvc.Publish(ctx, id, &aid); err != nil {
		a.Log.Error("admin approve article", zap.Error(err), zap.Uint64("article_id", id))
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	art, _ = a.ArticleSvc.GetArticleByID(c.Request.Context(), id)
	u, _ := a.UserSvc.GetUserByID(c.Request.Context(), art.UserID)
	a.Log.Info("admin approved article", zap.Uint64("article_id", id), zap.Uint64("admin_id", adminID))
	resp.OK(c, adminArticleToJSON(art, user.DisplayUsername(u)))
}

// AdminRejectArticle POST /api/v1/admin/articles/:id/reject
func (a *API) AdminRejectArticle(c *gin.Context) {
	adminID, ok := middleware.AdminID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	var req adminArticleRejectReq
	_ = c.ShouldBindJSON(&req)
	reason := strings.TrimSpace(req.Reason)
	if reason == "" {
		reason = "内容不符合社区规范"
	}
	art, err := a.ArticleSvc.GetArticleByID(c.Request.Context(), id)
	if err != nil {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	if art.Status != article.StatusPendingReview {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	now := time.Now()
	if err := a.ArticleSvc.AdminUpdateArticle(c.Request.Context(), id, map[string]interface{}{
		"status":               article.StatusRejected,
		"fail_reason":          reason,
		"published_at":         nil,
		"reviewed_at":          now,
		"reviewed_by_admin_id": adminID,
	}); err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	if a.SearchSvc != nil && a.SearchSvc.Enabled() {
		ictx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		_ = a.SearchSvc.DeleteArticle(ictx, id)
		cancel()
	}
	art, _ = a.ArticleSvc.GetArticleByID(c.Request.Context(), id)
	u, _ := a.UserSvc.GetUserByID(c.Request.Context(), art.UserID)
	a.Log.Info("admin rejected article", zap.Uint64("article_id", id), zap.Uint64("admin_id", adminID))
	resp.OK(c, adminArticleToJSON(art, user.DisplayUsername(u)))
}

// AdminDeleteArticle POST /api/v1/admin/articles/:id/delete or DELETE /api/v1/admin/articles/:id
func (a *API) AdminDeleteArticle(c *gin.Context) {
	adminID, ok := middleware.AdminID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	art, err := a.ArticleSvc.GetArticleByID(c.Request.Context(), id)
	if err != nil {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	if art.Status != article.StatusPublished && art.Status != article.StatusRejected {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	if err := a.ArticleSvc.AdminDeleteArticleCascade(c.Request.Context(), id, func(tx *gorm.DB) error {
		return deleteArticleCascade(tx, id)
	}); err != nil {
		a.Log.Error("admin delete article", zap.Error(err), zap.Uint64("article_id", id))
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	purgeArticleOSSObjects(a.Cfg, a.OSS, a.Log, *art)
	a.esDeleteArticle(id)
	a.Log.Info("admin deleted article",
		zap.Uint64("article_id", id),
		zap.Uint64("admin_id", adminID),
		zap.String("status", art.Status),
	)
	resp.OK(c, okResponse{OK: true})
}
