package service

import (
	"cakecake/internal/model/video"
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func newEngagementService(t *testing.T) *EngagementService {
	t.Helper()
	db := newAgentTestDB(t)
	_, rdb := newAgentTestRedis(t)
	return NewEngagementService(db, rdb, zapNop(), NewUserProvider(db), NewVideoProvider(db))
}

func TestEngagementService_Coins(t *testing.T) {
	s := newEngagementService(t)
	ctx := context.Background()
	seedUser(t, s.db, 1, "alice")
	seedUser(t, s.db, 2, "bob")
	seedVideoForFav(t, s.db, 10, 2, true)

	require.False(t, s.HasCoined(ctx, 1, 10))
	require.Equal(t, int64(230), s.GetUserCoinBalance(ctx, 1))
	require.Zero(t, s.GetUserCoinBalance(ctx, 999))

	require.NoError(t, s.IncrementVideoCoinCount(ctx, 10, 2))
	var v video.Video
	require.NoError(t, s.db.First(&v, 10).Error)
	require.Equal(t, uint64(2), v.CoinCount)

	require.NoError(t, s.DecrementUserCoins(ctx, 1, 1))
	var usr struct{ CoinBalanceTenths int64 }
	require.NoError(t, s.db.Raw("SELECT coin_balance_tenths FROM users WHERE id = 1").Scan(&usr).Error)
	require.Equal(t, int64(220), usr.CoinBalanceTenths)

	// PostVideoCoin full flow.
	res, err := s.PostVideoCoin(ctx, 1, 10, 2, 1)
	require.NoError(t, err)
	require.True(t, res.Coined)
	require.Equal(t, 1, res.Amount)
	require.Equal(t, 1, res.MyCoinAmount)

	// Second coin (amount 1) upgrades to 2 total.
	res, err = s.PostVideoCoin(ctx, 1, 10, 2, 1)
	require.NoError(t, err)
	require.Equal(t, 1, res.Amount)
	require.Equal(t, 2, res.MyCoinAmount)
	require.Equal(t, float64(20.0), res.CoinBalance)

	// Already maxed -> nil result.
	res, err = s.PostVideoCoin(ctx, 1, 10, 2, 2)
	require.NoError(t, err)
	require.Nil(t, res)

	// Invalid amount defaults to 1 for a fresh video.
	seedVideoForFav(t, s.db, 11, 2, true)
	res, err = s.PostVideoCoin(ctx, 1, 11, 2, 99)
	require.NoError(t, err)
	require.Equal(t, 1, res.Amount)

	// Batch helpers.
	require.Equal(t, map[uint64]bool{10: true, 11: true}, s.BatchHasCoined(ctx, 1, []uint64{10, 11}))
	require.Equal(t, map[uint64]int{10: 2}, s.BatchCoinedByUser(ctx, 1, []uint64{10, 99}))
	require.Empty(t, s.BatchHasCoined(ctx, 0, []uint64{10}))
	require.Empty(t, s.BatchCoinedByUser(ctx, 1, nil))
}

func TestEngagementService_WatchLater(t *testing.T) {
	s := newEngagementService(t)
	ctx := context.Background()
	seedVideoForFav(t, s.db, 10, 2, true)
	seedVideoForFav(t, s.db, 11, 2, true)

	added, err := s.ToggleWatchLater(ctx, 1, 10)
	require.NoError(t, err)
	require.True(t, added)
	added, err = s.ToggleWatchLater(ctx, 1, 10)
	require.NoError(t, err)
	require.False(t, added)

	_, err = s.ToggleWatchLater(ctx, 1, 10)
	require.NoError(t, err)
	require.NoError(t, s.MarkWatchLaterWatched(ctx, 1, 10))
	list, total, err := s.ListWatchLater(ctx, 1, 1, 10)
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.Equal(t, int64(1), total)

	require.NoError(t, s.ClearWatchedWatchLater(ctx, 1))
	list, _, err = s.ListWatchLater(ctx, 1, 1, 10)
	require.NoError(t, err)
	require.Empty(t, list)

	_, err = s.ToggleWatchLater(ctx, 1, 10)
	require.NoError(t, err)
	require.NoError(t, s.ClearWatchLater(ctx, 1))
	_, total, err = s.ListWatchLater(ctx, 1, 1, 10)
	require.NoError(t, err)
	require.Zero(t, total)

	// BatchWatchLater.
	_, err = s.ToggleWatchLater(ctx, 1, 10)
	require.NoError(t, err)
	require.Equal(t, map[uint64]bool{10: true}, s.BatchWatchLater(ctx, 1, []uint64{10, 11}))
	require.Empty(t, s.BatchWatchLater(ctx, 0, []uint64{10}))
}

func TestEngagementService_Favorites(t *testing.T) {
	s := newEngagementService(t)
	ctx := context.Background()
	seedUser(t, s.db, 1, "alice")
	seedUser(t, s.db, 2, "bob")
	seedVideoForFav(t, s.db, 10, 2, true)

	liked, count, err := s.ToggleFavorite(ctx, 1, 10)
	require.NoError(t, err)
	require.True(t, liked)
	require.Equal(t, uint64(1), count)

	liked, count, err = s.ToggleVideoFavoriteWithFolder(ctx, 1, 10, 0)
	require.NoError(t, err)
	require.False(t, liked)
	require.Zero(t, count)

	require.Equal(t, map[uint64]bool{}, s.BatchFavoritedByUser(ctx, 1, []uint64{10}))

	folder := &video.FavoriteFolder{UserID: 1, Title: "favs"}
	require.NoError(t, s.db.Create(folder).Error)
	require.NoError(t, s.db.Create(&video.VideoFavorite{UserID: 1, VideoID: 10, FolderID: folder.ID}).Error)
	require.NoError(t, s.db.Create(&video.VideoFavorite{UserID: 1, VideoID: 10, FolderID: 0}).Error)

	// MoveFavoritesBetweenFolders.
	moved, err := s.MoveFavoritesBetweenFolders(ctx, 1, folder.ID, 0)
	require.NoError(t, err)
	require.Zero(t, moved)

	fc, err := s.VideoFavCount(ctx, 10)
	require.NoError(t, err)
	require.Zero(t, fc)

	require.NoError(t, s.AdjustVideoFavCount(ctx, 10, 5))
	fc, err = s.VideoFavCount(ctx, 10)
	require.NoError(t, err)
	require.Equal(t, uint64(5), fc)
	require.NoError(t, s.AdjustVideoFavCount(ctx, 10, -2))
	fc, err = s.VideoFavCount(ctx, 10)
	require.NoError(t, err)
	require.Equal(t, uint64(3), fc)

	n, err := s.UserFavoriteCount(ctx, 1, 10)
	require.NoError(t, err)
	require.Equal(t, int64(1), n)

	// BatchVideoLikes.
	require.NoError(t, s.db.Create(&video.VideoLike{UserID: 1, VideoID: 10}).Error)
	require.Equal(t, map[uint64]bool{10: true}, s.BatchVideoLikes(ctx, 1, []uint64{10, 11}))
	require.Empty(t, s.BatchVideoLikes(ctx, 0, []uint64{10}))
}

func TestEngagementService_ListWithVideos(t *testing.T) {
	s := newEngagementService(t)
	ctx := context.Background()
	seedUser(t, s.db, 1, "alice")
	seedUser(t, s.db, 2, "bob")
	seedVideoForFav(t, s.db, 10, 2, true)
	seedVideoForFav(t, s.db, 11, 2, true)

	_, err := s.ToggleWatchLater(ctx, 1, 10)
	require.NoError(t, err)
	_, err = s.PostVideoCoin(ctx, 1, 10, 2, 1)
	require.NoError(t, err)
	_, err = s.PostVideoCoin(ctx, 1, 11, 2, 1)
	require.NoError(t, err)

	items, total, err := s.ListWatchLaterWithVideos(ctx, 1, 1, 10)
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, items, 1)
	require.Equal(t, "v", items[0].Title)

	coins, total, err := s.ListUserCoinedVideos(ctx, 1, 10)
	require.NoError(t, err)
	require.Equal(t, int64(2), total)
	require.Len(t, coins, 2)
}
