package data

import (
	"cakecake/internal/model/admin"
	"cakecake/internal/model/article"
	"cakecake/internal/model/comment"
	"cakecake/internal/model/danmaku"
	"cakecake/internal/model/notification"
	"cakecake/internal/model/user"
	"cakecake/internal/model/video"
	"errors"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"cakecake/internal/config"
)

func setupDataDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	return db
}

func TestAutoMigrateAll(t *testing.T) {
	db := setupDataDB(t)
	lg := zap.NewNop()
	err := AutoMigrateAll(db, lg)
	require.NoError(t, err)

	assert.True(t, db.Migrator().HasTable(&user.User{}))
	assert.True(t, db.Migrator().HasTable(&video.Video{}))
	assert.True(t, db.Migrator().HasTable(&comment.Comment{}))
	assert.True(t, db.Migrator().HasTable(&article.Article{}))
	assert.True(t, db.Migrator().HasTable(&admin.Admin{}))
	assert.True(t, db.Migrator().HasTable(&danmaku.Danmaku{}))
	assert.True(t, db.Migrator().HasTable(&user.UserFollow{}))
	assert.True(t, db.Migrator().HasTable(&notification.Notification{}))
	assert.True(t, db.Migrator().HasTable(&admin.HomeBanner{}))
}

func TestAutoMigrateAll_Idempotent(t *testing.T) {
	db := setupDataDB(t)
	lg := zap.NewNop()
	err := AutoMigrateAll(db, lg)
	require.NoError(t, err)
	err = AutoMigrateAll(db, lg)
	require.NoError(t, err)
}

func TestAutoMigrateAll_NilLogger(t *testing.T) {
	db := setupDataDB(t)
	err := AutoMigrateAll(db, nil)
	require.NoError(t, err)
	assert.True(t, db.Migrator().HasTable(&user.User{}))
}

func TestSeedDefaultAdmin(t *testing.T) {
	db := setupDataDB(t)
	require.NoError(t, db.AutoMigrate(&admin.Admin{}))

	cfg := &config.C{AdminSeedUsername: "admin", AdminSeedPassword: "secret123"}
	err := SeedDefaultAdmin(db, cfg, zap.NewNop())
	require.NoError(t, err)

	var admins []admin.Admin
	db.Find(&admins)
	require.Len(t, admins, 1)
	assert.Equal(t, "admin", admins[0].Username)
	assert.NotEmpty(t, admins[0].PasswordHash)
}

func TestSeedDefaultAdmin_SkipNoConfig(t *testing.T) {
	db := setupDataDB(t)
	require.NoError(t, db.AutoMigrate(&admin.Admin{}))

	err := SeedDefaultAdmin(db, nil, zap.NewNop())
	require.NoError(t, err)

	var count int64
	db.Model(&admin.Admin{}).Count(&count)
	assert.Equal(t, int64(0), count)
}

func TestSeedDefaultAdmin_Idempotent(t *testing.T) {
	db := setupDataDB(t)
	require.NoError(t, db.AutoMigrate(&admin.Admin{}))

	cfg := &config.C{AdminSeedUsername: "admin", AdminSeedPassword: "pass"}
	err := SeedDefaultAdmin(db, cfg, zap.NewNop())
	require.NoError(t, err)

	err = SeedDefaultAdmin(db, cfg, zap.NewNop())
	require.NoError(t, err)

	var count int64
	db.Model(&admin.Admin{}).Count(&count)
	assert.Equal(t, int64(1), count)
}

func TestDBColumnExists(t *testing.T) {
	db := setupDataDB(t)
	require.NoError(t, db.AutoMigrate(&video.Video{}, &comment.Comment{}))

	assert.True(t, dbColumnExists(db, "videos", "comments_closed"))
	assert.True(t, dbColumnExists(db, "videos", "comments_curated"))
	assert.True(t, dbColumnExists(db, "videos", "danmaku_closed"))
	assert.True(t, dbColumnExists(db, "comments", "approved"))
	assert.True(t, dbColumnExists(db, "comments", "curated_ignored"))

	assert.False(t, dbColumnExists(db, "videos", "nonexistent_column"))
	assert.False(t, dbColumnExists(db, "nonexistent_table", "id"))
}

func TestBackfillUserCakeIDs(t *testing.T) {
	db := setupDataDB(t)
	require.NoError(t, db.AutoMigrate(&user.User{}))

	db.Create(&user.User{ID: 1, Username: "user1", Nickname: "User One"})
	db.Create(&user.User{ID: 2, Username: "user2", Nickname: "User Two"})
	db.Create(&user.User{ID: 3, Username: "user3", Nickname: "User Three", CakeID: "existing_cake"})

	err := backfillUserCakeIDs(db, zap.NewNop())
	require.NoError(t, err)

	var u1, u2, u3 user.User
	db.First(&u1, 1)
	db.First(&u2, 2)
	db.First(&u3, 3)

	assert.NotEmpty(t, u1.CakeID)
	assert.NotEmpty(t, u2.CakeID)
	assert.Equal(t, "existing_cake", u3.CakeID)
}

func TestNewDB_EmptyDSN(t *testing.T) {
	_, err := NewDB("", zap.NewNop(), true)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "MYSQL_DSN is empty")
}

func TestIsIgnorableAddColumnErr(t *testing.T) {
	tests := []struct {
		err  error
		want bool
	}{
		{nil, false},
		{errors.New("duplicate column name"), true},
		{errors.New("Duplicate column"), true},
		{errors.New("table not found"), false},
		{errors.New("syntax error"), false},
	}
	for _, tc := range tests {
		got := isIgnorableAddColumnErr(tc.err)
		assert.Equal(t, tc.want, got, "isIgnorableAddColumnErr(%v)", tc.err)
	}
}

func TestNewRedis_InvalidConfig(t *testing.T) {
	cfg := &config.C{
		RedisAddr:  "127.0.0.1:1",
		RedisDial:  time.Second,
		RedisRead:  time.Second,
		RedisWrite: time.Second,
	}
	_, err := NewRedis(cfg)
	require.Error(t, err)
}
