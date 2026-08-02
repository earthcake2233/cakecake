//go:build integration

package handler

import (
	"cakecake/internal/model/user"
	"cakecake/internal/model/video"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestFavoriteFolder_CRUD(t *testing.T) {
	api, r, token := setupHandlerIntegrationDB(t)
	require.NoError(t, api.DB.Create(&user.User{ID: 1, Username: "u1", PasswordHash: "x", CoinBalanceTenths: 230}).Error)

	// List empty -> default folder may exist.
	w := doReq(r, "GET", "/api/v1/users/me/favorite-folders", token, "", nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	// Create.
	w = doJSON(r, "POST", "/api/v1/users/me/favorite-folders", token, map[string]interface{}{
		"title": "my folder", "description": "desc", "is_public": true,
	})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var cr struct {
		Code int `json:"code"`
		Data struct {
			ID uint64 `json:"id"`
		} `json:"data"`
	}
	decodeBody(t, w, &cr)
	require.Equal(t, 0, cr.Code)
	require.NotZero(t, cr.Data.ID)
	folderID := cr.Data.ID

	// Create with empty title -> bad request.
	w = doJSON(r, "POST", "/api/v1/users/me/favorite-folders", token, map[string]interface{}{"title": ""})
	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())

	// Update.
	w = doJSON(r, "PUT", fmt.Sprintf("/api/v1/users/me/favorite-folders/%d", folderID), token, map[string]interface{}{
		"title": "renamed", "is_public": false,
	})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var f video.FavoriteFolder
	require.NoError(t, api.DB.First(&f, folderID).Error)
	require.Equal(t, "renamed", f.Title)
	require.False(t, f.IsPublic)

	// Update another user's folder -> not found.
	require.NoError(t, api.DB.Create(&video.FavoriteFolder{ID: 99, UserID: 2, Title: "other"}).Error)
	w = doJSON(r, "PUT", "/api/v1/users/me/favorite-folders/99", token, map[string]interface{}{"title": "x"})
	require.Equal(t, http.StatusNotFound, w.Code, w.Body.String())

	// Delete.
	w = doReq(r, "DELETE", fmt.Sprintf("/api/v1/users/me/favorite-folders/%d", folderID), token, "", nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.Error(t, api.DB.First(&video.FavoriteFolder{}, folderID).Error)
}

func TestFavoriteFolder_VideoOps(t *testing.T) {
	api, r, token := setupHandlerIntegrationDB(t)
	require.NoError(t, api.DB.Create(&user.User{ID: 1, Username: "u1", PasswordHash: "x", CoinBalanceTenths: 230}).Error)
	require.NoError(t, api.DB.Create(&video.Video{
		ID: 10, UserID: 2, Title: "v", Status: video.StatusPublished, CreatedAt: time.Now(),
	}).Error)
	folder := video.FavoriteFolder{UserID: 1, Title: "favs"}
	require.NoError(t, api.DB.Create(&folder).Error)

	// Add video to folder.
	w := doJSON(r, "POST", fmt.Sprintf("/api/v1/videos/10/favorite-folders/%d", folder.ID), token, nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var n int64
	require.NoError(t, api.DB.Model(&video.VideoFavorite{}).Where("folder_id = ?", folder.ID).Count(&n).Error)
	require.Equal(t, int64(1), n)

	// List my favorites.
	w = doReq(r, "GET", "/api/v1/users/me/favorites", token, "", nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	// Favorite picker for the video.
	w = doReq(r, "GET", "/api/v1/videos/10/favorite-picker", token, "", nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	// Remove video from folder.
	w = doReq(r, "DELETE", fmt.Sprintf("/api/v1/videos/10/favorite-folders/%d", folder.ID), token, "", nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NoError(t, api.DB.Model(&video.VideoFavorite{}).Where("folder_id = ?", folder.ID).Count(&n).Error)
	require.Zero(t, n)

	// Batch remove.
	require.NoError(t, api.DB.Create(&video.VideoFavorite{UserID: 1, VideoID: 10, FolderID: folder.ID}).Error)
	w = doJSON(r, "POST", fmt.Sprintf("/api/v1/users/me/favorite-folders/%d/batch-remove", folder.ID), token, map[string]interface{}{
		"video_ids": []uint64{10},
	})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NoError(t, api.DB.Model(&video.VideoFavorite{}).Where("folder_id = ?", folder.ID).Count(&n).Error)
	require.Zero(t, n)

	// Move favorites between folders.
	folder2 := video.FavoriteFolder{UserID: 1, Title: "favs2"}
	require.NoError(t, api.DB.Create(&folder2).Error)
	require.NoError(t, api.DB.Create(&video.VideoFavorite{UserID: 1, VideoID: 10, FolderID: folder.ID}).Error)
	w = doJSON(r, "PUT", "/api/v1/videos/10/favorite-folders/move", token, map[string]interface{}{
		"from_folder_id": folder.ID, "to_folder_id": folder2.ID,
	})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	// Clear invalid favorites.
	w = doReq(r, "DELETE", fmt.Sprintf("/api/v1/users/me/favorite-folders/%d/invalid-favorites", folder2.ID), token, "", nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
}

func TestFavoriteFolder_PublicLists(t *testing.T) {
	api, r, token := setupHandlerIntegrationDB(t)
	require.NoError(t, api.DB.Create(&user.User{ID: 1, Username: "u1", PasswordHash: "x", CoinBalanceTenths: 230}).Error)
	require.NoError(t, api.DB.Create(&video.Video{ID: 10, UserID: 1, Title: "v", Status: video.StatusPublished}).Error)
	folder := video.FavoriteFolder{UserID: 1, Title: "public", IsPublic: true}
	require.NoError(t, api.DB.Create(&folder).Error)
	require.NoError(t, api.DB.Create(&video.VideoFavorite{UserID: 1, VideoID: 10, FolderID: folder.ID}).Error)

	// Public space lists.
	w := doReq(r, "GET", "/api/v1/space/1/favorite-folders", token, "", nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	w = doReq(r, "GET", "/api/v1/space/1/favorites", token, "", nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
}
