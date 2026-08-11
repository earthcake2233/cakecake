//go:build integration

package handler

import (
	"cakecake/internal/model/admin"
	"cakecake/internal/model/comment"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

func code(t *testing.T, w *httptest.ResponseRecorder) int {
	t.Helper()
	var r struct {
		Code int `json:"code"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &r))
	return r.Code
}

// Notification like flow
func Test_NotifLikeFlow(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "nlf1", "NLF1", 0)
	u2 := seedUser(t, api, "nlf2", "NLF2", 0)
	v := seedVideoWithAPI(t, api, u2.ID, "NLF Video")
	tk := tok(t, api, u.ID)

	// Create a comment on u2 video
	body := `{"content":"nice video"}`
	w := srve(r, areq("POST", "/api/v1/videos/"+strconv.FormatUint(v.ID, 10)+"/comments", tk, body))
	require.Equal(t, 0, code(t, w), w.Body.String())
	var cm struct {
		Data comment.Comment `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &cm))
	require.NotZero(t, cm.Data.ID, w.Body.String())

	// Like the comment (should create notification for u2)
	srveOK(t, r, areq("POST", "/api/v1/comments/"+strconv.FormatUint(cm.Data.ID, 10)+"/like", tk, nil), http.StatusOK)
	srveOK(t, r, areq("POST", "/api/v1/comments/"+strconv.FormatUint(cm.Data.ID, 10)+"/dislike", tk, nil), http.StatusOK)

	// Check notifications as u2
	tk2 := tok(t, api, u2.ID)
	srveOK(t, r, areq("GET", "/api/v1/notifications", tk2, nil), http.StatusOK)

	// Mark notification category as read
	srveOK(t, r, areq("PATCH", "/api/v1/notifications/read-by-category?category=like", tk2, nil), http.StatusOK)

	// Get like likers
	srveOK(t, r, areq("GET", "/api/v1/notifications/99999/like-likers", tk2, nil), http.StatusNotFound)
}

// DM flow with conversation management
func Test_DMFullFlow(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "dmff1", "DMFF1", 10)
	u2 := seedUser(t, api, "dmff2", "DMFF2", 10)
	tk := tok(t, api, u.ID)

	w := srve(r, areq("POST", "/api/v1/dm/conversations", tk, fmt.Sprintf(`{"peer_id":%d}`, u2.ID)))
	var conv struct {
		Data struct {
			ID uint64 `json:"id"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &conv))
	require.NotZero(t, conv.Data.ID, w.Body.String())

	cid := conv.Data.ID
	srveOK(t, r, areq("POST", fmt.Sprintf("/api/v1/dm/conversations/%d/messages", cid), tk, `{"content":"hello there"}`), http.StatusOK)
	srveOK(t, r, areq("GET", fmt.Sprintf("/api/v1/dm/conversations/%d/messages", cid), tk, nil), http.StatusOK)
	srveOK(t, r, areq("GET", "/api/v1/dm/conversations", tk, nil), http.StatusOK)
}

// Creator comments list flow
func Test_CreatorCommentsList(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "ccl1", "CCL1", 0)
	v := seedVideoWithAPI(t, api, u.ID, "CCL Video")
	tk := tok(t, api, u.ID)

	// Post a comment first
	srveOK(t, r, areq("POST", "/api/v1/videos/"+strconv.FormatUint(v.ID, 10)+"/comments", tk, `{"content":"test comment"}`), http.StatusCreated)

	// List creator video comments
	srveOK(t, r, areq("GET", "/api/v1/users/me/creator/comments?page=1&page_size=10&type=video", tk, nil), http.StatusOK)
}

// Article engagement flow
func Test_ArticleEngagement(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "aef1", "AEF1", 100)
	u2 := seedUser(t, api, "aef2", "AEF2", 100)
	art := seedArticle(t, api, u2.ID, "AE Article")
	tk := tok(t, api, u.ID)

	// Toggle favorite article
	w := srve(r, areq("POST", fmt.Sprintf("/api/v1/articles/%d/favorite", art.ID), tk, nil))
	_ = code(t, w)

	// Coin article
	srveOK(t, r, areq("POST", fmt.Sprintf("/api/v1/articles/%d/coin", art.ID), tk, `{"amount":1}`), http.StatusOK)
}

// Video engagement: watch later full flow
func Test_WatchLaterFlow(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "wlf1", "WLF1", 0)
	v := seedVideoWithAPI(t, api, u.ID, "WL Video")
	tk := tok(t, api, u.ID)

	srveOK(t, r, areq("POST", fmt.Sprintf("/api/v1/videos/%d/watch-later", v.ID), tk, nil), http.StatusOK)
	srveOK(t, r, areq("GET", "/api/v1/users/me/watch-later", tk, nil), http.StatusOK)
	srveOK(t, r, areq("POST", fmt.Sprintf("/api/v1/videos/%d/watch-later", v.ID), tk, nil), http.StatusOK)
	srveOK(t, r, areq("GET", "/api/v1/users/me/watch-later", tk, nil), http.StatusOK)
	srveOK(t, r, areq("DELETE", "/api/v1/users/me/watch-later/watched", tk, nil), http.StatusOK)
	srveOK(t, r, areq("DELETE", "/api/v1/users/me/watch-later", tk, nil), http.StatusOK)
}

// Multiple favorite folders
func Test_MultiFavoriteFolders(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "mff1", "MFF1", 0)
	v := seedVideoWithAPI(t, api, u.ID, "MFF Video")
	tk := tok(t, api, u.ID)

	// Create two folders
	srveOK(t, r, areq("POST", "/api/v1/users/me/favorite-folders", tk, `{"title":"Folder A"}`), http.StatusOK)
	srveOK(t, r, areq("POST", "/api/v1/users/me/favorite-folders", tk, `{"title":"Folder B"}`), http.StatusOK)

	// List folders
	srveOK(t, r, areq("GET", "/api/v1/users/me/favorite-folders", tk, nil), http.StatusOK)

	// Get video favorite status
	srveOK(t, r, areq("GET", fmt.Sprintf("/api/v1/videos/%d/favorite-picker", v.ID), tk, nil), http.StatusOK)
}

// User space detailed endpoints
func Test_UserSpaceDetailed(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "usd1", "USD1", 0)

	srveOK(t, r, areq("GET", fmt.Sprintf("/api/v1/space/%d", u.ID), "", nil), http.StatusOK)
	srveOK(t, r, areq("GET", fmt.Sprintf("/api/v1/space/%d/videos?page=1&page_size=10", u.ID), "", nil), http.StatusOK)
	srveOK(t, r, areq("GET", fmt.Sprintf("/api/v1/space/%d/articles?page=1&page_size=10", u.ID), "", nil), http.StatusOK)

	tk := tok(t, api, u.ID)
	srveOK(t, r, areq("GET", fmt.Sprintf("/api/v1/space/%d/followers", u.ID), tk, nil), http.StatusOK)
	srveOK(t, r, areq("GET", fmt.Sprintf("/api/v1/space/%d/following", u.ID), tk, nil), http.StatusOK)
}

// User settings updates
func Test_UserSettings(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "ust1", "UST1", 0)
	tk := tok(t, api, u.ID)

	// Update gender/birthday privacy
	srveOK(t, r, areq("PUT", "/api/v1/users/me/space-privacy", tk, `{"public_favorites":false,"public_birthday":false}`), http.StatusOK)

	// Update view history settings
	srveOK(t, r, areq("GET", "/api/v1/users/me/view-history/settings", tk, nil), http.StatusOK)
	srveOK(t, r, areq("PUT", "/api/v1/users/me/view-history/settings", tk, `{"paused":false}`), http.StatusOK)
}

// User avatar update
func Test_UserAvatarUpdate(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "uau1", "UAU1", 0)
	tk := tok(t, api, u.ID)

	// Calling without multipart should return 400
	srveOK(t, r, areq("POST", "/api/v1/users/me/avatar", tk, nil), http.StatusBadRequest)
}

// Admin hot search operations
func Test_AdminHotSearchOps(t *testing.T) {
	api, r, _ := newTestAPI(t)
	// Create an admin user
	adm := admin.Admin{Username: "admin_hs", PasswordHash: "hash"}
	require.NoError(t, api.DB.Create(&adm).Error)

	atk := admintok(t, api)
	srveOK(t, r, areq("GET", "/api/v1/admin/hot-search/preview", atk, nil), http.StatusOK)
}

// Admin video/article review flow
func Test_AdminReviewOps(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "aro1", "ARO1", 0)
	v := seedVideoWithAPI(t, api, u.ID, "ARO Video")
	art := seedArticle(t, api, u.ID, "ARO Article")
	atk := admintok(t, api)

	srveOK(t, r, areq("GET", "/api/v1/admin/videos", atk, nil), http.StatusOK)
	srveOK(t, r, areq("GET", "/api/v1/admin/articles", atk, nil), http.StatusOK)
	_ = v
	_ = art
}
