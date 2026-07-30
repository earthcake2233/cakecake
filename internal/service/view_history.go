package service

import (
	"minibili/internal/model/article"
	"minibili/internal/model/extra"
	"minibili/internal/model/user"
	"minibili/internal/model/video"
	"context"
	"sort"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"

)

// ViewHistoryService handles video view history operations.
type ViewHistoryService struct {
	db  *gorm.DB
	rdb *redis.Client
	log *zap.Logger
}

func NewViewHistoryService(db *gorm.DB, rdb *redis.Client, log *zap.Logger) *ViewHistoryService {
	return &ViewHistoryService{db: db, rdb: rdb, log: log}
}

// RecordViewHistory creates or updates a view history entry.
func (s *ViewHistoryService) RecordViewHistory(ctx context.Context, userID, videoID uint64) error {
	var existing extra.VideoViewHistory
	if err := s.db.WithContext(ctx).Where("user_id = ? AND video_id = ?", userID, videoID).First(&existing).Error; err == nil {
		return s.db.WithContext(ctx).Model(&existing).Update("viewed_at", s.db.NowFunc()).Error
	}
	vh := extra.VideoViewHistory{UserID: userID, VideoID: videoID, ViewedAt: s.db.NowFunc()}
	return s.db.WithContext(ctx).Create(&vh).Error
}

// RecordVideoViewHistoryWithProgress creates or updates view history with progress details.
func (s *ViewHistoryService) RecordVideoViewHistoryWithProgress(ctx context.Context, userID, videoID uint64, progressSec, durationSec float64, device string, viewedAt time.Time) error {
	var row extra.VideoViewHistory
	err := s.db.WithContext(ctx).Where("user_id = ? AND video_id = ?", userID, videoID).Limit(1).Find(&row).Error
	if err != nil {
		return err
	}
	if row.ID == 0 {
		row = extra.VideoViewHistory{
			UserID:      userID,
			VideoID:     videoID,
			ProgressSec: progressSec,
			DurationSec: durationSec,
			Device:      device,
			ViewedAt:    viewedAt,
		}
		return s.db.WithContext(ctx).Create(&row).Error
	}
	return s.db.WithContext(ctx).Model(&row).Updates(map[string]interface{}{
		"progress_sec": progressSec,
		"duration_sec": durationSec,
		"device":       device,
		"viewed_at":    viewedAt,
	}).Error
}

// GetUserViewHistoryPaused returns whether the user has view-history recording paused.
func (s *ViewHistoryService) GetUserViewHistoryPaused(ctx context.Context, userID uint64) (bool, error) {
	var u user.User
	if err := s.db.WithContext(ctx).Select("id", "view_history_paused").First(&u, userID).Error; err != nil {
		return false, err
	}
	return u.ViewHistoryPaused, nil
}

// RecordArticleViewHistory creates or updates an article view history entry.
func (s *ViewHistoryService) RecordArticleViewHistory(ctx context.Context, userID, articleID uint64, device string) {
	if userID == 0 || articleID == 0 {
		return
	}
	paused, err := s.GetUserViewHistoryPaused(ctx, userID)
	if err != nil || paused {
		return
	}
	if device != "mobile" {
		device = "web"
	}
	now := time.Now()
	var row extra.ArticleViewHistory
	_ = s.db.WithContext(ctx).Where("user_id = ? AND article_id = ?", userID, articleID).Limit(1).Find(&row).Error
	if row.ID == 0 {
		row = extra.ArticleViewHistory{
			UserID:    userID,
			ArticleID: articleID,
			Device:    device,
			ViewedAt:  now,
		}
		_ = s.db.WithContext(ctx).Create(&row).Error
	} else {
		_ = s.db.WithContext(ctx).Model(&row).Updates(map[string]interface{}{
			"device":    device,
			"viewed_at": now,
		}).Error
	}
}

// TrimViewHistoryCombined trims combined video+article view history to maxItems.
func (s *ViewHistoryService) TrimViewHistoryCombined(ctx context.Context, userID uint64, maxItems int) {
	type dropRec struct {
		kind string
		id   uint64
		at   time.Time
	}
	var recs []dropRec
	var vrows []extra.VideoViewHistory
	_ = s.db.WithContext(ctx).Where("user_id = ?", userID).Find(&vrows).Error
	for i := range vrows {
		recs = append(recs, dropRec{"video", vrows[i].ID, vrows[i].ViewedAt})
	}
	var arows []extra.ArticleViewHistory
	_ = s.db.WithContext(ctx).Where("user_id = ?", userID).Find(&arows).Error
	for i := range arows {
		recs = append(recs, dropRec{"article", arows[i].ID, arows[i].ViewedAt})
	}
	if len(recs) <= maxItems {
		return
	}
	sort.Slice(recs, func(i, j int) bool {
		if recs[i].at.Equal(recs[j].at) {
			return recs[i].id < recs[j].id
		}
		return recs[i].at.Before(recs[j].at)
	})
	excess := len(recs) - maxItems
	for i := 0; i < excess; i++ {
		switch recs[i].kind {
		case "video":
			_ = s.db.WithContext(ctx).Delete(&extra.VideoViewHistory{}, recs[i].id).Error
		case "article":
			_ = s.db.WithContext(ctx).Delete(&extra.ArticleViewHistory{}, recs[i].id).Error
		}
	}
}

// ListViewHistory returns view history with pagination.
func (s *ViewHistoryService) ListViewHistory(ctx context.Context, userID uint64, page, pageSize int) ([]extra.VideoViewHistory, int64, error) {
	var total int64
	_ = s.db.WithContext(ctx).Model(&extra.VideoViewHistory{}).Where("user_id = ?", userID).Count(&total).Error
	offset := (page - 1) * pageSize
	var list []extra.VideoViewHistory
	if err := s.db.WithContext(ctx).Where("user_id = ?", userID).Order("updated_at DESC").Offset(offset).Limit(pageSize).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

// ListVideoViewHistory queries video view history with optional keyword search.
func (s *ViewHistoryService) ListVideoViewHistory(ctx context.Context, userID uint64, keyword string) ([]extra.VideoViewHistory, error) {
	q := s.db.WithContext(ctx).Where("user_id = ?", userID)
	if keyword != "" {
		sub := s.db.WithContext(ctx).Model(&video.Video{}).
			Select("id").
			Where("title LIKE ? OR description LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
		q = q.Where("video_id IN (?)", sub)
	}
	var list []extra.VideoViewHistory
	if err := q.Order("viewed_at DESC, id DESC").Limit(500).Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

// ListArticleViewHistory queries article view history with optional keyword search.
func (s *ViewHistoryService) ListArticleViewHistory(ctx context.Context, userID uint64, keyword string) ([]extra.ArticleViewHistory, error) {
	q := s.db.WithContext(ctx).Where("user_id = ?", userID)
	if keyword != "" {
		sub := s.db.WithContext(ctx).Model(&article.Article{}).
			Select("id").
			Where("title LIKE ? OR content LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
		q = q.Where("article_id IN (?)", sub)
	}
	var list []extra.ArticleViewHistory
	if err := q.Order("viewed_at DESC, id DESC").Limit(500).Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

// BatchFetchVideosByIDs returns a map of video id to Video for the given ids.
func (s *ViewHistoryService) BatchFetchVideosByIDs(ctx context.Context, ids []uint64) (map[uint64]video.Video, error) {
	result := make(map[uint64]video.Video, len(ids))
	if len(ids) == 0 {
		return result, nil
	}
	var list []video.Video
	if err := s.db.WithContext(ctx).Where("id IN ?", ids).Find(&list).Error; err != nil {
		return nil, err
	}
	for i := range list {
		result[list[i].ID] = list[i]
	}
	return result, nil
}

// BatchFetchArticlesByIDs returns a map of article id to Article for the given ids.
func (s *ViewHistoryService) BatchFetchArticlesByIDs(ctx context.Context, ids []uint64) (map[uint64]article.Article, error) {
	result := make(map[uint64]article.Article, len(ids))
	if len(ids) == 0 {
		return result, nil
	}
	var list []article.Article
	if err := s.db.WithContext(ctx).Where("id IN ?", ids).Find(&list).Error; err != nil {
		return nil, err
	}
	for i := range list {
		result[list[i].ID] = list[i]
	}
	return result, nil
}

// BatchFetchUsersByIDs returns a map of user id to User for the given ids.
func (s *ViewHistoryService) BatchFetchUsersByIDs(ctx context.Context, ids []uint64) (map[uint64]user.User, error) {
	result := make(map[uint64]user.User, len(ids))
	if len(ids) == 0 {
		return result, nil
	}
	var list []user.User
	if err := s.db.WithContext(ctx).Where("id IN ?", ids).Find(&list).Error; err != nil {
		return nil, err
	}
	for i := range list {
		result[list[i].ID] = list[i]
	}
	return result, nil
}

// DeleteViewHistoryEntry removes a single history entry.
func (s *ViewHistoryService) DeleteViewHistoryEntry(ctx context.Context, userID, historyID uint64) error {
	return s.db.WithContext(ctx).Where("id = ? AND user_id = ?", historyID, userID).Delete(&extra.VideoViewHistory{}).Error
}

// DeleteVideoHistoryByVideo removes view history for a specific video.
func (s *ViewHistoryService) DeleteVideoHistoryByVideo(ctx context.Context, userID, videoID uint64) error {
	return s.db.WithContext(ctx).Where("user_id = ? AND video_id = ?", userID, videoID).Delete(&extra.VideoViewHistory{}).Error
}

// DeleteArticleHistoryByArticle removes view history for a specific article.
func (s *ViewHistoryService) DeleteArticleHistoryByArticle(ctx context.Context, userID, articleID uint64) error {
	return s.db.WithContext(ctx).Where("user_id = ? AND article_id = ?", userID, articleID).Delete(&extra.ArticleViewHistory{}).Error
}

// ClearViewHistory removes all history entries for a user.
func (s *ViewHistoryService) ClearViewHistory(ctx context.Context, userID uint64) error {
	return s.db.WithContext(ctx).Where("user_id = ?", userID).Delete(&extra.VideoViewHistory{}).Error
}

// ClearArticleViewHistory removes all article history entries for a user.
func (s *ViewHistoryService) ClearArticleViewHistory(ctx context.Context, userID uint64) error {
	return s.db.WithContext(ctx).Where("user_id = ?", userID).Delete(&extra.ArticleViewHistory{}).Error
}

// UpdateViewHistorySettings updates view history settings.
func (s *ViewHistoryService) UpdateViewHistorySettings(ctx context.Context, userID uint64, paused bool) error {
	return s.db.WithContext(ctx).Model(&user.User{}).Where("id = ?", userID).Update("view_history_paused", paused).Error
}

// GetViewHistorySettings returns view history settings.
func (s *ViewHistoryService) GetViewHistorySettings(ctx context.Context, userID uint64) (bool, error) {
	var u user.User
	if err := s.db.WithContext(ctx).Select("view_history_paused").First(&u, userID).Error; err != nil {
		return false, err
	}
	return u.ViewHistoryPaused, nil
}


