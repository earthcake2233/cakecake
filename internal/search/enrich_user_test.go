package search

import (
	"minibili/internal/model/article"
	"minibili/internal/model/user"
	"minibili/internal/model/video"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

)

func setupEnrichDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&video.Video{}, &article.Article{}, &user.User{}, &user.UserFollow{}))
	return db
}

func TestRecentArchivesForUser_NoVideosNoArticles(t *testing.T) {
	db := setupEnrichDB(t)
	items := recentArchivesForUser(db, 999, 3)
	assert.Empty(t, items)
}

func TestRecentArchivesForUser_ZeroLimit(t *testing.T) {
	db := setupEnrichDB(t)
	items := recentArchivesForUser(db, 1, 0)
	assert.Nil(t, items)
}

func TestRecentArchivesForUser_NegativeLimit(t *testing.T) {
	db := setupEnrichDB(t)
	items := recentArchivesForUser(db, 1, -1)
	assert.Nil(t, items)
}

func TestRecentArchivesForUser_OnlyVideos(t *testing.T) {
	db := setupEnrichDB(t)
	now := time.Now()
	v := video.Video{UserID: 1, Title: "Test Video", CoverURL: "cover.jpg", Status: "published", CreatedAt: now}
	require.NoError(t, db.Create(&v).Error)

	items := recentArchivesForUser(db, 1, 3)
	require.Len(t, items, 1)
	assert.Equal(t, v.ID, items[0].Aid)
	assert.Equal(t, "Test Video", items[0].Title)
	assert.Equal(t, "cover.jpg", items[0].Pic)
	assert.Equal(t, "video", items[0].Rtype)
}

func TestRecentArchivesForUser_LimitVideos(t *testing.T) {
	db := setupEnrichDB(t)
	now := time.Now()
	for i := 0; i < 5; i++ {
		v := video.Video{UserID: 1, Title: "V", CoverURL: "c.jpg", Status: "published", CreatedAt: now.Add(time.Duration(i) * time.Second)}
		require.NoError(t, db.Create(&v).Error)
	}

	items := recentArchivesForUser(db, 1, 3)
	assert.Len(t, items, 3)
}

func TestRecentArchivesForUser_VideosAndArticles(t *testing.T) {
	db := setupEnrichDB(t)
	now := time.Now()
	v := video.Video{UserID: 1, Title: "Video", CoverURL: "v.jpg", Status: "published", CreatedAt: now}
	require.NoError(t, db.Create(&v).Error)
	a := article.Article{UserID: 1, Title: "Article", CoverURL: "a.jpg", Status: "published", CreatedAt: now}
	require.NoError(t, db.Create(&a).Error)

	items := recentArchivesForUser(db, 1, 3)
	require.Len(t, items, 2)
}

func TestRecentArchivesForUser_OnlyUnpublished(t *testing.T) {
	db := setupEnrichDB(t)
	now := time.Now()
	v := video.Video{UserID: 1, Title: "Draft", CoverURL: "c.jpg", Status: "draft", CreatedAt: now}
	require.NoError(t, db.Create(&v).Error)
	a := article.Article{UserID: 1, Title: "Draft Article", CoverURL: "c.jpg", Status: "draft", CreatedAt: now}
	require.NoError(t, db.Create(&a).Error)

	items := recentArchivesForUser(db, 1, 3)
	assert.Empty(t, items)
}
