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

func TestDynamicComment_Endpoints(t *testing.T) {
	api, r, token := setupHandlerIntegrationDB(t)
	require.NoError(t, api.DB.Create(&user.User{ID: 1, Username: "u1", PasswordHash: "x", CoinBalanceTenths: 230}).Error)
	require.NoError(t, api.DB.Create(&dynamic.UserDynamic{ID: 30, UserID: 1, Title: "d"}).Error)

	// Post dynamic comment.
	w := doJSON(r, "POST", "/api/v1/user-dynamics/30/comments", token, map[string]interface{}{
		"content": "nice dynamic", "parent_id": 0,
	})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var cr struct {
		Data struct {
			ID uint64 `json:"id"`
		} `json:"data"`
	}
	decodeBody(t, w, &cr)
	dcID := cr.Data.ID

	// List comments.
	w = doReq(r, "GET", "/api/v1/user-dynamics/30/comments", token, "", nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	// Like + dislike.
	w = doJSON(r, "POST", fmt.Sprintf("/api/v1/dynamic-comments/%d/like", dcID), token, nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	w = doJSON(r, "POST", fmt.Sprintf("/api/v1/dynamic-comments/%d/dislike", dcID), token, nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	// Approve + ignore.
	w = doJSON(r, "POST", fmt.Sprintf("/api/v1/dynamic-comments/%d/approve", dcID), token, nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	w = doJSON(r, "POST", fmt.Sprintf("/api/v1/dynamic-comments/%d/ignore-curated", dcID), token, nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	// Missing dynamic.
	w = doJSON(r, "POST", "/api/v1/user-dynamics/999/comments", token, map[string]interface{}{"content": "x"})
	require.Equal(t, http.StatusNotFound, w.Code, w.Body.String())

	// Delete.
	w = doReq(r, "DELETE", fmt.Sprintf("/api/v1/dynamic-comments/%d", dcID), token, "", nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
}
