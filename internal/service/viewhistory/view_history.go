package viewhistory

import (
	"cakecake/internal/model/article"
	"cakecake/internal/model/extra"
	"cakecake/internal/model/user"
	"cakecake/internal/model/video"
	"context"
	"sort"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// ViewHistoryService handles video view history operations.
type ViewHistoryService struct {
	store ViewHistoryStore
	rdb   *redis.Client
	log   *zap.Logger
}

func NewViewHistoryService(db *gorm.DB, rdb *redis.Client, log *zap.Logger) *ViewHistoryService {
	return &ViewHistoryService{store: NewViewHistoryStore(db), rdb: rdb, log: log}
}

// RecordViewHistory creates or updates a view history entry.
func (s *ViewHistoryService) RecordViewHistory(ctx context.Context, userID, videoID uint64) error {
	return s.store.UpsertVideoViewHistory(ctx, userID, videoID)
}

// RecordVideoViewHistoryWithProgress creates or updates view history with progress details.
func (s *ViewHistoryService) RecordVideoViewHistoryWithProgress(ctx context.Context, userID, videoID uint64, progressSec, durationSec float64, device string, viewedAt time.Time) error {
	return s.store.UpsertVideoViewHistoryWithProgress(ctx, userID, videoID, progressSec, durationSec, device, viewedAt)
}

// GetUserViewHistoryPaused returns whether the user has view-history recording paused.
func (s *ViewHistoryService) GetUserViewHistoryPaused(ctx context.Context, userID uint64) (bool, error) {
	return s.store.GetUserViewHistoryPaused(ctx, userID)
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
	_ = s.store.UpsertArticleViewHistory(ctx, userID, articleID, device, now)
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
	vrows, _ = s.store.ListVideoViewHistoryRows(ctx, userID)
	for i := range vrows {
		recs = append(recs, dropRec{"video", vrows[i].ID, vrows[i].ViewedAt})
	}
	var arows []extra.ArticleViewHistory
	arows, _ = s.store.ListArticleViewHistoryRows(ctx, userID)
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
			_ = s.store.DeleteVideoViewHistoryByID(ctx, recs[i].id)
		case "article":
			_ = s.store.DeleteArticleViewHistoryByID(ctx, recs[i].id)
		}
	}
}

// ListViewHistory returns view history with pagination.
func (s *ViewHistoryService) ListViewHistory(ctx context.Context, userID uint64, page, pageSize int) ([]extra.VideoViewHistory, int64, error) {
	return s.store.ListViewHistory(ctx, userID, page, pageSize)
}

// ListVideoViewHistory queries video view history with optional keyword search.
func (s *ViewHistoryService) ListVideoViewHistory(ctx context.Context, userID uint64, keyword string) ([]extra.VideoViewHistory, error) {
	return s.store.ListVideoViewHistory(ctx, userID, keyword)
}

// ListArticleViewHistory queries article view history with optional keyword search.
func (s *ViewHistoryService) ListArticleViewHistory(ctx context.Context, userID uint64, keyword string) ([]extra.ArticleViewHistory, error) {
	return s.store.ListArticleViewHistory(ctx, userID, keyword)
}

// BatchFetchVideosByIDs returns a map of video id to Video for the given ids.
func (s *ViewHistoryService) BatchFetchVideosByIDs(ctx context.Context, ids []uint64) (map[uint64]video.Video, error) {
	return s.store.BatchFetchVideosByIDs(ctx, ids)
}

// BatchFetchArticlesByIDs returns a map of article id to Article for the given ids.
func (s *ViewHistoryService) BatchFetchArticlesByIDs(ctx context.Context, ids []uint64) (map[uint64]article.Article, error) {
	return s.store.BatchFetchArticlesByIDs(ctx, ids)
}

// BatchFetchUsersByIDs returns a map of user id to User for the given ids.
func (s *ViewHistoryService) BatchFetchUsersByIDs(ctx context.Context, ids []uint64) (map[uint64]user.User, error) {
	return s.store.BatchFetchUsersByIDs(ctx, ids)
}

// DeleteViewHistoryEntry removes a single history entry.
func (s *ViewHistoryService) DeleteViewHistoryEntry(ctx context.Context, userID, historyID uint64) error {
	return s.store.DeleteViewHistoryEntry(ctx, userID, historyID)
}

// DeleteVideoHistoryByVideo removes view history for a specific video.
func (s *ViewHistoryService) DeleteVideoHistoryByVideo(ctx context.Context, userID, videoID uint64) error {
	return s.store.DeleteVideoHistoryByVideo(ctx, userID, videoID)
}

// DeleteArticleHistoryByArticle removes view history for a specific article.
func (s *ViewHistoryService) DeleteArticleHistoryByArticle(ctx context.Context, userID, articleID uint64) error {
	return s.store.DeleteArticleHistoryByArticle(ctx, userID, articleID)
}

// ClearViewHistory removes all history entries for a user.
func (s *ViewHistoryService) ClearViewHistory(ctx context.Context, userID uint64) error {
	return s.store.ClearViewHistory(ctx, userID)
}

// ClearArticleViewHistory removes all article history entries for a user.
func (s *ViewHistoryService) ClearArticleViewHistory(ctx context.Context, userID uint64) error {
	return s.store.ClearArticleViewHistory(ctx, userID)
}

// UpdateViewHistorySettings updates view history settings.
func (s *ViewHistoryService) UpdateViewHistorySettings(ctx context.Context, userID uint64, paused bool) error {
	return s.store.UpdateViewHistorySettings(ctx, userID, paused)
}

// GetViewHistorySettings returns view history settings.
func (s *ViewHistoryService) GetViewHistorySettings(ctx context.Context, userID uint64) (bool, error) {
	return s.store.GetViewHistorySettings(ctx, userID)
}
