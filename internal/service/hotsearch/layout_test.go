package hotsearch

import (
	"cakecake/internal/model/admin"
	"context"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupHotSearchDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&admin.HotSearchDisplayLayout{}, &admin.HotSearchOp{}))
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
	layout := admin.HotSearchDisplayLayout{ID: 1, OrderJSON: "[]"}
	err := db.Save(&layout).Error
	require.NoError(t, err)

	entries := loadHotSearchLayout(db)
	require.Nil(t, entries)
}

func TestLoadHotSearchLayout_InvalidJSON(t *testing.T) {
	db := setupHotSearchDB(t)
	layout := admin.HotSearchDisplayLayout{ID: 1, OrderJSON: "not-json"}
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
