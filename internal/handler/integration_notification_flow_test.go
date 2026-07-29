//go:build integration

package handler

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"minibili/internal/model"
)

func Test_NotificationFullFlow_SeedAndList(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "nff1", "NFF1", 10)
	tk := tok(t, api, u.ID)
	for i := 0; i < 5; i++ { seedNotification(t, api, u.ID, "reply", uint64(i+100)) }
	for i := 0; i < 3; i++ {
		n := model.Notification{
			RecipientID: u.ID,
			Type: "like_aggregation",
			RelatedID: uint64(i + 200),
			SenderNamesJSON: `["user_a","user_b","user_c"]`,
			TotalLikes: 3 + i,
			CommentPreview: "Nice!",
			PayloadJSON: `{"like_subject":"comment"}`,
			IsRead: false,
			CreatedAt: time.Now(),
		}
		require.NoError(t, api.DB.Create(&n).Error)
	}
	srve(r, areq("GET", "/api/v1/notifications", tk, nil))
	srve(r, areq("GET", "/api/v1/notifications/unread-summary", tk, nil))
}

func Test_NotificationReadAndMute(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "nrm2", "NRM2", 10)
	tk := tok(t, api, u.ID)
	nid := seedNotification(t, api, u.ID, "reply", 0)
	srve(r, areq("PATCH", fmt.Sprintf("/api/v1/notifications/%d/read", nid), tk, nil))
	srve(r, areq("POST", fmt.Sprintf("/api/v1/notifications/%d/mute-likes", nid), tk, nil))
	srve(r, areq("PATCH", "/api/v1/notifications/read-batch", tk, fmt.Sprintf(`{"ids":[%d]}`, nid)))
	srve(r, areq("PATCH", "/api/v1/notifications/read-by-category", tk, `{"type":"reply"}`))
	srve(r, areq("DELETE", fmt.Sprintf("/api/v1/notifications/%d", nid), tk, nil))
}

func Test_NotificationCommentReplyAndLike(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "ncr1", "NCR1", 10)
	tk := tok(t, api, u.ID)
	n := model.Notification{
		RecipientID: u.ID,
		Type: "reply",
		RelatedID: 0,
		SenderNamesJSON: `["someone"]`,
		PayloadJSON: `{"reply_body":"Check this out","video_id":0}`,
		IsRead: false,
		CreatedAt: time.Now(),
	}
	require.NoError(t, api.DB.Create(&n).Error)
	srve(r, areq("POST", fmt.Sprintf("/api/v1/notifications/%d/comment-like", n.ID), tk, nil))
	srve(r, areq("POST", fmt.Sprintf("/api/v1/notifications/%d/comment-reply", n.ID), tk, `{"content":"Thanks!"}`))
}

func Test_NotificationLikeLikers(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "nll1", "NLL1", 10)
	tk := tok(t, api, u.ID)
	n := model.Notification{
		RecipientID: u.ID,
		Type: "like_aggregation",
		RelatedID: 0,
		SenderNamesJSON: `["liker_a","liker_b"]`,
		TotalLikes: 2,
		CommentPreview: "Cool!",
		PayloadJSON: `{"like_subject":"comment"}`,
		IsRead: false,
		CreatedAt: time.Now(),
	}
	require.NoError(t, api.DB.Create(&n).Error)
	srve(r, areq("GET", fmt.Sprintf("/api/v1/notifications/%d/like-likers", n.ID), tk, nil))
}

func Test_DmConversationMultipleMessages(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "dmm1", "DMM1", 10)
	u2 := seedUser(t, api, "dmm2", "DMM2", 10)
	tk := tok(t, api, u.ID)
	tk2 := tok(t, api, u2.ID)
	w := srve(r, areq("POST", "/api/v1/dm/conversations", tk, fmt.Sprintf(`{"user_id":%d}`, u2.ID)))
	var dcr struct { Code int `json:"code"`; Data struct { ID uint64 `json:"id"`} }
	json.Unmarshal(w.Body.Bytes(), &dcr)
	if dcr.Code == 0 && dcr.Data.ID > 0 {
		cid := dcr.Data.ID
		for i := 0; i < 3; i++ {
			srve(r, areq("POST", fmt.Sprintf("/api/v1/dm/conversations/%d/messages", cid), tk, fmt.Sprintf(`{"content":"U1 msg %d"}`, i)))
			srve(r, areq("POST", fmt.Sprintf("/api/v1/dm/conversations/%d/messages", cid), tk2, fmt.Sprintf(`{"content":"U2 reply %d"}`, i)))
		}
		srve(r, areq("GET", "/api/v1/dm/conversations", tk, nil))
		srve(r, areq("GET", fmt.Sprintf("/api/v1/dm/conversations/%d/messages", cid), tk, nil))
		srve(r, areq("PATCH", fmt.Sprintf("/api/v1/dm/conversations/%d/settings", cid), tk, `{"is_agent":true}`))
		srve(r, areq("POST", fmt.Sprintf("/api/v1/dm/conversations/%d/reset", cid), tk, nil))
		srve(r, areq("DELETE", fmt.Sprintf("/api/v1/dm/conversations/%d", cid), tk, nil))
	}
}

func Test_UserDynamicPostUpdateDelete(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "udp1", "UDP1", 10)
	tk := tok(t, api, u.ID)
	w := srve(r, areq("POST", "/api/v1/users/me/dynamics", tk, `{"title":"Test Dynamic","content":"Test content"}`))
	var dr struct { Code int `json:"code"`; Data struct { ID uint64 `json:"id"`} }
	json.Unmarshal(w.Body.Bytes(), &dr)
	if dr.Code == 0 && dr.Data.ID > 0 {
		did := dr.Data.ID
		srve(r, areq("PUT", fmt.Sprintf("/api/v1/users/me/dynamics/%d", did), tk, `{"title":"Updated","content":"Updated"}`))
		srve(r, areq("PATCH", fmt.Sprintf("/api/v1/users/me/dynamics/%d/playback", did), tk, `{"current_time":25.0}`))
		srve(r, areq("GET", "/api/v1/users/me/dynamics?page=1&page_size=10", tk, nil))
		srve(r, areq("DELETE", fmt.Sprintf("/api/v1/users/me/dynamics/%d", did), tk, nil))
	}
}

func Test_VideoEngagementFullFolderFlow(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "vef2", "VEF2", 10)
	u2 := seedUser(t, api, "vef3", "VEF3", 10)
	tk := tok(t, api, u.ID)
	v := seedVideoWithAPI(t, api, u2.ID, "VEF2 Video")
	w := srve(r, areq("POST", "/api/v1/users/me/favorite-folders", tk, `{"title":"VEF Folder"}`))
	var fr struct { Code int `json:"code"`; Data struct { ID uint64 `json:"id"`} }
	json.Unmarshal(w.Body.Bytes(), &fr)
	if fr.Code == 0 && fr.Data.ID > 0 {
		fid := fr.Data.ID
		srve(r, areq("POST", fmt.Sprintf("/api/v1/videos/%d/favorite", v.ID), tk, nil))
		srve(r, areq("POST", fmt.Sprintf("/api/v1/videos/%d/favorite-folders/%d", v.ID, fid), tk, nil))
		srve(r, areq("GET", fmt.Sprintf("/api/v1/videos/%d/favorite-picker", v.ID), tk, nil))
		srve(r, areq("PUT", fmt.Sprintf("/api/v1/videos/%d/favorite-folders", v.ID), tk, fmt.Sprintf(`{"folder_ids":[%d]}`, fid)))
		srve(r, areq("PUT", fmt.Sprintf("/api/v1/videos/%d/favorite-folders/move", v.ID), tk, fmt.Sprintf(`{"folder_ids":[%d]}`, fid)))
		srve(r, areq("DELETE", fmt.Sprintf("/api/v1/videos/%d/favorite-folders/%d", v.ID, fid), tk, nil))
		srve(r, areq("POST", fmt.Sprintf("/api/v1/videos/%d/favorite", v.ID), tk, nil))
	}
}

func Test_VideoDraftListAndSource(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "vdu1", "VDU1", 10)
	tk := tok(t, api, u.ID)
	srve(r, areq("GET", "/api/v1/users/me/video-drafts?page=1&page_size=10", tk, nil))
	srve(r, areq("GET", "/api/v1/users/me/videos/99999/draft-source", tk, nil))
}

func Test_ViewHistoryRecordAndClear(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "vhr1", "VHR1", 10)
	u2 := seedUser(t, api, "vhr2", "VHR2", 10)
	tk := tok(t, api, u.ID)
	v := seedVideoWithAPI(t, api, u2.ID, "VHR Video")
	srve(r, areq("POST", fmt.Sprintf("/api/v1/videos/%d/view", v.ID), tk, `{"current_time":10.0,"duration":200.0}`))
	srve(r, areq("GET", "/api/v1/users/me/view-history?page=1&page_size=10", tk, nil))
	srve(r, areq("DELETE", fmt.Sprintf("/api/v1/users/me/view-history/videos/%d", v.ID), tk, nil))
	srve(r, areq("DELETE", "/api/v1/users/me/view-history", tk, nil))
}

func Test_ArticleCommentReplyFlow(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "acf1", "ACF1", 10)
	u2 := seedUser(t, api, "acf2", "ACF2", 10)
	tk := tok(t, api, u.ID)
	art := seedArticle(t, api, u.ID, "ACF Article")
	w := srve(r, areq("POST", fmt.Sprintf("/api/v1/articles/%d/comments", art.ID), tok(t, api, u2.ID), `{"content":"Parent article comment"}`))
	var acr struct { Code int `json:"code"`; Data struct { ID uint64 `json:"id"`} }
	json.Unmarshal(w.Body.Bytes(), &acr)
	if acr.Code == 0 && acr.Data.ID > 0 {
		pcid := acr.Data.ID
		srve(r, areq("POST", fmt.Sprintf("/api/v1/articles/%d/comments", art.ID), tk, fmt.Sprintf(`{"content":"Owner reply","parent_id":%d}`, pcid)))
		srve(r, areq("POST", fmt.Sprintf("/api/v1/article-comments/%d/approve", pcid), tk, nil))
		srve(r, areq("POST", fmt.Sprintf("/api/v1/article-comments/%d/like", pcid), tk, nil))
		srve(r, areq("POST", fmt.Sprintf("/api/v1/article-comments/%d/pin", pcid), tk, nil))
		srve(r, areq("GET", fmt.Sprintf("/api/v1/articles/%d/comments", art.ID), "", nil))
	}
}

func Test_VideoUploadAndUpdate(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "vuu1", "VUU1", 10)
	tk := tok(t, api, u.ID)
	srve(r, areq("POST", "/api/v1/videos", tk, `{"title":"Test Upload","description":"Test desc","status":"draft"}`))
	srve(r, areq("GET", "/api/v1/users/me/videos?page=1&page_size=10", tk, nil))
}

func Test_AdminHotSearchAllOps(t *testing.T) {
	api, r, _ := newTestAPI(t)
	at := admintok(t, api)
	w := srve(r, areq("POST", "/api/v1/admin/hot-search/ops", at, `{"keyword":"summer","group":"seasonal","score":80}`))
	var hopr struct { Code int `json:"code"`; Data struct { ID uint64 `json:"id"`} }
	json.Unmarshal(w.Body.Bytes(), &hopr)
	if hopr.Code == 0 && hopr.Data.ID > 0 {
		srve(r, areq("PUT", fmt.Sprintf("/api/v1/admin/hot-search/ops/%d", hopr.Data.ID), at, `{"keyword":"summer_upd","score":90}`))
		srve(r, areq("DELETE", fmt.Sprintf("/api/v1/admin/hot-search/ops/%d", hopr.Data.ID), at, nil))
	}
	srve(r, areq("GET", "/api/v1/admin/hot-search/ops", at, nil))
	srve(r, areq("GET", "/api/v1/admin/hot-search/dashboard", at, nil))
	srve(r, areq("GET", "/api/v1/admin/hot-search/preview", at, nil))
	srve(r, areq("POST", "/api/v1/admin/hot-search/reorder", at, `{"ids":[]}`))
	srve(r, areq("POST", "/api/v1/admin/hot-search/display-order/reset", at, nil))
	srve(r, areq("POST", "/api/v1/admin/hot-search/redis/remove", at, `{"keyword":"test"}`))
	srve(r, areq("POST", "/api/v1/admin/hot-search/redis/boost", at, `{"keyword":"boosted","score":300}`))
}

func Test_AdminVideoFullCRUD(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "avc1", "AVC1", 10)
	at := admintok(t, api)
	v := model.Video{UserID: u.ID, Title: "Admin CRUD Video", Status: "pending_review", VideoURL: "https://example.com/avc.mp4", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	require.NoError(t, api.DB.Create(&v).Error)
	srve(r, areq("GET", "/api/v1/admin/videos?page=1&page_size=10", at, nil))
	srve(r, areq("GET", fmt.Sprintf("/api/v1/admin/videos/%d", v.ID), at, nil))
	srve(r, areq("POST", fmt.Sprintf("/api/v1/admin/videos/%d/approve", v.ID), at, nil))
	srve(r, areq("POST", fmt.Sprintf("/api/v1/admin/videos/%d/reject", v.ID), at, `{"reason":"Test"}`))
	srve(r, areq("DELETE", fmt.Sprintf("/api/v1/admin/videos/%d", v.ID), at, nil))
}

func Test_FollowGroupMemberManagement(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "fgm2", "FGM2a", 10)
	u2 := seedUser(t, api, "fgm3", "FGM3", 10)
	tk := tok(t, api, u.ID)
	srve(r, areq("POST", "/api/v1/users/me/follow", tk, fmt.Sprintf(`{"user_id":%d}`, u2.ID)))
	w := srve(r, areq("POST", "/api/v1/users/me/follow-groups", tk, `{"name":"Special Friends"}`))
	var fgr struct { Code int `json:"code"`; Data struct { ID uint64 `json:"id"`} }
	json.Unmarshal(w.Body.Bytes(), &fgr)
	if fgr.Code == 0 && fgr.Data.ID > 0 {
		gid := fgr.Data.ID
		srve(r, areq("POST", fmt.Sprintf("/api/v1/users/me/follow-groups/%d/members", gid), tk, fmt.Sprintf(`{"user_id":%d}`, u2.ID)))
		srve(r, areq("GET", fmt.Sprintf("/api/v1/users/me/follow-groups/%d/members", gid), tk, nil))
		srve(r, areq("DELETE", fmt.Sprintf("/api/v1/users/me/follow-groups/%d/members/%d", gid, u2.ID), tk, nil))
		srve(r, areq("PUT", fmt.Sprintf("/api/v1/users/me/follow-groups/%d/rename", gid), tk, `{"name":"Best Friends"}`))
		srve(r, areq("DELETE", fmt.Sprintf("/api/v1/users/me/follow-groups/%d", gid), tk, nil))
	}
}