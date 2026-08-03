//go:build integration

package handler

import (
	"cakecake/internal/model/admin"
	"cakecake/internal/model/user"
	"cakecake/internal/model/video"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestAdminBanner_UpdateDelete(t *testing.T) {
	api, r, jm := newTestAPI(t)
	access, _, _, err := jm.IssueAdminPair(1)
	require.NoError(t, err)
	require.NoError(t, api.DB.Create(&admin.HomeBanner{
		ID: 10, Title: "b", ImageURL: "https://ex.com/1.jpg", Enabled: true, SortOrder: 1,
	}).Error)

	// Update.
	w := doJSON(r, "PUT", "/api/v1/admin/home-banners/10", access, map[string]interface{}{
		"title": "renamed", "link_type": "url", "link_target": "https://ex.com/x",
		"sort_order": 2, "enabled": true,
	})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var b admin.HomeBanner
	require.NoError(t, api.DB.First(&b, 10).Error)
	require.Equal(t, "renamed", b.Title)

	// Update missing banner.
	w = doJSON(r, "PUT", "/api/v1/admin/home-banners/999", access, map[string]interface{}{"title": "x"})
	require.Equal(t, http.StatusNotFound, w.Code, w.Body.String())

	// Bad id.
	w = doJSON(r, "PUT", "/api/v1/admin/home-banners/abc", access, map[string]interface{}{"title": "x"})
	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())

	// Delete.
	w = doReq(r, "DELETE", "/api/v1/admin/home-banners/10", access, "", nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.Error(t, api.DB.First(&admin.HomeBanner{}, 10).Error)
	// Delete missing.
	w = doReq(r, "DELETE", "/api/v1/admin/home-banners/10", access, "", nil)
	require.Equal(t, http.StatusNotFound, w.Code, w.Body.String())
}

func TestUserSpace_PublicEndpoints(t *testing.T) {
	api, r, token := setupHandlerIntegrationDB(t)
	require.NoError(t, api.DB.Create(&user.User{ID: 1, Username: "u1", PasswordHash: "x", CoinBalanceTenths: 230}).Error)
	now := time.Now()
	require.NoError(t, api.DB.Create(&video.Video{ID: 10, UserID: 1, Title: "v", Status: video.StatusPublished, CreatedAt: now}).Error)

	// GetUserPublic.
	w := doReq(r, "GET", "/api/v1/space/1", token, "", nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.Contains(t, w.Body.String(), "u1")
	w = doReq(r, "GET", "/api/v1/space/999", token, "", nil)
	require.Equal(t, http.StatusNotFound, w.Code, w.Body.String())

	// ListUserPublishedVideos.
	w = doReq(r, "GET", fmt.Sprintf("/api/v1/space/1/videos?page=1&page_size=10"), token, "", nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.Contains(t, w.Body.String(), "v")
	w = doReq(r, "GET", "/api/v1/space/999/videos", token, "", nil)
	require.Equal(t, http.StatusNotFound, w.Code, w.Body.String())
}
