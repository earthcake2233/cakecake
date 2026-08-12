package handler

import (
	"cakecake/internal/errcode"
	"cakecake/internal/middleware"
	"cakecake/internal/model/video"
	"cakecake/internal/pkg/resp"
	vsvc "cakecake/internal/service/video"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type transcodeDeadLetterItem struct {
	ID            uint64  `json:"id"`
	VideoID       uint64  `json:"video_id"`
	Reason        string  `json:"reason"`
	RetryCount    int     `json:"retry_count"`
	PayloadJSON   string  `json:"payload_json"`
	CreatedAt     string  `json:"created_at"`
	ProcessedAt   *string `json:"processed_at"`
	RequeuedAt    *string `json:"requeued_at"`
	RequeuedCount int     `json:"requeued_count"`
}

func fmtTimePtr(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.Format(time.RFC3339)
	return &s
}

func deadLetterItemFromRow(r *video.TranscodeDeadLetter) transcodeDeadLetterItem {
	return transcodeDeadLetterItem{
		ID:            r.ID,
		VideoID:       r.VideoID,
		Reason:        r.Reason,
		RetryCount:    r.RetryCount,
		PayloadJSON:   r.PayloadJSON,
		CreatedAt:     r.CreatedAt.Format(time.RFC3339),
		ProcessedAt:   fmtTimePtr(r.ProcessedAt),
		RequeuedAt:    fmtTimePtr(r.RequeuedAt),
		RequeuedCount: r.RequeuedCount,
	}
}

// AdminListTranscodeDeadLetters lists dead-letter audit rows.
func (a *API) AdminListTranscodeDeadLetters(c *gin.Context) {
	page, pageSize := parsePagination(c, 20)
	f := vsvc.TranscodeDeadLetterFilter{
		Page:     page,
		PageSize: pageSize,
		Status:   c.Query("status"),
	}
	rows, total, err := a.VideoSvc.ListTranscodeDeadLetters(c.Request.Context(), f)
	if err != nil {
		a.Log.Error("list transcode dead letters", zap.Error(err))
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	items := make([]transcodeDeadLetterItem, 0, len(rows))
	for i := range rows {
		items = append(items, deadLetterItemFromRow(&rows[i]))
	}
	totalPages := 0
	if pageSize > 0 {
		totalPages = int((total + int64(pageSize) - 1) / int64(pageSize))
	}
	resp.OK(c, gin.H{
		"items":       items,
		"page":        page,
		"page_size":   pageSize,
		"total":       total,
		"total_pages": totalPages,
	})
}

// AdminRequeueTranscodeDeadLetter re-publishes a dead letter to the main queue.
func (a *API) AdminRequeueTranscodeDeadLetter(c *gin.Context) {
	if _, ok := middleware.AdminID(c); !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	if err := a.VideoSvc.RequeueTranscodeDeadLetter(c.Request.Context(), id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
			return
		}
		if errors.Is(err, vsvc.ErrRequeueSourceMissing) {
			resp.Err(c, http.StatusConflict, errcode.CodeRequeueSourceMissing)
			return
		}
		a.Log.Error("requeue transcode dead letter", zap.Error(err), zap.Uint64("dead_letter_id", id))
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	resp.OK(c, gin.H{"ok": true, "id": id})
}
