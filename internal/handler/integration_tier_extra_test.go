//go:build integration

package handler

import (
	"cakecake/internal/model/article"
	"cakecake/internal/model/dynamic"
	"cakecake/internal/model/extra"
	"cakecake/internal/model/user"
	"cakecake/internal/model/video"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestTierExtra_ArticleEngagementInDetail(t *testing.T) {
	api, r, token := setupHandlerIntegrationDB(t)
	require.NoError(t, api.DB.Create(&user.User{ID: 1, Username: "u1", PasswordHash: "x", CoinBalanceTenths: 230}).Error)
	require.NoError(t, api.DB.Create(&article.Article{ID: 70, UserID: 2, Title: "a", BodyMD: "b", Status: article.StatusPublished}).Error)

	// Favorite + coin, then GET detail with viewer token to exercise engagement mapping.
	w := doJSON(r, "POST", "/api/v1/articles/70/favorite", token, nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	w = doJSON(r, "POST", "/api/v1/articles/70/coin", token, map[string]interface{}{"amount": 2})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	w = doReq(r, "GET", "/api/v1/articles/70", token, "", nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.Contains(t, w.Body.String(), "favorited_by_me")
}

func TestTierExtra_WatchLaterAndFolderEdges(t *testing.T) {
	api, r, token := setupHandlerIntegrationDB(t)
	require.NoError(t, api.DB.Create(&user.User{ID: 1, Username: "u1", PasswordHash: "x", CoinBalanceTenths: 230}).Error)
	require.NoError(t, api.DB.Create(&video.Video{ID: 10, UserID: 2, Title: "v", Status: video.StatusPublished}).Error)

	// Watch-later list with an entry referencing a published video.
	require.NoError(t, api.DB.Create(&video.WatchLater{UserID: 1, VideoID: 10}).Error)
	w := doReq(r, "GET", "/api/v1/users/me/watch-later", token, "", nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.Contains(t, w.Body.String(), "v")

	// Set favorite folders to a non-owned folder -> bad request.
	w = doJSON(r, "PUT", "/api/v1/videos/10/favorite-folders", token, map[string]interface{}{
		"folder_ids": []uint64{999},
	})
	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())

	// Move with missing video.
	w = doJSON(r, "PUT", "/api/v1/videos/999/favorite-folders/move", token, map[string]interface{}{
		"from_folder_id": 1, "to_folder_id": 2,
	})
	require.Equal(t, http.StatusNotFound, w.Code, w.Body.String())
}

func TestTierExtra_ViewHistoryList(t *testing.T) {
	api, r, token := setupHandlerIntegrationDB(t)
	require.NoError(t, api.DB.Create(&user.User{ID: 1, Username: "u1", PasswordHash: "x", CoinBalanceTenths: 230}).Error)
	require.NoError(t, api.DB.Create(&user.User{ID: 2, Username: "u2", PasswordHash: "x", CoinBalanceTenths: 230}).Error)
	require.NoError(t, api.DB.Create(&video.Video{ID: 10, UserID: 2, Title: "watched video", Status: video.StatusPublished}).Error)
	require.NoError(t, api.DB.Create(&article.Article{ID: 20, UserID: 2, Title: "read article", BodyMD: "b", Status: article.StatusPublished}).Error)
	now := time.Now()
	require.NoError(t, api.DB.Create(&extra.VideoViewHistory{UserID: 1, VideoID: 10, ViewedAt: now}).Error)
	require.NoError(t, api.DB.Create(&extra.ArticleViewHistory{UserID: 1, ArticleID: 20, ViewedAt: now.Add(-time.Minute)}).Error)

	w := doReq(r, "GET", "/api/v1/users/me/view-history", token, "", nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.Contains(t, w.Body.String(), "watched video")
	require.Contains(t, w.Body.String(), "read article")

	// Keyword filter.
	w = doReq(r, "GET", "/api/v1/users/me/view-history?q=watched", token, "", nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
}

func TestTierExtra_DailyRewards(t *testing.T) {
	api, r, token := setupHandlerIntegrationDB(t)
	require.NoError(t, api.DB.Create(&user.User{ID: 1, Username: "u1", PasswordHash: "x", CoinBalanceTenths: 230}).Error)

	w := doReq(r, "GET", "/api/v1/users/me/daily-rewards", token, "", nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	w = doJSON(r, "POST", "/api/v1/users/me/daily-rewards/watch", token, nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
}

func TestTierExtra_DynamicKeepImages(t *testing.T) {
	api, r, token := setupHandlerIntegrationDB(t)
	require.NoError(t, api.DB.Create(&user.User{ID: 1, Username: "u1", PasswordHash: "x", CoinBalanceTenths: 230}).Error)
	require.NoError(t, api.DB.Create(&dynamic.UserDynamic{ID: 30, UserID: 1, Title: "d", Content: "x"}).Error)

	// Update with keep_images (no new uploads; OSS nil so keep URLs pass through).
	w := doMultipart(r, "PUT", "/api/v1/users/me/dynamics/30", token, map[string]string{
		"title":       "renamed",
		"content":     "updated",
		"keep_images": "https://cdn.example.com/a.jpg",
	})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
}
