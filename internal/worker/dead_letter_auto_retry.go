package worker

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"cakecake/internal/model/video"
	"cakecake/internal/queue"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

const (
	deadLetterAutoRetryInterval = time.Minute
	deadLetterAutoRetryMax      = 3
	// deadLetterAutoRetryTotalMax bounds automatic retries across the whole
	// lifetime of a video: every failed replay creates a NEW dead-letter row
	// with count 0, so per-row limits alone would loop forever under a
	// persistent transient failure (e.g. OSS down for hours).
	deadLetterAutoRetryTotalMax = 3
	deadLetterAutoRetryBatch    = 20
	deadLetterAutoRetryBase     = time.Minute
	deadLetterAutoRetryCap      = 15 * time.Minute
)

// autoRetryableKeywords mark transient failures worth an automatic requeue.
// Permanent failures (ffmpeg corruption, unsupported codecs) never match.
var autoRetryableKeywords = []string{
	"下载", "上传", "oss", "OSS",
	"记录转码产物失败", "发布视频失败",
	"timeout", "deadline", "connection", "broker", "network",
}

// autoRetryNonRetryableKeywords are configuration/operator errors that retry
// cannot fix; they must never match the transient keyword list.
var autoRetryNonRetryableKeywords = []string{
	"OSS 未配置", "未配置", "not configured", "configuration",
}

// errAutoRetrySkip rolls back the per-row transaction when another replay is
// already in flight (video is processing); the row is left pending so the
// in-flight job's outcome decides its fate.
var errAutoRetrySkip = errors.New("auto retry skip: replay already in flight")

// errAutoRetryResolve rolls back the per-row transaction when the video is
// deleted or in a terminal state; the row is then marked processed so it can
// be archived instead of dangling as pending forever.
var errAutoRetryResolve = errors.New("auto retry resolve: video not retryable")

// StartTranscodeDeadLetterAutoRetry periodically requeues unresolved dead
// letters whose reason is transient. Each row gets a bounded number of auto
// retries with exponential backoff. Replays are written to the outbox in the
// same DB transaction as the dead-letter/video updates, so publish and state
// changes are atomic; the outbox relay owns reliable delivery.
func StartTranscodeDeadLetterAutoRetry(ctx context.Context, db *gorm.DB, lg *zap.Logger) {
	if db == nil {
		return
	}
	t := time.NewTicker(deadLetterAutoRetryInterval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if _, err := autoRetryDeadLettersOnce(ctx, db, lg); err != nil {
				lg.Warn("auto retry transcode dead letters", zap.Error(err))
			}
		}
	}
}

// autoRetryDeadLettersOnce schedules one batch of auto-retryable dead letters
// and returns how many were scheduled into the outbox.
func autoRetryDeadLettersOnce(ctx context.Context, db *gorm.DB, lg *zap.Logger) (int, error) {
	var rows []video.TranscodeDeadLetter
	if err := db.WithContext(ctx).
		Where("archived_at IS NULL AND processed_at IS NULL AND requeued_at IS NULL").
		Order("id ASC").
		Limit(deadLetterAutoRetryBatch).
		Find(&rows).Error; err != nil {
		return 0, err
	}
	requeued := 0
	resolve := func(rowID uint64, why string) {
		if err := resolveTranscodeDeadLetter(db.WithContext(ctx), rowID); err != nil {
			lg.Warn("resolve transcode dead letter",
				zap.Uint64("dead_letter_id", rowID),
				zap.String("why", why),
				zap.Error(err))
		}
	}
	for i := range rows {
		row := rows[i]
		if !autoRetryableReason(row.Reason) {
			resolve(row.ID, "not transient")
			continue
		}
		if row.AutoRetryCount >= deadLetterAutoRetryMax {
			resolve(row.ID, "per-row budget exhausted")
			continue
		}
		backoff := deadLetterAutoRetryBase << row.AutoRetryCount
		if backoff > deadLetterAutoRetryCap {
			backoff = deadLetterAutoRetryCap
		}
		if row.LastAutoRetryAt != nil && time.Since(*row.LastAutoRetryAt) < backoff {
			continue
		}
		var payload struct {
			Job queue.TranscodeJob `json:"job"`
		}
		if err := json.Unmarshal([]byte(row.PayloadJSON), &payload); err != nil || payload.Job.VideoID == 0 {
			resolve(row.ID, "invalid payload")
			continue
		}
		// Lifetime bound: sum auto retries across all un-archived dead-letter
		// rows of this video. A failed replay produces a fresh row with count
		// 0, so without this check retry rounds would be unbounded.
		var totalAutoRetries int64
		if err := db.WithContext(ctx).Model(&video.TranscodeDeadLetter{}).
			Where("video_id = ? AND archived_at IS NULL", payload.Job.VideoID).
			Select("COALESCE(SUM(auto_retry_count), 0)").
			Scan(&totalAutoRetries).Error; err != nil {
			lg.Warn("sum auto retry budget", zap.Uint64("video_id", payload.Job.VideoID), zap.Error(err))
			continue
		}
		if totalAutoRetries >= deadLetterAutoRetryTotalMax {
			resolve(row.ID, "lifetime budget exhausted")
			continue
		}
		job := payload.Job
		job.RetryCount = 0
		// Fresh JobID: the original attempt's dedup row (job_id, 0) must not
		// swallow this replay. TraceID is preserved for correlation.
		job.JobID = uuid.NewString()
		body, err := json.Marshal(job)
		if err != nil {
			continue
		}
		now := time.Now()
		txErr := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			// P3: never publish for a deleted or terminal video. Check inside
			// the transaction so the state we validated is the state we update.
			var v video.Video
			if err := tx.WithContext(ctx).First(&v, job.VideoID).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return errAutoRetryResolve
				}
				return err
			}
			// Only failed videos are retryable. processing means a replay (or
			// a normal job) is already in flight; scheduling again here would
			// double-transcode the same video in the minute-level window
			// before the in-flight job resolves this row.
			if v.Status != video.StatusFailed {
				if v.Status == video.StatusProcessing {
					return errAutoRetrySkip
				}
				return errAutoRetryResolve
			}
			if err := tx.WithContext(ctx).Create(&video.TranscodeOutbox{
				JobID:   job.JobID,
				VideoID: job.VideoID,
				Payload: string(body),
				Status:  video.OutboxStatusPending,
			}).Error; err != nil {
				return err
			}
			if err := tx.WithContext(ctx).Model(&video.TranscodeDeadLetter{}).
				Where("id = ? AND archived_at IS NULL", row.ID).
				Updates(map[string]interface{}{
					"auto_retry_count":   gorm.Expr("auto_retry_count + 1"),
					"last_auto_retry_at": now,
				}).Error; err != nil {
				return err
			}
			if err := tx.WithContext(ctx).Model(&video.Video{}).
				Where("id = ?", job.VideoID).
				Updates(map[string]interface{}{
					"status":      video.StatusProcessing,
					"fail_reason": "",
					"video_url":   "",
					"cover_url":   "",
				}).Error; err != nil {
				return err
			}
			return tx.WithContext(ctx).Create(&video.TranscodeEvent{
				VideoID:    job.VideoID,
				JobID:      job.JobID,
				FromStatus: video.StatusFailed,
				ToStatus:   video.StatusProcessing,
				Reason:     "auto_retry",
			}).Error
		})
		if errors.Is(txErr, errAutoRetrySkip) {
			continue
		}
		if errors.Is(txErr, errAutoRetryResolve) {
			resolve(row.ID, "video terminal or missing")
			continue
		}
		if txErr != nil {
			lg.Warn("schedule auto retry",
				zap.Uint64("dead_letter_id", row.ID),
				zap.Uint64("video_id", job.VideoID),
				zap.Error(txErr))
			continue
		}
		lg.Info("auto retried transcode dead letter",
			zap.Uint64("dead_letter_id", row.ID),
			zap.Uint64("video_id", job.VideoID),
			zap.Int("attempt", row.AutoRetryCount+1))
		requeued++
	}
	return requeued, nil
}

func autoRetryableReason(reason string) bool {
	for _, k := range autoRetryNonRetryableKeywords {
		if strings.Contains(reason, k) {
			return false
		}
	}
	for _, k := range autoRetryableKeywords {
		if strings.Contains(reason, k) {
			return true
		}
	}
	return false
}
