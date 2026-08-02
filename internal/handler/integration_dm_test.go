//go:build integration

package handler

import (
	"cakecake/internal/model/dm"
	"cakecake/internal/model/user"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDm_ConversationsAndMessages(t *testing.T) {
	api, r, token := setupHandlerIntegrationDB(t)
	require.NoError(t, api.DB.Create(&user.User{ID: 1, Username: "u1", PasswordHash: "x", CoinBalanceTenths: 230}).Error)
	require.NoError(t, api.DB.Create(&user.User{ID: 2, Username: "u2", PasswordHash: "x", CoinBalanceTenths: 230}).Error)

	// Create conversation with peer 2.
	w := doJSON(r, "POST", "/api/v1/dm/conversations", token, map[string]interface{}{"peer_id": 2})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var cr struct {
		Data struct {
			ID uint64 `json:"id"`
		} `json:"data"`
	}
	decodeBody(t, w, &cr)
	convID := cr.Data.ID

	// Self-conversation -> bad request.
	w = doJSON(r, "POST", "/api/v1/dm/conversations", token, map[string]interface{}{"peer_id": 1})
	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
	// Missing peer -> bad request.
	w = doJSON(r, "POST", "/api/v1/dm/conversations", token, map[string]interface{}{})
	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
	// Nonexistent peer -> not found.
	w = doJSON(r, "POST", "/api/v1/dm/conversations", token, map[string]interface{}{"peer_id": 999})
	require.Equal(t, http.StatusNotFound, w.Code, w.Body.String())

	// List conversations.
	w = doReq(r, "GET", "/api/v1/dm/conversations", token, "", nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	// Post message.
	w = doJSON(r, "POST", fmt.Sprintf("/api/v1/dm/conversations/%d/messages", convID), token, map[string]interface{}{
		"content": "hello", "peer_id": 2,
	})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	// List messages.
	w = doReq(r, "GET", fmt.Sprintf("/api/v1/dm/conversations/%d/messages", convID), token, "", nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	// Patch settings.
	w = doJSON(r, "PATCH", fmt.Sprintf("/api/v1/dm/conversations/%d/settings", convID), token, map[string]interface{}{"pinned": true})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var part dm.DmParticipant
	require.NoError(t, api.DB.Where("conversation_id = ? AND user_id = ?", convID, 1).First(&part).Error)
	require.True(t, part.Pinned)

	// Reset agent conversation.
	w = doJSON(r, "POST", fmt.Sprintf("/api/v1/dm/conversations/%d/reset", convID), token, nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	// Delete conversation.
	w = doReq(r, "DELETE", fmt.Sprintf("/api/v1/dm/conversations/%d", convID), token, "", nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
}
