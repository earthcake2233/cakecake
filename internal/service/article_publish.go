package service

import (
	"context"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"minibili/internal/model"
	"minibili/internal/search"
)

// PublishArticle marks an article published and indexes search (post-review or direct publish).
func PublishArticle(ctx context.Context, db *gorm.DB, esc *search.Client, log *zap.Logger, articleID uint64, adminID *uint64) error {
	var art model.Article
	if err := db.First(&art, articleID).Error; err != nil {
		return err
	}
	if art.Status == "published" {
		return nil
	}
	now := time.Now()
	updates := map[string]any{
		"status":       "published",
		"published_at": now,
		"reviewed_at":  now,
		"fail_reason":  "",
	}
	if adminID != nil && *adminID > 0 {
		updates["reviewed_by_admin_id"] = *adminID
	}
	if err := db.Model(&art).Updates(updates).Error; err != nil {
		return err
	}
	if esc != nil && esc.Enabled() {
		ictx, cancel := context.WithTimeout(ctx, 15*time.Second)
		defer cancel()
		if err := esc.IndexArticleFromDB(ictx, db, articleID); err != nil && log != nil {
			log.Warn("elasticsearch index article on publish", zap.Uint64("article_id", articleID), zap.Error(err))
		}
	}
	return nil
}
