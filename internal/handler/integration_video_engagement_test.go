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

func TestVideoEngagement_LikeFavCoinWatch(t *testing.T) {
	api, r, token := setupHandlerIntegrationDB(t)
	require.NoError(t, api.DB.Create(&user.User{ID: 1, Username: "u1", PasswordHash: "x", CoinBalanceTenths: 230}).Error)
	require.NoError(t, api.DB.Create(&video.Video{
		ID: 10, UserID: 2, Title: "v", Status: video.StatusPublished, CreatedAt: time.Now(),
	}).Error)

	// Like toggle.
	w := doJSON(r, "POST", "/api/v1/videos/10/like", token, nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	w = doJSON(r, "POST", "/api/v1/videos/10/like", token, nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	// Favorite toggle (creates default folder).
	w = doJSON(r, "POST", "/api/v1/videos/10/favorite", token, nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	// Favorite picker.
	w = doReq(r, "GET", "/api/v1/videos/10/favorite-picker", token, "", nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	// Set favorite folders.
	folder := video.FavoriteFolder{UserID: 1, Title: "favs"}
	require.NoError(t, api.DB.Create(&folder).Error)
	w = doJSON(r, "PUT", "/api/v1/videos/10/favorite-folders", token, map[string]interface{}{
		"folder_ids": []uint64{folder.ID},
	})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	// Move between folders.
	folder2 := video.FavoriteFolder{UserID: 1, Title: "favs2"}
	require.NoError(t, api.DB.Create(&folder2).Error)
	w = doJSON(r, "PUT", "/api/v1/videos/10/favorite-folders/move", token, map[string]interface{}{
		"from_folder_id": folder.ID, "to_folder_id": folder2.ID,
	})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	// Coin (video 10 by uploader 2; user 1 has coins).
	w = doJSON(r, "POST", "/api/v1/videos/10/coin", token, map[string]interface{}{"amount": 1})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	// Watch later toggle + list + mark watched + clear watched + clear all.
	w = doJSON(r, "POST", "/api/v1/videos/10/watch-later", token, nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	w = doReq(r, "GET", "/api/v1/users/me/watch-later", token, "", nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	w = doJSON(r, "POST", "/api/v1/users/me/watch-later/10/watched", token, nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	w = doReq(r, "DELETE", "/api/v1/users/me/watch-later/watched", token, "", nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	w = doReq(r, "DELETE", "/api/v1/users/me/watch-later", token, "", nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	// Error paths.
	w = doJSON(r, "POST", "/api/v1/videos/999/like", token, nil)
	require.Equal(t, http.StatusNotFound, w.Code, w.Body.String())
	w = doJSON(r, "POST", "/api/v1/videos/999/coin", token, map[string]interface{}{"amount": 1})
	require.Equal(t, http.StatusNotFound, w.Code, w.Body.String())
	w = doJSON(r, "POST", fmt.Sprintf("/api/v1/videos/10/favorite-folders/%d", folder.ID), token, nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	w = doReq(r, "DELETE", fmt.Sprintf("/api/v1/videos/10/favorite-folders/%d", folder.ID), token, "", nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
}
