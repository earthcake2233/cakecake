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
	"minibili/internal/pkg/resp"
	"minibili/internal/service"
)

func adminVideoStatusFilter(q string) []string {
	switch strings.TrimSpace(q) {
	case "", "all":
		return nil
	case "pending_review", "pending":
		return []string{"pending_review"}
	case "published", "passed":
		return []string{"published"}
	case "rejected":
		return []string{"rejected"}
	case "processing":
		return []string{"processing"}
	case "failed":
		return []string{"failed"}
	default:
		return []string{strings.TrimSpace(q)}
	}
}

func adminVideoToJSON(v *model.Video, uploaderName string) gin.H {
	out := gin.H{
		"id":            v.ID,
		"title":         v.Title,
		"description":   v.Description,
		"status":        v.Status,
		"fail_reason":   strings.TrimSpace(v.FailReason),
		"cover_url":     v.CoverURL,
		"video_url":     v.VideoURL,
		"duration_sec":  v.DurationSec,
		"zone":          v.Zone,
		"user_id":       v.UserID,
		"uploader_name": uploaderName,
		"play_count":    v.PlayCount,
		"created_at":    v.CreatedAt,
		"updated_at":    v.UpdatedAt,
	}
	if v.ReviewedAt != nil {
		out["reviewed_at"] = v.ReviewedAt
	}
	if v.ReviewedByAdminID != nil {
		out["reviewed_by_admin_id"] = *v.ReviewedByAdminID
	}
	return out
}

// AdminListVideos GET /api/v1/admin/videos
func (a *API) AdminListVideos(c *gin.Context) {
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

	q := a.DB.Model(&model.Video{})
	if sts := adminVideoStatusFilter(statusQ); len(sts) > 0 {
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
	var rows []model.Video
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
		items = append(items, adminVideoToJSON(&rows[i], names[rows[i].UserID]))
	}
	var pending int64
	_ = a.DB.Model(&model.Video{}).Where("status = ?", "pending_review").Count(&pending).Error
	resp.OK(c, gin.H{
		"items":         items,
		"page":          page,
		"page_size":     pageSize,
		"total":         total,
		"total_pages":   totalPages,
		"pending_count": pending,
	})
}

// AdminGetVideo GET /api/v1/admin/videos/:id
func (a *API) AdminGetVideo(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	var v model.Video
	if err := a.DB.First(&v, id).Error; err != nil {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	var u model.User
	_ = a.DB.First(&u, v.UserID).Error
	resp.OK(c, adminVideoToJSON(&v, model.DisplayUsername(&u)))
}

type adminVideoRejectReq struct {
	Reason string `json:"reason"`
}

// AdminApproveVideo POST /api/v1/admin/videos/:id/approve
func (a *API) AdminApproveVideo(c *gin.Context) {
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
	var v model.Video
	if err := a.DB.First(&v, id).Error; err != nil {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	if v.Status != "pending_review" {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	if strings.TrimSpace(v.VideoURL) == "" {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 20*time.Second)
	defer cancel()
	aid := adminID
	if err := service.PublishVideo(ctx, a.DB, a.ES, a.Log, id, &aid); err != nil {
		a.Log.Error("admin approve video", zap.Error(err), zap.Uint64("video_id", id))
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	_ = a.DB.First(&v, id)
	var u model.User
	_ = a.DB.First(&u, v.UserID).Error
	a.Log.Info("admin approved video", zap.Uint64("video_id", id), zap.Uint64("admin_id", adminID))
	resp.OK(c, adminVideoToJSON(&v, model.DisplayUsername(&u)))
}

// AdminRejectVideo POST /api/v1/admin/videos/:id/reject
func (a *API) AdminRejectVideo(c *gin.Context) {
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
	var req adminVideoRejectReq
	_ = c.ShouldBindJSON(&req)
	reason := strings.TrimSpace(req.Reason)
	if reason == "" {
		reason = "内容不符合社区规范"
	}
	var v model.Video
	if err := a.DB.First(&v, id).Error; err != nil {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	if v.Status != "pending_review" {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	now := time.Now()
	if err := a.DB.Model(&v).Updates(map[string]any{
		"status":               "rejected",
		"fail_reason":          reason,
		"reviewed_at":          now,
		"reviewed_by_admin_id": adminID,
	}).Error; err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	if a.ES != nil && a.ES.Enabled() {
		ictx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		_ = a.ES.DeleteVideo(ictx, id)
		cancel()
	}
	_ = a.DB.First(&v, id)
	var u model.User
	_ = a.DB.First(&u, v.UserID).Error
	a.Log.Info("admin rejected video", zap.Uint64("video_id", id), zap.Uint64("admin_id", adminID))
	resp.OK(c, adminVideoToJSON(&v, model.DisplayUsername(&u)))
}

// AdminDeleteVideo POST /api/v1/admin/videos/:id/delete 或 DELETE /api/v1/admin/videos/:id
// Removes published or rejected videos from DB and OSS (same cascade as uploader delete).
func (a *API) AdminDeleteVideo(c *gin.Context) {
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
	var v model.Video
	if err := a.DB.First(&v, id).Error; err != nil {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	if v.Status != "published" && v.Status != "rejected" {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	removeVideoDraftFiles(v)
	if err := a.DB.Transaction(func(tx *gorm.DB) error {
		return deleteVideoCascade(tx, id)
	}); err != nil {
		a.Log.Error("admin delete video", zap.Error(err), zap.Uint64("video_id", id))
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	purgeVideoOSSObjects(a.Cfg, a.OSS, a.Log, v)
	a.esDeleteVideo(id)
	a.Log.Info("admin deleted video",
		zap.Uint64("video_id", id),
		zap.Uint64("admin_id", adminID),
		zap.String("status", v.Status),
	)
	resp.OK(c, gin.H{"ok": true})
}
