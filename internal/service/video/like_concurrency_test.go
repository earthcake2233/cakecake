package video

import (
	"cakecake/internal/data"
	"cakecake/internal/model/video"
	"cakecake/internal/service/servicetest"
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// newConcurrentDB opens a named in-memory SQLite database with a shared cache
// so parallel GORM connections observe the same data (plain ":memory:" would
// give every connection its own private database).
func newConcurrentDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:like_conc_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, data.AutoMigrateAll(db, zap.NewNop()))
	return db
}

// TestVideoLike_ConcurrentDistinctUsers verifies that N concurrent likes from
// distinct users all succeed and the video counter ends at exactly N.
func TestVideoLike_ConcurrentDistinctUsers(t *testing.T) {
	db := newConcurrentDB(t)
	ctx := context.Background()
	servicetest.SeedUser(t, db, 1, "owner")
	require.NoError(t, db.Create(&video.Video{ID: 10, UserID: 1, Title: "v", Status: video.StatusPublished}).Error)
	s := NewVideoService(db, nil, zap.NewNop(), nil, nil)

	const n = 10
	var wg sync.WaitGroup
	errs := make(chan error, n)
	for i := 0; i < n; i++ {
		uid := uint64(100 + i)
		servicetest.SeedUser(t, db, uid, fmt.Sprintf("user%d", i))
		wg.Add(1)
		go func(uid uint64) {
			defer wg.Done()
			liked, err := s.ToggleVideoLike(ctx, uid, 10)
			if err != nil {
				errs <- err
				return
			}
			if !liked {
				errs <- fmt.Errorf("user %d expected liked=true", uid)
			}
		}(uid)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}

	var v video.Video
	require.NoError(t, db.First(&v, 10).Error)
	require.Equal(t, uint64(n), v.LikeCount)
	var rows int64
	require.NoError(t, db.Model(&video.VideoLike{}).Where("video_id = ?", 10).Count(&rows).Error)
	require.Equal(t, int64(n), rows)
}

// TestVideoLike_ConcurrentSameUserNoDuplicate verifies that concurrent toggles
// from the same user never create duplicate like rows or a negative counter.
func TestVideoLike_ConcurrentSameUserNoDuplicate(t *testing.T) {
	db := newConcurrentDB(t)
	ctx := context.Background()
	servicetest.SeedUser(t, db, 1, "alice")
	require.NoError(t, db.Create(&video.Video{ID: 20, UserID: 1, Title: "v", Status: video.StatusPublished}).Error)
	s := NewVideoService(db, nil, zap.NewNop(), nil, nil)

	const n = 8
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = s.ToggleVideoLike(ctx, 1, 20)
		}()
	}
	wg.Wait()

	var rows int64
	require.NoError(t, db.Model(&video.VideoLike{}).Where("user_id = ? AND video_id = ?", 1, 20).Count(&rows).Error)
	require.LessOrEqual(t, rows, int64(1), "concurrent same-user toggles must never create duplicate like rows")

	var v video.Video
	require.NoError(t, db.First(&v, 20).Error)
	require.LessOrEqual(t, v.LikeCount, uint64(1), "like_count must stay within the single-row invariant")
}
