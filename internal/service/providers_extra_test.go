package service

import (
	"cakecake/internal/model/article"
	"cakecake/internal/model/dynamic"
	"cakecake/internal/model/user"
	"cakecake/internal/model/video"
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestProviders_AuthorsAndLookups(t *testing.T) {
	db := newAgentTestDB(t)
	ctx := context.Background()
	videoProv := NewVideoProvider(db)
	articleProv := NewArticleProvider(db)
	dynamicProv := NewDynamicProvider(db)

	require.NoError(t, db.Create(&video.Video{ID: 10, UserID: 5, Title: "v", Status: video.StatusPublished}).Error)
	require.NoError(t, db.Create(&article.Article{ID: 20, UserID: 6, Title: "a", BodyMD: "b", Status: article.StatusPublished}).Error)
	require.NoError(t, db.Create(&dynamic.UserDynamic{ID: 30, UserID: 7, Title: "d"}).Error)

	author, err := videoProv.GetVideoAuthor(ctx, 10)
	require.NoError(t, err)
	require.Equal(t, uint64(5), author)
	_, err = videoProv.GetVideoAuthor(ctx, 999)
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)

	author, err = articleProv.GetArticleAuthor(ctx, 20)
	require.NoError(t, err)
	require.Equal(t, uint64(6), author)

	author, err = dynamicProv.GetDynamicAuthor(ctx, 30)
	require.NoError(t, err)
	require.Equal(t, uint64(7), author)

	// GetPublishedVideo rejects non-published.
	draft := video.Video{ID: 11, UserID: 5, Title: "draft", Status: video.StatusDraft}
	require.NoError(t, db.Create(&draft).Error)
	_, err = videoProv.GetPublishedVideo(ctx, 11)
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)

	// Batch with empty ids.
	vm, err := videoProv.BatchGetPublishedVideos(ctx, nil)
	require.NoError(t, err)
	require.Nil(t, vm)
	um, err := NewUserProvider(db).GetUsersByIDs(ctx, nil)
	require.NoError(t, err)
	require.Nil(t, um)
}

func TestUserProvider_LevelsAndInfo(t *testing.T) {
	db := newAgentTestDB(t)
	ctx := context.Background()
	require.NoError(t, db.Create(&user.User{ID: 1, Username: "alice", PasswordHash: "x", Experience: 3000}).Error)

	p := NewUserProvider(db)
	levels, err := p.BatchCurrentLevels(ctx, []uint64{1})
	require.NoError(t, err)
	require.Equal(t, 6, levels[1])

	u, err := p.GetUser(ctx, 1)
	require.NoError(t, err)
	require.Equal(t, "alice", u.Username)
	_, err = p.GetUser(ctx, 999)
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)
}
