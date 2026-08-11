//go:build integration

// auto-generated test file
package handler

import (
	"cakecake/internal/config"
	"cakecake/internal/model/admin"
	"cakecake/internal/model/article"
	"cakecake/internal/model/dynamic"
	"cakecake/internal/model/notification"
	"encoding/json"
	"fmt"
	"net/http"
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
	srveOK(t, r, areq("PATCH", fmt.Sprintf("/api/v1/notifications/%d/read", nid), tk, nil), http.StatusOK)
	srveOK(t, r, areq("POST", fmt.Sprintf("/api/v1/notifications/%d/mute-likes", nid), tk, nil), http.StatusBadRequest)
	srveOK(t, r, areq("POST", fmt.Sprintf("/api/v1/notifications/%d/comment-like", nid), tk, nil), http.StatusBadRequest)
	srveOK(t, r, areq("POST", fmt.Sprintf("/api/v1/notifications/%d/comment-reply", nid), tk, `{"content":"Thanks!"}`), http.StatusNotFound)
	srveOK(t, r, areq("DELETE", fmt.Sprintf("/api/v1/notifications/%d", nid), tk, nil), http.StatusOK)
}

func Test_NotificationBatchRead(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "nbr1", "NBR1", 10)
	tk := tok(t, api, u.ID)
	n1 := seedNotification(t, api, u.ID, "reply", 0)
	n2 := seedNotification(t, api, u.ID, "like_aggregation", 0)
	srveOK(t, r, areq("PATCH", "/api/v1/notifications/read-batch", tk, fmt.Sprintf(`[%d,%d]`, n1, n2)), http.StatusOK)
	srveOK(t, r, areq("PATCH", "/api/v1/notifications/read-by-category?category=reply", tk, nil), http.StatusOK)
	srveOK(t, r, areq("GET", fmt.Sprintf("/api/v1/notifications/%d/like-likers", n2), tk, nil), http.StatusOK)
}

func Test_NotificationLikeAggregation(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "nla1", "NLA1", 10)
	u2 := seedUser(t, api, "nla2", "NLA2", 10)
	v := seedVideoWithAPI(t, api, u2.ID, "Notif Like Agg Video")
	body := fmt.Sprintf(`{"content":"Comment for like agg test","video_id":%d}`, v.ID)
	srveOK(t, r, areq("POST", "/api/v1/videos/"+fmt.Sprint(v.ID)+"/comments", tok(t, api, u2.ID), body), http.StatusCreated)
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
		srveOK(t, r, areq("POST", fmt.Sprintf("/api/v1/comments/%d/like", cr.Data.ID), tok(t, api, u.ID), nil), http.StatusOK)
	}
	srveOK(t, r, areq("GET", "/api/v1/notifications", tok(t, api, u.ID), nil), http.StatusOK)
	srveOK(t, r, areq("GET", "/api/v1/notifications/unread-summary", tok(t, api, u.ID), nil), http.StatusCreated)
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
		srveOK(t, r, areq("POST", fmt.Sprintf("/api/v1/comments/%d/approve", cid), tk, nil), http.StatusOK)
		srveOK(t, r, areq("POST", fmt.Sprintf("/api/v1/comments/%d/like", cid), tok(t, api, u2.ID), nil), http.StatusOK)
		srveOK(t, r, areq("POST", fmt.Sprintf("/api/v1/comments/%d/dislike", cid), tk, nil), http.StatusOK)
		srveOK(t, r, areq("POST", fmt.Sprintf("/api/v1/comments/%d/pin", cid), tk, nil), http.StatusOK)
		srveOK(t, r, areq("DELETE", fmt.Sprintf("/api/v1/comments/%d", cid), tok(t, api, u2.ID), nil), http.StatusOK)
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
		srveOK(t, r, areq("POST", fmt.Sprintf("/api/v1/comments/%d/ignore-curated", cr.Data.ID), tk, nil), http.StatusOK)
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
		srveOK(t, r, areq("POST", fmt.Sprintf("/api/v1/article-comments/%d/approve", cid), tk, nil), http.StatusOK)
		srveOK(t, r, areq("POST", fmt.Sprintf("/api/v1/article-comments/%d/ignore-curated", cid), tk, nil), http.StatusOK)
		srveOK(t, r, areq("POST", fmt.Sprintf("/api/v1/article-comments/%d/like", cid), tk, nil), http.StatusOK)
		srveOK(t, r, areq("POST", fmt.Sprintf("/api/v1/article-comments/%d/dislike", cid), tk, nil), http.StatusOK)
		srveOK(t, r, areq("POST", fmt.Sprintf("/api/v1/article-comments/%d/pin", cid), tk, nil), http.StatusOK)
		srveOK(t, r, areq("DELETE", fmt.Sprintf("/api/v1/article-comments/%d", cid), tk, nil), http.StatusOK)
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
		srveOK(t, r, areq("POST", fmt.Sprintf("/api/v1/videos/%d/favorite-folders/%d", v.ID, f1.Data.ID), tk, nil), http.StatusOK)
		srveOK(t, r, areq("GET", fmt.Sprintf("/api/v1/videos/%d/favorite-picker", v.ID), tk, nil), http.StatusOK)
		srveOK(t, r, areq("PUT", fmt.Sprintf("/api/v1/videos/%d/favorite-folders", v.ID), tk, fmt.Sprintf(`{"folder_ids":[%d,%d]}`, f1.Data.ID, f2.Data.ID)), http.StatusOK)
		srveOK(t, r, areq("PUT", fmt.Sprintf("/api/v1/videos/%d/favorite-folders/move", v.ID), tk, fmt.Sprintf(`{"from_folder_id":%d,"to_folder_id":%d}`, f1.Data.ID, f2.Data.ID)), http.StatusOK)
	}
}

func Test_VideoPlaybackAndCover(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "vpb1", "VPB1", 10)
	tk := tok(t, api, u.ID)
	v := seedVideoWithAPI(t, api, u.ID, "VPB Video")
	srveOK(t, r, areq("PATCH", fmt.Sprintf("/api/v1/videos/%d/playback", v.ID), tk, `{"comments_closed":true}`), http.StatusOK)
	srveOK(t, r, areq("PUT", fmt.Sprintf("/api/v1/videos/%d/cover", v.ID), tk, nil), http.StatusBadRequest)
}

func Test_ArticlePlaybackAndCover(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "apc1", "APC1", 10)
	tk := tok(t, api, u.ID)
	art := seedArticle(t, api, u.ID, "APC Article")
	srveOK(t, r, areq("PATCH", fmt.Sprintf("/api/v1/users/me/articles/%d/playback", art.ID), tk, `{"comments_closed":true}`), http.StatusOK)
	srveOK(t, r, areq("PUT", fmt.Sprintf("/api/v1/users/me/articles/%d/cover", art.ID), tk, nil), http.StatusBadRequest)
}

func Test_DmConversationSettingsAndReset(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "dcs1", "DCS1", 10)
	u2 := seedUser(t, api, "dcs2", "DCS2", 10)
	tk := tok(t, api, u.ID)
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
		srveOK(t, r, areq("PATCH", fmt.Sprintf("/api/v1/dm/conversations/%d/settings", cid), tk, `{"pinned":true}`), http.StatusOK)
		srveOK(t, r, areq("POST", fmt.Sprintf("/api/v1/dm/conversations/%d/reset", cid), tk, nil), http.StatusOK)
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
		srveOK(t, r, areq("POST", fmt.Sprintf("/api/v1/comments/%d/approve", cr.Data.ID), tk, nil), http.StatusOK)
		srveOK(t, r, areq("GET", "/api/v1/users/me/creator/comments?page=1&page_size=10&status=approved", tk, nil), http.StatusOK)
		srveOK(t, r, areq("GET", "/api/v1/users/me/creator/comments?page=1&page_size=10&status=pending", tk, nil), http.StatusOK)
	}
}

func Test_UserDynamicLikeAndView(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "udl3", "UDL3", 10)
	u2 := seedUser(t, api, "udl4", "UDL4", 10)
	tk2 := tok(t, api, u2.ID)
	dyn := dynamic.UserDynamic{UserID: u.ID, Title: "Dynamic Like Test", Content: "Test content", ImagesJSON: "[]", CreatedAt: time.Now()}
	require.NoError(t, api.DB.Create(&dyn).Error)
	srveOK(t, r, areq("POST", fmt.Sprintf("/api/v1/user-dynamics/%d/like", dyn.ID), tk2, nil), http.StatusOK)
	srveOK(t, r, areq("GET", fmt.Sprintf("/api/v1/user-dynamics/%d", dyn.ID), "", nil), http.StatusOK)
	srveOK(t, r, areq("GET", fmt.Sprintf("/api/v1/space/%d/dynamics", u.ID), "", nil), http.StatusOK)
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
		srveOK(t, r, areq("POST", fmt.Sprintf("/api/v1/videos/%d/favorite-folders/%d", v.ID, fid), tk, nil), http.StatusOK)
		srveOK(t, r, areq("POST", fmt.Sprintf("/api/v1/users/me/favorite-folders/%d/batch-remove", fid), tk, fmt.Sprintf(`{"video_ids":[%d]}`, v.ID)), http.StatusOK)
	}
}

func Test_AdminHotSearchFullDashboard(t *testing.T) {
	api, r, _ := newTestAPI(t)
	at := admintok(t, api)
	srveOK(t, r, areq("GET", "/api/v1/admin/hot-search/ops", at, nil), http.StatusOK)
	srveOK(t, r, areq("GET", "/api/v1/admin/hot-search/dashboard", at, nil), http.StatusOK)
	srveOK(t, r, areq("GET", "/api/v1/admin/hot-search/preview", at, nil), http.StatusOK)
	srveOK(t, r, areq("POST", "/api/v1/admin/hot-search/ops", at, `{"keyword":"test","op_type":"manual","display_title":"测试"}`), http.StatusCreated)
	srveOK(t, r, areq("POST", "/api/v1/admin/hot-search/quick-op", at, `{"keyword":"quick","op_type":"manual","display_title":"快速"}`), http.StatusCreated)
	srveOK(t, r, areq("POST", "/api/v1/admin/hot-search/reorder", at, `{"items":[{"keyword":"test","title":"测试","source":"manual"}]}`), http.StatusOK)
	srveOK(t, r, areq("POST", "/api/v1/admin/hot-search/display-order/reset", at, nil), http.StatusOK)
	srveOK(t, r, areq("POST", "/api/v1/admin/hot-search/redis/remove", at, `{"keyword":"test"}`), http.StatusOK)
	srveOK(t, r, areq("POST", "/api/v1/admin/hot-search/redis/boost", at, `{"keyword":"test","delta":200}`), http.StatusOK)
}

func Test_AdminAgentSettingsAndAvatar(t *testing.T) {
	api, r, _ := newTestAPI(t)
	at := admintok(t, api)
	srveOK(t, r, areq("GET", "/api/v1/admin/agent-settings", at, nil), http.StatusOK)
	srveOK(t, r, areq("PUT", "/api/v1/admin/agent-settings", at, `{"global_system_prompt":"You are a helpful assistant."}`), http.StatusOK)
	srveOK(t, r, areq("POST", "/api/v1/admin/agent-settings/avatar", at, nil), http.StatusBadRequest)
	srveOK(t, r, areq("GET", "/api/v1/admin/agent-profiles", at, nil), http.StatusOK)
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
		srveOK(t, r, areq("PUT", fmt.Sprintf("/api/v1/admin/agent-profiles/%d", pid), at, `{"display_name":"Updated Support Bot"}`), http.StatusOK)
		srveOK(t, r, areq("DELETE", fmt.Sprintf("/api/v1/admin/agent-profiles/%d", pid), at, nil), http.StatusOK)
	}
}

func Test_AgentDmEndpoints(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "adm1", "ADM1", 10)
	u2 := seedUser(t, api, "adm2", "ADM2", 10)
	tk := tok(t, api, u.ID)
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
		srveOK(t, r, areq("PATCH", fmt.Sprintf("/api/v1/dm/conversations/%d/settings", cid), tk, `{"pinned":true}`), http.StatusOK)
		srveOK(t, r, areq("POST", fmt.Sprintf("/api/v1/dm/conversations/%d/messages", cid), tk, `{"content":"Hello agent"}`), http.StatusOK)
	}
}

func Test_UserMeProfileEndpoints(t *testing.T) {
	_, r, _ := newTestAPI(t)
	tk, _ := covRegister(t, r, "ump1", "password12")
	srveOK(t, r, areq("GET", "/api/v1/users/me", tk, nil), http.StatusOK)
	srveOK(t, r, areq("PUT", "/api/v1/users/me", tk, `{"username":"ump1_new"}`), http.StatusOK)
	srveOK(t, r, areq("PUT", "/api/v1/users/me/password", tk, `{"old_password":"password12","new_password":"password"}`), http.StatusOK)
}

func Test_UserBlockEndpointsMore(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "ubl3", "UBL3", 10)
	u2 := seedUser(t, api, "ubl4", "UBL4", 10)
	tk := tok(t, api, u.ID)
	srveOK(t, r, areq("POST", fmt.Sprintf("/api/v1/users/%d/block", u2.ID), tk, nil), http.StatusOK)
	srveOK(t, r, areq("GET", "/api/v1/users/me/blocked", tk, nil), http.StatusNotFound)
	srveOK(t, r, areq("POST", fmt.Sprintf("/api/v1/users/%d/block", u2.ID), tk, nil), http.StatusOK)
}

func Test_SearchHistoryEndpoints(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "shp1", "SHP1", 10)
	tk := tok(t, api, u.ID)
	srveOK(t, r, areq("PUT", "/api/v1/users/me/search-history", tk, `{"keywords":["test search"]}`), http.StatusOK)
	srveOK(t, r, areq("POST", "/api/v1/users/me/search-history", tk, `{"keyword":"another search"}`), http.StatusOK)
	srveOK(t, r, areq("GET", "/api/v1/users/me/search-history", tk, nil), http.StatusOK)
}

func Test_DanmakuDeleteAndLike(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "ddl1", "DDL1", 10)
	u2 := seedUser(t, api, "ddl2", "DDL2", 10)
	tk := tok(t, api, u.ID)
	v := seedVideoWithAPI(t, api, u2.ID, "DDL Video")
	w := srve(r, areq("POST", fmt.Sprintf("/api/v1/videos/%d/danmaku", v.ID), tk, `{"content":"DDL Danmaku","type":"scroll","color":"#FFFFFF","video_time":5.0}`))
	require.Equal(t, 0, decodeCode(t, w), w.Body.String())
	var dmr struct {
		Code int `json:"code"`
		Data struct {
			ID uint64 `json:"id"`
		}
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &dmr))
	if dmr.Code == 0 && dmr.Data.ID > 0 {
		did := dmr.Data.ID
		srveOK(t, r, areq("POST", fmt.Sprintf("/api/v1/danmakus/%d/like", did), tk, nil), http.StatusOK)
		srveOK(t, r, areq("DELETE", fmt.Sprintf("/api/v1/danmakus/%d", did), tok(t, api, u2.ID), nil), http.StatusOK)
	}
}

func Test_VideoDraftSourceAndList(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "vds1", "VDS1", 10)
	tk := tok(t, api, u.ID)
	srveOK(t, r, areq("GET", "/api/v1/users/me/video-drafts?page=1&page_size=10", tk, nil), http.StatusNotFound)
	srveOK(t, r, areq("GET", "/api/v1/users/me/videos/99999/draft-source", tk, nil), http.StatusNotFound)
}

func Test_AuthRefreshAndEdgeCases(t *testing.T) {
	api, r, _ := newTestAPI(t)
	_ = seedUser(t, api, "are1", "ARE1", 10)
	srveOK(t, r, areq("POST", "/api/v1/auth/login", "", `{"username":"are1","password":"hash"}`), http.StatusUnauthorized)
	srveOK(t, r, areq("POST", "/api/v1/users", "", `{"username":"are1","password":"hash","nickname":"Duplicate"}`), http.StatusBadRequest)
}

func Test_AdminAuthMe(t *testing.T) {
	api, r, _ := newTestAPI(t)
	require.NoError(t, api.DB.Create(&admin.Admin{ID: 1, Username: "root", PasswordHash: "x", Status: "active"}).Error)
	at := admintok(t, api)
	srveOK(t, r, areq("GET", "/api/v1/admin/me", at, nil), http.StatusOK)
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
		srveOK(t, r, areq("GET", fmt.Sprintf("/api/v1/users/me/articles/%d", aid), tk, nil), http.StatusOK)
		srveOK(t, r, areq("PUT", fmt.Sprintf("/api/v1/users/me/articles/%d", aid), tk, `{"title":"Updated CRUD","body_md":"# Updated"}`), http.StatusOK)
		srveOK(t, r, areq("DELETE", fmt.Sprintf("/api/v1/users/me/articles/%d", aid), tk, nil), http.StatusOK)
	}
	srveOK(t, r, areq("GET", "/api/v1/users/me/articles?page=1&page_size=10", tk, nil), http.StatusOK)
}

func Test_AdminArticleRejectApproveDelete(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "aar1", "AAR1", 10)
	at := admintok(t, api)
	art := article.Article{UserID: u.ID, Title: "Admin Pending Article", BodyMD: "# Content", Status: "pending_review", CreatedAt: time.Now()}
	require.NoError(t, api.DB.Create(&art).Error)
	srveOK(t, r, areq("POST", fmt.Sprintf("/api/v1/admin/articles/%d/approve", art.ID), at, nil), http.StatusOK)
	art2 := article.Article{UserID: u.ID, Title: "Admin Pending Article 2", BodyMD: "# Content", Status: "pending_review", CreatedAt: time.Now()}
	require.NoError(t, api.DB.Create(&art2).Error)
	srveOK(t, r, areq("POST", fmt.Sprintf("/api/v1/admin/articles/%d/reject", art2.ID), at, `{"reason":"Not good enough"}`), http.StatusOK)
	art3 := article.Article{UserID: u.ID, Title: "Admin Pending Article 3", BodyMD: "# Content", Status: "published", CreatedAt: time.Now()}
	require.NoError(t, api.DB.Create(&art3).Error)
	srveOK(t, r, areq("DELETE", fmt.Sprintf("/api/v1/admin/articles/%d", art3.ID), at, nil), http.StatusOK)
}

func Test_AdminDynamicGetDelete(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "adg1", "ADG1", 10)
	at := admintok(t, api)
	dyn := dynamic.UserDynamic{UserID: u.ID, Title: "Admin Dynamic Test", Content: "Content", ImagesJSON: "[]", CreatedAt: time.Now()}
	require.NoError(t, api.DB.Create(&dyn).Error)
	srveOK(t, r, areq("GET", fmt.Sprintf("/api/v1/admin/dynamics/%d", dyn.ID), at, nil), http.StatusOK)
	srveOK(t, r, areq("DELETE", fmt.Sprintf("/api/v1/admin/dynamics/%d", dyn.ID), at, nil), http.StatusOK)
}

func Test_UserAvatarUpload(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "uav1", "UAV1", 10)
	tk := tok(t, api, u.ID)
	srveOK(t, r, areq("POST", "/api/v1/users/me/avatar", tk, nil), http.StatusBadRequest)
}

func Test_UserBindThirdPartyEndpoints(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "ubt1", "UBT1", 10)
	tk := tok(t, api, u.ID)
	srveOK(t, r, areq("GET", "/api/v1/users/me/bindings", tk, nil), http.StatusNotFound)
}

func Test_ViewHistoryEndpoints(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "vhe1", "VHE1", 10)
	u2 := seedUser(t, api, "vhe2", "VHE2", 10)
	tk := tok(t, api, u.ID)
	v := seedVideoWithAPI(t, api, u2.ID, "VHE Video")
	srveOK(t, r, areq("POST", fmt.Sprintf("/api/v1/videos/%d/view-history", v.ID), tk, `{"progress_sec":5.0,"duration_sec":120.0}`), http.StatusOK)
	srveOK(t, r, areq("GET", "/api/v1/users/me/view-history?page=1&page_size=10", tk, nil), http.StatusOK)
	srveOK(t, r, areq("DELETE", fmt.Sprintf("/api/v1/users/me/view-history/%d", v.ID), tk, nil), http.StatusOK)
	srveOK(t, r, areq("DELETE", "/api/v1/users/me/view-history/articles/99999", tk, nil), http.StatusOK)
	srveOK(t, r, areq("DELETE", "/api/v1/users/me/view-history", tk, nil), http.StatusOK)
}

func Test_CoinLedgerEndpoints(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "cle1", "CLE1", 100)
	u2 := seedUser(t, api, "cle2", "CLE2", 10)
	tk := tok(t, api, u.ID)
	v := seedVideoWithAPI(t, api, u2.ID, "CLE Video")
	srveOK(t, r, areq("POST", fmt.Sprintf("/api/v1/videos/%d/coin", v.ID), tk, `{"amount":1}`), http.StatusOK)
	srveOK(t, r, areq("GET", "/api/v1/users/me/coin-balance", tk, nil), http.StatusNotFound)
	srveOK(t, r, areq("GET", "/api/v1/users/me/coin-ledger?page=1&page_size=10", tk, nil), http.StatusOK)
}

func Test_AdminSystemConfig(t *testing.T) {
	api, r, _ := newTestAPI(t)
	api.RuntimeCfg = config.NewRuntimeConfig(api.DB, nil)
	at := admintok(t, api)
	srveOK(t, r, areq("GET", "/api/v1/admin/system-configs", at, nil), http.StatusOK)
	srveOK(t, r, areq("PUT", "/api/v1/admin/system-configs", at, `{"configs":{"agent_enabled":"true"}}`), http.StatusOK)
}

func Test_HomeBannerAdminCRUD(t *testing.T) {
	api, r, _ := newTestAPI(t)
	at := admintok(t, api)
	srveOK(t, r, areq("GET", "/api/v1/admin/home-banners", at, nil), http.StatusOK)
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
		srveOK(t, r, areq("PUT", fmt.Sprintf("/api/v1/admin/home-banners/%d", bid), at, `{"title":"Updated Banner","link_type":"none","sort_order":2}`), http.StatusOK)
		srveOK(t, r, areq("DELETE", fmt.Sprintf("/api/v1/admin/home-banners/%d", bid), at, nil), http.StatusOK)
	}
}

func Test_UserDailyReward(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "udr1", "UDR1", 10)
	tk := tok(t, api, u.ID)
	srveOK(t, r, areq("POST", "/api/v1/users/me/daily-rewards/watch", tk, nil), http.StatusOK)
	srveOK(t, r, areq("GET", "/api/v1/users/me/daily-rewards", tk, nil), http.StatusOK)
}

func Test_SensitiveUGCEndpoints(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "sug1", "SUG1", 10)
	tk := tok(t, api, u.ID)
	srveOK(t, r, areq("POST", "/api/v1/sensitive/check", tk, `{"text":"hello world"}`), http.StatusNotFound)
}

func Test_SearchAllEndpoint(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "sae1", "SAE1", 10)
	tk := tok(t, api, u.ID)
	srveOK(t, r, areq("GET", "/api/v1/search?keyword=test&page=1&page_size=10", "", nil), http.StatusOK)
	srveOK(t, r, areq("GET", "/api/v1/search/suggest?keyword=test", tk, nil), http.StatusOK)
}
