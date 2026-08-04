package service

import (
	"cakecake/internal/model/article"
	"cakecake/internal/model/user"
	"cakecake/internal/model/video"
	artsvc "cakecake/internal/service/article"
	"cakecake/internal/service/playcount"
	vsvc "cakecake/internal/service/video"
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/glebarez/sqlite"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// ---------- playcount.PlayCounter ----------

func TestPlayCounter_Incr(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	pc := &playcount.PlayCounter{Rdb: rdb}

	ctx := context.Background()
	err = pc.Incr(ctx, 42)
	require.NoError(t, err)

	// Verify the delta key exists
	val, err := rdb.Get(ctx, "videodelta:42").Uint64()
	require.NoError(t, err)
	require.Equal(t, uint64(1), val)

	// Verify dirty set
	member, err := rdb.SIsMember(ctx, "playcount:dirty", "42").Result()
	require.NoError(t, err)
	require.True(t, member)
}

func TestPlayCounter_Display(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	pc := &playcount.PlayCounter{Rdb: rdb}

	ctx := context.Background()
	v := &video.Video{ID: 42, PlayCount: 100}

	// No delta - should return play_count only
	n, err := pc.Display(ctx, v)
	require.NoError(t, err)
	require.Equal(t, uint64(100), n)

	// With delta
	_ = rdb.Set(ctx, "videodelta:42", 5, 0).Err()
	n, err = pc.Display(ctx, v)
	require.NoError(t, err)
	require.Equal(t, uint64(105), n)
}

func TestPlayCounter_Flush(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&video.Video{}))

	// Create a video record
	v := video.Video{ID: 1, Title: "Test", PlayCount: 50, Status: "published"}
	require.NoError(t, db.Create(&v).Error)

	pc := &playcount.PlayCounter{Rdb: rdb, Store: playcount.NewPlayCountStore(db)}
	ctx := context.Background()

	// Simulate view count increments via Redis
	_ = pc.Incr(ctx, 1)
	_ = pc.Incr(ctx, 1)
	_ = pc.Incr(ctx, 1)

	err = pc.Flush(ctx)
	require.NoError(t, err)

	// Verify DB was updated
	var updated video.Video
	require.NoError(t, db.First(&updated, 1).Error)
	require.Equal(t, uint64(53), updated.PlayCount)

	// Verify Redis keys cleaned
	exists, _ := rdb.Exists(ctx, "videodelta:1").Result()
	require.Equal(t, int64(0), exists)
	isDirty, _ := rdb.SIsMember(ctx, "playcount:dirty", "1").Result()
	require.False(t, isDirty)
}

// ---------- PublishArticle ----------

func TestPublishArticle_Success(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&article.Article{}))

	now := time.Now()
	art := article.Article{
		Title:     "Test Article",
		BodyMD:    "# Hello",
		Status:    "pending_review",
		UserID:    1,
		CreatedAt: now,
		UpdatedAt: now,
	}
	require.NoError(t, db.Create(&art).Error)

	log := zap.NewNop()
	ctx := context.Background()
	adminID := uint64(1)

	err = artsvc.NewArticleStore(db).PublishArticle(ctx, nil, log, art.ID, &adminID)
	require.NoError(t, err)

	var updated article.Article
	require.NoError(t, db.First(&updated, art.ID).Error)
	require.Equal(t, "published", updated.Status)
	require.NotNil(t, updated.PublishedAt)
	require.NotNil(t, updated.ReviewedAt)
}

func TestPublishArticle_AlreadyPublished(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&article.Article{}))

	now := time.Now()
	pubAt := now
	art := article.Article{
		Title:       "Published Article",
		BodyMD:      "# Done",
		Status:      "published",
		UserID:      1,
		PublishedAt: &pubAt,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	require.NoError(t, db.Create(&art).Error)

	log := zap.NewNop()
	ctx := context.Background()

	err = artsvc.NewArticleStore(db).PublishArticle(ctx, nil, log, art.ID, nil)
	require.NoError(t, err) // should be no-op
}

func TestPublishArticle_NotFound(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&article.Article{}))

	log := zap.NewNop()
	ctx := context.Background()

	err = artsvc.NewArticleStore(db).PublishArticle(ctx, nil, log, 999, nil)
	require.Error(t, err)
}

// ---------- PublishVideo ----------

func TestPublishVideo_Success(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&video.Video{}, &user.User{}))

	now := time.Now()
	u := user.User{Username: "testuser", PasswordHash: "hash", CoinBalanceTenths: 230}
	require.NoError(t, db.Create(&u).Error)

	v := video.Video{
		Title:       "Test Video",
		VideoURL:    "https://cdn.example.com/v.mp4",
		Status:      "pending_review",
		UserID:      u.ID,
		DurationSec: 120,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	require.NoError(t, db.Create(&v).Error)

	log := zap.NewNop()
	ctx := context.Background()
	adminID := uint64(1)

	err = vsvc.PublishVideo(ctx, db, nil, log, v.ID, &adminID)
	require.NoError(t, err)

	var updated video.Video
	require.NoError(t, db.First(&updated, v.ID).Error)
	require.Equal(t, "published", updated.Status)
	require.NotNil(t, updated.ReviewedAt)
}

func TestPublishVideo_AlreadyPublished(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&video.Video{}, &user.User{}))

	now := time.Now()
	u := user.User{Username: "testuser2", PasswordHash: "hash", CoinBalanceTenths: 230}
	require.NoError(t, db.Create(&u).Error)

	v := video.Video{
		Title:     "Published Video",
		VideoURL:  "https://cdn.example.com/v.mp4",
		Status:    "published",
		UserID:    u.ID,
		CreatedAt: now,
		UpdatedAt: now,
	}
	require.NoError(t, db.Create(&v).Error)

	log := zap.NewNop()
	ctx := context.Background()

	err = vsvc.PublishVideo(ctx, db, nil, log, v.ID, nil)
	require.NoError(t, err) // no-op
}

func TestPublishVideo_NotFound(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&video.Video{}))

	log := zap.NewNop()
	ctx := context.Background()

	err = vsvc.PublishVideo(ctx, db, nil, log, 999, nil)
	require.Error(t, err)
}
