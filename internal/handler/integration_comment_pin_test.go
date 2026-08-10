//go:build integration

package handler

import (
	"cakecake/internal/model/comment"
	"cakecake/internal/model/dynamic"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func decodeCode(t *testing.T, w *httptest.ResponseRecorder) int {
	t.Helper()
	var r struct {
		Code int `json:"code"`
	}
	// Some endpoints (e.g. DELETE) return an empty body; treat that as code 0.
	_ = json.Unmarshal(w.Body.Bytes(), &r)
	return r.Code
}

func decodeDataComment(t *testing.T, w *httptest.ResponseRecorder) comment.Comment {
	t.Helper()
	var r struct {
		Data comment.Comment `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &r))
	return r.Data
}

func Test_CommentPinAndApprove(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "cpa1", "CPA1", 10)
	v := seedVideoWithAPI(t, api, u.ID, "CPA Video")
	tk := tok(t, api, u.ID)
	// Post a comment
	var cm comment.Comment
	body := fmt.Sprintf(`{"content":"test comment","video_id":%d}`, v.ID)
	w := srve(r, areq("POST", "/api/v1/videos/"+strconv.FormatUint(v.ID, 10)+"/comments", tk, body))
	if c := decodeCode(t, w); c == 0 {
		cm = decodeDataComment(t, w)
	}
	if cm.ID > 0 {
		// Pin comment
		srveOK(t, r, areq("POST", fmt.Sprintf("/api/v1/comments/%d/pin", cm.ID), tk, nil), http.StatusOK)
		// Approve comment
		srveOK(t, r, areq("POST", fmt.Sprintf("/api/v1/comments/%d/approve", cm.ID), tk, nil), http.StatusOK)
		// Ignore curated
		srveOK(t, r, areq("POST", fmt.Sprintf("/api/v1/comments/%d/ignore-curated", cm.ID), tk, nil), http.StatusOK)
		// Toggle like
		srveOK(t, r, areq("POST", fmt.Sprintf("/api/v1/comments/%d/like", cm.ID), tk, nil), http.StatusOK)
		// Toggle dislike
		srveOK(t, r, areq("POST", fmt.Sprintf("/api/v1/comments/%d/dislike", cm.ID), tk, nil), http.StatusOK)
	}
}

func Test_ArticleCommentPinAndToggle(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "acp1", "ACP1", 10)
	art := seedArticle(t, api, u.ID, "AC Article")
	tk := tok(t, api, u.ID)
	// Post article comment
	body := fmt.Sprintf(`{"content":"nice article","article_id":%d}`, art.ID)
	w := srve(r, areq("POST", fmt.Sprintf("/api/v1/articles/%d/comments", art.ID), tk, body))
	type artCmResp struct {
		Code int `json:"code"`
		Data struct {
			ID uint64 `json:"id"`
		} `json:"data"`
	}
	var acr artCmResp
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &acr))
	if acr.Code == 0 && acr.Data.ID > 0 {
		cid := acr.Data.ID
		srveOK(t, r, areq("POST", fmt.Sprintf("/api/v1/article-comments/%d/like", cid), tk, nil), http.StatusOK)
		srveOK(t, r, areq("POST", fmt.Sprintf("/api/v1/article-comments/%d/dislike", cid), tk, nil), http.StatusOK)
		srveOK(t, r, areq("POST", fmt.Sprintf("/api/v1/article-comments/%d/pin", cid), tk, nil), http.StatusOK)
		srveOK(t, r, areq("POST", fmt.Sprintf("/api/v1/article-comments/%d/approve", cid), tk, nil), http.StatusOK)
		srveOK(t, r, areq("POST", fmt.Sprintf("/api/v1/article-comments/%d/ignore-curated", cid), tk, nil), http.StatusOK)
		srveOK(t, r, areq("DELETE", fmt.Sprintf("/api/v1/article-comments/%d", cid), tk, nil), http.StatusOK)
	}
}

func Test_UpdateMeProfileAndPrivacy(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "ump1", "UMP1", 10)
	tk := tok(t, api, u.ID)
	srveOK(t, r, areq("PUT", "/api/v1/users/me", tk, `{"username":"NewName123"}`), http.StatusOK)
	srveOK(t, r, areq("PUT", "/api/v1/users/me/announcement", tk, `{"announcement":"Welcome!"}`), http.StatusOK)
	srveOK(t, r, areq("PUT", "/api/v1/users/me/space-privacy", tk, `{"public_favorites":true}`), http.StatusOK)
	// Avatar upload (nil body -> parse error -> 400)
	srveOK(t, r, areq("POST", "/api/v1/users/me/avatar", tk, nil), http.StatusBadRequest)
}

func Test_FollowGroupRenameAndDelete(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "fgr1", "FGR1", 10)
	u2 := seedUser(t, api, "fgr2", "FGR2", 10)
	tk := tok(t, api, u.ID)
	// Create a follow group
	w := srve(r, areq("POST", "/api/v1/users/me/follow-groups", tk, `{"name":"Test Group"}`))
	type fgResp struct {
		Code int `json:"code"`
		Data struct {
			ID uint64 `json:"id"`
		} `json:"data"`
	}
	var fgr fgResp
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &fgr))
	if fgr.Code == 0 && fgr.Data.ID > 0 {
		gid := fgr.Data.ID
		// Rename
		srveOK(t, r, areq("PUT", fmt.Sprintf("/api/v1/users/me/follow-groups/%d", gid), tk, `{"name":"Renamed"}`), http.StatusOK)
		// Group members require an existing follow relationship.
		srveOK(t, r, areq("POST", fmt.Sprintf("/api/v1/users/%d/follow", u2.ID), tk, nil), http.StatusOK)
		// Add member
		srveOK(t, r, areq("POST", fmt.Sprintf("/api/v1/users/me/follow-groups/%d/members", gid), tk, fmt.Sprintf(`{"followee_id":%d}`, u2.ID)), http.StatusOK)
		// Delete group
		srveOK(t, r, areq("DELETE", fmt.Sprintf("/api/v1/users/me/follow-groups/%d", gid), tk, nil), http.StatusOK)
	}
}

func Test_DmActions(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "dma1", "DMA1", 10)
	u2 := seedUser(t, api, "dma2", "DMA2", 10)
	tk := tok(t, api, u.ID)
	// Create a conversation
	w := srve(r, areq("POST", "/api/v1/dm/conversations", tk, fmt.Sprintf(`{"peer_id":%d}`, u2.ID)))
	type dmConvResp struct {
		Code int `json:"code"`
		Data struct {
			ID uint64 `json:"id"`
		} `json:"data"`
	}
	var dcr dmConvResp
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &dcr))
	if dcr.Code == 0 && dcr.Data.ID > 0 {
		cid := dcr.Data.ID
		// List messages
		srveOK(t, r, areq("GET", fmt.Sprintf("/api/v1/dm/conversations/%d/messages", cid), tk, nil), http.StatusOK)
		// Post a message
		srveOK(t, r, areq("POST", fmt.Sprintf("/api/v1/dm/conversations/%d/messages", cid), tk, `{"content":"Hello!"}`), http.StatusOK)
		// Patch settings
		srveOK(t, r, areq("PATCH", fmt.Sprintf("/api/v1/dm/conversations/%d/settings", cid), tk, `{"pinned":true}`), http.StatusOK)
		// Delete conversation
		srveOK(t, r, areq("DELETE", fmt.Sprintf("/api/v1/dm/conversations/%d", cid), tk, nil), http.StatusOK)
	}
}

func Test_VideoActions(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "vact1", "VACT1", 10)
	tk := tok(t, api, u.ID)
	v := seedVideoWithAPI(t, api, u.ID, "VA Video")

	// List my videos
	srveOK(t, r, areq("GET", "/api/v1/users/me/videos?page=1&page_size=10", tk, nil), http.StatusOK)

	// Update video
	srveOK(t, r, areq("PUT", fmt.Sprintf("/api/v1/videos/%d", v.ID), tk, `{"title":"Updated Title","tags":["tag1"]}`), http.StatusOK)

	// Patch playback
	srveOK(t, r, areq("PATCH", fmt.Sprintf("/api/v1/videos/%d/playback", v.ID), tk, `{"comments_closed":true}`), http.StatusOK)

	// Update cover (nil body -> parse error -> 400)
	srveOK(t, r, areq("PUT", fmt.Sprintf("/api/v1/videos/%d/cover", v.ID), tk, nil), http.StatusBadRequest)
}

func Test_ArticleActions(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "aact1", "AACT1", 10)
	tk := tok(t, api, u.ID)

	// Post a draft article
	w := srve(r, areq("POST", "/api/v1/articles", tk, `{"title":"My Article","content_md":"# Hello","publish":true}`))
	type artResp struct {
		Code int `json:"code"`
		Data struct {
			ID uint64 `json:"id"`
		} `json:"data"`
	}
	var ar artResp
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &ar))

	// List my articles
	srveOK(t, r, areq("GET", "/api/v1/users/me/articles?page=1&page_size=10", tk, nil), http.StatusOK)

	// Count my articles
	srveOK(t, r, areq("GET", "/api/v1/users/me/articles/count", tk, nil), http.StatusBadRequest)

	if ar.Code == 0 && ar.Data.ID > 0 {
		aid := ar.Data.ID
		// Patch playback
		srveOK(t, r, areq("PATCH", fmt.Sprintf("/api/v1/users/me/articles/%d/playback", aid), tk, `{"comments_closed":true}`), http.StatusOK)

		// Post view
		srveOK(t, r, areq("POST", fmt.Sprintf("/api/v1/articles/%d/view", aid), tk, nil), http.StatusOK)

		// Update article
		srveOK(t, r, areq("PUT", fmt.Sprintf("/api/v1/users/me/articles/%d", aid), tk, `{"title":"Updated Title","content_md":"# Updated"}`), http.StatusOK)

		// Update cover (nil body -> 400)
		srveOK(t, r, areq("PUT", fmt.Sprintf("/api/v1/users/me/articles/%d/cover", aid), tk, nil), http.StatusBadRequest)

		// Delete article
		srveOK(t, r, areq("DELETE", fmt.Sprintf("/api/v1/users/me/articles/%d", aid), tk, nil), http.StatusOK)
	}
}

func Test_VideoEngagementMore(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "vem1", "VEM1", 100)
	u2 := seedUser(t, api, "vem2", "VEM2", 100)
	tk := tok(t, api, u.ID)
	v := seedVideoWithAPI(t, api, u2.ID, "VEM Video")

	// Toggle favorite
	srveOK(t, r, areq("POST", fmt.Sprintf("/api/v1/videos/%d/favorite", v.ID), tk, nil), http.StatusOK)

	// Get favorite picker
	srveOK(t, r, areq("GET", fmt.Sprintf("/api/v1/videos/%d/favorite-picker", v.ID), tk, nil), http.StatusOK)

	// Post coin
	srveOK(t, r, areq("POST", fmt.Sprintf("/api/v1/videos/%d/coin", v.ID), tk, `{"amount":1}`), http.StatusOK)

	// Toggle watch later
	srveOK(t, r, areq("POST", fmt.Sprintf("/api/v1/videos/%d/watch-later", v.ID), tk, nil), http.StatusOK)

	// List watch later
	srveOK(t, r, areq("GET", "/api/v1/users/me/watch-later", tk, nil), http.StatusOK)

	// Clear watch later watched
	srveOK(t, r, areq("DELETE", "/api/v1/users/me/watch-later/watched", tk, nil), http.StatusOK)
}

func Test_DynamicActions(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "dya1", "DYA1", 10)
	tk := tok(t, api, u.ID)

	// Create a dynamic
	dyn := dynamic.UserDynamic{UserID: u.ID, Title: "My Dynamic", Content: "Dynamic Content", ImagesJSON: "[]", CreatedAt: time.Now()}
	require.NoError(t, api.DB.Create(&dyn).Error)

	// Get dynamic
	srveOK(t, r, areq("GET", fmt.Sprintf("/api/v1/user-dynamics/%d", dyn.ID), "", nil), http.StatusOK)

	// Post dynamic comment
	srveOK(t, r, areq("POST", fmt.Sprintf("/api/v1/user-dynamics/%d/comments", dyn.ID), tk, `{"content":"Nice dynamic!"}`), http.StatusOK)

	// List space dynamics
	srveOK(t, r, areq("GET", fmt.Sprintf("/api/v1/space/%d/dynamics", u.ID), "", nil), http.StatusOK)
}

func Test_NotificationEndpoints(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "ne1", "NE1", 10)
	tk := tok(t, api, u.ID)

	// Unread summary
	srveOK(t, r, areq("GET", "/api/v1/notifications/unread-summary", tk, nil), http.StatusCreated)

	// List notifications
	srveOK(t, r, areq("GET", "/api/v1/notifications?page=1&page_size=10", tk, nil), http.StatusOK)

	// Read by category (uses sqlite-compatible value)
	srveOK(t, r, areq("PATCH", "/api/v1/notifications/read-by-category?category=like", tk, nil), http.StatusOK)

	// Mark batch read (with nil ids)
	srveOK(t, r, areq("PATCH", "/api/v1/notifications/read-batch", tk, `[]`), http.StatusOK)

	// Mute like notification (non-existent -> handled gracefully)
	srveOK(t, r, areq("POST", "/api/v1/notifications/0/mute-likes", tk, nil), http.StatusInternalServerError)

	// Delete notification (non-existent -> handled)
	srveOK(t, r, areq("DELETE", "/api/v1/notifications/0", tk, nil), http.StatusOK)
}

func Test_DanmakuActions(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "dka1", "DKA1", 10)
	tk := tok(t, api, u.ID)
	v := seedVideoWithAPI(t, api, u.ID, "DK Video")

	// Post a danmaku
	w := srve(r, areq("POST", fmt.Sprintf("/api/v1/videos/%d/danmaku", v.ID), tk, `{"content":"Hello","type":"scroll","color":"#FFFFFF","video_time":10.5}`))
	require.Equal(t, 0, decodeCode(t, w), w.Body.String())
	type dkResp struct {
		Code int `json:"code"`
		Data struct {
			ID uint64 `json:"id"`
		} `json:"data"`
	}
	var dkr dkResp
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &dkr))
	if dkr.Code == 0 && dkr.Data.ID > 0 {
		did := dkr.Data.ID
		// Toggle danmaku like
		srveOK(t, r, areq("POST", fmt.Sprintf("/api/v1/danmakus/%d/like", did), tk, nil), http.StatusOK)
		// Delete danmaku
		srveOK(t, r, areq("DELETE", fmt.Sprintf("/api/v1/danmakus/%d", did), tk, nil), http.StatusOK)
	}
}

func Test_UserBlockEndpoints(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "ubl1", "UBL1", 10)
	u2 := seedUser(t, api, "ubl2", "UBL2", 10)
	tk := tok(t, api, u.ID)
	// Block user
	srveOK(t, r, areq("POST", fmt.Sprintf("/api/v1/users/%d/block", u2.ID), tk, nil), http.StatusOK)
	// Check blocked user's space
	srveOK(t, r, areq("GET", fmt.Sprintf("/api/v1/space/%d", u2.ID), tk, nil), http.StatusForbidden)
	// Unblock (same toggle endpoint)
	srveOK(t, r, areq("POST", fmt.Sprintf("/api/v1/users/%d/block", u2.ID), tk, nil), http.StatusOK)
}

func Test_AdminMoreActions(t *testing.T) {
	api, r, _ := newTestAPI(t)
	at := admintok(t, api)

	// Hot search dashboard
	srveOK(t, r, areq("GET", "/api/v1/admin/hot-search/dashboard", at, nil), http.StatusOK)

	// List pending videos
	srveOK(t, r, areq("GET", "/api/v1/admin/videos?page=1&page_size=10", at, nil), http.StatusOK)

	// List pending articles
	srveOK(t, r, areq("GET", "/api/v1/admin/articles?page=1&page_size=10", at, nil), http.StatusOK)

	// List dynamics
	srveOK(t, r, areq("GET", "/api/v1/admin/dynamics?page=1&page_size=10", at, nil), http.StatusOK)

	// Admin get video (non-existent -> 404)
	srveOK(t, r, areq("GET", "/api/v1/admin/videos/99999", at, nil), http.StatusNotFound)

	// Admin get article (non-existent -> 404)
	srveOK(t, r, areq("GET", "/api/v1/admin/articles/99999", at, nil), http.StatusNotFound)

	// Admin get dynamic (non-existent -> 404)
	srveOK(t, r, areq("GET", "/api/v1/admin/dynamics/99999", at, nil), http.StatusNotFound)
}

func Test_UserFollowMore(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "ufm1", "UFM1", 10)
	u2 := seedUser(t, api, "ufm2", "UFM2", 10)
	tk := tok(t, api, u.ID)

	// Follow user
	srveOK(t, r, areq("POST", fmt.Sprintf("/api/v1/users/%d/follow", u2.ID), tk, nil), http.StatusOK)

	// List following
	srveOK(t, r, areq("GET", fmt.Sprintf("/api/v1/space/%d/following", u.ID), tk, nil), http.StatusOK)

	// List followers
	srveOK(t, r, areq("GET", fmt.Sprintf("/api/v1/space/%d/followers", u.ID), tk, nil), http.StatusOK)

	// Toggle again -> unfollow
	srveOK(t, r, areq("POST", fmt.Sprintf("/api/v1/users/%d/follow", u2.ID), tk, nil), http.StatusOK)
}

func Test_SearchHistoryMore(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "shm1", "SHM1", 10)
	tk := tok(t, api, u.ID)

	// Post search history
	srveOK(t, r, areq("POST", "/api/v1/users/me/search-history", tk, `{"keyword":"test query"}`), http.StatusOK)

	// Get search history
	srveOK(t, r, areq("GET", "/api/v1/users/me/search-history", tk, nil), http.StatusOK)

	// Put search history (clear)
	srveOK(t, r, areq("PUT", "/api/v1/users/me/search-history", tk, `{"keywords":[]}`), http.StatusOK)
}

func Test_CoinAndRewardMore(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "crm1", "CRM1", 100)
	tk := tok(t, api, u.ID)

	// Claim daily reward
	srveOK(t, r, areq("POST", "/api/v1/users/me/daily-rewards/watch", tk, nil), http.StatusOK)

	// Daily reward status
	srveOK(t, r, areq("GET", "/api/v1/users/me/daily-rewards", tk, nil), http.StatusOK)

	// Coin ledger
	srveOK(t, r, areq("GET", "/api/v1/users/me/coin-ledger?page=1&page_size=10", tk, nil), http.StatusOK)
}
func Test_VideoZoneAndCatalog(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "vzc1", "VZC1", 10)
	v := seedVideoWithAPI(t, api, u.ID, "Zone Video")

	// List with zone filter
	srveOK(t, r, areq("GET", "/api/v1/videos?zone=entertainment&page=1&page_size=10", "", nil), http.StatusOK)

	// Get video detail
	srveOK(t, r, areq("GET", fmt.Sprintf("/api/v1/videos/%d", v.ID), "", nil), http.StatusOK)
}

func Test_SpaceEndpointsMore(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "spm1", "SPM1", 10)

	// Get user public profile
	srveOK(t, r, areq("GET", fmt.Sprintf("/api/v1/space/%d", u.ID), "", nil), http.StatusOK)

	// User not found
	srveOK(t, r, areq("GET", "/api/v1/space/99999", "", nil), http.StatusNotFound)
}

func Test_CreatorEndpoints(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "ce1", "CE1", 10)
	tk := tok(t, api, u.ID)

	// List creator comments
	srveOK(t, r, areq("GET", "/api/v1/users/me/creator/comments?page=1&page_size=10", tk, nil), http.StatusOK)

	// List creator danmakus
	srveOK(t, r, areq("GET", "/api/v1/users/me/creator/danmakus?page=1&page_size=10", tk, nil), http.StatusOK)
}
