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

func TestVideo_ListGetSpace(t *testing.T) {
	api, r, token := setupHandlerIntegrationDB(t)
	require.NoError(t, api.DB.Create(&user.User{ID: 1, Username: "u1", PasswordHash: "x", CoinBalanceTenths: 230}).Error)
	now := time.Now()
	require.NoError(t, api.DB.Create(&video.Video{
		ID: 10, UserID: 1, Title: "v1", Status: video.StatusPublished, Zone: "动画",
		PlayCount: 5, CreatedAt: now,
	}).Error)
	require.NoError(t, api.DB.Create(&video.Video{
		ID: 11, UserID: 1, Title: "v2", Status: video.StatusPublished, Zone: "动画",
		CreatedAt: now,
	}).Error)
	require.NoError(t, api.DB.Create(&video.Video{ID: 12, UserID: 2, Title: "draft", Status: video.StatusDraft}).Error)

	// List published.
	w := doReq(r, "GET", "/api/v1/videos?page=1&page_size=10", token, "", nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	// Zone filter.
	w = doReq(r, "GET", "/api/v1/videos?zone=动画", token, "", nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	// Get video.
	w = doReq(r, "GET", "/api/v1/videos/10", token, "", nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	// Unpublished -> not found.
	w = doReq(r, "GET", "/api/v1/videos/12", token, "", nil)
	require.Equal(t, http.StatusNotFound, w.Code, w.Body.String())
	w = doReq(r, "GET", "/api/v1/videos/999", token, "", nil)
	require.Equal(t, http.StatusNotFound, w.Code, w.Body.String())

	// Space list.
	w = doReq(r, "GET", "/api/v1/space/1/videos", token, "", nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	// Stats and banners.
	w = doReq(r, "GET", "/api/v1/stats/home", token, "", nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	w = doReq(r, "GET", "/api/v1/home-banners", token, "", nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
}

func TestVideo_UpdateDeletePlayback(t *testing.T) {
	api, r, token := setupHandlerIntegrationDB(t)
	require.NoError(t, api.DB.Create(&user.User{ID: 1, Username: "u1", PasswordHash: "x", CoinBalanceTenths: 230}).Error)
	require.NoError(t, api.DB.Create(&video.Video{ID: 10, UserID: 1, Title: "v", Status: video.StatusPublished}).Error)
	require.NoError(t, api.DB.Create(&video.Video{ID: 11, UserID: 2, Title: "other", Status: video.StatusPublished}).Error)

	// Update my video.
	w := doJSON(r, "PUT", "/api/v1/videos/10", token, map[string]interface{}{
		"title": "renamed", "description": "d", "tags": []string{"a"}, "zone": "动画",
	})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var v video.Video
	require.NoError(t, api.DB.First(&v, 10).Error)
	require.Equal(t, "renamed", v.Title)

	// Update someone else's -> forbidden.
	w = doJSON(r, "PUT", "/api/v1/videos/11", token, map[string]interface{}{"title": "x"})
	require.Equal(t, http.StatusForbidden, w.Code, w.Body.String())

	// Playback patch.
	w = doJSON(r, "PATCH", "/api/v1/videos/10/playback", token, map[string]interface{}{"comments_closed": true})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NoError(t, api.DB.First(&v, 10).Error)
	require.True(t, v.CommentsClosed)
	// Empty body -> bad request.
	w = doJSON(r, "PATCH", "/api/v1/videos/10/playback", token, map[string]interface{}{})
	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())

	// Delete my video.
	w = doReq(r, "DELETE", "/api/v1/videos/10", token, "", nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.Error(t, api.DB.First(&video.Video{}, 10).Error)
	// Delete someone else's -> forbidden.
	w = doReq(r, "DELETE", "/api/v1/videos/11", token, "", nil)
	require.Equal(t, http.StatusForbidden, w.Code, w.Body.String())

	// My videos list.
	w = doReq(r, "GET", fmt.Sprintf("/api/v1/users/me/videos?page=1&page_size=10&status=%s", video.StatusPublished), token, "", nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
}
