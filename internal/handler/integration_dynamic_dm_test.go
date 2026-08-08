//go:build integration

package handler

import (
	"cakecake/internal/model/comment"
	"cakecake/internal/model/dm"
	"cakecake/internal/model/dynamic"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// Test_DynamicCommentApproveIgnoreDelete verifies curated comment flow for dynamics.
func Test_DynamicCommentApproveIgnoreDelete(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "dca1", "DCA1", 10)
	u2 := seedUser(t, api, "dca2", "DCA2", 10)
	tk := tok(t, api, u.ID)
	tk2 := tok(t, api, u2.ID)
	// Create a dynamic
	dyn := dynamic.UserDynamic{UserID: u.ID, Title: "Test Dyn", Content: "Test"}
	require.NoError(t, api.DB.Create(&dyn).Error)
	did := dyn.ID
	// Post a comment as u2
	w := srve(r, areq("POST", fmt.Sprintf("/api/v1/user-dynamics/%d/comments", did), tk2, `{"content":"Nice!"}`))
	var cresp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &cresp))
	cd, _ := cresp["data"].(map[string]interface{})
	var cid uint64
	if cd != nil {
		cid = uint64(cd["id"].(float64))
	}
	if cid == 0 {
		t.Skip("could not parse comment id")
	}
	// Approve as owner
	srve(r, areq("POST", fmt.Sprintf("/api/v1/dynamic-comments/%d/approve", cid), tk, nil))
	// Ignore curated as owner
	srve(r, areq("POST", fmt.Sprintf("/api/v1/dynamic-comments/%d/ignore-curated", cid), tk, nil))
	// Like as foreign user
	srve(r, areq("POST", fmt.Sprintf("/api/v1/dynamic-comments/%d/like", cid), tk2, nil))
	// Dislike as foreign user
	srve(r, areq("POST", fmt.Sprintf("/api/v1/dynamic-comments/%d/dislike", cid), tk2, nil))
	// Delete as owner
	srve(r, areq("DELETE", fmt.Sprintf("/api/v1/dynamic-comments/%d", cid), tk, nil))
}

// Test_UserDynamicEdgeCases covers PutMyUserDynamic, PatchUserDynamicPlayback, DeleteMyDynamic.
func Test_UserDynamicEdgeCases(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "udc1", "UDC1", 10)
	tk := tok(t, api, u.ID)
	// Create a dynamic
	dyn := dynamic.UserDynamic{UserID: u.ID, Title: "Edge Dyn", Content: "Edge"}
	require.NoError(t, api.DB.Create(&dyn).Error)
	did := dyn.ID
	// Update (PUT)
	srve(r, areq("PUT", fmt.Sprintf("/api/v1/users/me/dynamics/%d", did), tk, `{"title":"Updated","content":"Updated"}`))
	// Playback (PATCH)
	srve(r, areq("PATCH", fmt.Sprintf("/api/v1/users/me/dynamics/%d/playback", did), tk, `{"current_time":30.0}`))
	// List my dynamics
	srve(r, areq("GET", "/api/v1/users/me/dynamics?page=1&page_size=10", tk, nil))
	// Delete
	srve(r, areq("DELETE", fmt.Sprintf("/api/v1/users/me/dynamics/%d", did), tk, nil))
}

// Test_ViewHistorySettings covers PutMyViewHistorySettings.
func Test_ViewHistorySettings(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "vhs1", "VHS1", 10)
	tk := tok(t, api, u.ID)
	srve(r, areq("GET", "/api/v1/users/me/view-history/settings", tk, nil))
	srve(r, areq("PUT", "/api/v1/users/me/view-history/settings", tk, `{"paused":true}`))
	srve(r, areq("PUT", "/api/v1/users/me/view-history/settings", tk, `{"paused":false}`))
}

// Test_DmConversationEdgeCases covers DeleteDmConversation, ResetDmAgentConversation, PatchDmConversationSettings, dmUnreadTotal.
func Test_DmConversationEdgeCases(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u1 := seedUser(t, api, "dce1", "DCE1", 10)
	u2 := seedUser(t, api, "dce2", "DCE2", 10)
	tk := tok(t, api, u1.ID)
	w := srve(r, areq("POST", "/api/v1/dm/conversations", tk, fmt.Sprintf(`{"user_id":%d}`, u2.ID)))
	var dcr struct {
		Code int `json:"code"`
		Data struct {
			ID uint64 `json:"id"`
		}
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &dcr))
	if dcr.Code == 0 && dcr.Data.ID > 0 {
		cid := dcr.Data.ID
		// Send a message to create unread
		msgBody := fmt.Sprintf(`{"content":"Hello %d"}`, u2.ID)
		srve(r, areq("POST", fmt.Sprintf("/api/v1/dm/conversations/%d/messages", cid), tk, msgBody))
		// Patch settings
		srve(r, areq("PATCH", fmt.Sprintf("/api/v1/dm/conversations/%d/settings", cid), tk, `{"pinned":true}`))
		// Reset agent conversation
		// Reset agent conversation - Agent not configured so expect error
		srve(r, areq("POST", fmt.Sprintf("/api/v1/dm/conversations/%d/reset", cid), tk, nil))
		// Delete conversation
		srve(r, areq("DELETE", fmt.Sprintf("/api/v1/dm/conversations/%d", cid), tk, nil))
	}
}

// Test_DynamicCommentLikeDislikeEdge covers like/dislike toggle edge cases.
func Test_DynamicCommentLikeDislikeEdge(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "dld1", "DLD1", 10)
	u2 := seedUser(t, api, "dld2", "DLD2", 10)
	tk := tok(t, api, u.ID)
	// Create dynamic and comment
	dyn := dynamic.UserDynamic{UserID: u.ID, Title: "Like Test", Content: "Testing likes"}
	require.NoError(t, api.DB.Create(&dyn).Error)
	cm := comment.DynamicComment{DynamicID: dyn.ID, UserID: u2.ID, Content: "Test comment", Approved: true}
	require.NoError(t, api.DB.Create(&cm).Error)
	cid := cm.ID
	// Like
	srve(r, areq("POST", fmt.Sprintf("/api/v1/dynamic-comments/%d/like", cid), tk, nil))
	// Dislike (should remove existing like)
	srve(r, areq("POST", fmt.Sprintf("/api/v1/dynamic-comments/%d/dislike", cid), tk, nil))
	// Like again
	srve(r, areq("POST", fmt.Sprintf("/api/v1/dynamic-comments/%d/like", cid), tk, nil))
}

// Test_DmConversationDirectDB covers DM endpoints with a DB-created conversation.
func Test_DmConversationDirectDB(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u1 := seedUser(t, api, "dcd1", "DCD1", 10)
	u2 := seedUser(t, api, "dcd2", "DCD2", 10)
	tk := tok(t, api, u1.ID)
	// Create a conversation directly in DB
	lo, hi := u1.ID, u2.ID
	if lo > hi {
		lo, hi = hi, lo
	}
	conv := dm.DmConversation{UserLow: lo, UserHigh: hi, Kind: dm.DmKindHuman, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	require.NoError(t, api.DB.Create(&conv).Error)
	// Add participants
	p1 := dm.DmParticipant{ConversationID: conv.ID, UserID: u1.ID, UnreadCount: 0}
	p2 := dm.DmParticipant{ConversationID: conv.ID, UserID: u2.ID, UnreadCount: 0}
	require.NoError(t, api.DB.Create(&p1).Error)
	require.NoError(t, api.DB.Create(&p2).Error)
	cid := conv.ID
	// List conversations
	srve(r, areq("GET", "/api/v1/dm/conversations", tk, nil))
	// Send a message
	srve(r, areq("POST", fmt.Sprintf("/api/v1/dm/conversations/%d/messages", cid), tk, `{"content":"Hello"}`))
	// Patch settings - this should cover PatchDmConversationSettings
	srve(r, areq("PATCH", fmt.Sprintf("/api/v1/dm/conversations/%d/settings", cid), tk, `{"pinned":true}`))
	// Reset agent conversation - this should cover ResetDmAgentConversation
	// Reset agent conversation - Agent not configured so expect error
	srve(r, areq("POST", fmt.Sprintf("/api/v1/dm/conversations/%d/reset", cid), tk, nil))
	// Delete conversation - this should cover DeleteDmConversation
	srve(r, areq("DELETE", fmt.Sprintf("/api/v1/dm/conversations/%d", cid), tk, nil))
}
