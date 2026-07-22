package service

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/glebarez/sqlite"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"minibili/internal/model"
)

// ---------- PlayCounter ----------

func TestPlayCounter_Incr(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	pc := &PlayCounter{Rdb: rdb}

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
	pc := &PlayCounter{Rdb: rdb}

	ctx := context.Background()
	v := &model.Video{ID: 42, PlayCount: 100}

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
	require.NoError(t, db.AutoMigrate(&model.Video{}))

	// Create a video record
	v := model.Video{ID: 1, Title: "Test", PlayCount: 50, Status: "published"}
	require.NoError(t, db.Create(&v).Error)

	pc := &PlayCounter{Rdb: rdb, DB: db}
	ctx := context.Background()

	// Simulate view count increments via Redis
	_ = pc.Incr(ctx, 1)
	_ = pc.Incr(ctx, 1)
	_ = pc.Incr(ctx, 1)

	err = pc.Flush(ctx)
	require.NoError(t, err)

	// Verify DB was updated
	var updated model.Video
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
	require.NoError(t, db.AutoMigrate(&model.Article{}))

	now := time.Now()
	art := model.Article{
		Title:   "Test Article",
		BodyMD:  "# Hello",
		Status:  "pending_review",
		UserID:  1,
		CreatedAt: now,
		UpdatedAt: now,
	}
	require.NoError(t, db.Create(&art).Error)

	log := zap.NewNop()
	ctx := context.Background()
	adminID := uint64(1)

	err = PublishArticle(ctx, db, nil, log, art.ID, &adminID)
	require.NoError(t, err)

	var updated model.Article
	require.NoError(t, db.First(&updated, art.ID).Error)
	require.Equal(t, "published", updated.Status)
	require.NotNil(t, updated.PublishedAt)
	require.NotNil(t, updated.ReviewedAt)
}

func TestPublishArticle_AlreadyPublished(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.Article{}))

	now := time.Now()
	pubAt := now
	art := model.Article{
		Title:     "Published Article",
		BodyMD:    "# Done",
		Status:    "published",
		UserID:    1,
		PublishedAt: &pubAt,
		CreatedAt: now,
		UpdatedAt: now,
	}
	require.NoError(t, db.Create(&art).Error)

	log := zap.NewNop()
	ctx := context.Background()

	err = PublishArticle(ctx, db, nil, log, art.ID, nil)
	require.NoError(t, err) // should be no-op
}

func TestPublishArticle_NotFound(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.Article{}))

	log := zap.NewNop()
	ctx := context.Background()

	err = PublishArticle(ctx, db, nil, log, 999, nil)
	require.Error(t, err)
}

// ---------- PublishVideo ----------

func TestPublishVideo_Success(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.Video{}, &model.User{}))

	now := time.Now()
	u := model.User{Username: "testuser", PasswordHash: "hash", CoinBalanceTenths: 230}
	require.NoError(t, db.Create(&u).Error)

	v := model.Video{
		Title:     "Test Video",
		VideoURL:  "https://cdn.example.com/v.mp4",
		Status:    "pending_review",
		UserID:    u.ID,
		DurationSec: 120,
		CreatedAt: now,
		UpdatedAt: now,
	}
	require.NoError(t, db.Create(&v).Error)

	log := zap.NewNop()
	ctx := context.Background()
	adminID := uint64(1)

	err = PublishVideo(ctx, db, nil, log, v.ID, &adminID)
	require.NoError(t, err)

	var updated model.Video
	require.NoError(t, db.First(&updated, v.ID).Error)
	require.Equal(t, "published", updated.Status)
	require.NotNil(t, updated.ReviewedAt)
}

func TestPublishVideo_AlreadyPublished(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.Video{}, &model.User{}))

	now := time.Now()
	u := model.User{Username: "testuser2", PasswordHash: "hash", CoinBalanceTenths: 230}
	require.NoError(t, db.Create(&u).Error)

	v := model.Video{
		Title:    "Published Video",
		VideoURL: "https://cdn.example.com/v.mp4",
		Status:   "published",
		UserID:   u.ID,
		CreatedAt: now,
		UpdatedAt: now,
	}
	require.NoError(t, db.Create(&v).Error)

	log := zap.NewNop()
	ctx := context.Background()

	err = PublishVideo(ctx, db, nil, log, v.ID, nil)
	require.NoError(t, err) // no-op
}

func TestPublishVideo_NotFound(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.Video{}))

	log := zap.NewNop()
	ctx := context.Background()

	err = PublishVideo(ctx, db, nil, log, 999, nil)
	require.Error(t, err)
}

// ---------- SearchSuggest ----------

func TestSearchSuggest_EmptyDB(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.UserSearchHistory{}, &model.HotSearchOp{}))

	ctx := context.Background()
	results := SearchSuggest(ctx, db, nil, 0, "", 10)
	require.NotNil(t, results)
	require.Empty(t, results)
}

func TestSearchSuggest_WithUserHistory(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.UserSearchHistory{}, &model.HotSearchOp{}))

	// Add user search history
	h := model.UserSearchHistory{
		UserID:   1,
		Keyword:  "golang testing",
	}
	require.NoError(t, db.Create(&h).Error)

	ctx := context.Background()
	results := SearchSuggest(ctx, db, nil, 1, "golang", 10)
	require.NotNil(t, results)
	require.GreaterOrEqual(t, len(results), 1)
	require.Contains(t, results[0].Name, "golang")
	require.Contains(t, results[0].Value, "golang")
}

func TestSearchSuggest_TermTooLong(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.UserSearchHistory{}, &model.HotSearchOp{}))

	ctx := context.Background()
	longTerm := string(make([]rune, 60))
	results := SearchSuggest(ctx, db, nil, 0, longTerm, 10)
	require.NotNil(t, results)
	require.Empty(t, results)
}

func TestSearchSuggest_LimitBounds(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.UserSearchHistory{}, &model.HotSearchOp{}))

	ctx := context.Background()

	// Zero limit should become 10
	results := SearchSuggest(ctx, db, nil, 0, "", 0)
	require.NotNil(t, results)
	require.LessOrEqual(t, len(results), 10)

	// Large limit should be capped at 20
	results = SearchSuggest(ctx, db, nil, 0, "", 100)
	require.NotNil(t, results)
	require.LessOrEqual(t, len(results), 20)
}

// ---------- ValidateSuggestTerm ----------

func TestValidateSuggestTerm_Edge(t *testing.T) {
	require.True(t, ValidateSuggestTerm("short"))
	require.True(t, ValidateSuggestTerm(""))
	require.True(t, ValidateSuggestTerm("  "))
	require.True(t, ValidateSuggestTerm("a"))
	term50 := string(make([]rune, 50))
	require.True(t, ValidateSuggestTerm(term50))
	term51 := string(make([]rune, 51))
	require.False(t, ValidateSuggestTerm(term51))
}

// ---------- HighlightSuggestKeyword ----------

func TestHighlightSuggestKeyword_Edge(t *testing.T) {
	require.Empty(t, HighlightSuggestKeyword("", "x"))
	require.Empty(t, HighlightSuggestKeyword("  ", "x"))
	require.Equal(t, "hello", HighlightSuggestKeyword("hello", ""))
	require.Equal(t, "<em class=\"suggest_high_light\">he</em>llo", HighlightSuggestKeyword("hello", "he"))
	require.Equal(t, "h<em class=\"suggest_high_light\">el</em>lo", HighlightSuggestKeyword("hello", "el"))
	require.Equal(t, "hel<em class=\"suggest_high_light\">lo</em>", HighlightSuggestKeyword("hello", "lo"))
	// Case insensitive
	require.Equal(t, "<em class=\"suggest_high_light\">HE</em>LLO", HighlightSuggestKeyword("HELLO", "he"))
}

// ---------- escapeHTML ----------

func TestEscapeHTML_Edge(t *testing.T) {
	require.Equal(t, "a&amp;b", escapeHTML("a&b"))
	require.Equal(t, "&lt;tag&gt;", escapeHTML("<tag>"))
	require.Equal(t, "&quot;quote&quot;", escapeHTML(`"quote"`))
	require.Equal(t, "no change", escapeHTML("no change"))
	require.Empty(t, escapeHTML(""))
}
