package handler

import (
	"encoding/json"
	"github.com/stretchr/testify/require"
	"net/http"
	"testing"
)

func TestUserProfileAndMisc(t *testing.T) {
	api, r, _ := newTestAPI(t)
	tokenA, uidA := covRegister(t, r, "covua", "password12")
	_ = uidA
	covOK(t, covReq(t, r, "PUT", "/api/v1/users/me", tokenA, map[string]any{"username": "newcovname"}), http.StatusOK)
	covOK(t, covReq(t, r, "PUT", "/api/v1/users/me/profile", tokenA, map[string]any{"sign": "hello world", "nickname": "nick"}), http.StatusOK)
	covOK(t, covReq(t, r, "PUT", "/api/v1/users/me/announcement", tokenA, map[string]any{"announcement": "ann"}), http.StatusOK)
	covOK(t, covReq(t, r, "PUT", "/api/v1/users/me/password", tokenA, map[string]any{"old_password": "password12", "new_password": "password34"}), http.StatusOK)
	covOK(t, covReq(t, r, "GET", "/api/v1/users/me/space-privacy", tokenA, nil), http.StatusOK)
	covOK(t, covReq(t, r, "PUT", "/api/v1/users/me/space-privacy", tokenA, map[string]any{"public_favorites": true}), http.StatusOK)
	covOK(t, covReq(t, r, "GET", "/api/v1/users/me/daily-rewards", tokenA, nil), http.StatusOK)
	covOK(t, covReq(t, r, "POST", "/api/v1/users/me/daily-rewards/watch", tokenA, map[string]any{}), http.StatusOK)
	covOK(t, covReq(t, r, "GET", "/api/v1/users/me/coin-ledger", tokenA, nil), http.StatusOK)
	covOK(t, covReq(t, r, "GET", "/api/v1/notifications", tokenA, nil), http.StatusOK)
	covOK(t, covReq(t, r, "GET", "/api/v1/notifications/unread-summary", tokenA, nil), http.StatusCreated)
	covOK(t, covReq(t, r, "PATCH", "/api/v1/notifications/read-by-category?category=like", tokenA, nil), http.StatusOK)
	covOK(t, covReq(t, r, "POST", "/api/v1/users/me/deletion/request", tokenA, map[string]any{"password": "password34"}), http.StatusOK)
	covOK(t, covReq(t, r, "POST", "/api/v1/users/me/deletion/revoke", tokenA, map[string]any{}), http.StatusOK)
	covOK(t, covReq(t, r, "GET", "/api/v1/health", "", nil), http.StatusOK)
	_ = api
}

func TestAuthRefresh(t *testing.T) {
	_, r, _ := newTestAPI(t)
	covOK(t, covReq(t, r, "POST", "/api/v1/users", "", map[string]string{"username": "refreshuser", "password": "password12"}), http.StatusCreated)
	lw := covReq(t, r, "POST", "/api/v1/auth/login", "", map[string]string{"username": "refreshuser", "password": "password12"})
	covOK(t, lw, http.StatusOK)
	var out struct {
		Data struct {
			AccessToken  string `json:"access_token"`
			RefreshToken string `json:"refresh_token"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(lw.Body.Bytes(), &out))
	require.NotEmpty(t, out.Data.RefreshToken)
	covOK(t, covReq(t, r, "POST", "/api/v1/auth/refresh", "", map[string]any{"refresh_token": out.Data.RefreshToken}), http.StatusOK)
	covOK(t, covReq(t, r, "POST", "/api/v1/auth/refresh", "", map[string]any{"refresh_token": "invalid"}), http.StatusUnauthorized)
}
