//go:build integration

package handler

import (
	"cakecake/internal/model/article"
	"cakecake/internal/model/user"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestArticleFavorites_Lists(t *testing.T) {
	api, r, token := setupHandlerIntegrationDB(t)
	require.NoError(t, api.DB.Create(&user.User{ID: 1, Username: "u1", PasswordHash: "x", CoinBalanceTenths: 230}).Error)
	require.NoError(t, api.DB.Create(&article.Article{ID: 20, UserID: 2, Title: "a", BodyMD: "b", Status: article.StatusPublished}).Error)
	require.NoError(t, api.DB.Create(&article.ArticleFavorite{UserID: 1, ArticleID: 20}).Error)

	// My favorites.
	w := doReq(r, "GET", "/api/v1/users/me/article-favorites", token, "", nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.Contains(t, w.Body.String(), "a")

	// Public space favorites.
	w = doReq(r, "GET", "/api/v1/space/1/article-favorites", token, "", nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.Contains(t, w.Body.String(), "a")
}
