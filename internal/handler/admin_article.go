package handler

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"minibili/internal/errcode"
	"minibili/internal/middleware"
	"minibili/internal/model"
	"minibili/internal/pkg/markdown"
	"minibili/internal/pkg/resp"
	"minibili/internal/service"
)

func adminArticleStatusFilter(q string) []string {
	switch strings.TrimSpace(q) {
	case "", "all":
		return nil
	case "pending_review", "pending":
		return []string{"pending_review"}
	case "published", "passed":
		return []string{"published"}
	case "rejected":
		return []string{"rejected"}
	default:
		return []string{strings.TrimSpace(q)}
	}
}

func adminArticleToJSON(art *model.Article, uploaderName string) gin.H {
	bodyHTML, _, _ := markdown.Render(art.BodyMD)
	pubAt := ""
	if art.PublishedAt != nil {
		pubAt = art.PublishedAt.Format("2006-01-02 15:04:05")
	}
	out := gin.H{
		"id":            art.ID,
		"title":         art.Title,
		"cover_url":     art.CoverURL,
		"body_md":       art.BodyMD,
		"body_html":     bodyHTML,
		"status":        art.Status,
		"fail_reason":   strings.TrimSpace(art.FailReason),
		"user_id":       art.UserID,
		"uploader_name": uploaderName,
		"view_count":    art.ViewCount,
		"comment_count": art.CommentCount,
		"published_at":  pubAt,
		"created_at":    art.CreatedAt,
		"updated_at":    art.UpdatedAt,
	}
	if art.ReviewedAt != nil {
		out["reviewed_at"] = art.ReviewedAt
	}
	if art.ReviewedByAdminID != nil {
		out["reviewed_by_admin_id"] = *art.ReviewedByAdminID
	}
	return out
}

// AdminListArticles GET /api/v1/admin/articles
func (a *API) AdminListArticles(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 50 {
		pageSize = 20
	}
	statusQ := c.DefaultQuery("status", "pending_review")
	titleQ := strings.TrimSpace(c.Query("q"))

	q := a.DB.Model(&model.Article{})
	if sts := adminArticleStatusFilter(statusQ); len(sts) > 0 {
		q = q.Where("status IN ?", sts)
	}
	if titleQ != "" {
		q = q.Where("title LIKE ?", "%"+titleQ+"%")
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	totalPages := int((total + int64(pageSize) - 1) / int64(pageSize))
	if totalPages < 1 {
		totalPages = 1
	}
	if page > totalPages {
		page = totalPages
	}
	offset := (page - 1) * pageSize
	var rows []model.Article
	if err := q.Order("created_at DESC, id DESC").Offset(offset).Limit(pageSize).Find(&rows).Error; err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	uids := make([]uint64, 0, len(rows))
	for i := range rows {
		uids = append(uids, rows[i].UserID)
	}
	names := map[uint64]string{}
	if len(uids) > 0 {
		var users []model.User
		_ = a.DB.Where("id IN ?", uids).Find(&users).Error
		for i := range users {
			names[users[i].ID] = model.DisplayUsername(&users[i])
		}
	}
	items := make([]gin.H, 0, len(rows))
	for i := range rows {
		items = append(items, adminArticleToJSON(&rows[i], names[rows[i].UserID]))
	}
	var pending int64
	_ = a.DB.Model(&model.Article{}).Where("status = ?", "pending_review").Count(&pending).Error
	resp.OK(c, gin.H{
		"items":         items,
		"page":          page,
		"page_size":     pageSize,
		"total":         total,
		"total_pages":   totalPages,
		"pending_count": pending,
	})
}

// AdminGetArticle GET /api/v1/admin/articles/:id
func (a *API) AdminGetArticle(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	var art model.Article
	if err := a.DB.First(&art, id).Error; err != nil {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	var u model.User
	_ = a.DB.First(&u, art.UserID).Error
	resp.OK(c, adminArticleToJSON(&art, model.DisplayUsername(&u)))
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
	var art model.Article
	if err := a.DB.First(&art, id).Error; err != nil {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	if art.Status != articleStatusPendingReview {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 20*time.Second)
	defer cancel()
	aid := adminID
	if err := service.PublishArticle(ctx, a.DB, a.ES, a.Log, id, &aid); err != nil {
		a.Log.Error("admin approve article", zap.Error(err), zap.Uint64("article_id", id))
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	_ = a.DB.First(&art, id)
	var u model.User
	_ = a.DB.First(&u, art.UserID).Error
	a.Log.Info("admin approved article", zap.Uint64("article_id", id), zap.Uint64("admin_id", adminID))
	resp.OK(c, adminArticleToJSON(&art, model.DisplayUsername(&u)))
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
	var art model.Article
	if err := a.DB.First(&art, id).Error; err != nil {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	if art.Status != articleStatusPendingReview {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	now := time.Now()
	if err := a.DB.Model(&art).Updates(map[string]any{
		"status":               articleStatusRejected,
		"fail_reason":          reason,
		"published_at":         nil,
		"reviewed_at":          now,
		"reviewed_by_admin_id": adminID,
	}).Error; err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	if a.ES != nil && a.ES.Enabled() {
		ictx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		_ = a.ES.DeleteArticle(ictx, id)
		cancel()
	}
	_ = a.DB.First(&art, id)
	var u model.User
	_ = a.DB.First(&u, art.UserID).Error
	a.Log.Info("admin rejected article", zap.Uint64("article_id", id), zap.Uint64("admin_id", adminID))
	resp.OK(c, adminArticleToJSON(&art, model.DisplayUsername(&u)))
}

// AdminDeleteArticle POST /api/v1/admin/articles/:id/delete 或 DELETE /api/v1/admin/articles/:id
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
	var art model.Article
	if err := a.DB.First(&art, id).Error; err != nil {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	if art.Status != articleStatusPublished && art.Status != articleStatusRejected {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	if err := a.DB.Transaction(func(tx *gorm.DB) error {
		return deleteArticleCascade(tx, id)
	}); err != nil {
		a.Log.Error("admin delete article", zap.Error(err), zap.Uint64("article_id", id))
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	purgeArticleOSSObjects(a.Cfg, a.OSS, a.Log, art)
	a.esDeleteArticle(id)
	a.Log.Info("admin deleted article",
		zap.Uint64("article_id", id),
		zap.Uint64("admin_id", adminID),
		zap.String("status", art.Status),
	)
	resp.OK(c, gin.H{"ok": true})
}
