package data

import (
	"testing"

	"cakecake/internal/config"
	"cakecake/internal/model/danmaku"
	"cakecake/internal/model/user"
	"cakecake/internal/model/video"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestSeedDemoData_Disabled(t *testing.T) {
	db := setupDataDB(t)
	require.NoError(t, AutoMigrateAll(db, nil))

	err := SeedDemoData(db, &config.C{SeedDemoData: false}, zap.NewNop())
	require.NoError(t, err)

	var n int64
	require.NoError(t, db.Model(&video.Video{}).Count(&n).Error)
	assert.Zero(t, n)
}

func TestSeedDemoData_SeedsContent(t *testing.T) {
	db := setupDataDB(t)
	require.NoError(t, AutoMigrateAll(db, nil))

	err := SeedDemoData(db, &config.C{SeedDemoData: true, DemoUserPassword: "demo123456"}, zap.NewNop())
	require.NoError(t, err)

	var videoCount, userCount, danmakuCount int64
	require.NoError(t, db.Model(&video.Video{}).Count(&videoCount).Error)
	require.NoError(t, db.Model(&user.User{}).Count(&userCount).Error)
	require.NoError(t, db.Model(&danmaku.Danmaku{}).Count(&danmakuCount).Error)
	assert.EqualValues(t, len(demoVideoSeeds), videoCount)
	assert.EqualValues(t, len(demoVideoSeeds), userCount)
	assert.EqualValues(t, len(demoDanmakuSeeds), danmakuCount)

	var videos []video.Video
	require.NoError(t, db.Find(&videos).Error)
	require.Len(t, videos, len(demoVideoSeeds))
	for i, v := range videos {
		assert.Equal(t, video.StatusPublished, v.Status)
		assert.NotEmpty(t, v.VideoURL)
		assert.NotEmpty(t, v.CoverURL)
		assert.NotEmpty(t, v.Zone)
		// Row-backed counters must stay in sync with seeded data: only the first
		// video carries danmaku, and no comments/likes/favorites/coins are seeded.
		if i == 0 {
			assert.EqualValues(t, len(demoDanmakuSeeds), v.DanmakuCount)
		} else {
			assert.Zero(t, v.DanmakuCount)
		}
		assert.Zero(t, v.CommentCount)
		assert.Zero(t, v.LikeCount)
		assert.Zero(t, v.FavCount)
		assert.Zero(t, v.CoinCount)
	}

	var users []user.User
	require.NoError(t, db.Find(&users).Error)
	for _, u := range users {
		assert.NotEmpty(t, u.CakeID, "user %q should have a cake_id", u.Username)
	}

	var dms []danmaku.Danmaku
	require.NoError(t, db.Find(&dms).Error)
	for _, d := range dms {
		assert.Equal(t, "scroll", d.Type)
		assert.Regexp(t, `^#[0-9A-Fa-f]{6}$`, d.Color)
	}
}

func TestSeedDemoData_Idempotent(t *testing.T) {
	db := setupDataDB(t)
	require.NoError(t, AutoMigrateAll(db, nil))
	cfg := &config.C{SeedDemoData: true, DemoUserPassword: "demo123456"}

	require.NoError(t, SeedDemoData(db, cfg, zap.NewNop()))
	require.NoError(t, SeedDemoData(db, cfg, zap.NewNop()))

	var n int64
	require.NoError(t, db.Model(&video.Video{}).Count(&n).Error)
	assert.EqualValues(t, len(demoVideoSeeds), n)
}

func TestSeedDemoData_SkipsWhenContentExists(t *testing.T) {
	db := setupDataDB(t)
	require.NoError(t, AutoMigrateAll(db, nil))
	require.NoError(t, db.Create(&video.Video{
		UserID: 1,
		Title:  "existing",
		Status: video.StatusDraft,
	}).Error)

	require.NoError(t, SeedDemoData(db, &config.C{SeedDemoData: true, DemoUserPassword: "demo123456"}, zap.NewNop()))

	var n int64
	require.NoError(t, db.Model(&video.Video{}).Count(&n).Error)
	assert.EqualValues(t, 1, n)
}

func TestSeedDemoData_NilInputs(t *testing.T) {
	require.NoError(t, SeedDemoData(nil, &config.C{SeedDemoData: true}, nil))
	require.NoError(t, SeedDemoData(nil, nil, nil))
}

func TestSeedDemoData_RollsBackOnMidSeedFailure(t *testing.T) {
	db := setupDataDB(t)
	require.NoError(t, AutoMigrateAll(db, nil))

	demoSeedFailAfterVideos = 2
	t.Cleanup(func() { demoSeedFailAfterVideos = -1 })

	err := SeedDemoData(db, &config.C{SeedDemoData: true, DemoUserPassword: "demo123456"}, zap.NewNop())
	require.Error(t, err)

	// All-or-nothing: the partial users/videos/danmaku must be rolled back so the
	// next startup retries the seed instead of skipping it forever.
	var videoCount, userCount, danmakuCount int64
	require.NoError(t, db.Model(&video.Video{}).Count(&videoCount).Error)
	require.NoError(t, db.Model(&user.User{}).Count(&userCount).Error)
	require.NoError(t, db.Model(&danmaku.Danmaku{}).Count(&danmakuCount).Error)
	assert.Zero(t, videoCount)
	assert.Zero(t, userCount)
	assert.Zero(t, danmakuCount)
}
