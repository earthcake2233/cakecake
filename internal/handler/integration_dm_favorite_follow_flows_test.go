//go:build integration

package handler

import (
	"cakecake/internal/model/dynamic"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func Test_DMFlow(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "dmf1", "DMF1", 10)
	u2 := seedUser(t, api, "dmf2", "DMF2", 10)
	tk := tok(t, api, u.ID)

	// Create conversation
	w := srve(r, areq("POST", "/api/v1/dm/conversations", tk, fmt.Sprintf(`{"peer_id":%d}`, u2.ID)))
	type convResp struct {
		Code int `json:"code"`
		Data struct {
			ID uint64 `json:"id"`
		} `json:"data"`
	}
	var cr convResp
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &cr))
	require.Equal(t, 0, cr.Code, w.Body.String())
	require.NotZero(t, cr.Data.ID, w.Body.String())
	cid := cr.Data.ID

	// List conversations
	srveOK(t, r, areq("GET", "/api/v1/dm/conversations", tk, nil), http.StatusOK)

	// Send message
	srveOK(t, r, areq("POST", fmt.Sprintf("/api/v1/dm/conversations/%d/messages", cid), tk, `{"content":"hello"}`), http.StatusOK)

	// List messages
	srveOK(t, r, areq("GET", fmt.Sprintf("/api/v1/dm/conversations/%d/messages", cid), tk, nil), http.StatusOK)
}

// ==================== 2. favorite_folder.go ====================

func Test_FavoriteFolderCRUD(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "ffc1", "FFC1", 10)
	tk := tok(t, api, u.ID)

	// Create folder
	w := srve(r, areq("POST", "/api/v1/users/me/favorite-folders", tk, `{"title":"My Folder","description":"Test folder"}`))
	type ffResp struct {
		Code int `json:"code"`
		Data struct {
			ID uint64 `json:"id"`
		} `json:"data"`
	}
	var ffr ffResp
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &ffr))
	require.Equal(t, 0, ffr.Code, w.Body.String())
	require.NotZero(t, ffr.Data.ID, w.Body.String())
	fid := ffr.Data.ID

	// List folders
	srveOK(t, r, areq("GET", "/api/v1/users/me/favorite-folders", tk, nil), http.StatusOK)

	// Rename folder (UpdateFavoriteFolder)
	srveOK(t, r, areq("PUT", fmt.Sprintf("/api/v1/users/me/favorite-folders/%d", fid), tk, `{"title":"Renamed","description":"Updated desc"}`), http.StatusOK)

	// Delete folder
	srveOK(t, r, areq("DELETE", fmt.Sprintf("/api/v1/users/me/favorite-folders/%d", fid), tk, nil), http.StatusOK)
}

// ==================== 3. user_dynamic.go ====================

func Test_UserDynamicFlow(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "udf1", "UDF1", 10)
	tk := tok(t, api, u.ID)

	// Seed a dynamic directly (PostUserDynamic/PutMyUserDynamic are multipart-only)
	dyn := dynamic.UserDynamic{UserID: u.ID, Title: "My Dynamic", Content: "Dynamic content", ImagesJSON: "[]", CreatedAt: time.Now()}
	require.NoError(t, api.DB.Create(&dyn).Error)

	// Get dynamic
	srveOK(t, r, areq("GET", fmt.Sprintf("/api/v1/user-dynamics/%d", dyn.ID), "", nil), http.StatusOK)

	// Patch playback (comments_closed)
	srveOK(t, r, areq("PATCH", fmt.Sprintf("/api/v1/users/me/dynamics/%d/playback", dyn.ID), tk, `{"comments_closed":true}`), http.StatusOK)

	// List dynamics from space
	srveOK(t, r, areq("GET", fmt.Sprintf("/api/v1/space/%d/dynamics", u.ID), "", nil), http.StatusOK)

	// Delete dynamic
	srveOK(t, r, areq("DELETE", fmt.Sprintf("/api/v1/users/me/dynamics/%d", dyn.ID), tk, nil), http.StatusOK)
}

// ==================== 4. article.go ====================

func Test_ArticleCRUD(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "acr1", "ACR1", 10)
	tk := tok(t, api, u.ID)

	// Post article
	w := srve(r, areq("POST", "/api/v1/articles", tk, `{"title":"My Article","body_md":"# Hello","publish":true}`))
	type artResp struct {
		Code int `json:"code"`
		Data struct {
			ID uint64 `json:"id"`
		} `json:"data"`
	}
	var ar artResp
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &ar))
	require.Equal(t, 0, ar.Code, w.Body.String())
	require.NotZero(t, ar.Data.ID, w.Body.String())
	aid := ar.Data.ID

	// Get article (public)
	srveOK(t, r, areq("GET", fmt.Sprintf("/api/v1/articles/%d", aid), "", nil), http.StatusOK)

	// Put (update) article
	srveOK(t, r, areq("PUT", fmt.Sprintf("/api/v1/users/me/articles/%d", aid), tk, `{"title":"Updated Title","body_md":"# Updated"}`), http.StatusOK)

	// Patch playback
	srveOK(t, r, areq("PATCH", fmt.Sprintf("/api/v1/users/me/articles/%d/playback", aid), tk, `{"comments_closed":true}`), http.StatusOK)

	// Update article cover (nil body -> multipart parse error -> 400)
	srveOK(t, r, areq("PUT", fmt.Sprintf("/api/v1/users/me/articles/%d/cover", aid), tk, nil), http.StatusBadRequest)

	// Delete article
	srveOK(t, r, areq("DELETE", fmt.Sprintf("/api/v1/users/me/articles/%d", aid), tk, nil), http.StatusOK)
}

// ==================== 5. video.go ====================

func Test_VideoUpdateDelete(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "vud1", "VUD1", 10)
	tk := tok(t, api, u.ID)
	v := seedVideoWithAPI(t, api, u.ID, "Original Title")

	// Get video
	srveOK(t, r, areq("GET", fmt.Sprintf("/api/v1/videos/%d", v.ID), "", nil), http.StatusOK)

	// Update video
	srveOK(t, r, areq("PUT", fmt.Sprintf("/api/v1/videos/%d", v.ID), tk, `{"title":"Updated Title","description":"New desc"}`), http.StatusOK)

	// Patch playback
	srveOK(t, r, areq("PATCH", fmt.Sprintf("/api/v1/videos/%d/playback", v.ID), tk, `{"comments_closed":true,"danmaku_closed":false}`), http.StatusOK)

	// Update video cover (nil body -> multipart parse error -> 400)
	srveOK(t, r, areq("PUT", fmt.Sprintf("/api/v1/videos/%d/cover", v.ID), tk, nil), http.StatusBadRequest)

	// Delete video
	srveOK(t, r, areq("DELETE", fmt.Sprintf("/api/v1/videos/%d", v.ID), tk, nil), http.StatusOK)
}

// ==================== 6. view_history.go ====================

func Test_ViewHistoryFlow(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "vhf1", "VHF1", 10)
	v := seedVideoWithAPI(t, api, u.ID, "VH Video")
	tk := tok(t, api, u.ID)

	// Post view history
	srveOK(t, r, areq("POST", fmt.Sprintf("/api/v1/videos/%d/view-history", v.ID), tk, nil), http.StatusOK)

	// List view history
	srveOK(t, r, areq("GET", "/api/v1/users/me/view-history", tk, nil), http.StatusOK)

	// Get settings
	srveOK(t, r, areq("GET", "/api/v1/users/me/view-history/settings", tk, nil), http.StatusOK)

	// Update settings
	srveOK(t, r, areq("PUT", "/api/v1/users/me/view-history/settings", tk, `{"paused":true}`), http.StatusOK)

	// Delete entry
	srveOK(t, r, areq("DELETE", fmt.Sprintf("/api/v1/users/me/view-history/%d", v.ID), tk, nil), http.StatusOK)
}

// ==================== 7. user_me.go ====================

func Test_UserMeFlow(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "umf1", "UMF1", 50)
	tk := tok(t, api, u.ID)

	// GetMe
	srveOK(t, r, areq("GET", "/api/v1/users/me", tk, nil), http.StatusOK)

	// Update profile
	srveOK(t, r, areq("PUT", "/api/v1/users/me/profile", tk, `{"nickname":"NewNick","sign":"Hello world","gender":"male","birthday":"2000-01-01"}`), http.StatusOK)

	// Update announcement
	srveOK(t, r, areq("PUT", "/api/v1/users/me/announcement", tk, `{"announcement":"Welcome to my space!"}`), http.StatusOK)

	// Update username (use same username to test no-op path)
	srveOK(t, r, areq("PUT", "/api/v1/users/me", tk, fmt.Sprintf(`{"username":"%s"}`, u.Username)), http.StatusOK)

	// Update password with wrong old password -> 403
	srveOK(t, r, areq("PUT", "/api/v1/users/me/password", tk, `{"old_password":"wrong","new_password":"password"}`), http.StatusForbidden)
}

// ==================== 8. user_follow.go ====================

func Test_UserFollowFlow(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "uff1", "UFF1", 10)
	u2 := seedUser(t, api, "uff2", "UFF2", 10)
	tk := tok(t, api, u.ID)

	// Follow user (POST /api/v1/users/:userId/follow)
	srveOK(t, r, areq("POST", fmt.Sprintf("/api/v1/users/%d/follow", u2.ID), tk, nil), http.StatusOK)

	// List following
	srveOK(t, r, areq("GET", fmt.Sprintf("/api/v1/space/%d/following", u.ID), tk, nil), http.StatusOK)

	// List followers
	srveOK(t, r, areq("GET", fmt.Sprintf("/api/v1/space/%d/followers", u.ID), tk, nil), http.StatusOK)

	// Unfollow (POST same endpoint toggles)
	srveOK(t, r, areq("POST", fmt.Sprintf("/api/v1/users/%d/follow", u2.ID), tk, nil), http.StatusOK)
}

// ==================== 9. follow_group.go ====================

func Test_FollowGroupFlow(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "fgf1", "FGF1", 10)
	u2 := seedUser(t, api, "fgf2", "FGF2", 10)
	tk := tok(t, api, u.ID)

	// First follow user2 so we can add to group
	srveOK(t, r, areq("POST", fmt.Sprintf("/api/v1/users/%d/follow", u2.ID), tk, nil), http.StatusOK)

	// Create group
	w := srve(r, areq("POST", "/api/v1/users/me/follow-groups", tk, `{"name":"Test Group"}`))
	type fgResp struct {
		Code int `json:"code"`
		Data struct {
			ID uint64 `json:"id"`
		} `json:"data"`
	}
	var fgr fgResp
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &fgr))
	require.Equal(t, 0, fgr.Code, w.Body.String())
	require.NotZero(t, fgr.Data.ID, w.Body.String())
	gid := fgr.Data.ID

	// List groups
	srveOK(t, r, areq("GET", "/api/v1/users/me/follow-groups", tk, nil), http.StatusOK)

	// Add member
	srveOK(t, r, areq("POST", fmt.Sprintf("/api/v1/users/me/follow-groups/%d/members", gid), tk, fmt.Sprintf(`{"followee_id":%d}`, u2.ID)), http.StatusOK)

	// Remove member
	srveOK(t, r, areq("DELETE", fmt.Sprintf("/api/v1/users/me/follow-groups/%d/members/%d", gid, u2.ID), tk, nil), http.StatusOK)

	// Delete group
	srveOK(t, r, areq("DELETE", fmt.Sprintf("/api/v1/users/me/follow-groups/%d", gid), tk, nil), http.StatusOK)
}

// ==================== 10. video_engagement.go ====================

func Test_VideoEngagementFlow(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "vef1", "VEF1", 100)
	u2 := seedUser(t, api, "vef2", "VEF2", 100)
	v := seedVideoWithAPI(t, api, u2.ID, "Engagement Video")
	tk := tok(t, api, u.ID)

	// Create a favorite folder first
	w := srve(r, areq("POST", "/api/v1/users/me/favorite-folders", tk, `{"title":"Test Folder"}`))
	type ffResp struct {
		Code int `json:"code"`
		Data struct {
			ID uint64 `json:"id"`
		} `json:"data"`
	}
	var ffr ffResp
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &ffr))

	// Toggle video favorite
	srveOK(t, r, areq("POST", fmt.Sprintf("/api/v1/videos/%d/favorite", v.ID), tk, nil), http.StatusOK)

	if ffr.Code == 0 && ffr.Data.ID > 0 {
		fid := ffr.Data.ID
		// Add video to folder
		srveOK(t, r, areq("POST", fmt.Sprintf("/api/v1/videos/%d/favorite-folders/%d", v.ID, fid), tk, nil), http.StatusOK)

		// Remove video from folder
		srveOK(t, r, areq("DELETE", fmt.Sprintf("/api/v1/videos/%d/favorite-folders/%d", v.ID, fid), tk, nil), http.StatusOK)
	}

	// Post coin (need another user's video, can't coin self)
	srveOK(t, r, areq("POST", fmt.Sprintf("/api/v1/videos/%d/coin", v.ID), tk, `{"amount":1}`), http.StatusOK)

	// Toggle watch later
	srveOK(t, r, areq("POST", fmt.Sprintf("/api/v1/videos/%d/watch-later", v.ID), tk, nil), http.StatusOK)

	// List watch later
	srveOK(t, r, areq("GET", "/api/v1/users/me/watch-later", tk, nil), http.StatusOK)

	// Clear watch later
	srveOK(t, r, areq("DELETE", "/api/v1/users/me/watch-later", tk, nil), http.StatusOK)
}
