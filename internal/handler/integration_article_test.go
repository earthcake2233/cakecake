//go:build integration

package handler

import (
	"cakecake/internal/model/article"
	"cakecake/internal/model/user"
	"cakecake/internal/model/video"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestArticle_CRUD(t *testing.T) {
	api, r, token := setupHandlerIntegrationDB(t)
	require.NoError(t, api.DB.Create(&user.User{ID: 1, Username: "u1", PasswordHash: "x", CoinBalanceTenths: 230}).Error)

	// Create draft (not published).
	w := doJSON(r, "POST", "/api/v1/articles", token, map[string]interface{}{
		"title": "my article", "body_md": "hello world", "publish": false,
	})
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	var cr struct {
		Code int `json:"code"`
		Data struct {
			ID uint64 `json:"id"`
		} `json:"data"`
	}
	decodeBody(t, w, &cr)
	require.NotZero(t, cr.Data.ID)
	artID := cr.Data.ID

	// List my articles.
	w = doReq(r, "GET", "/api/v1/users/me/articles", token, "", nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	// Get my article.
	w = doReq(r, "GET", fmt.Sprintf("/api/v1/users/me/articles/%d", artID), token, "", nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	// Public GET of unpublished article -> not found.
	w = doReq(r, "GET", fmt.Sprintf("/api/v1/articles/%d", artID), token, "", nil)
	require.Equal(t, http.StatusNotFound, w.Code, w.Body.String())

	// Publish via PUT.
	w = doJSON(r, "PUT", fmt.Sprintf("/api/v1/users/me/articles/%d", artID), token, map[string]interface{}{
		"title": "published", "publish": true,
	})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var a article.Article
	require.NoError(t, api.DB.First(&a, artID).Error)
	require.Equal(t, article.StatusPublished, a.Status)

	// Public GET now works.
	w = doReq(r, "GET", fmt.Sprintf("/api/v1/articles/%d", artID), token, "", nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	// View count increments.
	w = doJSON(r, "POST", fmt.Sprintf("/api/v1/articles/%d/view", artID), token, nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NoError(t, api.DB.First(&a, artID).Error)
	require.Positive(t, a.ViewCount)

	// Another user's published article for favorite/coin (cannot coin self).
	require.NoError(t, api.DB.Create(&article.Article{ID: 70, UserID: 2, Title: "other", BodyMD: "b", Status: article.StatusPublished}).Error)

	// Favorite toggle.
	w = doJSON(r, "POST", "/api/v1/articles/70/favorite", token, nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	// Coin.
	w = doJSON(r, "POST", "/api/v1/articles/70/coin", token, map[string]interface{}{"amount": 1})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	// Coin self -> bad request.
	w = doJSON(r, "POST", fmt.Sprintf("/api/v1/articles/%d/coin", artID), token, map[string]interface{}{"amount": 1})
	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())

	// Playback patch.
	w = doJSON(r, "PATCH", fmt.Sprintf("/api/v1/users/me/articles/%d/playback", artID), token, map[string]interface{}{
		"comments_closed": false,
	})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	// Article favorites list.
	w = doReq(r, "GET", "/api/v1/users/me/article-favorites", token, "", nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	// Delete.
	w = doReq(r, "DELETE", fmt.Sprintf("/api/v1/users/me/articles/%d", artID), token, "", nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.Error(t, api.DB.First(&article.Article{}, artID).Error)
}

func TestArticle_Validation(t *testing.T) {
	api, r, token := setupHandlerIntegrationDB(t)
	require.NoError(t, api.DB.Create(&user.User{ID: 1, Username: "u1", PasswordHash: "x", CoinBalanceTenths: 230}).Error)

	// Empty draft (no title, no body) -> bad request.
	w := doJSON(r, "POST", "/api/v1/articles", token, map[string]interface{}{"body_md": "", "publish": false})
	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())

	// Someone else's article -> not found.
	require.NoError(t, api.DB.Create(&article.Article{ID: 50, UserID: 2, Title: "other", Status: article.StatusDraft}).Error)
	w = doJSON(r, "PUT", "/api/v1/users/me/articles/50", token, map[string]interface{}{"title": "x"})
	require.Equal(t, http.StatusNotFound, w.Code, w.Body.String())
	w = doReq(r, "DELETE", "/api/v1/users/me/articles/50", token, "", nil)
	require.Equal(t, http.StatusNotFound, w.Code, w.Body.String())

	// Unauthorized.
	w = doReq(r, "GET", "/api/v1/users/me/articles", "", "", nil)
	require.Equal(t, http.StatusUnauthorized, w.Code, w.Body.String())

	// Space list.
	require.NoError(t, api.DB.Create(&video.Video{ID: 10, UserID: 1, Title: "v", Status: video.StatusPublished}).Error)
	w = doReq(r, "GET", "/api/v1/space/1/articles", token, "", nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
}
