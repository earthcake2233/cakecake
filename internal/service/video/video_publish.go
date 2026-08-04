package video

import (
	"context"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"cakecake/internal/search"
)

// PublishVideo marks a video published and indexes search (post-review or direct publish).
func PublishVideo(ctx context.Context, db *gorm.DB, esc *search.Client, log *zap.Logger, videoID uint64, adminID *uint64) error {
	return NewVideoProvider(db).PublishVideo(ctx, esc, log, videoID, adminID)
}
