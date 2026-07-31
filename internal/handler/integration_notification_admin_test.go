//go:build integration

package handler

import (
	"encoding/json"
	"fmt"
	"minibili/internal/model/article"
	"minibili/internal/model/video"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func Test_NotificationFromComment(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "nfc1", "NFC1", 10)
	u2 := seedUser(t, api, "nfc2", "NFC2", 10)
	tk := tok(t, api, u2.ID)
	v := seedVideoWithAPI(t, api, u.ID, "Notif Video")

	// Post comment from u2 on u1 video
	body := fmt.Sprintf(`{"content":"Nice video!","video_id":%d}`, v.ID)
	srve(r, areq("POST", "/api/v1/videos/"+fmt.Sprint(v.ID)+"/comments", tk, body))

	// Check notifications for u1
	tk1 := tok(t, api, u.ID)
	srve(r, areq("GET", "/api/v1/notifications", tk1, nil))
	srve(r, areq("GET", "/api/v1/notifications/unread-summary", tk1, nil))
}

func Test_FollowGroupManagement(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "fgm1", "FGM1", 10)
	u2 := seedUser(t, api, "fgm2", "FGM2", 10)
	tk := tok(t, api, u.ID)

	// Follow u2 first
	srve(r, areq("POST", "/api/v1/users/me/follow", tk, fmt.Sprintf(`{"user_id":%d}`, u2.ID)))

	// Create follow group
	w := srve(r, areq("POST", "/api/v1/users/me/follow-groups", tk, `{"name":"Close Friends"}`))
	var fgr struct {
		Code int `json:"code"`
		Data struct {
			ID uint64 `json:"id"`
		} `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &fgr)
	if fgr.Code == 0 && fgr.Data.ID > 0 {
		gid := fgr.Data.ID
		// Rename group
		srve(r, areq("PUT", fmt.Sprintf("/api/v1/users/me/follow-groups/%d/rename", gid), tk, `{"name":"Best Friends"}`))
		// Add member
		srve(r, areq("POST", fmt.Sprintf("/api/v1/users/me/follow-groups/%d/members", gid), tk, fmt.Sprintf(`{"user_id":%d}`, u2.ID)))
		// Remove member
		srve(r, areq("DELETE", fmt.Sprintf("/api/v1/users/me/follow-groups/%d/members/%d", gid, u2.ID), tk, nil))
		// Delete group
		srve(r, areq("DELETE", fmt.Sprintf("/api/v1/users/me/follow-groups/%d", gid), tk, nil))
	}
}

func Test_AdminReviewFlow(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "arf1", "ARF1", 10)
	at := admintok(t, api)

	// Create pending video
	v := video.Video{UserID: u.ID, Title: "Pending Video", Status: "pending_review", VideoURL: "https://cdn.example.com/p.mp4", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	require.NoError(t, api.DB.Create(&v).Error)

	// Create pending article
	art := article.Article{UserID: u.ID, Title: "Pending Article", BodyMD: "# Content", Status: "pending_review", CreatedAt: time.Now()}
	require.NoError(t, api.DB.Create(&art).Error)

	// List pending videos
	srve(r, areq("GET", "/api/v1/admin/videos?page=1&page_size=10", at, nil))

	// Get specific pending video
	srve(r, areq("GET", fmt.Sprintf("/api/v1/admin/videos/%d", v.ID), at, nil))

	// List pending articles
	srve(r, areq("GET", "/api/v1/admin/articles?page=1&page_size=10", at, nil))

	// Get specific pending article
	srve(r, areq("GET", fmt.Sprintf("/api/v1/admin/articles/%d", art.ID), at, nil))

	// List dynamics
	srve(r, areq("GET", "/api/v1/admin/dynamics?page=1&page_size=10", at, nil))

	// Reject video (wrong status path)
	srve(r, areq("POST", fmt.Sprintf("/api/v1/admin/videos/%d/reject", v.ID), at, `{"reason":"Test reject"}`))

	// Approve video (wrong status path)
	srve(r, areq("POST", fmt.Sprintf("/api/v1/admin/videos/%d/approve", v.ID), at, nil))
}

func Test_CommentReplyFlow(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "crf1", "CRF1", 10)
	u2 := seedUser(t, api, "crf2", "CRF2", 10)
	tk := tok(t, api, u.ID)
	tk2 := tok(t, api, u2.ID)
	v := seedVideoWithAPI(t, api, u2.ID, "Reply Video")

	// Post parent comment
	body := fmt.Sprintf(`{"content":"Parent comment","video_id":%d}`, v.ID)
	w := srve(r, areq("POST", "/api/v1/videos/"+fmt.Sprint(v.ID)+"/comments", tk, body))
	var pcr struct {
		Code int `json:"code"`
		Data struct {
			ID uint64 `json:"id"`
		} `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &pcr)
	if pcr.Code == 0 && pcr.Data.ID > 0 {
		pcid := pcr.Data.ID
		// Reply from another user
		replyBody := fmt.Sprintf(`{"content":"Reply to comment","video_id":%d,"parent_id":%d}`, v.ID, pcid)
		srve(r, areq("POST", "/api/v1/videos/"+fmt.Sprint(v.ID)+"/comments", tk2, replyBody))
		// List comments to see the reply thread
		srve(r, areq("GET", fmt.Sprintf("/api/v1/videos/%d/comments", v.ID), "", nil))
	}
}

func Test_CoinAndWatchLaterEdge(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "cwe1", "CWE1", 100)
	u2 := seedUser(t, api, "cwe2", "CWE2", 100)
	tk := tok(t, api, u.ID)
	v := seedVideoWithAPI(t, api, u2.ID, "CWE Video")

	// Coin with amount=2
	srve(r, areq("POST", fmt.Sprintf("/api/v1/videos/%d/coin", v.ID), tk, `{"amount":2}`))

	// Toggle watch later
	srve(r, areq("POST", fmt.Sprintf("/api/v1/videos/%d/watch-later", v.ID), tk, nil))

	// List watch later
	srve(r, areq("GET", "/api/v1/users/me/watch-later", tk, nil))

	// Mark as watched
	srve(r, areq("POST", fmt.Sprintf("/api/v1/users/me/watch-later/%d/watched", v.ID), tk, nil))

	// Clear watched
	srve(r, areq("DELETE", "/api/v1/users/me/watch-later/watched", tk, nil))

	// Clear all watch later
	srve(r, areq("DELETE", "/api/v1/users/me/watch-later", tk, nil))
}

func Test_CreatorEndpointsFull(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "cef1", "CEF1", 10)
	u2 := seedUser(t, api, "cef2", "CEF2", 10)
	tk := tok(t, api, u.ID)
	v := seedVideoWithAPI(t, api, u.ID, "Creator Video")

	// Post a comment on the video from another user
	tk2 := tok(t, api, u2.ID)
	body := fmt.Sprintf(`{"content":"Creator comment","video_id":%d}`, v.ID)
	srve(r, areq("POST", "/api/v1/videos/"+fmt.Sprint(v.ID)+"/comments", tk2, body))

	// List creator comments (as the video owner)
	srve(r, areq("GET", "/api/v1/users/me/creator/comments?page=1&page_size=10", tk, nil))

	// List creator danmakus
	srve(r, areq("GET", "/api/v1/users/me/creator/danmakus?page=1&page_size=10", tk, nil))
}

func Test_HomeStatsAndBanners(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "hsb1", "HSB1", 10)
	_ = seedVideoWithAPI(t, api, u.ID, "HS Video")

	// Home stats should work
	srve(r, areq("GET", "/api/v1/stats/home", "", nil))

	// Home banners
	srve(r, areq("GET", "/api/v1/home-banners", "", nil))
}

func Test_VideoDraftValidation(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "vdv1", "VDV1", 10)
	tk := tok(t, api, u.ID)

	// List video drafts
	srve(r, areq("GET", "/api/v1/users/me/video-drafts?page=1&page_size=10", tk, nil))

	// Get draft source for non-existent (404)
	srve(r, areq("GET", "/api/v1/users/me/videos/99999/draft-source", tk, nil))
}
