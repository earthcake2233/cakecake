package data

import (
	"cakecake/internal/model/extra"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func newSearchHistoryMigrateDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&extra.UserSearchHistory{}))
	return db
}

func TestCleanupUserSearchHistory(t *testing.T) {
	db := newSearchHistoryMigrateDB(t)
	now := time.Now()
	require.NoError(t, db.Create(&extra.UserSearchHistory{UserID: 1, Keyword: "Golang", UpdatedAt: now}).Error)
	require.NoError(t, db.Create(&extra.UserSearchHistory{UserID: 1, Keyword: "golang", UpdatedAt: now.Add(time.Minute)}).Error)
	require.NoError(t, db.Create(&extra.UserSearchHistory{UserID: 1, Keyword: "   ", UpdatedAt: now.Add(2 * time.Minute)}).Error)
	require.NoError(t, db.Create(&extra.UserSearchHistory{UserID: 2, Keyword: "rust", UpdatedAt: now.Add(3 * time.Minute)}).Error)

	require.NoError(t, CleanupUserSearchHistory(db, nil))

	var rows []extra.UserSearchHistory
	require.NoError(t, db.Find(&rows).Error)
	require.Len(t, rows, 2) // one per user, blank removed
	for _, r := range rows {
		require.NotEmpty(t, r.KeywordNorm)
	}
}

func TestMigrateUserSearchHistory(t *testing.T) {
	db := newSearchHistoryMigrateDB(t)
	require.NoError(t, db.Create(&extra.UserSearchHistory{UserID: 1, Keyword: "dup", UpdatedAt: time.Now()}).Error)
	require.NoError(t, db.Create(&extra.UserSearchHistory{UserID: 1, Keyword: "DUP", UpdatedAt: time.Now().Add(time.Minute)}).Error)

	require.NoError(t, migrateUserSearchHistory(db, nil))

	// Unique index on (user_id, keyword_norm) now exists; duplicates deduped.
	require.True(t, db.Migrator().HasIndex(&extra.UserSearchHistory{}, "idx_user_search_norm"))
	var n int64
	require.NoError(t, db.Model(&extra.UserSearchHistory{}).Count(&n).Error)
	require.Equal(t, int64(1), n)
}

func TestMigrateUserSearchHistory_NoTable(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, migrateUserSearchHistory(db, nil))
}
