package service

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/glebarez/sqlite"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"minibili/internal/model"
	"minibili/internal/ws"
)

// ---------- DanmakuRelay ----------

func TestNewDanmakuRelay(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	hub := ws.NewHub()
	log := zap.NewNop()

	relay := NewDanmakuRelay(rdb, hub, log)
	require.NotNil(t, relay)
	require.Equal(t, rdb, relay.Rdb)
	require.Equal(t, hub, relay.Hub)
	require.Equal(t, log, relay.Log)
}

func TestNewDanmakuRelay_NilLog(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	hub := ws.NewHub()

	relay := NewDanmakuRelay(rdb, hub, nil)
	require.NotNil(t, relay)
	require.NotNil(t, relay.Log) // should be zap.NewNop()
}


func TestDanmakuRelay_Publish_Simple(t *testing.T) {
    mr, err := miniredis.Run()
    require.NoError(t, err)
    defer mr.Close()

    rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
    hub := ws.NewHub()
    relay := NewDanmakuRelay(rdb, hub, zap.NewNop())

    ctx := context.Background()
    err = relay.Publish(ctx, uint64(100), map[string]interface{}{"text": "hello"})
    require.NoError(t, err)
}
func TestDanmakuRelay_Publish_Error(t *testing.T) {
	// Use a disconnected client to simulate publish failure
	rdb := redis.NewClient(&redis.Options{Addr: "127.0.0.1:1"})
	hub := ws.NewHub()
	relay := NewDanmakuRelay(rdb, hub, zap.NewNop())

	ctx := context.Background()
	err := relay.Publish(ctx, uint64(1), "test")
	require.Error(t, err)
}

// ---------- HotSearchLayout ----------

func setupHotSearchDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.HotSearchDisplayLayout{}, &model.HotSearchOp{}))
	return db
}

func TestLoadHotSearchLayout_NoDB(t *testing.T) {
	entries := loadHotSearchLayout(nil)
	require.Nil(t, entries)
}

func TestLoadHotSearchLayout_EmptyDB(t *testing.T) {
	db := setupHotSearchDB(t)
	entries := loadHotSearchLayout(db)
	require.Nil(t, entries)
}

func TestSaveAndLoadHotSearchLayout(t *testing.T) {
	db := setupHotSearchDB(t)

	entries := []HotSearchLayoutEntry{
		{Keyword: "kw1", Title: "Title 1"},
		{Keyword: "kw2", Title: "Title 2"},
	}

	err := SaveHotSearchDisplayLayout(db, entries)
	require.NoError(t, err)

	loaded := loadHotSearchLayout(db)
	require.NotNil(t, loaded)
	require.Len(t, loaded, 2)
	require.Equal(t, "kw1", loaded[0].Keyword)
	require.Equal(t, "Title 1", loaded[0].Title)
}

func TestHasHotSearchLayout_Absent(t *testing.T) {
	db := setupHotSearchDB(t)
	require.False(t, HasHotSearchDisplayLayout(db))
}

func TestHasHotSearchLayout_Present(t *testing.T) {
	db := setupHotSearchDB(t)
	err := SaveHotSearchDisplayLayout(db, []HotSearchLayoutEntry{
		{Keyword: "test", Title: "Test"},
	})
	require.NoError(t, err)
	require.True(t, HasHotSearchDisplayLayout(db))
}

func TestClearHotSearchDisplayLayout(t *testing.T) {
	db := setupHotSearchDB(t)

	// Save then clear
	err := SaveHotSearchDisplayLayout(db, []HotSearchLayoutEntry{
		{Keyword: "kw", Title: "Title"},
	})
	require.NoError(t, err)
	require.True(t, HasHotSearchDisplayLayout(db))

	err = ClearHotSearchDisplayLayout(db)
	require.NoError(t, err)
	require.False(t, HasHotSearchDisplayLayout(db))
}

func TestClearHotSearchDisplayLayout_NoDB(t *testing.T) {
	err := ClearHotSearchDisplayLayout(nil)
	require.NoError(t, err)
}

func TestSaveHotSearchDisplayLayout_NoDB(t *testing.T) {
	err := SaveHotSearchDisplayLayout(nil, []HotSearchLayoutEntry{{Keyword: "kw"}})
	require.NoError(t, err)
}

func TestLoadHotSearchLayout_EmptyJSON(t *testing.T) {
	db := setupHotSearchDB(t)
	// Insert a row with empty OrderJSON
	layout := model.HotSearchDisplayLayout{ID: 1, OrderJSON: "[]"}
	err := db.Save(&layout).Error
	require.NoError(t, err)

	entries := loadHotSearchLayout(db)
	require.Nil(t, entries)
}

func TestLoadHotSearchLayout_InvalidJSON(t *testing.T) {
	db := setupHotSearchDB(t)
	layout := model.HotSearchDisplayLayout{ID: 1, OrderJSON: "not-json"}
	err := db.Save(&layout).Error
	require.NoError(t, err)

	entries := loadHotSearchLayout(db)
	require.Nil(t, entries)
}

func TestApplyHotSearchLayoutMove_ToFront(t *testing.T) {
	db := setupHotSearchDB(t)

	entries := []HotSearchLayoutEntry{
		{Keyword: "a", Title: "A"},
		{Keyword: "b", Title: "B"},
		{Keyword: "c", Title: "C"},
	}
	err := SaveHotSearchDisplayLayout(db, entries)
	require.NoError(t, err)

	err = ApplyHotSearchLayoutMove(db, "c", "C", 1)
	require.NoError(t, err)

	loaded := loadHotSearchLayout(db)
	require.Len(t, loaded, 3)
	require.Equal(t, "c", loaded[0].Keyword)
	require.Equal(t, "a", loaded[1].Keyword)
	require.Equal(t, "b", loaded[2].Keyword)
}

func TestApplyHotSearchLayoutMove_ToEnd(t *testing.T) {
	db := setupHotSearchDB(t)

	entries := []HotSearchLayoutEntry{
		{Keyword: "a", Title: "A"},
		{Keyword: "b", Title: "B"},
		{Keyword: "c", Title: "C"},
	}
	err := SaveHotSearchDisplayLayout(db, entries)
	require.NoError(t, err)

	// Move 'a' to position 3 (end)
	err = ApplyHotSearchLayoutMove(db, "a", "A", 3)
	require.NoError(t, err)

	loaded := loadHotSearchLayout(db)
	require.Len(t, loaded, 3)
	require.Equal(t, "b", loaded[0].Keyword)
	require.Equal(t, "c", loaded[1].Keyword)
	require.Equal(t, "a", loaded[2].Keyword)
}

func TestApplyHotSearchLayoutMove_NoDB(t *testing.T) {
	err := ApplyHotSearchLayoutMove(nil, "kw", "Title", 1)
	require.NoError(t, err)
}

func TestApplyHotSearchLayoutMove_EmptyLayout(t *testing.T) {
	db := setupHotSearchDB(t)
	err := ApplyHotSearchLayoutMove(db, "kw", "Title", 1)
	require.NoError(t, err) // layout is empty, should be no-op
}

func TestRemoveHotSearchLayoutEntry(t *testing.T) {
	db := setupHotSearchDB(t)

	entries := []HotSearchLayoutEntry{
		{Keyword: "a", Title: "A"},
		{Keyword: "b", Title: "B"},
		{Keyword: "c", Title: "C"},
	}
	err := SaveHotSearchDisplayLayout(db, entries)
	require.NoError(t, err)

	err = RemoveHotSearchLayoutEntry(db, "b")
	require.NoError(t, err)

	loaded := loadHotSearchLayout(db)
	require.Len(t, loaded, 2)
	require.Equal(t, "a", loaded[0].Keyword)
	require.Equal(t, "c", loaded[1].Keyword)
}

func TestRemoveHotSearchLayoutEntry_LastItem(t *testing.T) {
	db := setupHotSearchDB(t)

	err := SaveHotSearchDisplayLayout(db, []HotSearchLayoutEntry{
		{Keyword: "only", Title: "Only"},
	})
	require.NoError(t, err)

	err = RemoveHotSearchLayoutEntry(db, "only")
	require.NoError(t, err)

	require.False(t, HasHotSearchDisplayLayout(db))
}

func TestRemoveHotSearchLayoutEntry_NoDB(t *testing.T) {
	err := RemoveHotSearchLayoutEntry(nil, "kw")
	require.NoError(t, err)
}

func TestRemoveHotSearchLayoutEntry_EmptyLayout(t *testing.T) {
	db := setupHotSearchDB(t)
	err := RemoveHotSearchLayoutEntry(db, "kw")
	require.NoError(t, err) // no-op
}

func TestEnsureHotSearchLayoutFromMerged_NoDB(t *testing.T) {
	err := EnsureHotSearchLayoutFromMerged(context.Background(), nil, nil, 10)
	require.NoError(t, err)
}

func TestEnsureHotSearchLayoutFromMerged_AlreadyExists(t *testing.T) {
	db := setupHotSearchDB(t)

	err := SaveHotSearchDisplayLayout(db, []HotSearchLayoutEntry{
		{Keyword: "existing", Title: "Existing"},
	})
	require.NoError(t, err)

	// Should be no-op since layout already exists
	err = EnsureHotSearchLayoutFromMerged(context.Background(), db, nil, 10)
	require.NoError(t, err)

	loaded := loadHotSearchLayout(db)
	require.Len(t, loaded, 1)
	require.Equal(t, "existing", loaded[0].Keyword)
}

func TestResolveHotSearchEntry_Blocked(t *testing.T) {
	pools := buildHotSearchMergePools(context.Background(), nil, nil, 10)
	pools.blocked["testkeyword"] = struct{}{}

	_, ok := resolveHotSearchEntry("testkeyword", "Test", pools)
	require.False(t, ok)
}

func TestResolveHotSearchEntry_EmptyNorm(t *testing.T) {
	pools := buildHotSearchMergePools(context.Background(), nil, nil, 10)
	_, ok := resolveHotSearchEntry("", "Title", pools)
	require.False(t, ok)
}

func TestMergeHotSearchFromLayout_NoLayout(t *testing.T) {
	db := setupHotSearchDB(t)
	result, ok := mergeHotSearchFromLayout(context.Background(), db, nil, 10)
	require.False(t, ok)
	require.Nil(t, result)
}

func TestBuildHotSearchMergePools_NilDB(t *testing.T) {
	pools := buildHotSearchMergePools(context.Background(), nil, nil, 10)
	require.NotNil(t, pools.blocked)
	require.NotNil(t, pools.opByNorm)
	require.NotNil(t, pools.autoBy)
}
