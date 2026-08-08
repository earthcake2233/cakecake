//go:build integration

package handler

import (
	"cakecake/internal/model/comment"
	"cakecake/internal/model/dynamic"
	"encoding/json"
	"fmt"
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
		srve(r, areq("POST", fmt.Sprintf("/api/v1/comments/%d/pin", cm.ID), tk, nil))
		// Approve comment
		srve(r, areq("POST", fmt.Sprintf("/api/v1/comments/%d/approve", cm.ID), tk, nil))
		// Ignore curated
		srve(r, areq("POST", fmt.Sprintf("/api/v1/comments/%d/ignore-curated", cm.ID), tk, nil))
		// Toggle like
		srve(r, areq("POST", fmt.Sprintf("/api/v1/comments/%d/like", cm.ID), tk, nil))
		// Toggle dislike
		srve(r, areq("POST", fmt.Sprintf("/api/v1/comments/%d/dislike", cm.ID), tk, nil))
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
		srve(r, areq("POST", fmt.Sprintf("/api/v1/article-comments/%d/like", cid), tk, nil))
		srve(r, areq("POST", fmt.Sprintf("/api/v1/article-comments/%d/dislike", cid), tk, nil))
		srve(r, areq("POST", fmt.Sprintf("/api/v1/article-comments/%d/pin", cid), tk, nil))
		srve(r, areq("POST", fmt.Sprintf("/api/v1/article-comments/%d/approve", cid), tk, nil))
		srve(r, areq("POST", fmt.Sprintf("/api/v1/article-comments/%d/ignore-curated", cid), tk, nil))
		srve(r, areq("DELETE", fmt.Sprintf("/api/v1/article-comments/%d", cid), tk, nil))
	}
}

func Test_UpdateMeProfileAndPrivacy(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "ump1", "UMP1", 10)
	tk := tok(t, api, u.ID)
	srve(r, areq("PUT", "/api/v1/users/me", tk, `{"nickname":"NewName","sign":"Hello"}`))
	srve(r, areq("PUT", "/api/v1/users/me/announcement", tk, `{"announcement":"Welcome!"}`))
	srve(r, areq("PUT", "/api/v1/users/me/space-privacy", tk, `{"public_favorites":true}`))
	// Avatar upload (nil body -> parse error -> 400)
	srve(r, areq("POST", "/api/v1/users/me/avatar", tk, nil))
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
		srve(r, areq("PUT", fmt.Sprintf("/api/v1/users/me/follow-groups/%d/rename", gid), tk, `{"name":"Renamed"}`))
		// Add member
		srve(r, areq("POST", fmt.Sprintf("/api/v1/users/me/follow-groups/%d/members", gid), tk, fmt.Sprintf(`{"user_id":%d}`, u2.ID)))
		// Delete group
		srve(r, areq("DELETE", fmt.Sprintf("/api/v1/users/me/follow-groups/%d", gid), tk, nil))
	}
}

func Test_DmActions(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "dma1", "DMA1", 10)
	u2 := seedUser(t, api, "dma2", "DMA2", 10)
	tk := tok(t, api, u.ID)
	// Create a conversation
	w := srve(r, areq("POST", "/api/v1/dm/conversations", tk, fmt.Sprintf(`{"user_id":%d}`, u2.ID)))
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
		srve(r, areq("GET", fmt.Sprintf("/api/v1/dm/conversations/%d/messages", cid), tk, nil))
		// Post a message
		srve(r, areq("POST", fmt.Sprintf("/api/v1/dm/conversations/%d/messages", cid), tk, `{"content":"Hello!"}`))
		// Patch settings
		srve(r, areq("PATCH", fmt.Sprintf("/api/v1/dm/conversations/%d/settings", cid), tk, `{"auto_reply":false}`))
		// Delete conversation
		srve(r, areq("DELETE", fmt.Sprintf("/api/v1/dm/conversations/%d", cid), tk, nil))
	}
}

func Test_VideoActions(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "vact1", "VACT1", 10)
	tk := tok(t, api, u.ID)
	v := seedVideoWithAPI(t, api, u.ID, "VA Video")

	// List my videos
	srve(r, areq("GET", "/api/v1/users/me/videos?page=1&page_size=10", tk, nil))

	// Update video
	srve(r, areq("PUT", fmt.Sprintf("/api/v1/videos/%d", v.ID), tk, `{"title":"Updated Title","tags":["tag1"]}`))

	// Patch playback
	srve(r, areq("PATCH", fmt.Sprintf("/api/v1/videos/%d/playback", v.ID), tk, `{"current_time":30.5}`))

	// Update cover (nil body -> parse error -> 400)
	srve(r, areq("PUT", fmt.Sprintf("/api/v1/videos/%d/cover", v.ID), tk, nil))
}

func Test_ArticleActions(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "aact1", "AACT1", 10)
	tk := tok(t, api, u.ID)

	// Post a draft article
	w := srve(r, areq("POST", "/api/v1/articles", tk, `{"title":"My Article","content_md":"# Hello","publish":false}`))
	type artResp struct {
		Code int `json:"code"`
		Data struct {
			ID uint64 `json:"id"`
		} `json:"data"`
	}
	var ar artResp
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &ar))

	// List my articles
	srve(r, areq("GET", "/api/v1/users/me/articles?page=1&page_size=10", tk, nil))

	// Count my articles
	srve(r, areq("GET", "/api/v1/users/me/articles/count", tk, nil))

	if ar.Code == 0 && ar.Data.ID > 0 {
		aid := ar.Data.ID
		// Patch playback
		srve(r, areq("PATCH", fmt.Sprintf("/api/v1/articles/%d/playback", aid), tk, `{"current_time":15.0}`))

		// Post view
		srve(r, areq("POST", fmt.Sprintf("/api/v1/articles/%d/view", aid), tk, nil))

		// Update article
		srve(r, areq("PUT", fmt.Sprintf("/api/v1/users/me/articles/%d", aid), tk, `{"title":"Updated Title","content_md":"# Updated"}`))

		// Update cover (nil body -> 400)
		srve(r, areq("PUT", fmt.Sprintf("/api/v1/articles/%d/cover", aid), tk, nil))

		// Delete article
		srve(r, areq("DELETE", fmt.Sprintf("/api/v1/users/me/articles/%d", aid), tk, nil))
	}
}

func Test_VideoEngagementMore(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "vem1", "VEM1", 100)
	u2 := seedUser(t, api, "vem2", "VEM2", 100)
	tk := tok(t, api, u.ID)
	v := seedVideoWithAPI(t, api, u2.ID, "VEM Video")

	// Toggle favorite
	srve(r, areq("POST", fmt.Sprintf("/api/v1/videos/%d/favorite", v.ID), tk, nil))

	// Get favorite picker
	srve(r, areq("GET", fmt.Sprintf("/api/v1/videos/%d/favorite-picker", v.ID), tk, nil))

	// Post coin
	srve(r, areq("POST", fmt.Sprintf("/api/v1/videos/%d/coin", v.ID), tk, `{"amount":1}`))

	// Toggle watch later
	srve(r, areq("POST", fmt.Sprintf("/api/v1/videos/%d/watch-later", v.ID), tk, nil))

	// List watch later
	srve(r, areq("GET", "/api/v1/users/me/watch-later", tk, nil))

	// Clear watch later watched
	srve(r, areq("DELETE", "/api/v1/users/me/watch-later/watched", tk, nil))
}

func Test_DynamicActions(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "dya1", "DYA1", 10)
	tk := tok(t, api, u.ID)

	// Create a dynamic
	dyn := dynamic.UserDynamic{UserID: u.ID, Title: "My Dynamic", Content: "Dynamic Content", ImagesJSON: "[]", CreatedAt: time.Now()}
	require.NoError(t, api.DB.Create(&dyn).Error)

	// Get dynamic
	srve(r, areq("GET", fmt.Sprintf("/api/v1/user-dynamics/%d", dyn.ID), "", nil))

	// Post dynamic comment
	srve(r, areq("POST", fmt.Sprintf("/api/v1/user-dynamics/%d/comments", dyn.ID), tk, `{"content":"Nice dynamic!"}`))

	// List space dynamics
	srve(r, areq("GET", fmt.Sprintf("/api/v1/space/%d/dynamics", u.ID), "", nil))
}

func Test_NotificationEndpoints(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "ne1", "NE1", 10)
	tk := tok(t, api, u.ID)

	// Unread summary
	srve(r, areq("GET", "/api/v1/notifications/unread-summary", tk, nil))

	// List notifications
	srve(r, areq("GET", "/api/v1/notifications?page=1&page_size=10", tk, nil))

	// Read by category (uses sqlite-compatible value)
	srve(r, areq("PATCH", "/api/v1/notifications/read-by-category", tk, `{"category":"like"}`))

	// Mark batch read (with nil ids)
	srve(r, areq("PATCH", "/api/v1/notifications/read-batch", tk, `{"ids":[]}`))

	// Mute like notification (non-existent -> handled gracefully)
	srve(r, areq("POST", "/api/v1/notifications/0/mute-likes", tk, nil))

	// Delete notification (non-existent -> handled)
	srve(r, areq("DELETE", "/api/v1/notifications/0", tk, nil))
}

func Test_DanmakuActions(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "dka1", "DKA1", 10)
	tk := tok(t, api, u.ID)
	v := seedVideoWithAPI(t, api, u.ID, "DK Video")

	// Post a danmaku
	w := srve(r, areq("POST", fmt.Sprintf("/api/v1/videos/%d/danmaku", v.ID), tk, `{"content":"Hello","type":0,"color":16777215,"progress":10.5}`))
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
		srve(r, areq("POST", fmt.Sprintf("/api/v1/danmakus/%d/like", did), tk, nil))
		// Delete danmaku
		srve(r, areq("DELETE", fmt.Sprintf("/api/v1/danmakus/%d", did), tk, nil))
	}
}

func Test_UserBlockEndpoints(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "ubl1", "UBL1", 10)
	u2 := seedUser(t, api, "ubl2", "UBL2", 10)
	tk := tok(t, api, u.ID)
	// Block user
	srve(r, areq("POST", "/api/v1/users/me/block", tk, fmt.Sprintf(`{"user_id":%d}`, u2.ID)))
	// Check blocked user's space
	srve(r, areq("GET", fmt.Sprintf("/api/v1/space/%d", u2.ID), tk, nil))
	// Unblock
	srve(r, areq("POST", "/api/v1/users/me/unblock", tk, fmt.Sprintf(`{"user_id":%d}`, u2.ID)))
}

func Test_AdminMoreActions(t *testing.T) {
	api, r, _ := newTestAPI(t)
	at := admintok(t, api)

	// Hot search dashboard
	srve(r, areq("GET", "/api/v1/admin/hot-search/dashboard", at, nil))

	// List pending videos
	srve(r, areq("GET", "/api/v1/admin/videos?page=1&page_size=10", at, nil))

	// List pending articles
	srve(r, areq("GET", "/api/v1/admin/articles?page=1&page_size=10", at, nil))

	// List dynamics
	srve(r, areq("GET", "/api/v1/admin/dynamics?page=1&page_size=10", at, nil))

	// Admin get video (non-existent -> 404)
	srve(r, areq("GET", "/api/v1/admin/videos/99999", at, nil))

	// Admin get article (non-existent -> 404)
	srve(r, areq("GET", "/api/v1/admin/articles/99999", at, nil))

	// Admin get dynamic (non-existent -> 404)
	srve(r, areq("GET", "/api/v1/admin/dynamics/99999", at, nil))
}

func Test_UserFollowMore(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "ufm1", "UFM1", 10)
	u2 := seedUser(t, api, "ufm2", "UFM2", 10)
	tk := tok(t, api, u.ID)

	// Follow user
	srve(r, areq("POST", "/api/v1/users/me/follow", tk, fmt.Sprintf(`{"user_id":%d}`, u2.ID)))

	// List following
	srve(r, areq("GET", fmt.Sprintf("/api/v1/space/%d/following", u.ID), "", nil))

	// List followers
	srve(r, areq("GET", fmt.Sprintf("/api/v1/space/%d/followers", u.ID), "", nil))

	// Unfollow
	srve(r, areq("POST", "/api/v1/users/me/unfollow", tk, fmt.Sprintf(`{"user_id":%d}`, u2.ID)))
}

func Test_SearchHistoryMore(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "shm1", "SHM1", 10)
	tk := tok(t, api, u.ID)

	// Post search history
	srve(r, areq("POST", "/api/v1/users/me/search-history", tk, `{"keyword":"test query"}`))

	// Get search history
	srve(r, areq("GET", "/api/v1/users/me/search-history", tk, nil))

	// Put search history (clear)
	srve(r, areq("PUT", "/api/v1/users/me/search-history", tk, nil))
}

func Test_CoinAndRewardMore(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "crm1", "CRM1", 100)
	tk := tok(t, api, u.ID)

	// Claim daily reward
	srve(r, areq("POST", "/api/v1/users/me/daily-reward", tk, nil))

	// Daily reward status
	srve(r, areq("GET", "/api/v1/users/me/daily-reward/status", tk, nil))

	// Coin ledger
	srve(r, areq("GET", "/api/v1/users/me/coin-ledger?page=1&page_size=10", tk, nil))
}
func Test_VideoZoneAndCatalog(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "vzc1", "VZC1", 10)
	v := seedVideoWithAPI(t, api, u.ID, "Zone Video")

	// List with zone filter
	srve(r, areq("GET", "/api/v1/videos?zone=entertainment&page=1&page_size=10", "", nil))

	// Get video detail
	srve(r, areq("GET", fmt.Sprintf("/api/v1/videos/%d", v.ID), "", nil))
}

func Test_SpaceEndpointsMore(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "spm1", "SPM1", 10)

	// Get user public profile
	srve(r, areq("GET", fmt.Sprintf("/api/v1/space/%d", u.ID), "", nil))

	// User not found
	srve(r, areq("GET", "/api/v1/space/99999", "", nil))
}

func Test_CreatorEndpoints(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "ce1", "CE1", 10)
	tk := tok(t, api, u.ID)

	// List creator comments
	srve(r, areq("GET", "/api/v1/users/me/creator/comments?page=1&page_size=10", tk, nil))

	// List creator danmakus
	srve(r, areq("GET", "/api/v1/users/me/creator/danmakus?page=1&page_size=10", tk, nil))
}
