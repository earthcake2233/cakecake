package search

import (
	"cakecake/internal/model/article"
	"cakecake/internal/model/user"
	"cakecake/internal/model/video"
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

func TestEnrichUserHits_Normal(t *testing.T) {
	db := setupEnrichDB(t)
	now := time.Now()
	require.NoError(t, db.Create(&user.User{ID: 1, Username: "alice", Nickname: "Alice", Sign: "hello", Experience: 100}).Error)
	require.NoError(t, db.Create(&video.Video{ID: 10, UserID: 1, Title: "v1", Status: video.StatusPublished, CoverURL: "c", CreatedAt: now}).Error)
	require.NoError(t, db.Create(&article.Article{ID: 20, UserID: 1, Title: "a1", BodyMD: "b", Status: article.StatusPublished}).Error)
	require.NoError(t, db.Create(&user.UserFollow{FollowerID: 2, FolloweeID: 1}).Error)

	hits := EnrichUserHits(db, 2, []UserHit{{Mid: 1, Uname: "alice"}})
	require.Len(t, hits, 1)
	require.Equal(t, "alice", hits[0].Uname) // uname comes from the ES hit, untouched
	require.Equal(t, "hello", hits[0].Usign)
	require.Positive(t, hits[0].Level)
	require.Equal(t, 2, hits[0].Archives)
	require.Equal(t, 1, hits[0].Fans)
	require.True(t, hits[0].FollowedByMe)
	require.Len(t, hits[0].Recent, 2)
}

func TestEnrichUserHits_NotFollowedAndMissing(t *testing.T) {
	db := setupEnrichDB(t)
	require.NoError(t, db.Create(&user.User{ID: 1, Username: "alice", Sign: "sign"}).Error)

	hits := EnrichUserHits(db, 1, []UserHit{{Mid: 1, Uname: "alice"}, {Mid: 999}})
	require.Len(t, hits, 2)
	require.False(t, hits[0].FollowedByMe)
	require.Equal(t, "alice", hits[0].Uname)
	// Missing user: hit left unchanged.
	require.Equal(t, uint64(999), hits[1].Mid)
}

func TestEnrichUserHits_Anonymized(t *testing.T) {
	db := setupEnrichDB(t)
	now := time.Now()
	require.NoError(t, db.Create(&user.User{ID: 1, Username: "gone", AnonymizedAt: &now}).Error)
	hits := EnrichUserHits(db, 0, []UserHit{{Mid: 1, Uname: "old"}})
	require.Len(t, hits, 1)
	require.Equal(t, "已注销用户", hits[0].Uname)
	require.Empty(t, hits[0].Usign)
	require.Nil(t, hits[0].Recent)
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
