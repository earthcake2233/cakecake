//go:build integration

package handler

import (
	"cakecake/internal/model/article"
	"cakecake/internal/model/video"
	"encoding/json"
	"fmt"
	"net/http"
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
	srveOK(t, r, areq("POST", "/api/v1/videos/"+fmt.Sprint(v.ID)+"/comments", tk, body), http.StatusCreated)

	// Check notifications for u1
	tk1 := tok(t, api, u.ID)
	srveOK(t, r, areq("GET", "/api/v1/notifications", tk1, nil), http.StatusOK)
	srveOK(t, r, areq("GET", "/api/v1/notifications/unread-summary", tk1, nil), http.StatusCreated)
}

func Test_FollowGroupManagement(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "fgm1", "FGM1", 10)
	u2 := seedUser(t, api, "fgm2", "FGM2", 10)
	tk := tok(t, api, u.ID)

	// Follow u2 first
	srveOK(t, r, areq("POST", fmt.Sprintf("/api/v1/users/%d/follow", u2.ID), tk, nil), http.StatusOK)

	// Create follow group
	w := srve(r, areq("POST", "/api/v1/users/me/follow-groups", tk, `{"name":"Close Friends"}`))
	var fgr struct {
		Code int `json:"code"`
		Data struct {
			ID uint64 `json:"id"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &fgr))
	if fgr.Code == 0 && fgr.Data.ID > 0 {
		gid := fgr.Data.ID
		// Rename group
		srveOK(t, r, areq("PUT", fmt.Sprintf("/api/v1/users/me/follow-groups/%d", gid), tk, `{"name":"Best Friends"}`), http.StatusOK)
		// Add member
		srveOK(t, r, areq("POST", fmt.Sprintf("/api/v1/users/me/follow-groups/%d/members", gid), tk, fmt.Sprintf(`{"followee_id":%d}`, u2.ID)), http.StatusOK)
		// Remove member
		srveOK(t, r, areq("DELETE", fmt.Sprintf("/api/v1/users/me/follow-groups/%d/members/%d", gid, u2.ID), tk, nil), http.StatusOK)
		// Delete group
		srveOK(t, r, areq("DELETE", fmt.Sprintf("/api/v1/users/me/follow-groups/%d", gid), tk, nil), http.StatusOK)
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
	srveOK(t, r, areq("GET", "/api/v1/admin/videos?page=1&page_size=10", at, nil), http.StatusOK)

	// Get specific pending video
	srveOK(t, r, areq("GET", fmt.Sprintf("/api/v1/admin/videos/%d", v.ID), at, nil), http.StatusOK)

	// List pending articles
	srveOK(t, r, areq("GET", "/api/v1/admin/articles?page=1&page_size=10", at, nil), http.StatusOK)

	// Get specific pending article
	srveOK(t, r, areq("GET", fmt.Sprintf("/api/v1/admin/articles/%d", art.ID), at, nil), http.StatusOK)

	// List dynamics
	srveOK(t, r, areq("GET", "/api/v1/admin/dynamics?page=1&page_size=10", at, nil), http.StatusOK)

	// Reject video (wrong status path)
	srveOK(t, r, areq("POST", fmt.Sprintf("/api/v1/admin/videos/%d/reject", v.ID), at, `{"reason":"Test reject"}`), http.StatusOK)

	// Approve video (wrong status path)
	v2 := video.Video{UserID: u.ID, Title: "Pending Video 2", Status: "pending_review", VideoURL: "https://cdn.example.com/p2.mp4", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	require.NoError(t, api.DB.Create(&v2).Error)
	srveOK(t, r, areq("POST", fmt.Sprintf("/api/v1/admin/videos/%d/approve", v2.ID), at, nil), http.StatusOK)
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
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &pcr))
	if pcr.Code == 0 && pcr.Data.ID > 0 {
		pcid := pcr.Data.ID
		// Reply from another user
		replyBody := fmt.Sprintf(`{"content":"Reply to comment","video_id":%d,"parent_id":%d}`, v.ID, pcid)
		srveOK(t, r, areq("POST", "/api/v1/videos/"+fmt.Sprint(v.ID)+"/comments", tk2, replyBody), http.StatusCreated)
		// List comments to see the reply thread
		srveOK(t, r, areq("GET", fmt.Sprintf("/api/v1/videos/%d/comments", v.ID), "", nil), http.StatusOK)
	}
}

func Test_CoinAndWatchLaterEdge(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "cwe1", "CWE1", 100)
	u2 := seedUser(t, api, "cwe2", "CWE2", 100)
	tk := tok(t, api, u.ID)
	v := seedVideoWithAPI(t, api, u2.ID, "CWE Video")

	// Coin with amount=2
	srveOK(t, r, areq("POST", fmt.Sprintf("/api/v1/videos/%d/coin", v.ID), tk, `{"amount":2}`), http.StatusOK)

	// Toggle watch later
	srveOK(t, r, areq("POST", fmt.Sprintf("/api/v1/videos/%d/watch-later", v.ID), tk, nil), http.StatusOK)

	// List watch later
	srveOK(t, r, areq("GET", "/api/v1/users/me/watch-later", tk, nil), http.StatusOK)

	// Mark as watched
	srveOK(t, r, areq("POST", fmt.Sprintf("/api/v1/users/me/watch-later/%d/watched", v.ID), tk, nil), http.StatusOK)

	// Clear watched
	srveOK(t, r, areq("DELETE", "/api/v1/users/me/watch-later/watched", tk, nil), http.StatusOK)

	// Clear all watch later
	srveOK(t, r, areq("DELETE", "/api/v1/users/me/watch-later", tk, nil), http.StatusOK)
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
	srveOK(t, r, areq("POST", "/api/v1/videos/"+fmt.Sprint(v.ID)+"/comments", tk2, body), http.StatusCreated)

	// List creator comments (as the video owner)
	srveOK(t, r, areq("GET", "/api/v1/users/me/creator/comments?page=1&page_size=10", tk, nil), http.StatusOK)

	// List creator danmakus
	srveOK(t, r, areq("GET", "/api/v1/users/me/creator/danmakus?page=1&page_size=10", tk, nil), http.StatusOK)
}

func Test_HomeStatsAndBanners(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "hsb1", "HSB1", 10)
	_ = seedVideoWithAPI(t, api, u.ID, "HS Video")

	// Home stats should work
	srveOK(t, r, areq("GET", "/api/v1/stats/home", "", nil), http.StatusOK)

	// Home banners
	srveOK(t, r, areq("GET", "/api/v1/home-banners", "", nil), http.StatusOK)
}

func Test_VideoDraftValidation(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "vdv1", "VDV1", 10)
	tk := tok(t, api, u.ID)

	// List video drafts
	srveOK(t, r, areq("GET", "/api/v1/users/me/video-drafts?page=1&page_size=10", tk, nil), http.StatusNotFound)

	// Get draft source for non-existent (404)
	srveOK(t, r, areq("GET", "/api/v1/users/me/videos/99999/draft-source", tk, nil), http.StatusNotFound)
}
