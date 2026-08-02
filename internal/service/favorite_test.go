package service

import (
	"cakecake/internal/model/video"
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func newFavoriteService(t *testing.T) *FavoriteService {
	t.Helper()
	db := newAgentTestDB(t)
	_, rdb := newAgentTestRedis(t)
	return NewFavoriteService(db, rdb, zapNop(), NewUserProvider(db), NewVideoProvider(db))
}

func seedVideoForFav(t *testing.T, db *gorm.DB, id, owner uint64, published bool) {
	t.Helper()
	status := video.StatusPublished
	if !published {
		status = video.StatusDraft
	}
	require.NoError(t, db.Create(&video.Video{
		ID: id, UserID: owner, Title: "v", Status: status, CoverURL: "cover.jpg",
	}).Error)
}

func TestFavoriteService_FolderCRUD(t *testing.T) {
	s := newFavoriteService(t)
	ctx := context.Background()

	f := &video.FavoriteFolder{UserID: 1, Title: "my folder", IsPublic: true}
	require.NoError(t, s.CreateFolder(ctx, f))
	require.NotZero(t, f.ID)

	got, err := s.GetFolderByID(ctx, f.ID)
	require.NoError(t, err)
	require.Equal(t, "my folder", got.Title)
	_, err = s.GetFolderByID(ctx, 999)
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)

	list, err := s.ListFoldersByUser(ctx, 1)
	require.NoError(t, err)
	require.Len(t, list, 1)

	require.NoError(t, s.UpdateFolder(ctx, f.ID, map[string]interface{}{"title": "renamed"}))
	got, err = s.GetFolderByID(ctx, f.ID)
	require.NoError(t, err)
	require.Equal(t, "renamed", got.Title)

	require.NoError(t, s.UpdateFolderCover(ctx, f.ID, "newcover.jpg"))
	got, err = s.GetFolderByID(ctx, f.ID)
	require.NoError(t, err)
	require.Equal(t, "newcover.jpg", got.CoverURL)

	n, err := s.CountFoldersByUser(ctx, 1)
	require.NoError(t, err)
	require.Equal(t, int64(1), n)
	n, err = s.CountPublicFoldersByUser(ctx, 1)
	require.NoError(t, err)
	require.Equal(t, int64(1), n)

	// SearchFolders.
	searched, err := s.SearchFolders(ctx, 1, "renam", 10)
	require.NoError(t, err)
	require.Len(t, searched, 1)

	require.NoError(t, s.DeleteFolder(ctx, f.ID))
	_, err = s.GetFolderByID(ctx, f.ID)
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)
}

func TestFavoriteService_Favorites(t *testing.T) {
	s := newFavoriteService(t)
	ctx := context.Background()
	seedUser(t, s.db, 1, "alice")
	seedVideoForFav(t, s.db, 10, 2, true)
	folder := &video.FavoriteFolder{UserID: 1, Title: "favs"}
	require.NoError(t, s.CreateFolder(ctx, folder))

	// AddFavorite + checks.
	require.NoError(t, s.AddFavorite(ctx, folder.ID, 10, 1))
	require.True(t, s.IsFavorited(ctx, 1, 10))
	require.Equal(t, int64(1), s.CountByUser(ctx, 1))
	require.Equal(t, map[uint64]bool{10: true}, s.BatchFavorited(ctx, 1, []uint64{10, 11}))

	exists, err := s.CheckFavoriteExists(ctx, 1, folder.ID, 10)
	require.NoError(t, err)
	require.True(t, exists)

	list, total, err := s.ListFavoritesByFolder(ctx, folder.ID, 1, 10)
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.Equal(t, int64(1), total)

	_, total, vids, err := s.ListFavoritesByFolderWithVideoIds(ctx, folder.ID, 1, 10)
	require.NoError(t, err)
	require.Equal(t, []uint64{10}, vids)
	require.Equal(t, int64(1), total)

	n, err := s.CountFavoritesInFolder(ctx, folder.ID)
	require.NoError(t, err)
	require.Equal(t, int64(1), n)

	// RemoveFavorite + DeleteFavoriteByVideo.
	require.NoError(t, s.RemoveFavorite(ctx, folder.ID, 10))
	require.False(t, s.IsFavorited(ctx, 1, 10))

	require.NoError(t, s.AddFavorite(ctx, folder.ID, 10, 1))
	require.NoError(t, s.DeleteFavoriteByVideo(ctx, 1, folder.ID, 10))
	require.NoError(t, s.AddFavorite(ctx, folder.ID, 10, 1))
	require.NoError(t, s.DeleteFavoritesByFolder(ctx, folder.ID))

	// DeleteFavoriteEntry by primary key.
	fav := video.VideoFavorite{UserID: 1, VideoID: 10, FolderID: folder.ID}
	require.NoError(t, s.db.Create(&fav).Error)
	require.NoError(t, s.DeleteFavoriteEntry(ctx, fav.ID))
}

func TestFavoriteService_FolderCoverFromVideos(t *testing.T) {
	s := newFavoriteService(t)
	ctx := context.Background()
	seedVideoForFav(t, s.db, 10, 2, true)

	// No favorites -> empty cover.
	require.Empty(t, s.FolderCoverFromVideos(ctx, 1))

	folder := &video.FavoriteFolder{UserID: 1, Title: "favs"}
	require.NoError(t, s.CreateFolder(ctx, folder))
	require.NoError(t, s.AddFavorite(ctx, folder.ID, 10, 1))
	require.Equal(t, "cover.jpg", s.FolderCoverFromVideos(ctx, folder.ID))
}

func TestFavoriteService_SetVideoFavoriteFolders(t *testing.T) {
	s := newFavoriteService(t)
	ctx := context.Background()
	seedUser(t, s.db, 1, "alice")
	seedVideoForFav(t, s.db, 10, 2, true)
	f1 := &video.FavoriteFolder{UserID: 1, Title: "f1"}
	f2 := &video.FavoriteFolder{UserID: 1, Title: "f2"}
	require.NoError(t, s.CreateFolder(ctx, f1))
	require.NoError(t, s.CreateFolder(ctx, f2))

	// Add to two folders.
	res, err := s.SetVideoFavoriteFolders(ctx, 1, 10, []uint64{f1.ID, f2.ID})
	require.NoError(t, err)
	require.True(t, res.Favorited)
	require.ElementsMatch(t, []uint64{f1.ID, f2.ID}, res.FolderIDs)

	// Requesting a folder the user does not own -> nil result.
	res, err = s.SetVideoFavoriteFolders(ctx, 1, 10, []uint64{999})
	require.NoError(t, err)
	require.Nil(t, res)

	// Remove all.
	res, err = s.SetVideoFavoriteFolders(ctx, 1, 10, nil)
	require.NoError(t, err)
	require.False(t, res.Favorited)
	var n int64
	require.NoError(t, s.db.Model(&video.VideoFavorite{}).Count(&n).Error)
	require.Zero(t, n)
}

func TestFavoriteService_ListUserFavoriteVideos(t *testing.T) {
	s := newFavoriteService(t)
	ctx := context.Background()
	seedUser(t, s.db, 1, "alice")
	seedUser(t, s.db, 2, "bob")
	seedVideoForFav(t, s.db, 10, 2, true)
	seedVideoForFav(t, s.db, 11, 2, true)
	folder := &video.FavoriteFolder{UserID: 1, Title: "favs"}
	require.NoError(t, s.CreateFolder(ctx, folder))
	require.NoError(t, s.AddFavorite(ctx, folder.ID, 10, 1))
	require.NoError(t, s.AddFavorite(ctx, folder.ID, 11, 1))

	res, err := s.ListUserFavoriteVideos(ctx, 1, 10, 0, false)
	require.NoError(t, err)
	require.Equal(t, int64(2), res.Total)
	require.Len(t, res.Items, 2)
	require.Equal(t, "bob", res.Items[0].UploaderName)

	// filterFolder with a different folder -> empty.
	res, err = s.ListUserFavoriteVideos(ctx, 1, 10, 999, true)
	require.NoError(t, err)
	require.Empty(t, res.Items)
}

func TestFavoriteService_MigrateAndDeleteCascade(t *testing.T) {
	s := newFavoriteService(t)
	ctx := context.Background()
	seedUser(t, s.db, 1, "alice")
	seedVideoForFav(t, s.db, 10, 2, true)
	from := &video.FavoriteFolder{UserID: 1, Title: "from"}
	to := &video.FavoriteFolder{UserID: 1, Title: "to"}
	require.NoError(t, s.CreateFolder(ctx, from))
	require.NoError(t, s.CreateFolder(ctx, to))
	require.NoError(t, s.AddFavorite(ctx, 0, 10, 1)) // orphan favorite (folder_id=0)

	require.NoError(t, s.MigrateOrphanFavorites(ctx, 1, to.ID))
	var n int64
	require.NoError(t, s.db.Model(&video.VideoFavorite{}).Where("folder_id = ?", to.ID).Count(&n).Error)
	require.Equal(t, int64(1), n)

	require.NoError(t, s.DeleteFolderCascade(ctx, to.ID, 1))
	require.NoError(t, s.db.Model(&video.VideoFavorite{}).Count(&n).Error)
	require.Zero(t, n)
}

func TestFavoriteService_FilterPublishedVideoIDs(t *testing.T) {
	s := newFavoriteService(t)
	ctx := context.Background()
	seedVideoForFav(t, s.db, 10, 2, true)
	seedVideoForFav(t, s.db, 11, 2, false)

	ids, err := s.FilterPublishedVideoIDs(ctx, []uint64{10, 11, 99})
	require.NoError(t, err)
	require.Equal(t, []uint64{10}, ids)

	ids, err = s.FilterPublishedVideoIDs(ctx, nil)
	require.NoError(t, err)
	require.Nil(t, ids)
}
