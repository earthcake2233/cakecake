package engagement

import (
	"cakecake/internal/model/video"
	"cakecake/internal/service"
	"cakecake/internal/service/servicetest"
	vsvc "cakecake/internal/service/video"
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func newEngagementService(t *testing.T) (*EngagementService, *gorm.DB) {
	t.Helper()
	db := servicetest.NewDB(t)
	_, rdb := servicetest.NewRedis(t)
	return NewEngagementService(db, rdb, servicetest.ZapNop(), service.NewUserProvider(db), vsvc.NewVideoProvider(db)), db
}

func TestEngagementService_Coins(t *testing.T) {
	s, db := newEngagementService(t)
	ctx := context.Background()
	servicetest.SeedUser(t, db, 1, "alice")
	servicetest.SeedUser(t, db, 2, "bob")
	servicetest.SeedVideoForFav(t, db, 10, 2, true)

	require.False(t, s.HasCoined(ctx, 1, 10))
	require.Equal(t, int64(230), s.GetUserCoinBalance(ctx, 1))
	require.Zero(t, s.GetUserCoinBalance(ctx, 999))

	require.NoError(t, s.IncrementVideoCoinCount(ctx, 10, 2))
	var v video.Video
	require.NoError(t, db.First(&v, 10).Error)
	require.Equal(t, uint64(2), v.CoinCount)

	require.NoError(t, s.DecrementUserCoins(ctx, 1, 1))
	var usr struct{ CoinBalanceTenths int64 }
	require.NoError(t, db.Raw("SELECT coin_balance_tenths FROM users WHERE id = 1").Scan(&usr).Error)
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
	servicetest.SeedVideoForFav(t, db, 11, 2, true)
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
	s, db := newEngagementService(t)
	ctx := context.Background()
	servicetest.SeedVideoForFav(t, db, 10, 2, true)
	servicetest.SeedVideoForFav(t, db, 11, 2, true)

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

func TestEngagementService_ListWithVideos(t *testing.T) {
	s, db := newEngagementService(t)
	ctx := context.Background()
	servicetest.SeedUser(t, db, 1, "alice")
	servicetest.SeedUser(t, db, 2, "bob")
	servicetest.SeedVideoForFav(t, db, 10, 2, true)
	servicetest.SeedVideoForFav(t, db, 11, 2, true)

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

func TestEngagementService_BatchVideoLikes(t *testing.T) {
	s, db := newEngagementService(t)
	ctx := context.Background()
	servicetest.SeedUser(t, db, 1, "alice")
	servicetest.SeedVideoForFav(t, db, 10, 2, true)
	require.NoError(t, db.Create(&video.VideoLike{UserID: 1, VideoID: 10}).Error)
	require.Equal(t, map[uint64]bool{10: true}, s.BatchVideoLikes(ctx, 1, []uint64{10, 11}))
	require.Empty(t, s.BatchVideoLikes(ctx, 0, []uint64{10}))
}
