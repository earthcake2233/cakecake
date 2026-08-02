package handler

import (
	"cakecake/internal/model/user"
	"cakecake/internal/model/video"
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
	"cakecake/internal/pkg/resp"
	"cakecake/internal/service"
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

type adminVideoItem struct {
	ID                uint64     `json:"id"`
	Title             string     `json:"title"`
	Description       string     `json:"description"`
	Status            string     `json:"status"`
	FailReason        string     `json:"fail_reason"`
	CoverURL          string     `json:"cover_url"`
	VideoURL          string     `json:"video_url"`
	DurationSec       float64    `json:"duration_sec"`
	Zone              string     `json:"zone"`
	UserID            uint64     `json:"user_id"`
	UploaderName      string     `json:"uploader_name"`
	PlayCount         uint64     `json:"play_count"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
	ReviewedAt        *time.Time `json:"reviewed_at,omitempty"`
	ReviewedByAdminID *uint64    `json:"reviewed_by_admin_id,omitempty"`
}

type adminVideoListResponse struct {
	Items        []adminVideoItem `json:"items"`
	Page         int              `json:"page"`
	PageSize     int              `json:"page_size"`
	Total        int64            `json:"total"`
	TotalPages   int              `json:"total_pages"`
	PendingCount int64            `json:"pending_count"`
}

func adminVideoToJSON(v *video.Video, uploaderName string) adminVideoItem {
	item := adminVideoItem{
		ID:           v.ID,
		Title:        v.Title,
		Description:  v.Description,
		Status:       v.Status,
		FailReason:   strings.TrimSpace(v.FailReason),
		CoverURL:     v.CoverURL,
		VideoURL:     v.VideoURL,
		DurationSec:  v.DurationSec,
		Zone:         v.Zone,
		UserID:       v.UserID,
		UploaderName: uploaderName,
		PlayCount:    v.PlayCount,
		CreatedAt:    v.CreatedAt,
		UpdatedAt:    v.UpdatedAt,
	}
	item.ReviewedAt = v.ReviewedAt
	item.ReviewedByAdminID = v.ReviewedByAdminID
	return item
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
	items := make([]adminVideoItem, 0, len(result.Rows))
	for i := range result.Rows {
		items = append(items, adminVideoToJSON(&result.Rows[i], names[result.Rows[i].UserID]))
	}

	resp.OK(c, adminVideoListResponse{
		Items:        items,
		Page:         page,
		PageSize:     pageSize,
		Total:        result.Total,
		TotalPages:   totalPages,
		PendingCount: result.PendingCount,
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
	resp.OK(c, okResponse{OK: true})
}
