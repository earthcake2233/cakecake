package service

import (
	"cakecake/internal/model/article"
	"cakecake/internal/model/extra"
	"cakecake/internal/model/user"
	"cakecake/internal/model/video"
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func newViewHistoryService(t *testing.T) *ViewHistoryService {
	t.Helper()
	db := newAgentTestDB(t)
	_, rdb := newAgentTestRedis(t)
	return NewViewHistoryService(db, rdb, zapNop())
}

func TestViewHistory_RecordAndList(t *testing.T) {
	s := newViewHistoryService(t)
	ctx := context.Background()

	require.NoError(t, s.RecordViewHistory(ctx, 1, 10))
	require.NoError(t, s.RecordViewHistory(ctx, 1, 10)) // update path
	require.NoError(t, s.RecordViewHistory(ctx, 1, 11))

	list, total, err := s.ListViewHistory(ctx, 1, 1, 10)
	require.NoError(t, err)
	require.Equal(t, int64(2), total)
	require.Len(t, list, 2)

	var n int64
	require.NoError(t, s.db.Model(&extra.VideoViewHistory{}).Where("user_id = ?", 1).Count(&n).Error)
	require.Equal(t, int64(2), n)
}

func TestViewHistory_RecordWithProgress(t *testing.T) {
	s := newViewHistoryService(t)
	ctx := context.Background()
	at := time.Now()

	require.NoError(t, s.RecordVideoViewHistoryWithProgress(ctx, 1, 10, 12.5, 100, "mobile", at))
	require.NoError(t, s.RecordVideoViewHistoryWithProgress(ctx, 1, 10, 50, 100, "web", at.Add(time.Minute)))

	var row extra.VideoViewHistory
	require.NoError(t, s.db.Where("user_id = ? AND video_id = ?", 1, 10).First(&row).Error)
	require.Equal(t, float64(50), row.ProgressSec)
	require.Equal(t, "web", row.Device)
}

func TestViewHistory_PausedSettings(t *testing.T) {
	s := newViewHistoryService(t)
	ctx := context.Background()
	seedUser(t, s.db, 1, "alice")

	paused, err := s.GetUserViewHistoryPaused(ctx, 1)
	require.NoError(t, err)
	require.False(t, paused)

	require.NoError(t, s.UpdateViewHistorySettings(ctx, 1, true))
	paused, err = s.GetViewHistorySettings(ctx, 1)
	require.NoError(t, err)
	require.True(t, paused)
	paused, err = s.GetUserViewHistoryPaused(ctx, 1)
	require.NoError(t, err)
	require.True(t, paused)
}

func TestViewHistory_RecordArticle(t *testing.T) {
	s := newViewHistoryService(t)
	ctx := context.Background()

	// userID 0 -> no-op.
	s.RecordArticleViewHistory(ctx, 0, 5, "mobile")
	var n int64
	require.NoError(t, s.db.Model(&extra.ArticleViewHistory{}).Count(&n).Error)
	require.Zero(t, n)

	seedUser(t, s.db, 1, "alice")
	s.RecordArticleViewHistory(ctx, 1, 5, "mobile")
	s.RecordArticleViewHistory(ctx, 1, 5, "desktop") // update path, device normalized to web

	var row extra.ArticleViewHistory
	require.NoError(t, s.db.Where("user_id = ? AND article_id = ?", 1, 5).First(&row).Error)
	require.Equal(t, "web", row.Device)

	// Paused user -> no record.
	require.NoError(t, s.db.Model(&user.User{}).Where("id = ?", 1).Update("view_history_paused", true).Error)
	s.RecordArticleViewHistory(ctx, 1, 6, "web")
	require.NoError(t, s.db.Model(&extra.ArticleViewHistory{}).Where("article_id = ?", 6).Count(&n).Error)
	require.Zero(t, n)
}

func TestViewHistory_TrimCombined(t *testing.T) {
	s := newViewHistoryService(t)
	ctx := context.Background()
	base := time.Now().Add(-time.Hour)
	for i := 0; i < 3; i++ {
		require.NoError(t, s.db.Create(&extra.VideoViewHistory{
			UserID: 1, VideoID: uint64(10 + i), ViewedAt: base.Add(time.Duration(i) * time.Minute),
		}).Error)
		require.NoError(t, s.db.Create(&extra.ArticleViewHistory{
			UserID: 1, ArticleID: uint64(20 + i), ViewedAt: base.Add(time.Duration(i) * time.Minute),
		}).Error)
	}
	s.TrimViewHistoryCombined(ctx, 1, 4)
	var vn, an int64
	require.NoError(t, s.db.Model(&extra.VideoViewHistory{}).Where("user_id = ?", 1).Count(&vn).Error)
	require.NoError(t, s.db.Model(&extra.ArticleViewHistory{}).Where("user_id = ?", 1).Count(&an).Error)
	require.Equal(t, int64(4), vn+an)
}

func TestViewHistory_KeywordSearch(t *testing.T) {
	s := newViewHistoryService(t)
	ctx := context.Background()

	require.NoError(t, s.db.Create(&video.Video{ID: 10, Title: "golang tutorial", Status: video.StatusPublished}).Error)
	require.NoError(t, s.db.Create(&article.Article{ID: 20, Title: "golang notes", Status: article.StatusPublished}).Error)
	require.NoError(t, s.db.Create(&extra.VideoViewHistory{UserID: 1, VideoID: 10, ViewedAt: time.Now()}).Error)
	require.NoError(t, s.db.Create(&extra.ArticleViewHistory{UserID: 1, ArticleID: 20, ViewedAt: time.Now()}).Error)

	vrows, err := s.ListVideoViewHistory(ctx, 1, "golang")
	require.NoError(t, err)
	require.Len(t, vrows, 1)

	arows, err := s.ListArticleViewHistory(ctx, 1, "golang")
	require.NoError(t, err)
	require.Len(t, arows, 1)

	arows, err = s.ListArticleViewHistory(ctx, 1, "")
	require.NoError(t, err)
	require.Len(t, arows, 1)

	// Video: no keyword -> everything.
	vrows, err = s.ListVideoViewHistory(ctx, 1, "")
	require.NoError(t, err)
	require.Len(t, vrows, 1)
}

func TestViewHistory_BatchFetches(t *testing.T) {
	s := newViewHistoryService(t)
	ctx := context.Background()

	require.NoError(t, s.db.Create(&video.Video{ID: 10, Title: "v", Status: video.StatusPublished}).Error)
	require.NoError(t, s.db.Create(&article.Article{ID: 20, Title: "a", Status: article.StatusPublished}).Error)
	seedUser(t, s.db, 30, "alice")

	vids, err := s.BatchFetchVideosByIDs(ctx, []uint64{10, 99})
	require.NoError(t, err)
	require.Contains(t, vids, uint64(10))
	require.NotContains(t, vids, uint64(99))
	empty, err := s.BatchFetchVideosByIDs(ctx, nil)
	require.NoError(t, err)
	require.Empty(t, empty)

	aids, err := s.BatchFetchArticlesByIDs(ctx, []uint64{20})
	require.NoError(t, err)
	require.Contains(t, aids, uint64(20))
	uids, err := s.BatchFetchUsersByIDs(ctx, []uint64{30})
	require.NoError(t, err)
	require.Contains(t, uids, uint64(30))
}

func TestViewHistory_DeleteAndClear(t *testing.T) {
	s := newViewHistoryService(t)
	ctx := context.Background()

	seedUser(t, s.db, 1, "alice")
	require.NoError(t, s.RecordViewHistory(ctx, 1, 10))
	require.NoError(t, s.RecordViewHistory(ctx, 1, 11))
	s.RecordArticleViewHistory(ctx, 1, 20, "web")

	list, _, err := s.ListViewHistory(ctx, 1, 1, 10)
	require.NoError(t, err)
	require.Len(t, list, 2)

	// Ownership-protected delete.
	require.NoError(t, s.DeleteViewHistoryEntry(ctx, 2, list[0].ID))
	list, _, err = s.ListViewHistory(ctx, 1, 1, 10)
	require.NoError(t, err)
	require.Len(t, list, 2)
	require.NoError(t, s.DeleteViewHistoryEntry(ctx, 1, list[0].ID))

	require.NoError(t, s.DeleteVideoHistoryByVideo(ctx, 1, 11))
	require.NoError(t, s.DeleteArticleHistoryByArticle(ctx, 1, 20))
	require.NoError(t, s.ClearViewHistory(ctx, 1))
	require.NoError(t, s.ClearArticleViewHistory(ctx, 1))

	var n int64
	require.NoError(t, s.db.Model(&extra.VideoViewHistory{}).Count(&n).Error)
	require.Zero(t, n)
	require.NoError(t, s.db.Model(&extra.ArticleViewHistory{}).Count(&n).Error)
	require.Zero(t, n)
}
