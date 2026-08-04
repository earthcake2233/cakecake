package danmaku

import (
	"cakecake/internal/model/danmaku"
	"cakecake/internal/model/video"
	"cakecake/internal/pkg/sensitive"
	"cakecake/internal/service"
	"cakecake/internal/service/servicetest"
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func newDanmakuService(t *testing.T) (*DanmakuService, *sensitive.Filter, *gorm.DB) {
	t.Helper()
	db := servicetest.NewDB(t)
	_, rdb := servicetest.NewRedis(t)

	f, err := os.CreateTemp("", "sensitive-*.txt")
	require.NoError(t, err)
	t.Cleanup(func() { os.Remove(f.Name()) })
	_, err = f.WriteString("badword\n")
	require.NoError(t, err)
	require.NoError(t, f.Close())

	filter := sensitive.NewFilter(f.Name(), zap.NewNop())
	require.NoError(t, filter.Reload())
	return NewDanmakuService(db, rdb, servicetest.ZapNop(), filter), filter, db
}

func TestDanmakuService_PostDanmaku(t *testing.T) {
	s, filter, db := newDanmakuService(t)
	ctx := context.Background()

	v := video.Video{ID: 10, UserID: 1, Title: "v", Status: video.StatusPublished}
	require.NoError(t, db.Create(&v).Error)
	servicetest.SeedUser(t, db, 1, "alice")

	// Missing video.
	_, err := s.PostDanmaku(ctx, 999, 1, "hi", "", "scroll", "", 1.0)
	require.Error(t, err)
	require.Equal(t, 40400, err.(*service.SvcError).Code)

	// Danmaku closed.
	require.NoError(t, db.Model(&video.Video{}).Where("id = ?", 10).Update("danmaku_closed", true).Error)
	_, err = s.PostDanmaku(ctx, 10, 1, "hi", "", "scroll", "", 1.0)
	require.Equal(t, 40304, err.(*service.SvcError).Code)
	require.NoError(t, db.Model(&video.Video{}).Where("id = ?", 10).Update("danmaku_closed", false).Error)

	// Content length invalid.
	_, err = s.PostDanmaku(ctx, 10, 1, "", "", "scroll", "", 1.0)
	require.ErrorIs(t, err, service.ErrParamError)
	long := make([]rune, 101)
	for i := range long {
		long[i] = 'a'
	}
	_, err = s.PostDanmaku(ctx, 10, 1, string(long), "", "scroll", "", 1.0)
	require.ErrorIs(t, err, service.ErrParamError)

	// Sensitive content blocked.
	_, err = s.PostDanmaku(ctx, 10, 1, "contains badword", "", "scroll", "", 1.0)
	require.Equal(t, 40022, err.(*service.SvcError).Code)

	// Success.
	res, err := s.PostDanmaku(ctx, 10, 1, "hello", "#fff", "scroll", "small", 2.5)
	require.NoError(t, err)
	require.Equal(t, "hello", res.Danmaku.Content)
	require.Equal(t, "alice", res.User.Username)

	// Cooldown blocks second post within TTL.
	_, err = s.PostDanmaku(ctx, 10, 1, "again", "", "scroll", "", 2.5)
	require.Equal(t, 40025, err.(*service.SvcError).Code)

	// Cleanup the cooldown key so the next test file instance is unaffected.
	_ = s.rdb.Del(ctx, "danmaku:cooldown:1:10").Err()
	_ = filter
}

func TestDanmakuService_ToggleLike(t *testing.T) {
	s, _, db := newDanmakuService(t)
	ctx := context.Background()

	// Missing danmaku.
	_, err := s.ToggleDanmakuLike(ctx, 999, 1)
	require.Equal(t, 40400, err.(*service.SvcError).Code)

	d := danmaku.Danmaku{VideoID: 10, UserID: 2, Content: "hello"}
	require.NoError(t, db.Create(&d).Error)

	res, err := s.ToggleDanmakuLike(ctx, d.ID, 1)
	require.NoError(t, err)
	require.True(t, res.Liked)
	require.Equal(t, uint64(1), res.LikeCount)

	res, err = s.ToggleDanmakuLike(ctx, d.ID, 1)
	require.NoError(t, err)
	require.False(t, res.Liked)
	require.Zero(t, res.LikeCount)
}

func TestDanmakuService_GetVideoAndUser(t *testing.T) {
	s, _, db := newDanmakuService(t)
	ctx := context.Background()
	servicetest.SeedUser(t, db, 1, "alice")
	require.NoError(t, db.Create(&video.Video{ID: 10, UserID: 1, Title: "v", Status: video.StatusPublished}).Error)

	v, u, err := s.GetVideoAndUser(ctx, 10)
	require.NoError(t, err)
	require.Equal(t, uint64(10), v.ID)
	require.Equal(t, uint64(1), u.ID)
}

func TestDanmakuService_ListCreatorDanmakus(t *testing.T) {
	s, _, db := newDanmakuService(t)
	ctx := context.Background()
	servicetest.SeedUser(t, db, 1, "alice")
	servicetest.SeedUser(t, db, 2, "bob")
	require.NoError(t, db.Create(&video.Video{ID: 10, UserID: 1, Title: "v1", CoverURL: "c", Status: video.StatusPublished}).Error)
	require.NoError(t, db.Create(&video.Video{ID: 11, UserID: 2, Title: "v2", Status: video.StatusPublished}).Error)
	require.NoError(t, db.Create(&danmaku.Danmaku{VideoID: 10, UserID: 2, Content: "hello world", Type: "scroll"}).Error)
	require.NoError(t, db.Create(&danmaku.Danmaku{VideoID: 10, UserID: 2, Content: "top stuff", Type: "top"}).Error)
	require.NoError(t, db.Create(&danmaku.Danmaku{VideoID: 11, UserID: 1, Content: "other", Type: "scroll"}).Error)

	// Filter video not owned -> error.
	_, err := s.ListCreatorDanmakus(ctx, 1, 10, "", "", 11, 1)
	require.Error(t, err)

	res, err := s.ListCreatorDanmakus(ctx, 1, 10, "", "", 0, 1)
	require.NoError(t, err)
	require.Equal(t, int64(2), res.Total)
	require.Len(t, res.Items, 2)
	require.Equal(t, "v1", res.Items[0].VideoTitle)

	res, err = s.ListCreatorDanmakus(ctx, 1, 10, "hello", "scroll", 10, 1)
	require.NoError(t, err)
	require.Len(t, res.Items, 1)
	require.Equal(t, "hello world", res.Items[0].Content)
}

func TestDanmakuService_DeleteCreatorDanmaku(t *testing.T) {
	s, _, db := newDanmakuService(t)
	ctx := context.Background()
	require.NoError(t, db.Create(&video.Video{ID: 10, UserID: 1, Title: "v", Status: video.StatusPublished}).Error)
	d := danmaku.Danmaku{VideoID: 10, UserID: 2, Content: "x", LikeCount: 3}
	require.NoError(t, db.Create(&d).Error)
	require.NoError(t, db.Create(&danmaku.DanmakuLike{UserID: 3, DanmakuID: d.ID}).Error)

	// Not the video owner.
	_, err := s.DeleteCreatorDanmaku(ctx, 2, d.ID)
	require.ErrorIs(t, err, service.ErrForbidden)

	// Success: danmaku + likes removed, video count decremented.
	del, err := s.DeleteCreatorDanmaku(ctx, 1, d.ID)
	require.NoError(t, err)
	require.Equal(t, d.ID, del.ID)
	var n int64
	require.NoError(t, db.Model(&danmaku.DanmakuLike{}).Count(&n).Error)
	require.Zero(t, n)

	// Missing danmaku.
	_, err = s.DeleteCreatorDanmaku(ctx, 1, 999)
	require.Error(t, err)
}

func TestDanmakuService_ListHistory(t *testing.T) {
	s, _, db := newDanmakuService(t)
	ctx := context.Background()
	require.NoError(t, db.Create(&danmaku.Danmaku{VideoID: 10, UserID: 1, Content: "a", VideoTime: 1.0}).Error)
	require.NoError(t, db.Create(&danmaku.Danmaku{VideoID: 10, UserID: 1, Content: "b", VideoTime: 5.0}).Error)
	require.NoError(t, db.Create(&danmaku.Danmaku{VideoID: 10, UserID: 1, Content: "c", VideoTime: 20.0}).Error)

	hist, err := s.ListHistory(ctx, 10, 4.0, 10)
	require.NoError(t, err)
	require.Len(t, hist, 2)

	hist, err = s.ListHistory(ctx, 10, 0, 10)
	require.NoError(t, err)
	require.Len(t, hist, 3)

}
