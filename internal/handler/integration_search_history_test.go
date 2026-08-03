//go:build integration

package handler

import (
	"cakecake/internal/model/user"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSearchHistory_Endpoints(t *testing.T) {
	api, r, token := setupHandlerIntegrationDB(t)
	require.NoError(t, api.DB.Create(&user.User{ID: 1, Username: "u1", PasswordHash: "x", CoinBalanceTenths: 230}).Error)

	// PUT replaces history.
	w := doJSON(r, "PUT", "/api/v1/users/me/search-history", token, map[string]interface{}{
		"keywords": []string{"golang", "rust"},
	})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	// GET lists it.
	w = doReq(r, "GET", "/api/v1/users/me/search-history", token, "", nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.Contains(t, w.Body.String(), "golang")
	require.Contains(t, w.Body.String(), "rust")

	// POST upserts a keyword.
	w = doJSON(r, "POST", "/api/v1/users/me/search-history", token, map[string]interface{}{"keyword": "redis"})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	w = doReq(r, "GET", "/api/v1/users/me/search-history", token, "", nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.Contains(t, w.Body.String(), "redis")

	// Unauthorized.
	w = doReq(r, "GET", "/api/v1/users/me/search-history", "", "", nil)
	require.Equal(t, http.StatusUnauthorized, w.Code, w.Body.String())
}
