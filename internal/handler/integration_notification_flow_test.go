//go:build integration

package handler

import (
	"cakecake/internal/model/notification"
	"cakecake/internal/model/video"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func Test_NotificationFullFlow_SeedAndList(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "nff1", "NFF1", 10)
	tk := tok(t, api, u.ID)
	for i := 0; i < 5; i++ {
		seedNotification(t, api, u.ID, "reply", uint64(i+100))
	}
	for i := 0; i < 3; i++ {
		n := notification.Notification{
			RecipientID:     u.ID,
			Type:            "like_aggregation",
			RelatedID:       uint64(i + 200),
			SenderNamesJSON: `["user_a","user_b","user_c"]`,
			TotalLikes:      3 + i,
			CommentPreview:  "Nice!",
			PayloadJSON:     `{"like_subject":"comment"}`,
			IsRead:          false,
			CreatedAt:       time.Now(),
		}
		require.NoError(t, api.DB.Create(&n).Error)
	}
	srveOK(t, r, areq("GET", "/api/v1/notifications", tk, nil), http.StatusOK)
	srveOK(t, r, areq("GET", "/api/v1/notifications/unread-summary", tk, nil), http.StatusCreated)
}

func Test_NotificationReadAndMute(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "nrm2", "NRM2", 10)
	tk := tok(t, api, u.ID)
	nid := seedNotification(t, api, u.ID, "reply", 0)
	srveOK(t, r, areq("PATCH", fmt.Sprintf("/api/v1/notifications/%d/read", nid), tk, nil), http.StatusOK)
	srveOK(t, r, areq("POST", fmt.Sprintf("/api/v1/notifications/%d/mute-likes", nid), tk, nil), http.StatusBadRequest)
	srveOK(t, r, areq("PATCH", "/api/v1/notifications/read-batch", tk, fmt.Sprintf(`[%d]`, nid)), http.StatusOK)
	srveOK(t, r, areq("PATCH", "/api/v1/notifications/read-by-category?category=reply", tk, nil), http.StatusOK)
	srveOK(t, r, areq("DELETE", fmt.Sprintf("/api/v1/notifications/%d", nid), tk, nil), http.StatusOK)
}

func Test_NotificationCommentReplyAndLike(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "ncr1", "NCR1", 10)
	tk := tok(t, api, u.ID)
	n := notification.Notification{
		RecipientID:     u.ID,
		Type:            "reply",
		RelatedID:       0,
		SenderNamesJSON: `["someone"]`,
		PayloadJSON:     `{"reply_body":"Check this out","video_id":0}`,
		IsRead:          false,
		CreatedAt:       time.Now(),
	}
	require.NoError(t, api.DB.Create(&n).Error)
	srveOK(t, r, areq("POST", fmt.Sprintf("/api/v1/notifications/%d/comment-like", n.ID), tk, nil), http.StatusBadRequest)
	srveOK(t, r, areq("POST", fmt.Sprintf("/api/v1/notifications/%d/comment-reply", n.ID), tk, `{"content":"Thanks!"}`), http.StatusNotFound)
}

func Test_NotificationLikeLikers(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "nll1", "NLL1", 10)
	tk := tok(t, api, u.ID)
	n := notification.Notification{
		RecipientID:     u.ID,
		Type:            "like_aggregation",
		RelatedID:       0,
		SenderNamesJSON: `["liker_a","liker_b"]`,
		TotalLikes:      2,
		CommentPreview:  "Cool!",
		PayloadJSON:     `{"like_subject":"comment"}`,
		IsRead:          false,
		CreatedAt:       time.Now(),
	}
	require.NoError(t, api.DB.Create(&n).Error)
	srveOK(t, r, areq("GET", fmt.Sprintf("/api/v1/notifications/%d/like-likers", n.ID), tk, nil), http.StatusOK)
}

func Test_DmConversationMultipleMessages(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "dmm1", "DMM1", 10)
	u2 := seedUser(t, api, "dmm2", "DMM2", 10)
	tk := tok(t, api, u.ID)
	tk2 := tok(t, api, u2.ID)
	w := srve(r, areq("POST", "/api/v1/dm/conversations", tk, fmt.Sprintf(`{"peer_id":%d}`, u2.ID)))
	var dcr struct {
		Code int `json:"code"`
		Data struct {
			ID uint64 `json:"id"`
		}
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &dcr))
	if dcr.Code == 0 && dcr.Data.ID > 0 {
		cid := dcr.Data.ID
		for i := 0; i < 3; i++ {
			srveOK(t, r, areq("POST", fmt.Sprintf("/api/v1/dm/conversations/%d/messages", cid), tk, fmt.Sprintf(`{"content":"U1 msg %d"}`, i)), http.StatusOK)
			srveOK(t, r, areq("POST", fmt.Sprintf("/api/v1/dm/conversations/%d/messages", cid), tk2, fmt.Sprintf(`{"content":"U2 reply %d"}`, i)), http.StatusOK)
		}
		srveOK(t, r, areq("GET", "/api/v1/dm/conversations", tk, nil), http.StatusOK)
		srveOK(t, r, areq("GET", fmt.Sprintf("/api/v1/dm/conversations/%d/messages", cid), tk, nil), http.StatusOK)
		srveOK(t, r, areq("PATCH", fmt.Sprintf("/api/v1/dm/conversations/%d/settings", cid), tk, `{"pinned":true}`), http.StatusOK)
		srveOK(t, r, areq("POST", fmt.Sprintf("/api/v1/dm/conversations/%d/reset", cid), tk, nil), http.StatusOK)
		srveOK(t, r, areq("DELETE", fmt.Sprintf("/api/v1/dm/conversations/%d", cid), tk, nil), http.StatusOK)
	}
}

func Test_UserDynamicPostUpdateDelete(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "udp1", "UDP1", 10)
	tk := tok(t, api, u.ID)
	w := doMultipart(r, "POST", "/api/v1/users/me/dynamics", tk, map[string]string{
		"title": "Test Dynamic", "content": "Test content",
	})
	covOK(t, w, http.StatusOK)
	var dr struct {
		Code int `json:"code"`
		Data struct {
			ID uint64 `json:"id"`
		}
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &dr))
	require.Equal(t, 0, dr.Code, w.Body.String())
	if dr.Code == 0 && dr.Data.ID > 0 {
		did := dr.Data.ID
		uw := doMultipart(r, "PUT", fmt.Sprintf("/api/v1/users/me/dynamics/%d", did), tk, map[string]string{
			"title": "Updated", "content": "Updated",
		})
		covOK(t, uw, http.StatusOK)
		pw := covReq(t, r, "PATCH", fmt.Sprintf("/api/v1/users/me/dynamics/%d/playback", did), tk, map[string]any{"comments_closed": true})
		covOK(t, pw, http.StatusOK)
		require.Contains(t, pw.Body.String(), `"comments_closed":true`)
		srveOK(t, r, areq("GET", "/api/v1/users/me/dynamics?page=1&page_size=10", tk, nil), http.StatusOK)
		srveOK(t, r, areq("DELETE", fmt.Sprintf("/api/v1/users/me/dynamics/%d", did), tk, nil), http.StatusOK)
	}
}

func Test_VideoEngagementFullFolderFlow(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "vef2", "VEF2", 10)
	u2 := seedUser(t, api, "vef3", "VEF3", 10)
	tk := tok(t, api, u.ID)
	v := seedVideoWithAPI(t, api, u2.ID, "VEF2 Video")
	w := srve(r, areq("POST", "/api/v1/users/me/favorite-folders", tk, `{"title":"VEF Folder"}`))
	var fr struct {
		Code int `json:"code"`
		Data struct {
			ID uint64 `json:"id"`
		}
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &fr))
	if fr.Code == 0 && fr.Data.ID > 0 {
		fid := fr.Data.ID
		folder2 := video.FavoriteFolder{UserID: u.ID, Title: "VEF Folder 2"}
		require.NoError(t, api.DB.Create(&folder2).Error)
		srveOK(t, r, areq("POST", fmt.Sprintf("/api/v1/videos/%d/favorite", v.ID), tk, nil), http.StatusOK)
		srveOK(t, r, areq("POST", fmt.Sprintf("/api/v1/videos/%d/favorite-folders/%d", v.ID, fid), tk, nil), http.StatusOK)
		srveOK(t, r, areq("GET", fmt.Sprintf("/api/v1/videos/%d/favorite-picker", v.ID), tk, nil), http.StatusOK)
		srveOK(t, r, areq("PUT", fmt.Sprintf("/api/v1/videos/%d/favorite-folders", v.ID), tk, fmt.Sprintf(`{"folder_ids":[%d]}`, fid)), http.StatusOK)
		srveOK(t, r, areq("PUT", fmt.Sprintf("/api/v1/videos/%d/favorite-folders/move", v.ID), tk, fmt.Sprintf(`{"from_folder_id":%d,"to_folder_id":%d}`, fid, folder2.ID)), http.StatusOK)
		srveOK(t, r, areq("DELETE", fmt.Sprintf("/api/v1/videos/%d/favorite-folders/%d", v.ID, fid), tk, nil), http.StatusOK)
		srveOK(t, r, areq("POST", fmt.Sprintf("/api/v1/videos/%d/favorite", v.ID), tk, nil), http.StatusOK)
	}
}

func Test_VideoDraftListAndSource(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "vdu1", "VDU1", 10)
	tk := tok(t, api, u.ID)
	srveOK(t, r, areq("GET", "/api/v1/users/me/video-drafts?page=1&page_size=10", tk, nil), http.StatusNotFound)
	srveOK(t, r, areq("GET", "/api/v1/users/me/videos/99999/draft-source", tk, nil), http.StatusNotFound)
}

func Test_ViewHistoryRecordAndClear(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "vhr1", "VHR1", 10)
	u2 := seedUser(t, api, "vhr2", "VHR2", 10)
	tk := tok(t, api, u.ID)
	v := seedVideoWithAPI(t, api, u2.ID, "VHR Video")
	srveOK(t, r, areq("POST", fmt.Sprintf("/api/v1/videos/%d/view-history", v.ID), tk, `{"progress_sec":10.0,"duration_sec":200.0}`), http.StatusOK)
	srveOK(t, r, areq("GET", "/api/v1/users/me/view-history?page=1&page_size=10", tk, nil), http.StatusOK)
	srveOK(t, r, areq("DELETE", fmt.Sprintf("/api/v1/users/me/view-history/%d", v.ID), tk, nil), http.StatusOK)
	srveOK(t, r, areq("DELETE", "/api/v1/users/me/view-history", tk, nil), http.StatusOK)
}

func Test_ArticleCommentReplyFlow(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "acf1", "ACF1", 10)
	u2 := seedUser(t, api, "acf2", "ACF2", 10)
	tk := tok(t, api, u.ID)
	art := seedArticle(t, api, u.ID, "ACF Article")
	w := srve(r, areq("POST", fmt.Sprintf("/api/v1/articles/%d/comments", art.ID), tok(t, api, u2.ID), `{"content":"Parent article comment"}`))
	var acr struct {
		Code int `json:"code"`
		Data struct {
			ID uint64 `json:"id"`
		}
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &acr))
	if acr.Code == 0 && acr.Data.ID > 0 {
		pcid := acr.Data.ID
		srveOK(t, r, areq("POST", fmt.Sprintf("/api/v1/articles/%d/comments", art.ID), tk, fmt.Sprintf(`{"content":"Owner reply","parent_id":%d}`, pcid)), http.StatusCreated)
		srveOK(t, r, areq("POST", fmt.Sprintf("/api/v1/article-comments/%d/approve", pcid), tk, nil), http.StatusOK)
		srveOK(t, r, areq("POST", fmt.Sprintf("/api/v1/article-comments/%d/like", pcid), tk, nil), http.StatusOK)
		srveOK(t, r, areq("POST", fmt.Sprintf("/api/v1/article-comments/%d/pin", pcid), tk, nil), http.StatusOK)
		srveOK(t, r, areq("GET", fmt.Sprintf("/api/v1/articles/%d/comments", art.ID), "", nil), http.StatusOK)
	}
}

func Test_VideoUploadAndUpdate(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "vuu1", "VUU1", 10)
	tk := tok(t, api, u.ID)
	srveOK(t, r, areq("POST", "/api/v1/videos", tk, `{"title":"Test Upload","description":"Test desc","status":"draft"}`), http.StatusBadRequest)
	srveOK(t, r, areq("GET", "/api/v1/users/me/videos?page=1&page_size=10", tk, nil), http.StatusOK)
}

func Test_AdminHotSearchAllOps(t *testing.T) {
	api, r, _ := newTestAPI(t)
	at := admintok(t, api)
	w := srve(r, areq("POST", "/api/v1/admin/hot-search/ops", at, `{"keyword":"summer","op_type":"manual","display_title":"夏季"}`))
	var hopr struct {
		Code int `json:"code"`
		Data struct {
			ID uint64 `json:"id"`
		}
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &hopr))
	if hopr.Code == 0 && hopr.Data.ID > 0 {
		srveOK(t, r, areq("PUT", fmt.Sprintf("/api/v1/admin/hot-search/ops/%d", hopr.Data.ID), at, `{"keyword":"summer_upd","score":90}`), http.StatusOK)
		srveOK(t, r, areq("DELETE", fmt.Sprintf("/api/v1/admin/hot-search/ops/%d", hopr.Data.ID), at, nil), http.StatusOK)
	}
	srveOK(t, r, areq("GET", "/api/v1/admin/hot-search/ops", at, nil), http.StatusOK)
	srveOK(t, r, areq("GET", "/api/v1/admin/hot-search/dashboard", at, nil), http.StatusOK)
	srveOK(t, r, areq("GET", "/api/v1/admin/hot-search/preview", at, nil), http.StatusOK)
	srveOK(t, r, areq("POST", "/api/v1/admin/hot-search/reorder", at, `{"items":[{"keyword":"summer","title":"夏季","source":"manual"}]}`), http.StatusOK)
	srveOK(t, r, areq("POST", "/api/v1/admin/hot-search/display-order/reset", at, nil), http.StatusOK)
	srveOK(t, r, areq("POST", "/api/v1/admin/hot-search/redis/remove", at, `{"keyword":"test"}`), http.StatusOK)
	srveOK(t, r, areq("POST", "/api/v1/admin/hot-search/redis/boost", at, `{"keyword":"boosted","delta":300}`), http.StatusOK)
}

func Test_AdminVideoFullCRUD(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "avc1", "AVC1", 10)
	at := admintok(t, api)
	v := video.Video{UserID: u.ID, Title: "Admin CRUD Video", Status: "pending_review", VideoURL: "https://example.com/avc.mp4", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	require.NoError(t, api.DB.Create(&v).Error)
	srveOK(t, r, areq("GET", "/api/v1/admin/videos?page=1&page_size=10", at, nil), http.StatusOK)
	srveOK(t, r, areq("GET", fmt.Sprintf("/api/v1/admin/videos/%d", v.ID), at, nil), http.StatusOK)
	srveOK(t, r, areq("POST", fmt.Sprintf("/api/v1/admin/videos/%d/approve", v.ID), at, nil), http.StatusOK)
	v2 := video.Video{UserID: u.ID, Title: "Admin CRUD Video 2", Status: "pending_review", VideoURL: "https://example.com/avc2.mp4", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	require.NoError(t, api.DB.Create(&v2).Error)
	srveOK(t, r, areq("POST", fmt.Sprintf("/api/v1/admin/videos/%d/reject", v2.ID), at, `{"reason":"Test"}`), http.StatusOK)
	srveOK(t, r, areq("DELETE", fmt.Sprintf("/api/v1/admin/videos/%d", v.ID), at, nil), http.StatusOK)
}

func Test_FollowGroupMemberManagement(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "fgm2", "FGM2a", 10)
	u2 := seedUser(t, api, "fgm3", "FGM3", 10)
	tk := tok(t, api, u.ID)
	srveOK(t, r, areq("POST", fmt.Sprintf("/api/v1/users/%d/follow", u2.ID), tk, nil), http.StatusOK)
	w := srve(r, areq("POST", "/api/v1/users/me/follow-groups", tk, `{"name":"Special Friends"}`))
	var fgr struct {
		Code int `json:"code"`
		Data struct {
			ID uint64 `json:"id"`
		}
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &fgr))
	if fgr.Code == 0 && fgr.Data.ID > 0 {
		gid := fgr.Data.ID
		srveOK(t, r, areq("POST", fmt.Sprintf("/api/v1/users/me/follow-groups/%d/members", gid), tk, fmt.Sprintf(`{"followee_id":%d}`, u2.ID)), http.StatusOK)
		srveOK(t, r, areq("GET", fmt.Sprintf("/api/v1/users/me/follow-groups/%d/members", gid), tk, nil), http.StatusNotFound)
		srveOK(t, r, areq("DELETE", fmt.Sprintf("/api/v1/users/me/follow-groups/%d/members/%d", gid, u2.ID), tk, nil), http.StatusOK)
		srveOK(t, r, areq("PUT", fmt.Sprintf("/api/v1/users/me/follow-groups/%d", gid), tk, `{"name":"Best Friends"}`), http.StatusOK)
		srveOK(t, r, areq("DELETE", fmt.Sprintf("/api/v1/users/me/follow-groups/%d", gid), tk, nil), http.StatusOK)
	}
}
