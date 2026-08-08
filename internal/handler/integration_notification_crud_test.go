//go:build integration

// auto-generated test file
package handler

import (
	"cakecake/internal/model/article"
	"cakecake/internal/model/dynamic"
	"cakecake/internal/model/notification"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func seedNotification(t *testing.T, api *API, recipientID uint64, notifType string, relatedID uint64) uint64 {
	t.Helper()
	payload := `{"like_subject":"comment","article_id":0,"article_title":"","cover_url":""}`
	n := notification.Notification{
		RecipientID:     recipientID,
		Type:            notifType,
		RelatedID:       relatedID,
		SenderNamesJSON: `["tester"]`,
		TotalLikes:      5,
		CommentPreview:  "Nice!",
		PayloadJSON:     payload,
		IsRead:          false,
		CreatedAt:       time.Now(),
	}
	require.NoError(t, api.DB.Create(&n).Error)
	return n.ID
}

func Test_NotificationReadMuteDelete(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "nrm1", "NRM1", 10)
	tk := tok(t, api, u.ID)
	nid := seedNotification(t, api, u.ID, "reply", 0)
	srve(r, areq("PATCH", fmt.Sprintf("/api/v1/notifications/%d/read", nid), tk, nil))
	srve(r, areq("POST", fmt.Sprintf("/api/v1/notifications/%d/mute-likes", nid), tk, nil))
	srve(r, areq("POST", fmt.Sprintf("/api/v1/notifications/%d/comment-like", nid), tk, nil))
	srve(r, areq("POST", fmt.Sprintf("/api/v1/notifications/%d/comment-reply", nid), tk, `{"content":"Thanks!"}`))
	srve(r, areq("DELETE", fmt.Sprintf("/api/v1/notifications/%d", nid), tk, nil))
}

func Test_NotificationBatchRead(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "nbr1", "NBR1", 10)
	tk := tok(t, api, u.ID)
	n1 := seedNotification(t, api, u.ID, "reply", 0)
	n2 := seedNotification(t, api, u.ID, "like_aggregation", 0)
	srve(r, areq("PATCH", "/api/v1/notifications/read-batch", tk, fmt.Sprintf(`{"ids":[%d,%d]}`, n1, n2)))
	srve(r, areq("PATCH", "/api/v1/notifications/read-by-category", tk, `{"type":"reply"}`))
	srve(r, areq("GET", fmt.Sprintf("/api/v1/notifications/%d/like-likers", n2), tk, nil))
}

func Test_NotificationLikeAggregation(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "nla1", "NLA1", 10)
	u2 := seedUser(t, api, "nla2", "NLA2", 10)
	v := seedVideoWithAPI(t, api, u2.ID, "Notif Like Agg Video")
	body := fmt.Sprintf(`{"content":"Comment for like agg test","video_id":%d}`, v.ID)
	srve(r, areq("POST", "/api/v1/videos/"+fmt.Sprint(v.ID)+"/comments", tok(t, api, u2.ID), body))
	body2 := fmt.Sprintf(`{"content":"Second comment for like agg","video_id":%d}`, v.ID)
	w := srve(r, areq("POST", "/api/v1/videos/"+fmt.Sprint(v.ID)+"/comments", tok(t, api, u2.ID), body2))
	var cr struct {
		Code int `json:"code"`
		Data struct {
			ID uint64 `json:"id"`
		}
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &cr))
	if cr.Code == 0 && cr.Data.ID > 0 {
		srve(r, areq("POST", fmt.Sprintf("/api/v1/comments/%d/like", cr.Data.ID), tok(t, api, u.ID), nil))
	}
	srve(r, areq("GET", "/api/v1/notifications", tok(t, api, u.ID), nil))
	srve(r, areq("GET", "/api/v1/notifications/unread-summary", tok(t, api, u.ID), nil))
}

func Test_CommentApproveIgnoreDelete(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "cai1", "CAI1", 10)
	u2 := seedUser(t, api, "cai2", "CAI2", 10)
	tk := tok(t, api, u.ID)
	v := seedVideoWithAPI(t, api, u.ID, "CAI Video")
	w := srve(r, areq("POST", "/api/v1/videos/"+fmt.Sprint(v.ID)+"/comments", tok(t, api, u2.ID), `{"content":"Needs approval"}`))
	var cr struct {
		Code int `json:"code"`
		Data struct {
			ID uint64 `json:"id"`
		}
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &cr))
	if cr.Code == 0 && cr.Data.ID > 0 {
		cid := cr.Data.ID
		srve(r, areq("POST", fmt.Sprintf("/api/v1/comments/%d/approve", cid), tk, nil))
		srve(r, areq("POST", fmt.Sprintf("/api/v1/comments/%d/like", cid), tok(t, api, u2.ID), nil))
		srve(r, areq("POST", fmt.Sprintf("/api/v1/comments/%d/dislike", cid), tk, nil))
		srve(r, areq("POST", fmt.Sprintf("/api/v1/comments/%d/pin", cid), tk, nil))
		srve(r, areq("DELETE", fmt.Sprintf("/api/v1/comments/%d", cid), tok(t, api, u2.ID), nil))
	}
}

func Test_CommentIgnoreCurated(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "cic1", "CIC1", 10)
	u2 := seedUser(t, api, "cic2", "CIC2", 10)
	tk := tok(t, api, u.ID)
	v := seedVideoWithAPI(t, api, u.ID, "CIC Video")
	w := srve(r, areq("POST", "/api/v1/videos/"+fmt.Sprint(v.ID)+"/comments", tok(t, api, u2.ID), `{"content":"Ignore me"}`))
	var cr struct {
		Code int `json:"code"`
		Data struct {
			ID uint64 `json:"id"`
		}
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &cr))
	if cr.Code == 0 && cr.Data.ID > 0 {
		srve(r, areq("POST", fmt.Sprintf("/api/v1/comments/%d/ignore-curated", cr.Data.ID), tk, nil))
	}
}

func Test_ArticleCommentIgnoreDelete(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "acd1", "ACD1", 10)
	u2 := seedUser(t, api, "acd2", "ACD2", 10)
	tk := tok(t, api, u.ID)
	art := seedArticle(t, api, u.ID, "ACD Article")
	w := srve(r, areq("POST", fmt.Sprintf("/api/v1/articles/%d/comments", art.ID), tok(t, api, u2.ID), `{"content":"Article comment test"}`))
	var acr struct {
		Code int `json:"code"`
		Data struct {
			ID uint64 `json:"id"`
		}
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &acr))
	if acr.Code == 0 && acr.Data.ID > 0 {
		cid := acr.Data.ID
		srve(r, areq("POST", fmt.Sprintf("/api/v1/article-comments/%d/approve", cid), tk, nil))
		srve(r, areq("POST", fmt.Sprintf("/api/v1/article-comments/%d/ignore-curated", cid), tk, nil))
		srve(r, areq("POST", fmt.Sprintf("/api/v1/article-comments/%d/like", cid), tk, nil))
		srve(r, areq("POST", fmt.Sprintf("/api/v1/article-comments/%d/dislike", cid), tk, nil))
		srve(r, areq("POST", fmt.Sprintf("/api/v1/article-comments/%d/pin", cid), tk, nil))
		srve(r, areq("DELETE", fmt.Sprintf("/api/v1/article-comments/%d", cid), tk, nil))
	}
}

func Test_VideoFavoritePickerAndFolderOps(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "vfp1", "VFP1", 10)
	u2 := seedUser(t, api, "vfp2", "VFP2", 10)
	tk := tok(t, api, u.ID)
	v := seedVideoWithAPI(t, api, u2.ID, "VFP Video")
	w1 := srve(r, areq("POST", "/api/v1/users/me/favorite-folders", tk, `{"title":"Folder A"}`))
	w2 := srve(r, areq("POST", "/api/v1/users/me/favorite-folders", tk, `{"title":"Folder B"}`))
	var f1 struct {
		Code int `json:"code"`
		Data struct {
			ID uint64 `json:"id"`
		}
	}
	var f2 struct {
		Code int `json:"code"`
		Data struct {
			ID uint64 `json:"id"`
		}
	}
	require.NoError(t, json.Unmarshal(w1.Body.Bytes(), &f1))
	require.NoError(t, json.Unmarshal(w2.Body.Bytes(), &f2))
	if f1.Code == 0 && f1.Data.ID > 0 && f2.Code == 0 && f2.Data.ID > 0 {
		srve(r, areq("POST", fmt.Sprintf("/api/v1/videos/%d/favorite-folders/%d", v.ID, f1.Data.ID), tk, nil))
		srve(r, areq("GET", fmt.Sprintf("/api/v1/videos/%d/favorite-picker", v.ID), tk, nil))
		srve(r, areq("PUT", fmt.Sprintf("/api/v1/videos/%d/favorite-folders", v.ID), tk, fmt.Sprintf(`{"folder_ids":[%d,%d]}`, f1.Data.ID, f2.Data.ID)))
		srve(r, areq("PUT", fmt.Sprintf("/api/v1/videos/%d/favorite-folders/move", v.ID), tk, fmt.Sprintf(`{"folder_ids":[%d]}`, f2.Data.ID)))
	}
}

func Test_VideoPlaybackAndCover(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "vpb1", "VPB1", 10)
	u2 := seedUser(t, api, "vpb2", "VPB2", 10)
	tk := tok(t, api, u.ID)
	v := seedVideoWithAPI(t, api, u2.ID, "VPB Video")
	srve(r, areq("PATCH", fmt.Sprintf("/api/v1/videos/%d/playback", v.ID), tk, `{"current_time":15.5}`))
	srve(r, areq("PUT", fmt.Sprintf("/api/v1/videos/%d/cover", v.ID), tk, nil))
}

func Test_ArticlePlaybackAndCover(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "apc1", "APC1", 10)
	tk := tok(t, api, u.ID)
	art := seedArticle(t, api, u.ID, "APC Article")
	srve(r, areq("PATCH", fmt.Sprintf("/api/v1/users/me/articles/%d/playback", art.ID), tk, `{"current_time":20.0}`))
	srve(r, areq("PUT", fmt.Sprintf("/api/v1/users/me/articles/%d/cover", art.ID), tk, nil))
}

func Test_DmConversationSettingsAndReset(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "dcs1", "DCS1", 10)
	u2 := seedUser(t, api, "dcs2", "DCS2", 10)
	tk := tok(t, api, u.ID)
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
		srve(r, areq("PATCH", fmt.Sprintf("/api/v1/dm/conversations/%d/settings", cid), tk, `{"is_agent":false}`))
		srve(r, areq("POST", fmt.Sprintf("/api/v1/dm/conversations/%d/reset", cid), tk, nil))
	}
}

func Test_CreatorCommentsWithApproval(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "ccw1", "CCW1", 10)
	u2 := seedUser(t, api, "ccw2", "CCW2", 10)
	tk := tok(t, api, u.ID)
	v := seedVideoWithAPI(t, api, u.ID, "CCW Video")
	w := srve(r, areq("POST", "/api/v1/videos/"+fmt.Sprint(v.ID)+"/comments", tok(t, api, u2.ID), `{"content":"Creator comment test"}`))
	var cr struct {
		Code int `json:"code"`
		Data struct {
			ID uint64 `json:"id"`
		}
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &cr))
	if cr.Code == 0 && cr.Data.ID > 0 {
		srve(r, areq("POST", fmt.Sprintf("/api/v1/comments/%d/approve", cr.Data.ID), tk, nil))
		srve(r, areq("GET", "/api/v1/users/me/creator/comments?page=1&page_size=10&status=approved", tk, nil))
		srve(r, areq("GET", "/api/v1/users/me/creator/comments?page=1&page_size=10&status=pending", tk, nil))
	}
}

func Test_UserDynamicLikeAndView(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "udl3", "UDL3", 10)
	u2 := seedUser(t, api, "udl4", "UDL4", 10)
	tk2 := tok(t, api, u2.ID)
	dyn := dynamic.UserDynamic{UserID: u.ID, Title: "Dynamic Like Test", Content: "Test content", ImagesJSON: "[]", CreatedAt: time.Now()}
	require.NoError(t, api.DB.Create(&dyn).Error)
	srve(r, areq("POST", fmt.Sprintf("/api/v1/user-dynamics/%d/like", dyn.ID), tk2, nil))
	srve(r, areq("GET", fmt.Sprintf("/api/v1/user-dynamics/%d", dyn.ID), "", nil))
	srve(r, areq("GET", fmt.Sprintf("/api/v1/space/%d/dynamics", u.ID), "", nil))
}

func Test_FavoriteFolderBatchRemove(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "fbr1", "FBR1", 10)
	u2 := seedUser(t, api, "fbr2", "FBR2", 10)
	tk := tok(t, api, u.ID)
	v := seedVideoWithAPI(t, api, u2.ID, "FBR Video")
	w := srve(r, areq("POST", "/api/v1/users/me/favorite-folders", tk, `{"title":"Batch Folder"}`))
	var fr struct {
		Code int `json:"code"`
		Data struct {
			ID uint64 `json:"id"`
		}
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &fr))
	if fr.Code == 0 && fr.Data.ID > 0 {
		fid := fr.Data.ID
		srve(r, areq("POST", fmt.Sprintf("/api/v1/videos/%d/favorite-folders/%d", v.ID, fid), tk, nil))
		srve(r, areq("POST", fmt.Sprintf("/api/v1/users/me/favorite-folders/%d/batch-remove", fid), tk, fmt.Sprintf(`{"video_ids":[%d]}`, v.ID)))
	}
}

func Test_AdminHotSearchFullDashboard(t *testing.T) {
	api, r, _ := newTestAPI(t)
	at := admintok(t, api)
	srve(r, areq("GET", "/api/v1/admin/hot-search/ops", at, nil))
	srve(r, areq("GET", "/api/v1/admin/hot-search/dashboard", at, nil))
	srve(r, areq("GET", "/api/v1/admin/hot-search/preview", at, nil))
	srve(r, areq("POST", "/api/v1/admin/hot-search/ops", at, `{"keyword":"test","group":"general","score":50}`))
	srve(r, areq("POST", "/api/v1/admin/hot-search/quick-op", at, `{"keyword":"quick","score":80}`))
	srve(r, areq("POST", "/api/v1/admin/hot-search/reorder", at, `{"ids":[]}`))
	srve(r, areq("POST", "/api/v1/admin/hot-search/display-order/reset", at, nil))
	srve(r, areq("POST", "/api/v1/admin/hot-search/redis/remove", at, `{"keyword":"test"}`))
	srve(r, areq("POST", "/api/v1/admin/hot-search/redis/boost", at, `{"keyword":"test","score":200}`))
}

func Test_AdminAgentSettingsAndAvatar(t *testing.T) {
	api, r, _ := newTestAPI(t)
	at := admintok(t, api)
	srve(r, areq("GET", "/api/v1/admin/agent-settings", at, nil))
	srve(r, areq("PUT", "/api/v1/admin/agent-settings", at, `{"model":"gpt-4o","system_prompt":"You are a helpful assistant.","max_tokens":1024}`))
	srve(r, areq("POST", "/api/v1/admin/agent-settings/avatar", at, nil))
	srve(r, areq("GET", "/api/v1/admin/agent-profiles", at, nil))
	w := srve(r, areq("POST", "/api/v1/admin/agent-profiles", at, `{"slug":"support-bot","display_name":"Support Bot","welcome_message":"How can I help?"}`))
	var apr struct {
		Code int `json:"code"`
		Data struct {
			ID uint64 `json:"id"`
		}
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &apr))
	if apr.Code == 0 && apr.Data.ID > 0 {
		pid := apr.Data.ID
		srve(r, areq("PUT", fmt.Sprintf("/api/v1/admin/agent-profiles/%d", pid), at, `{"display_name":"Updated Support Bot"}`))
		srve(r, areq("DELETE", fmt.Sprintf("/api/v1/admin/agent-profiles/%d", pid), at, nil))
	}
}

func Test_AgentDmEndpoints(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "adm1", "ADM1", 10)
	u2 := seedUser(t, api, "adm2", "ADM2", 10)
	tk := tok(t, api, u.ID)
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
		srve(r, areq("PATCH", fmt.Sprintf("/api/v1/dm/conversations/%d/settings", cid), tk, `{"is_agent":true}`))
		srve(r, areq("POST", fmt.Sprintf("/api/v1/dm/conversations/%d/messages", cid), tk, `{"content":"Hello agent"}`))
	}
}

func Test_UserMeProfileEndpoints(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "ump1", "UMP1", 10)
	tk := tok(t, api, u.ID)
	srve(r, areq("GET", "/api/v1/users/me", tk, nil))
	srve(r, areq("PUT", "/api/v1/users/me", tk, `{"nickname":"UpdatedName","bio":"Hello world"}`))
	srve(r, areq("POST", "/api/v1/users/me/change-password", tk, `{"old_password":"hash","new_password":"newhash123"}`))
}

func Test_UserBlockEndpointsMore(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "ubl3", "UBL3", 10)
	u2 := seedUser(t, api, "ubl4", "UBL4", 10)
	tk := tok(t, api, u.ID)
	srve(r, areq("POST", "/api/v1/users/me/block", tk, fmt.Sprintf(`{"user_id":%d}`, u2.ID)))
	srve(r, areq("GET", "/api/v1/users/me/blocked", tk, nil))
	srve(r, areq("POST", "/api/v1/users/me/unblock", tk, fmt.Sprintf(`{"user_id":%d}`, u2.ID)))
}

func Test_SearchHistoryEndpoints(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "shp1", "SHP1", 10)
	tk := tok(t, api, u.ID)
	srve(r, areq("PUT", "/api/v1/users/me/search-history", tk, `{"content":"test search"}`))
	srve(r, areq("POST", "/api/v1/users/me/search-history", tk, `{"content":"another search"}`))
	srve(r, areq("GET", "/api/v1/users/me/search-history", tk, nil))
}

func Test_DanmakuDeleteAndLike(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "ddl1", "DDL1", 10)
	u2 := seedUser(t, api, "ddl2", "DDL2", 10)
	tk := tok(t, api, u.ID)
	v := seedVideoWithAPI(t, api, u2.ID, "DDL Video")
	w := srve(r, areq("POST", fmt.Sprintf("/api/v1/videos/%d/danmaku", v.ID), tk, `{"content":"DDL Danmaku","type":0,"color":16777215,"progress":5.0}`))
	var dmr struct {
		Code int `json:"code"`
		Data struct {
			ID uint64 `json:"id"`
		}
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &dmr))
	if dmr.Code == 0 && dmr.Data.ID > 0 {
		did := dmr.Data.ID
		srve(r, areq("POST", fmt.Sprintf("/api/v1/danmakus/%d/like", did), tk, nil))
		srve(r, areq("DELETE", fmt.Sprintf("/api/v1/danmakus/%d", did), tk, nil))
	}
}

func Test_VideoDraftSourceAndList(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "vds1", "VDS1", 10)
	tk := tok(t, api, u.ID)
	srve(r, areq("GET", "/api/v1/users/me/video-drafts?page=1&page_size=10", tk, nil))
	srve(r, areq("GET", "/api/v1/users/me/videos/99999/draft-source", tk, nil))
}

func Test_AuthRefreshAndEdgeCases(t *testing.T) {
	api, r, _ := newTestAPI(t)
	_ = seedUser(t, api, "are1", "ARE1", 10)
	srve(r, areq("POST", "/api/v1/auth/login", "", `{"username":"are1","password":"hash"}`))
	srve(r, areq("POST", "/api/v1/users", "", `{"username":"are1","password":"hash","nickname":"Duplicate"}`))
}

func Test_AdminAuthMe(t *testing.T) {
	api, r, _ := newTestAPI(t)
	at := admintok(t, api)
	srve(r, areq("GET", "/api/v1/admin/me", at, nil))
}

func Test_ArticleFullCRUD(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "afc1", "AFC1", 10)
	tk := tok(t, api, u.ID)
	w := srve(r, areq("POST", "/api/v1/articles", tk, `{"title":"CRUD Article","body_md":"# Content","tags":["go","test"],"status":"draft"}`))
	var ar struct {
		Code int `json:"code"`
		Data struct {
			ID uint64 `json:"id"`
		}
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &ar))
	if ar.Code == 0 && ar.Data.ID > 0 {
		aid := ar.Data.ID
		srve(r, areq("GET", fmt.Sprintf("/api/v1/users/me/articles/%d", aid), tk, nil))
		srve(r, areq("PUT", fmt.Sprintf("/api/v1/users/me/articles/%d", aid), tk, `{"title":"Updated CRUD","body_md":"# Updated"}`))
		srve(r, areq("DELETE", fmt.Sprintf("/api/v1/users/me/articles/%d", aid), tk, nil))
	}
	srve(r, areq("GET", "/api/v1/users/me/articles?page=1&page_size=10", tk, nil))
}

func Test_AdminArticleRejectApproveDelete(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "aar1", "AAR1", 10)
	at := admintok(t, api)
	art := article.Article{UserID: u.ID, Title: "Admin Pending Article", BodyMD: "# Content", Status: "pending_review", CreatedAt: time.Now()}
	require.NoError(t, api.DB.Create(&art).Error)
	srve(r, areq("POST", fmt.Sprintf("/api/v1/admin/articles/%d/approve", art.ID), at, nil))
	art2 := article.Article{UserID: u.ID, Title: "Admin Pending Article 2", BodyMD: "# Content", Status: "pending_review", CreatedAt: time.Now()}
	require.NoError(t, api.DB.Create(&art2).Error)
	srve(r, areq("POST", fmt.Sprintf("/api/v1/admin/articles/%d/reject", art2.ID), at, `{"reason":"Not good enough"}`))
	art3 := article.Article{UserID: u.ID, Title: "Admin Pending Article 3", BodyMD: "# Content", Status: "pending_review", CreatedAt: time.Now()}
	require.NoError(t, api.DB.Create(&art3).Error)
	srve(r, areq("DELETE", fmt.Sprintf("/api/v1/admin/articles/%d", art3.ID), at, nil))
}

func Test_AdminDynamicGetDelete(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "adg1", "ADG1", 10)
	at := admintok(t, api)
	dyn := dynamic.UserDynamic{UserID: u.ID, Title: "Admin Dynamic Test", Content: "Content", ImagesJSON: "[]", CreatedAt: time.Now()}
	require.NoError(t, api.DB.Create(&dyn).Error)
	srve(r, areq("GET", fmt.Sprintf("/api/v1/admin/dynamics/%d", dyn.ID), at, nil))
	srve(r, areq("DELETE", fmt.Sprintf("/api/v1/admin/dynamics/%d", dyn.ID), at, nil))
}

func Test_UserAvatarUpload(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "uav1", "UAV1", 10)
	tk := tok(t, api, u.ID)
	srve(r, areq("POST", "/api/v1/users/me/avatar", tk, nil))
}

func Test_UserBindThirdPartyEndpoints(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "ubt1", "UBT1", 10)
	tk := tok(t, api, u.ID)
	srve(r, areq("GET", "/api/v1/users/me/bindings", tk, nil))
}

func Test_ViewHistoryEndpoints(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "vhe1", "VHE1", 10)
	u2 := seedUser(t, api, "vhe2", "VHE2", 10)
	tk := tok(t, api, u.ID)
	v := seedVideoWithAPI(t, api, u2.ID, "VHE Video")
	srve(r, areq("POST", fmt.Sprintf("/api/v1/videos/%d/view", v.ID), tk, `{"current_time":5.0,"duration":120.0}`))
	srve(r, areq("GET", "/api/v1/users/me/view-history?page=1&page_size=10", tk, nil))
	srve(r, areq("DELETE", fmt.Sprintf("/api/v1/users/me/view-history/videos/%d", v.ID), tk, nil))
	srve(r, areq("DELETE", "/api/v1/users/me/view-history/articles/99999", tk, nil))
	srve(r, areq("DELETE", "/api/v1/users/me/view-history", tk, nil))
}

func Test_CoinLedgerEndpoints(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "cle1", "CLE1", 100)
	u2 := seedUser(t, api, "cle2", "CLE2", 10)
	tk := tok(t, api, u.ID)
	v := seedVideoWithAPI(t, api, u2.ID, "CLE Video")
	srve(r, areq("POST", fmt.Sprintf("/api/v1/videos/%d/coin", v.ID), tk, `{"amount":1}`))
	srve(r, areq("GET", "/api/v1/users/me/coin-balance", tk, nil))
	srve(r, areq("GET", "/api/v1/users/me/coin-ledger?page=1&page_size=10", tk, nil))
}

func Test_AdminSystemConfig(t *testing.T) {
	api, r, _ := newTestAPI(t)
	at := admintok(t, api)
	srve(r, areq("GET", "/api/v1/admin/system-config", at, nil))
	srve(r, areq("PUT", "/api/v1/admin/system-config", at, `{"site_name":"MiniBili Test","site_description":"Test"}`))
}

func Test_HomeBannerAdminCRUD(t *testing.T) {
	api, r, _ := newTestAPI(t)
	at := admintok(t, api)
	srve(r, areq("GET", "/api/v1/admin/home-banners", at, nil))
	w := srve(r, areq("POST", "/api/v1/admin/home-banners", at, `{"title":"Test Banner","link_type":"none","sort_order":1}`))
	var br struct {
		Code int `json:"code"`
		Data struct {
			ID uint64 `json:"id"`
		}
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &br))
	if br.Code == 0 && br.Data.ID > 0 {
		bid := br.Data.ID
		srve(r, areq("PUT", fmt.Sprintf("/api/v1/admin/home-banners/%d", bid), at, `{"title":"Updated Banner","link_type":"none","sort_order":2}`))
		srve(r, areq("DELETE", fmt.Sprintf("/api/v1/admin/home-banners/%d", bid), at, nil))
	}
}

func Test_UserDailyReward(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "udr1", "UDR1", 10)
	tk := tok(t, api, u.ID)
	srve(r, areq("POST", "/api/v1/users/me/daily-reward/claim", tk, nil))
	srve(r, areq("GET", "/api/v1/users/me/daily-reward/status", tk, nil))
}

func Test_SensitiveUGCEndpoints(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "sug1", "SUG1", 10)
	tk := tok(t, api, u.ID)
	srve(r, areq("POST", "/api/v1/sensitive/check", tk, `{"text":"hello world"}`))
}

func Test_SearchAllEndpoint(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "sae1", "SAE1", 10)
	tk := tok(t, api, u.ID)
	srve(r, areq("GET", "/api/v1/search?q=test&page=1&page_size=10", "", nil))
	srve(r, areq("GET", "/api/v1/search/suggest?keyword=test", tk, nil))
}
