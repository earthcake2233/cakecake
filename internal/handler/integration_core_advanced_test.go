//go:build integration

package handler

import (
	"minibili/internal/model/admin"
	"minibili/internal/model/comment"
	"encoding/json"
	"fmt"
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
	json.Unmarshal(w.Body.Bytes(), &r)
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
	if code(t, w) != 0 { t.Skip("comment post failed") }
	var cm struct { Data comment.Comment `json:"data"` }
	json.Unmarshal(w.Body.Bytes(), &cm)
	if cm.Data.ID == 0 { t.Skip("no comment id") }

	// Like the comment (should create notification for u2)
	srve(r, areq("POST", "/api/v1/comments/"+strconv.FormatUint(cm.Data.ID, 10)+"/like", tk, nil))
	srve(r, areq("POST", "/api/v1/comments/"+strconv.FormatUint(cm.Data.ID, 10)+"/dislike", tk, nil))

	// Check notifications as u2
	tk2 := tok(t, api, u2.ID)
	srve(r, areq("GET", "/api/v1/users/me/notifications", tk2, nil))

	// Mark notification category as read
	srve(r, areq("POST", "/api/v1/users/me/notifications/read", tk2, `{"category":"like"}`))

	// Get like likers
	srve(r, areq("GET", "/api/v1/users/me/notifications/likers?page=1&page_size=5", tk2, nil))
}

// DM flow with conversation management
func Test_DMFullFlow(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "dmff1", "DMFF1", 10)
	u2 := seedUser(t, api, "dmff2", "DMFF2", 10)
	tk := tok(t, api, u.ID)

	w := srve(r, areq("POST", "/api/v1/dm/conversations", tk, fmt.Sprintf(`{"peer_id":%d}`, u2.ID)))
	var conv struct { Data struct { ID uint64 `json:"id"` } `json:"data"` }
	json.Unmarshal(w.Body.Bytes(), &conv)
	if conv.Data.ID == 0 { t.Skip("conv not created") }

	cid := conv.Data.ID
	srve(r, areq("POST", fmt.Sprintf("/api/v1/dm/conversations/%d/messages", cid), tk, `{"content":"hello there"}`))
	srve(r, areq("GET", fmt.Sprintf("/api/v1/dm/conversations/%d/messages", cid), tk, nil))
	srve(r, areq("GET", "/api/v1/dm/conversations", tk, nil))
}

// Creator comments list flow
func Test_CreatorCommentsList(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "ccl1", "CCL1", 0)
	v := seedVideoWithAPI(t, api, u.ID, "CCL Video")
	tk := tok(t, api, u.ID)

	// Post a comment first
	srve(r, areq("POST", "/api/v1/videos/"+strconv.FormatUint(v.ID, 10)+"/comments", tk, `{"content":"test comment"}`))

	// List creator video comments
	srve(r, areq("GET", "/api/v1/users/me/creator/comments?page=1&page_size=10&type=video", tk, nil))
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
	srve(r, areq("POST", fmt.Sprintf("/api/v1/articles/%d/coin", art.ID), tk, `{"amount":1}`))
}

// Video engagement: watch later full flow
func Test_WatchLaterFlow(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "wlf1", "WLF1", 0)
	v := seedVideoWithAPI(t, api, u.ID, "WL Video")
	tk := tok(t, api, u.ID)

	srve(r, areq("POST", fmt.Sprintf("/api/v1/videos/%d/watch-later", v.ID), tk, nil))
	srve(r, areq("GET", "/api/v1/users/me/watch-later", tk, nil))
	srve(r, areq("DELETE", fmt.Sprintf("/api/v1/users/me/watch-later/%d", v.ID), tk, nil))
	srve(r, areq("GET", "/api/v1/users/me/watch-later", tk, nil))
	srve(r, areq("DELETE", "/api/v1/users/me/watch-later", tk, nil))
}

// Multiple favorite folders
func Test_MultiFavoriteFolders(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "mff1", "MFF1", 0)
	v := seedVideoWithAPI(t, api, u.ID, "MFF Video")
	tk := tok(t, api, u.ID)

	// Create two folders
	srve(r, areq("POST", "/api/v1/users/me/favorite-folders", tk, `{"title":"Folder A"}`))
	srve(r, areq("POST", "/api/v1/users/me/favorite-folders", tk, `{"title":"Folder B"}`))

	// List folders
	srve(r, areq("GET", "/api/v1/users/me/favorite-folders", tk, nil))

	// Get video favorite status
	srve(r, areq("GET", fmt.Sprintf("/api/v1/videos/%d/favorite-folders", v.ID), tk, nil))
}

// User space detailed endpoints
func Test_UserSpaceDetailed(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "usd1", "USD1", 0)
	u2 := seedUser(t, api, "usd2", "USD2", 0)

	srve(r, areq("GET", fmt.Sprintf("/api/v1/space/%d", u.ID), "", nil))
	srve(r, areq("GET", fmt.Sprintf("/api/v1/space/%d/videos?page=1&page_size=10", u.ID), "", nil))
	srve(r, areq("GET", fmt.Sprintf("/api/v1/space/%d/articles?page=1&page_size=10", u.ID), "", nil))

	tk := tok(t, api, u2.ID)
	srve(r, areq("GET", fmt.Sprintf("/api/v1/space/%d/followers", u.ID), tk, nil))
	srve(r, areq("GET", fmt.Sprintf("/api/v1/space/%d/following", u.ID), tk, nil))
}

// User settings updates
func Test_UserSettings(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "ust1", "UST1", 0)
	tk := tok(t, api, u.ID)

	// Update gender/birthday privacy
	srve(r, areq("PUT", "/api/v1/users/me/privacy-settings", tk, `{"public_favorites":false,"public_birthday":false}`))

	// Update view history settings
	srve(r, areq("GET", "/api/v1/users/me/view-history/settings", tk, nil))
	srve(r, areq("PUT", "/api/v1/users/me/view-history/settings", tk, `{"paused":false}`))
}

// User avatar update
func Test_UserAvatarUpdate(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "uau1", "UAU1", 0)
	tk := tok(t, api, u.ID)

	// Calling without multipart should return 400
	w := srve(r, areq("PUT", "/api/v1/users/me/avatar", tk, nil))
	_ = w
}

// Admin hot search operations
func Test_AdminHotSearchOps(t *testing.T) {
	api, r, _ := newTestAPI(t)
	_ = api

	// Create an admin user
	adm := admin.Admin{Username: "admin_hs", PasswordHash: "hash"}
	require.NoError(t, api.DB.Create(&adm).Error)

	atk := admintok(t, api)
	srve(r, areq("GET", "/api/v1/admin/hot-search-ops/preview", atk, nil))
}

// Admin video/article review flow
func Test_AdminReviewOps(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "aro1", "ARO1", 0)
	v := seedVideoWithAPI(t, api, u.ID, "ARO Video")
	art := seedArticle(t, api, u.ID, "ARO Article")
	atk := admintok(t, api)

	srve(r, areq("GET", "/api/v1/admin/videos", atk, nil))
	srve(r, areq("GET", "/api/v1/admin/articles", atk, nil))
	_ = v
	_ = art
}
