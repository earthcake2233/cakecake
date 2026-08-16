package hotsearch

import (
	"cakecake/internal/model/admin"
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/glebarez/sqlite"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func newCacheTestEnv(t *testing.T) (*gorm.DB, *SearchHotRecorder, *HotSearchService) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&admin.HotSearchOp{}, &admin.HotSearchDisplayLayout{}))

	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	rec := &SearchHotRecorder{Rdb: rdb}
	return db, rec, NewHotSearchService(db, rec)
}

func seedCacheTestOp(t *testing.T, db *gorm.DB, op admin.HotSearchOp) {
	t.Helper()
	now := time.Now()
	end := now.Add(24 * time.Hour)
	op.Enabled = true
	op.StartAt = &now
	op.EndAt = &end
	require.NoError(t, db.Create(&op).Error)
}

func TestMergedDetailCache_HitAndInvalidate(t *testing.T) {
	db, _, svc := newCacheTestEnv(t)
	ctx := context.Background()
	p := svc.store.(*HotSearchProviderImpl)

	seedCacheTestOp(t, db, admin.HotSearchOp{Keyword: "golang", OpType: "manual", DisplayTitle: "Go", PinRank: 1})

	first, err := svc.ListMergedDetail(ctx, 10)
	require.NoError(t, err)
	require.NotEmpty(t, first)

	// Direct DB mutation bypasses service invalidation: cache should still hit.
	require.NoError(t, p.db.Delete(&admin.HotSearchOp{}, "keyword = ?", "golang").Error)
	second, err := svc.ListMergedDetail(ctx, 10)
	require.NoError(t, err)
	require.Equal(t, first, second, "cache hit should return stale data until invalidated")

	// A service mutation invalidates; the next read reflects the DB state.
	now := time.Now()
	end := now.Add(24 * time.Hour)
	require.NoError(t, svc.CreateOp(ctx, &admin.HotSearchOp{
		Keyword: "gin", OpType: "manual", DisplayTitle: "Gin", PinRank: 2,
		Enabled: true, StartAt: &now, EndAt: &end,
	}))
	third, err := svc.ListMergedDetail(ctx, 10)
	require.NoError(t, err)
	require.NotEqual(t, first, third)
	require.Len(t, third, 1)
	require.Equal(t, "Gin", third[0].Title)
}

func TestMergedDetailCache_NilRedis(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&admin.HotSearchOp{}))
	ctx := context.Background()
	svc := NewHotSearchService(db, nil)

	now := time.Now()
	end := now.Add(24 * time.Hour)
	require.NoError(t, svc.CreateOp(ctx, &admin.HotSearchOp{
		Keyword: "go", OpType: "manual", DisplayTitle: "Go", PinRank: 1,
		Enabled: true, StartAt: &now, EndAt: &end,
	}))
	items, err := svc.ListMergedDetail(ctx, 10)
	require.NoError(t, err)
	require.NotEmpty(t, items)
}

func TestMergedDetailCache_Disabled(t *testing.T) {
	db, rec, svc := newCacheTestEnv(t)
	ctx := context.Background()
	disabled := false
	rec.CacheEnabled = func() bool { return disabled }
	p := svc.store.(*HotSearchProviderImpl)

	seedCacheTestOp(t, db, admin.HotSearchOp{Keyword: "golang", OpType: "manual", DisplayTitle: "Go", PinRank: 1})
	first, err := svc.ListMergedDetail(ctx, 10)
	require.NoError(t, err)
	require.NotEmpty(t, first)

	// Cache disabled: a direct DB mutation is visible immediately.
	require.NoError(t, p.db.Delete(&admin.HotSearchOp{}, "keyword = ?", "golang").Error)
	second, err := svc.ListMergedDetail(ctx, 10)
	require.NoError(t, err)
	require.Empty(t, second)

	// Re-enable: cache is populated again and returns stale data until invalidation.
	disabled = true
	seedCacheTestOp(t, db, admin.HotSearchOp{Keyword: "gin", OpType: "manual", DisplayTitle: "Gin", PinRank: 2})
	_, err = svc.ListMergedDetail(ctx, 10)
	require.NoError(t, err)
	require.NoError(t, p.db.Delete(&admin.HotSearchOp{}, "keyword = ?", "gin").Error)
	third, err := svc.ListMergedDetail(ctx, 10)
	require.NoError(t, err)
	require.NotEmpty(t, third, "cache hit should return stale data when enabled")
}
