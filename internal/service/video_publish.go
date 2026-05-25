package service

import (
	"context"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"minibili/internal/model"
	"minibili/internal/search"
)

// PublishVideo marks a video published and indexes search (post-review or direct publish).
func PublishVideo(ctx context.Context, db *gorm.DB, esc *search.Client, log *zap.Logger, videoID uint64, adminID *uint64) error {
	var v model.Video
	if err := db.First(&v, videoID).Error; err != nil {
		return err
	}
	if v.Status == "published" {
		return nil
	}
	now := time.Now()
	updates := map[string]any{
		"status":      "published",
		"reviewed_at": now,
	}
	if adminID != nil && *adminID > 0 {
		updates["reviewed_by_admin_id"] = *adminID
	}
	if err := db.Model(&v).Updates(updates).Error; err != nil {
		return err
	}
	_ = db.Model(&model.User{}).
		Where("id = ? AND first_published_at IS NULL", v.UserID).
		Update("first_published_at", v.CreatedAt).Error
	if esc != nil && esc.Enabled() {
		ictx, cancel := context.WithTimeout(ctx, 15*time.Second)
		defer cancel()
		if err := esc.IndexVideoFromDB(ictx, db, videoID); err != nil && log != nil {
			log.Warn("elasticsearch index video on publish", zap.Uint64("video_id", videoID), zap.Error(err))
		}
	}
	return nil
}
