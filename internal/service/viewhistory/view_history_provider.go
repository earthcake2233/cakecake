package viewhistory

import (
	"cakecake/internal/model/article"
	"cakecake/internal/model/extra"
	"cakecake/internal/model/user"
	"cakecake/internal/model/video"
	"context"
	"time"

	"gorm.io/gorm"
)

// ViewHistoryStore is the view-history storage boundary.
// Phase 1: *gorm.DB impl. Phase 2+: replaced by gRPC client / per-domain store.
type ViewHistoryStore interface {
	UpsertVideoViewHistory(ctx context.Context, userID, videoID uint64) error
	UpsertVideoViewHistoryWithProgress(ctx context.Context, userID, videoID uint64, progressSec, durationSec float64, device string, viewedAt time.Time) error
	GetUserViewHistoryPaused(ctx context.Context, userID uint64) (bool, error)
	UpsertArticleViewHistory(ctx context.Context, userID, articleID uint64, device string, viewedAt time.Time) error
	ListVideoViewHistoryRows(ctx context.Context, userID uint64) ([]extra.VideoViewHistory, error)
	ListArticleViewHistoryRows(ctx context.Context, userID uint64) ([]extra.ArticleViewHistory, error)
	DeleteVideoViewHistoryByID(ctx context.Context, id uint64) error
	DeleteArticleViewHistoryByID(ctx context.Context, id uint64) error
	ListViewHistory(ctx context.Context, userID uint64, page, pageSize int) ([]extra.VideoViewHistory, int64, error)
	ListVideoViewHistory(ctx context.Context, userID uint64, keyword string) ([]extra.VideoViewHistory, error)
	ListArticleViewHistory(ctx context.Context, userID uint64, keyword string) ([]extra.ArticleViewHistory, error)
	BatchFetchVideosByIDs(ctx context.Context, ids []uint64) (map[uint64]video.Video, error)
	BatchFetchArticlesByIDs(ctx context.Context, ids []uint64) (map[uint64]article.Article, error)
	BatchFetchUsersByIDs(ctx context.Context, ids []uint64) (map[uint64]user.User, error)
	DeleteViewHistoryEntry(ctx context.Context, userID, historyID uint64) error
	DeleteVideoHistoryByVideo(ctx context.Context, userID, videoID uint64) error
	DeleteArticleHistoryByArticle(ctx context.Context, userID, articleID uint64) error
	ClearViewHistory(ctx context.Context, userID uint64) error
	ClearArticleViewHistory(ctx context.Context, userID uint64) error
	UpdateViewHistorySettings(ctx context.Context, userID uint64, paused bool) error
	GetViewHistorySettings(ctx context.Context, userID uint64) (bool, error)
}

// ViewHistoryStoreImpl implements ViewHistoryStore using *gorm.DB (Phase 1 monolith).
type ViewHistoryStoreImpl struct {
	db *gorm.DB
}

var _ ViewHistoryStore = (*ViewHistoryStoreImpl)(nil)

// NewViewHistoryStore creates a gorm-backed ViewHistoryStore implementation.
func NewViewHistoryStore(db *gorm.DB) *ViewHistoryStoreImpl {
	return &ViewHistoryStoreImpl{db: db}
}

// UpsertVideoViewHistory records a video view, refreshing the viewed timestamp on repeats.
func (p *ViewHistoryStoreImpl) UpsertVideoViewHistory(ctx context.Context, userID, videoID uint64) error {
	var existing extra.VideoViewHistory
	if err := p.db.WithContext(ctx).Where("user_id = ? AND video_id = ?", userID, videoID).First(&existing).Error; err == nil {
		return p.db.WithContext(ctx).Model(&existing).Update("viewed_at", p.db.NowFunc()).Error
	}
	vh := extra.VideoViewHistory{UserID: userID, VideoID: videoID, ViewedAt: p.db.NowFunc()}
	return p.db.WithContext(ctx).Create(&vh).Error
}

// UpsertVideoViewHistoryWithProgress records a video view with playback progress and device.
func (p *ViewHistoryStoreImpl) UpsertVideoViewHistoryWithProgress(ctx context.Context, userID, videoID uint64, progressSec, durationSec float64, device string, viewedAt time.Time) error {
	var row extra.VideoViewHistory
	err := p.db.WithContext(ctx).Where("user_id = ? AND video_id = ?", userID, videoID).Limit(1).Find(&row).Error
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
		return p.db.WithContext(ctx).Create(&row).Error
	}
	return p.db.WithContext(ctx).Model(&row).Updates(map[string]interface{}{
		"progress_sec": progressSec,
		"duration_sec": durationSec,
		"device":       device,
		"viewed_at":    viewedAt,
	}).Error
}

// GetUserViewHistoryPaused returns whether the user paused view-history recording.
func (p *ViewHistoryStoreImpl) GetUserViewHistoryPaused(ctx context.Context, userID uint64) (bool, error) {
	var u user.User
	if err := p.db.WithContext(ctx).Select("id", "view_history_paused").First(&u, userID).Error; err != nil {
		return false, err
	}
	return u.ViewHistoryPaused, nil
}

// UpsertArticleViewHistory records an article view.
func (p *ViewHistoryStoreImpl) UpsertArticleViewHistory(ctx context.Context, userID, articleID uint64, device string, viewedAt time.Time) error {
	var row extra.ArticleViewHistory
	_ = p.db.WithContext(ctx).Where("user_id = ? AND article_id = ?", userID, articleID).Limit(1).Find(&row).Error
	if row.ID == 0 {
		row = extra.ArticleViewHistory{
			UserID:    userID,
			ArticleID: articleID,
			Device:    device,
			ViewedAt:  viewedAt,
		}
		return p.db.WithContext(ctx).Create(&row).Error
	}
	return p.db.WithContext(ctx).Model(&row).Updates(map[string]interface{}{
		"device":    device,
		"viewed_at": viewedAt,
	}).Error
}

// ListVideoViewHistoryRows lists a user's video view-history rows.
func (p *ViewHistoryStoreImpl) ListVideoViewHistoryRows(ctx context.Context, userID uint64) ([]extra.VideoViewHistory, error) {
	var rows []extra.VideoViewHistory
	if err := p.db.WithContext(ctx).Where("user_id = ?", userID).Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// ListArticleViewHistoryRows lists a user's article view-history rows.
func (p *ViewHistoryStoreImpl) ListArticleViewHistoryRows(ctx context.Context, userID uint64) ([]extra.ArticleViewHistory, error) {
	var rows []extra.ArticleViewHistory
	if err := p.db.WithContext(ctx).Where("user_id = ?", userID).Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// DeleteVideoViewHistoryByID removes a video view-history row by id.
func (p *ViewHistoryStoreImpl) DeleteVideoViewHistoryByID(ctx context.Context, id uint64) error {
	return p.db.WithContext(ctx).Delete(&extra.VideoViewHistory{}, id).Error
}

// DeleteArticleViewHistoryByID removes an article view-history row by id.
func (p *ViewHistoryStoreImpl) DeleteArticleViewHistoryByID(ctx context.Context, id uint64) error {
	return p.db.WithContext(ctx).Delete(&extra.ArticleViewHistory{}, id).Error
}

// ListViewHistory pages a user's video view-history rows.
func (p *ViewHistoryStoreImpl) ListViewHistory(ctx context.Context, userID uint64, page, pageSize int) ([]extra.VideoViewHistory, int64, error) {
	var total int64
	_ = p.db.WithContext(ctx).Model(&extra.VideoViewHistory{}).Where("user_id = ?", userID).Count(&total).Error
	offset := (page - 1) * pageSize
	var list []extra.VideoViewHistory
	if err := p.db.WithContext(ctx).Where("user_id = ?", userID).Order("updated_at DESC").Offset(offset).Limit(pageSize).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

// ListVideoViewHistory returns a user's video view-history rows (keyword-filtered).
func (p *ViewHistoryStoreImpl) ListVideoViewHistory(ctx context.Context, userID uint64, keyword string) ([]extra.VideoViewHistory, error) {
	q := p.db.WithContext(ctx).Where("user_id = ?", userID)
	if keyword != "" {
		sub := p.db.WithContext(ctx).Model(&video.Video{}).
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

// ListArticleViewHistory returns a user's article view-history rows (keyword-filtered).
func (p *ViewHistoryStoreImpl) ListArticleViewHistory(ctx context.Context, userID uint64, keyword string) ([]extra.ArticleViewHistory, error) {
	q := p.db.WithContext(ctx).Where("user_id = ?", userID)
	if keyword != "" {
		sub := p.db.WithContext(ctx).Model(&article.Article{}).
			Select("id").
			Where("title LIKE ? OR body_md LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
		q = q.Where("article_id IN (?)", sub)
	}
	var list []extra.ArticleViewHistory
	if err := q.Order("viewed_at DESC, id DESC").Limit(500).Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

// BatchFetchVideosByIDs loads video rows by ids for history enrichment.
func (p *ViewHistoryStoreImpl) BatchFetchVideosByIDs(ctx context.Context, ids []uint64) (map[uint64]video.Video, error) {
	result := make(map[uint64]video.Video, len(ids))
	if len(ids) == 0 {
		return result, nil
	}
	var list []video.Video
	if err := p.db.WithContext(ctx).Where("id IN ?", ids).Find(&list).Error; err != nil {
		return nil, err
	}
	for i := range list {
		result[list[i].ID] = list[i]
	}
	return result, nil
}

// BatchFetchArticlesByIDs loads article rows by ids for history enrichment.
func (p *ViewHistoryStoreImpl) BatchFetchArticlesByIDs(ctx context.Context, ids []uint64) (map[uint64]article.Article, error) {
	result := make(map[uint64]article.Article, len(ids))
	if len(ids) == 0 {
		return result, nil
	}
	var list []article.Article
	if err := p.db.WithContext(ctx).Where("id IN ?", ids).Find(&list).Error; err != nil {
		return nil, err
	}
	for i := range list {
		result[list[i].ID] = list[i]
	}
	return result, nil
}

// BatchFetchUsersByIDs loads user rows by ids for history enrichment.
func (p *ViewHistoryStoreImpl) BatchFetchUsersByIDs(ctx context.Context, ids []uint64) (map[uint64]user.User, error) {
	result := make(map[uint64]user.User, len(ids))
	if len(ids) == 0 {
		return result, nil
	}
	var list []user.User
	if err := p.db.WithContext(ctx).Where("id IN ?", ids).Find(&list).Error; err != nil {
		return nil, err
	}
	for i := range list {
		result[list[i].ID] = list[i]
	}
	return result, nil
}

// DeleteViewHistoryEntry removes one of the user's view-history rows.
func (p *ViewHistoryStoreImpl) DeleteViewHistoryEntry(ctx context.Context, userID, historyID uint64) error {
	return p.db.WithContext(ctx).Where("id = ? AND user_id = ?", historyID, userID).Delete(&extra.VideoViewHistory{}).Error
}

// DeleteVideoHistoryByVideo removes a user's view-history rows for a video.
func (p *ViewHistoryStoreImpl) DeleteVideoHistoryByVideo(ctx context.Context, userID, videoID uint64) error {
	return p.db.WithContext(ctx).Where("user_id = ? AND video_id = ?", userID, videoID).Delete(&extra.VideoViewHistory{}).Error
}

// DeleteArticleHistoryByArticle removes a user's view-history rows for an article.
func (p *ViewHistoryStoreImpl) DeleteArticleHistoryByArticle(ctx context.Context, userID, articleID uint64) error {
	return p.db.WithContext(ctx).Where("user_id = ? AND article_id = ?", userID, articleID).Delete(&extra.ArticleViewHistory{}).Error
}

// ClearViewHistory removes all of a user's video view-history rows.
func (p *ViewHistoryStoreImpl) ClearViewHistory(ctx context.Context, userID uint64) error {
	return p.db.WithContext(ctx).Where("user_id = ?", userID).Delete(&extra.VideoViewHistory{}).Error
}

// ClearArticleViewHistory removes all of a user's article view-history rows.
func (p *ViewHistoryStoreImpl) ClearArticleViewHistory(ctx context.Context, userID uint64) error {
	return p.db.WithContext(ctx).Where("user_id = ?", userID).Delete(&extra.ArticleViewHistory{}).Error
}

// UpdateViewHistorySettings persists the user's view-history pause setting.
func (p *ViewHistoryStoreImpl) UpdateViewHistorySettings(ctx context.Context, userID uint64, paused bool) error {
	return p.db.WithContext(ctx).Model(&user.User{}).Where("id = ?", userID).Update("view_history_paused", paused).Error
}

// GetViewHistorySettings loads the user's view-history pause setting.
func (p *ViewHistoryStoreImpl) GetViewHistorySettings(ctx context.Context, userID uint64) (bool, error) {
	var u user.User
	if err := p.db.WithContext(ctx).Select("view_history_paused").First(&u, userID).Error; err != nil {
		return false, err
	}
	return u.ViewHistoryPaused, nil
}
