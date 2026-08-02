//go:build integration

package handler

import (
	"cakecake/internal/model/dynamic"
	"cakecake/internal/model/user"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUserDynamic_CRUD(t *testing.T) {
	api, r, token := setupHandlerIntegrationDB(t)
	require.NoError(t, api.DB.Create(&user.User{ID: 1, Username: "u1", PasswordHash: "x", CoinBalanceTenths: 230}).Error)

	// Create (multipart title/content).
	w := doMultipart(r, "POST", "/api/v1/users/me/dynamics", token, map[string]string{
		"title": "my dynamic", "content": "hello world",
	})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var cr struct {
		Code int `json:"code"`
		Data struct {
			ID uint64 `json:"id"`
		} `json:"data"`
	}
	decodeBody(t, w, &cr)
	require.NotZero(t, cr.Data.ID)
	dynID := cr.Data.ID

	// Empty dynamic -> bad request.
	w = doMultipart(r, "POST", "/api/v1/users/me/dynamics", token, map[string]string{})
	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())

	// List my dynamics.
	w = doReq(r, "GET", "/api/v1/users/me/dynamics", token, "", nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	// Get public dynamic.
	w = doReq(r, "GET", fmt.Sprintf("/api/v1/user-dynamics/%d", dynID), token, "", nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	// Update (multipart).
	w = doMultipart(r, "PUT", fmt.Sprintf("/api/v1/users/me/dynamics/%d", dynID), token, map[string]string{
		"title": "renamed",
	})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var d dynamic.UserDynamic
	require.NoError(t, api.DB.First(&d, dynID).Error)
	require.Equal(t, "renamed", d.Title)

	// Playback patch.
	w = doJSON(r, "PATCH", fmt.Sprintf("/api/v1/users/me/dynamics/%d/playback", dynID), token, map[string]interface{}{
		"comments_closed": false,
	})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	// Like toggle.
	w = doJSON(r, "POST", fmt.Sprintf("/api/v1/user-dynamics/%d/like", dynID), token, nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	// Space list.
	w = doReq(r, "GET", "/api/v1/space/1/dynamics", token, "", nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	// Delete.
	w = doReq(r, "DELETE", fmt.Sprintf("/api/v1/users/me/dynamics/%d", dynID), token, "", nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.Error(t, api.DB.First(&dynamic.UserDynamic{}, dynID).Error)

	// Missing dynamic.
	w = doReq(r, "GET", "/api/v1/user-dynamics/999", token, "", nil)
	require.Equal(t, http.StatusNotFound, w.Code, w.Body.String())
}
