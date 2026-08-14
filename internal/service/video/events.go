package video

import (
	"context"
	"strings"

	"cakecake/internal/logger"
	"cakecake/internal/model/video"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// RecordTranscodeEvent appends an append-only audit row for a status change.
// Audit writes are best-effort: a failure is logged, never fatal to the
// transcode pipeline.
func RecordTranscodeEvent(ctx context.Context, db *gorm.DB, videoID uint64, jobID, from, to, reason string) {
	if db == nil {
		return
	}
	ev := video.TranscodeEvent{
		VideoID:    videoID,
		JobID:      jobID,
		FromStatus: from,
		ToStatus:   to,
		Reason:     runeTruncate1900(strings.TrimSpace(reason)),
	}
	if err := db.WithContext(ctx).Create(&ev).Error; err != nil {
		if logger.L != nil {
			logger.L.Warn("record transcode event",
				zap.Uint64("video_id", videoID),
				zap.String("from", from),
				zap.String("to", to),
				zap.Error(err))
		}
	}
}

func runeTruncate1900(s string) string {
	r := []rune(s)
	if len(r) <= 1900 {
		return s
	}
	return string(r[:1900])
}
