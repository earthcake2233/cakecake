package handler

import (
	"context"
	"minibili/internal/model/user"
	"minibili/internal/model/video"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"minibili/internal/errcode"
	"minibili/internal/middleware"
	"minibili/internal/pkg/resp"
	"minibili/internal/service"
)

func adminVideoStatusFilter(q string) []string {
	switch strings.TrimSpace(q) {
	case "", "all":
		return nil
	case video.StatusPendingReview, "pending":
		return []string{video.StatusPendingReview}
	case video.StatusPublished, video.StatusPassed:
		return []string{video.StatusPublished}
	case video.StatusRejected:
		return []string{video.StatusRejected}
	case video.StatusProcessing:
		return []string{video.StatusProcessing}
	case video.StatusFailed:
		return []string{video.StatusFailed}
	default:
		return []string{strings.TrimSpace(q)}
	}
}

func adminVideoToJSON(v *video.Video, uploaderName string) gin.H {
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
	page, pageSize := parsePagination(c, 20)
	statusQ := c.DefaultQuery("status", video.StatusPendingReview)
	titleQ := strings.TrimSpace(c.Query("q"))

	statuses := adminVideoStatusFilter(statusQ)
	result, err := a.VideoSvc.AdminListVideos(c.Request.Context(), statuses, titleQ, page, pageSize)
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
	items := make([]gin.H, 0, len(result.Rows))
	for i := range result.Rows {
		items = append(items, adminVideoToJSON(&result.Rows[i], names[result.Rows[i].UserID]))
	}

	resp.OK(c, gin.H{
		"items":         items,
		"page":          page,
		"page_size":     pageSize,
		"total":         result.Total,
		"total_pages":   totalPages,
		"pending_count": result.PendingCount,
	})
}

// AdminGetVideo GET /api/v1/admin/videos/:id
func (a *API) AdminGetVideo(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	v, err := a.VideoSvc.GetVideoByID(c.Request.Context(), id)
	if err != nil {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	u, _ := a.UserSvc.GetUserByID(c.Request.Context(), v.UserID)
	resp.OK(c, adminVideoToJSON(v, user.DisplayUsername(u)))
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
	v, err := a.VideoSvc.GetVideoByID(c.Request.Context(), id)
	if err != nil {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	if v.Status != video.StatusPendingReview {
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
	v, _ = a.VideoSvc.GetVideoByID(c.Request.Context(), id)
	u, _ := a.UserSvc.GetUserByID(c.Request.Context(), v.UserID)
	a.Log.Info("admin approved video", zap.Uint64("video_id", id), zap.Uint64("admin_id", adminID))
	resp.OK(c, adminVideoToJSON(v, user.DisplayUsername(u)))
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
	v, err := a.VideoSvc.GetVideoByID(c.Request.Context(), id)
	if err != nil {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	if v.Status != video.StatusPendingReview {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	now := time.Now()
	if err := a.VideoSvc.AdminUpdateVideo(c.Request.Context(), id, map[string]interface{}{
		"status":               video.StatusRejected,
		"fail_reason":          reason,
		"reviewed_at":          now,
		"reviewed_by_admin_id": adminID,
	}); err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	if a.ES != nil && a.ES.Enabled() {
		ictx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		_ = a.ES.DeleteVideo(ictx, id)
		cancel()
	}
	v, _ = a.VideoSvc.GetVideoByID(c.Request.Context(), id)
	u, _ := a.UserSvc.GetUserByID(c.Request.Context(), v.UserID)
	a.Log.Info("admin rejected video", zap.Uint64("video_id", id), zap.Uint64("admin_id", adminID))
	resp.OK(c, adminVideoToJSON(v, user.DisplayUsername(u)))
}

// AdminDeleteVideo POST /api/v1/admin/videos/:id/delete or DELETE /api/v1/admin/videos/:id
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
	v, err := a.VideoSvc.GetVideoByID(c.Request.Context(), id)
	if err != nil {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	if v.Status != video.StatusPublished && v.Status != video.StatusRejected {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	removeVideoDraftFiles(*v)
	if err := a.VideoSvc.AdminDeleteVideoCascade(c.Request.Context(), id, func(tx *gorm.DB) error {
		return deleteVideoCascade(tx, id)
	}); err != nil {
		a.Log.Error("admin delete video", zap.Error(err), zap.Uint64("video_id", id))
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	purgeVideoOSSObjects(a.Cfg, a.OSS, a.Log, *v)
	a.esDeleteVideo(id)
	a.Log.Info("admin deleted video",
		zap.Uint64("video_id", id),
		zap.Uint64("admin_id", adminID),
		zap.String("status", v.Status),
	)
	resp.OK(c, gin.H{"ok": true})
}
