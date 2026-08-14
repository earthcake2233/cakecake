package worker

import (
	"context"
	"time"

	"cakecake/internal/model/video"
	"cakecake/internal/queue"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

const (
	outboxRelayInterval = time.Second
	outboxBatchSize     = 50
	outboxMaxBackoff    = 60 * time.Second
)

// StartTranscodeOutboxRelay polls the local message table and publishes
// pending jobs to RabbitMQ with publisher confirm. A row is marked sent only
// after confirmation; failures keep the row pending with exponential backoff.
func StartTranscodeOutboxRelay(ctx context.Context, db *gorm.DB, pub queue.TranscodePublisher, lg *zap.Logger) {
	if db == nil || pub == nil {
		return
	}
	t := time.NewTicker(outboxRelayInterval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if _, err := relayTranscodeOutboxOnce(ctx, db, pub, lg); err != nil {
				lg.Warn("relay transcode outbox", zap.Error(err))
			}
		}
	}
}

// relayTranscodeOutboxOnce publishes one batch of pending outbox rows and
// returns the number of rows marked sent.
func relayTranscodeOutboxOnce(ctx context.Context, db *gorm.DB, pub queue.TranscodePublisher, lg *zap.Logger) (int, error) {
	var rows []video.TranscodeOutbox
	if err := db.WithContext(ctx).
		Where("status = ? AND (next_retry_at IS NULL OR next_retry_at <= ?)", video.OutboxStatusPending, time.Now()).
		Order("id ASC").
		Limit(outboxBatchSize).
		Find(&rows).Error; err != nil {
		return 0, err
	}
	sent := 0
	for i := range rows {
		row := rows[i]
		if err := pub.PublishTranscode(ctx, []byte(row.Payload)); err != nil {
			lg.Warn("relay publish failed",
				zap.String("job_id", row.JobID),
				zap.Uint64("video_id", row.VideoID),
				zap.Error(err))
			attempts := row.Attempts + 1
			// Saturate the exponent: 1<<63 overflows and would make the
			// backoff negative, causing an immediate retry busy-loop.
			if attempts > 30 {
				attempts = 30
			}
			backoff := time.Duration(1<<attempts) * time.Second
			if backoff > outboxMaxBackoff {
				backoff = outboxMaxBackoff
			}
			if uerr := db.Model(&video.TranscodeOutbox{}).
				Where("id = ? AND status = ?", row.ID, video.OutboxStatusPending).
				Updates(map[string]interface{}{
					"attempts":      attempts,
					"next_retry_at": time.Now().Add(backoff),
				}).Error; uerr != nil {
				lg.Warn("update outbox retry state", zap.String("job_id", row.JobID), zap.Error(uerr))
			}
			continue
		}
		res := db.Model(&video.TranscodeOutbox{}).
			Where("id = ? AND status = ?", row.ID, video.OutboxStatusPending).
			Updates(map[string]interface{}{
				"status":  video.OutboxStatusSent,
				"sent_at": time.Now(),
			})
		if res.Error != nil {
			lg.Warn("mark outbox sent", zap.String("job_id", row.JobID), zap.Error(res.Error))
			continue
		}
		if res.RowsAffected == 1 {
			sent++
		}
	}
	return sent, nil
}
