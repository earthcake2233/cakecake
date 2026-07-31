package service

import (
	"minibili/internal/model/admin"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func svcDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&admin.HotSearchDisplayLayout{}, &admin.HotSearchOp{}))
	return db
}

func Test_HotSearchLayout_Clear(t *testing.T) {
	db := svcDB(t)
	require.False(t, HasHotSearchDisplayLayout(db))
	require.NoError(t, ClearHotSearchDisplayLayout(db))
}

func Test_HotSearchLayout_SaveAndLoad(t *testing.T) {
	db := svcDB(t)
	entries := []HotSearchLayoutEntry{
		{Keyword: "kw1", Title: "KW One"},
		{Keyword: "kw2", Title: "KW Two"},
	}
	require.NoError(t, SaveHotSearchDisplayLayout(db, entries))
	require.True(t, HasHotSearchDisplayLayout(db))

	loaded := loadHotSearchLayout(db)
	require.Len(t, loaded, 2)
	require.Equal(t, "kw1", loaded[0].Keyword)
}

func Test_HotSearchLayout_MoveAndRemove(t *testing.T) {
	db := svcDB(t)
	entries := []HotSearchLayoutEntry{
		{Keyword: "k1", Title: "K1"},
		{Keyword: "k2", Title: "K2"},
	}
	require.NoError(t, SaveHotSearchDisplayLayout(db, entries))

	require.NoError(t, ApplyHotSearchLayoutMove(db, "k1", "K1", 2))
	// After move: k2, k1
	require.NoError(t, RemoveHotSearchLayoutEntry(db, "k1"))
	// After remove k1: k2 still exists
	require.True(t, HasHotSearchDisplayLayout(db))
	require.NoError(t, RemoveHotSearchLayoutEntry(db, "k2"))
	// After removing all: layout cleared
	require.False(t, HasHotSearchDisplayLayout(db))
}
