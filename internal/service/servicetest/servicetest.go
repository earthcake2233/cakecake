// Package servicetest provides shared test helpers for the service domain
// packages (sqlite DB, miniredis, logger, and common seeders).
package servicetest

import (
	"cakecake/internal/data"
	"cakecake/internal/model/user"
	"cakecake/internal/model/video"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/glebarez/sqlite"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// NewDB opens an in-memory sqlite database with all models migrated.
func NewDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, data.AutoMigrateAll(db, zap.NewNop()))
	return db
}

// NewRedis starts an in-memory miniredis and returns it with a client.
func NewRedis(t *testing.T) (*miniredis.Miniredis, *redis.Client) {
	t.Helper()
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	return mr, rdb
}

// ZapNop returns a no-op logger for tests.
func ZapNop() *zap.Logger {
	return zap.NewNop()
}

// SeedUser inserts a minimal user row.
func SeedUser(t *testing.T, db *gorm.DB, id uint64, username string) *user.User {
	t.Helper()
	u := &user.User{ID: id, Username: username, PasswordHash: "x", CakeID: user.FormatCakeID(id)}
	require.NoError(t, db.Create(u).Error)
	return u
}

// SeedVideoForFav inserts a minimal video row (published unless noted).
func SeedVideoForFav(t *testing.T, db *gorm.DB, id, owner uint64, published bool) {
	t.Helper()
	status := video.StatusPublished
	if !published {
		status = video.StatusDraft
	}
	require.NoError(t, db.Create(&video.Video{
		ID: id, UserID: owner, Title: "v", Status: status, CoverURL: "cover.jpg",
	}).Error)
}
