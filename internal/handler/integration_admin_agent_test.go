//go:build integration

package handler

import (
	"cakecake/internal/model/agent"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAdminAgent_Profiles(t *testing.T) {
	_, r, jm := newTestAPI(t)
	access, _, _, err := jm.IssueAdminPair(1)
	require.NoError(t, err)

	// List empty.
	w := doReq(r, "GET", "/api/v1/admin/agent-profiles", access, "", nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	// Create.
	w = doJSON(r, "POST", "/api/v1/admin/agent-profiles", access, map[string]interface{}{
		"slug": "assistant", "display_name": "小助手", "system_prompt": "you are a helpful assistant",
		"welcome_messages": []string{"hello"}, "sort_order": 1, "enabled": true,
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
	profileID := cr.Data.ID

	// Invalid: short display name / short prompt.
	w = doJSON(r, "POST", "/api/v1/admin/agent-profiles", access, map[string]interface{}{
		"slug": "x", "display_name": "", "system_prompt": "short", "welcome_messages": []string{"hi"},
	})
	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())

	// Update.
	w = doJSON(r, "PUT", fmt.Sprintf("/api/v1/admin/agent-profiles/%d", profileID), access, map[string]interface{}{
		"display_name": "新名字", "system_prompt": "you are a helpful assistant v2",
		"welcome_messages": []string{"hi there"},
	})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var p agent.AgentProfile
	_ = p

	// Settings get/put.
	w = doReq(r, "GET", "/api/v1/admin/agent-settings", access, "", nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	w = doJSON(r, "PUT", "/api/v1/admin/agent-settings", access, map[string]interface{}{
		"display_name": "设置名", "system_prompt": "you are a helpful assistant v3",
		"welcome_message": "welcome", "assistant_enabled": true,
	})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	// Delete requires >1 active profile; create a second one first.
	w = doJSON(r, "POST", "/api/v1/admin/agent-profiles", access, map[string]interface{}{
		"slug": "assistant2", "display_name": "第二助手", "system_prompt": "you are another helpful assistant",
		"welcome_messages": []string{"hi"}, "enabled": true,
	})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	// Delete the first profile.
	w = doReq(r, "DELETE", fmt.Sprintf("/api/v1/admin/agent-profiles/%d", profileID), access, "", nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
}
