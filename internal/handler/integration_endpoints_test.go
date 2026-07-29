//go:build integration

package handler

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"minibili/internal/model"
)

func TestIntegration_BlockUser(t *testing.T) {
	api, r, _ := newTestAPI(t)

	u1 := model.User{Username: "blocker", PasswordHash: "hash", Nickname: "Blocker", CoinBalanceTenths: 100}
	require.NoError(t, api.DB.Create(&u1).Error)
	u2 := model.User{Username: "blocked", PasswordHash: "hash", Nickname: "Blocked", CoinBalanceTenths: 100}
	require.NoError(t, api.DB.Create(&u2).Error)

	token1, _, _, _ := api.JWT.IssuePair(u1.ID)

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/users/%d/block", u2.ID), nil)
	req.Header.Set("Authorization", "Bearer "+token1)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var block model.UserBlock
	err := api.DB.Where("blocker_id = ? AND blocked_id = ?", u1.ID, u2.ID).First(&block).Error
	require.NoError(t, err)
}

func TestIntegration_BlockSelf(t *testing.T) {
	api, r, tm := newTestAPI(t)
	u := model.User{Username: "self", PasswordHash: "hash", Nickname: "Self", CoinBalanceTenths: 100}
	require.NoError(t, api.DB.Create(&u).Error)
	token, _, _, _ := tm.IssuePair(u.ID)

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/users/%d/block", u.ID), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
}

func TestIntegration_GetMeSpacePrivacy(t *testing.T) {
	api, r, tm := newTestAPI(t)
	u := model.User{Username: "priv", PasswordHash: "hash", Nickname: "Priv", CoinBalanceTenths: 100,
		PrivacyPublicFavorites: true, PrivacyPublicBirthday: true}
	require.NoError(t, api.DB.Create(&u).Error)
	token, _, _, _ := tm.IssuePair(u.ID)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/me/space-privacy", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
}

func TestIntegration_UpdateMeSpacePrivacy(t *testing.T) {
	api, r, tm := newTestAPI(t)
	u := model.User{Username: "priv2", PasswordHash: "hash", Nickname: "Priv2", CoinBalanceTenths: 100}
	require.NoError(t, api.DB.Create(&u).Error)
	token, _, _, _ := tm.IssuePair(u.ID)

	body := `{"public_favorites":true,"public_recent_coins":true,"public_following":false}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/users/me/space-privacy", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
}

func TestIntegration_BlockUser_BlockTwice(t *testing.T) {
	api, r, tm := newTestAPI(t)
	u1 := model.User{Username: "bt1", PasswordHash: "hash", Nickname: "BT1", CoinBalanceTenths: 100}
	u2 := model.User{Username: "bt2", PasswordHash: "hash", Nickname: "BT2", CoinBalanceTenths: 100}
	require.NoError(t, api.DB.Create(&u1).Error)
	require.NoError(t, api.DB.Create(&u2).Error)
	token, _, _, _ := tm.IssuePair(u1.ID)

	// First block
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/users/%d/block", u2.ID), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	// Second block (already blocked)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
}

func TestIntegration_BlockUser_BlockNonexistent(t *testing.T) {
	api, r, tm := newTestAPI(t)
	u := model.User{Username: "bn1", PasswordHash: "hash", Nickname: "BN1", CoinBalanceTenths: 100}
	require.NoError(t, api.DB.Create(&u).Error)
	token, _, _, _ := tm.IssuePair(u.ID)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/users/99999/block", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusNotFound, w.Code, w.Body.String())
}

func TestIntegration_SearchHistory(t *testing.T) {
	api, r, tm := newTestAPI(t)
	u := model.User{Username: "sh1", PasswordHash: "hash", Nickname: "SH1", CoinBalanceTenths: 100}
	require.NoError(t, api.DB.Create(&u).Error)
	token, _, _, _ := tm.IssuePair(u.ID)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/me/search-history", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
}

func TestIntegration_UserFollow_Toggle(t *testing.T) {
	api, r, tm := newTestAPI(t)
	u1 := model.User{Username: "f1", PasswordHash: "hash", Nickname: "F1", CoinBalanceTenths: 100}
	u2 := model.User{Username: "f2", PasswordHash: "hash", Nickname: "F2", CoinBalanceTenths: 100}
	require.NoError(t, api.DB.Create(&u1).Error)
	require.NoError(t, api.DB.Create(&u2).Error)
	token, _, _, _ := tm.IssuePair(u1.ID)

	// Follow user 2
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/users/%d/follow", u2.ID), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	// Unfollow
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
}

func TestIntegration_PostSearchHistory(t *testing.T) {
	api, r, tm := newTestAPI(t)
	u := model.User{Username: "psh", PasswordHash: "hash", Nickname: "PSH", CoinBalanceTenths: 100}
	require.NoError(t, api.DB.Create(&u).Error)
	token, _, _, _ := tm.IssuePair(u.ID)

	body := strings.NewReader(`{"keyword":"test search"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users/me/search-history", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
}
