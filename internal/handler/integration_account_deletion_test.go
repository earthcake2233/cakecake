//go:build integration

package handler

import (
	"cakecake/internal/model/user"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

func TestAccountDeletion_RequestRevoke(t *testing.T) {
	api, r, token := setupHandlerIntegrationDB(t)
	hash, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.MinCost)
	require.NoError(t, err)
	require.NoError(t, api.DB.Create(&user.User{
		ID: 1, Username: "deleter", PasswordHash: string(hash), CoinBalanceTenths: 230,
	}).Error)

	// Missing password -> bad request.
	w := doJSON(r, "POST", "/api/v1/users/me/deletion/request", token, map[string]interface{}{})
	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())

	// Wrong password -> forbidden.
	w = doJSON(r, "POST", "/api/v1/users/me/deletion/request", token, map[string]interface{}{"password": "nope"})
	require.Equal(t, http.StatusForbidden, w.Code, w.Body.String())

	// Correct password -> pending.
	w = doJSON(r, "POST", "/api/v1/users/me/deletion/request", token, map[string]interface{}{"password": "password123"})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var resp struct {
		Code int `json:"code"`
		Data struct {
			OK       bool `json:"ok"`
			Pending  bool `json:"pending"`
			CoolDays int  `json:"cooling_days"`
		} `json:"data"`
	}
	decodeBody(t, w, &resp)
	require.Equal(t, 0, resp.Code)
	require.True(t, resp.Data.OK)
	require.True(t, resp.Data.Pending)
	require.Positive(t, resp.Data.CoolDays)

	// Re-request while pending -> pending response.
	w = doJSON(r, "POST", "/api/v1/users/me/deletion/request", token, map[string]interface{}{"password": "password123"})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	decodeBody(t, w, &resp)
	require.True(t, resp.Data.Pending)

	// Revoke -> OK.
	w = doJSON(r, "POST", "/api/v1/users/me/deletion/revoke", token, nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	// Revoke again (nothing pending) -> bad request.
	w = doJSON(r, "POST", "/api/v1/users/me/deletion/revoke", token, nil)
	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())

	// Unauthorized.
	w = doJSON(r, "POST", "/api/v1/users/me/deletion/request", "", map[string]interface{}{"password": "password123"})
	require.Equal(t, http.StatusUnauthorized, w.Code, w.Body.String())
}
