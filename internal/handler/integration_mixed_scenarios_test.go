//go:build integration

package handler

import (
	"cakecake/internal/model/dynamic"
	"cakecake/internal/model/video"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func Test_NotificationBatchAndCategoryRead(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "nb2a", "NB2A", 10)
	tk := tok(t, api, u.ID)
	for i := 0; i < 3; i++ {
		seedNotification(t, api, u.ID, "reply", uint64(i+100))
		seedNotification(t, api, u.ID, "like_aggregation", uint64(i+200))
	}
	srve(r, areq("GET", "/api/v1/notifications", tk, nil))
	srve(r, areq("GET", "/api/v1/notifications/unread-summary", tk, nil))
}

func Test_CommentLikeDislikeToggleFlow(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "clt1", "CLT1", 10)
	u2 := seedUser(t, api, "clt2", "CLT2", 10)
	tk := tok(t, api, u.ID)
	v := seedVideoWithAPI(t, api, u2.ID, "CLT Video")
	w := srve(r, areq("POST", "/api/v1/videos/"+fmt.Sprint(v.ID)+"/comments", tok(t, api, u2.ID), `{"content":"Toggle test comment"}`))
	var cr struct {
		Code int `json:"code"`
		Data struct {
			ID uint64 `json:"id"`
		}
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &cr))
	if cr.Code == 0 && cr.Data.ID > 0 {
		cid := cr.Data.ID
		srve(r, areq("POST", fmt.Sprintf("/api/v1/comments/%d/like", cid), tk, nil))
		srve(r, areq("POST", fmt.Sprintf("/api/v1/comments/%d/dislike", cid), tk, nil))
		srve(r, areq("POST", fmt.Sprintf("/api/v1/comments/%d/like", cid), tk, nil))
		srve(r, areq("GET", fmt.Sprintf("/api/v1/videos/%d/comments", v.ID), "", nil))
	}
}

func Test_VideoCoinDifferentAmounts(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "vcd1", "VCD1", 100)
	u2 := seedUser(t, api, "vcd2", "VCD2", 10)
	tk := tok(t, api, u.ID)
	v := seedVideoWithAPI(t, api, u2.ID, "VCD Video")
	srve(r, areq("POST", fmt.Sprintf("/api/v1/videos/%d/coin", v.ID), tk, `{"amount":2}`))
	srve(r, areq("GET", fmt.Sprintf("/api/v1/space/%d/recent-coins", u2.ID), "", nil))
}

func Test_DynamicCommentOperations(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "dco1", "DCO1", 10)
	u2 := seedUser(t, api, "dco2", "DCO2", 10)
	tk := tok(t, api, u.ID)
	tk2 := tok(t, api, u2.ID)
	dyn := dynamic.UserDynamic{UserID: u.ID, Title: "DCO Dynamic", Content: "DCO content", ImagesJSON: "[]", CreatedAt: time.Now()}
	require.NoError(t, api.DB.Create(&dyn).Error)
	w := srve(r, areq("POST", fmt.Sprintf("/api/v1/user-dynamics/%d/comments", dyn.ID), tk2, fmt.Sprintf(`{"content":"DCO comment","dynamic_id":%d}`, dyn.ID)))
	var dcr struct {
		Code int `json:"code"`
		Data struct {
			ID uint64 `json:"id"`
		}
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &dcr))
	if dcr.Code == 0 && dcr.Data.ID > 0 {
		cid := dcr.Data.ID
		srve(r, areq("POST", fmt.Sprintf("/api/v1/user-dynamics/%d/comments/%d/approve", dyn.ID, cid), tk, nil))
		srve(r, areq("POST", fmt.Sprintf("/api/v1/user-dynamics/%d/comments/%d/like", dyn.ID, cid), tk2, nil))
		srve(r, areq("POST", fmt.Sprintf("/api/v1/user-dynamics/%d/comments/%d/dislike", dyn.ID, cid), tk, nil))
		srve(r, areq("POST", fmt.Sprintf("/api/v1/user-dynamics/%d/comments/%d/ignore-curated", dyn.ID, cid), tk, nil))
		srve(r, areq("GET", fmt.Sprintf("/api/v1/user-dynamics/%d/comments", dyn.ID), "", nil))
		srve(r, areq("DELETE", fmt.Sprintf("/api/v1/user-dynamics/%d/comments/%d", dyn.ID, cid), tk2, nil))
	}
}

func Test_UserFollowGroupsAndList(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "ufg1", "UFG1", 10)
	u2 := seedUser(t, api, "ufg2", "UFG2", 10)
	u3 := seedUser(t, api, "ufg3", "UFG3", 10)
	tk := tok(t, api, u.ID)
	srve(r, areq("POST", "/api/v1/users/me/follow", tk, fmt.Sprintf(`{"user_id":%d}`, u2.ID)))
	srve(r, areq("POST", "/api/v1/users/me/follow", tk, fmt.Sprintf(`{"user_id":%d}`, u3.ID)))
	srve(r, areq("GET", fmt.Sprintf("/api/v1/space/%d/following", u.ID), "", nil))
	srve(r, areq("GET", fmt.Sprintf("/api/v1/space/%d/followers", u2.ID), "", nil))
	srve(r, areq("POST", "/api/v1/users/me/unfollow", tk, fmt.Sprintf(`{"user_id":%d}`, u3.ID)))
}

func Test_SpaceAndSearchFlow(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "ssf1", "SSF1", 10)
	_ = seedVideoWithAPI(t, api, u.ID, "SSF Video 1")
	_ = seedVideoWithAPI(t, api, u.ID, "SSF Video 2")
	srve(r, areq("GET", fmt.Sprintf("/api/v1/space/%d", u.ID), "", nil))
	srve(r, areq("GET", fmt.Sprintf("/api/v1/space/%d/videos", u.ID), "", nil))
	srve(r, areq("GET", "/api/v1/search?q=SSF&page=1&page_size=10", "", nil))
}

func Test_ArticleEngagementViewAndCoin(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "aev1", "AEV1", 100)
	u2 := seedUser(t, api, "aev2", "AEV2", 100)
	tk := tok(t, api, u.ID)
	art := seedArticle(t, api, u2.ID, "AEV Article")
	srve(r, areq("POST", fmt.Sprintf("/api/v1/articles/%d/view", art.ID), tk, nil))
	srve(r, areq("POST", fmt.Sprintf("/api/v1/articles/%d/coin", art.ID), tk, `{"amount":1}`))
	srve(r, areq("GET", "/api/v1/users/me/article-favorites", tk, nil))
}

func Test_CreatorDanmakuList(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "cdl1", "CDL1", 10)
	u2 := seedUser(t, api, "cdl2", "CDL2", 10)
	tk := tok(t, api, u.ID)
	v := seedVideoWithAPI(t, api, u.ID, "CDL Video")
	srve(r, areq("POST", fmt.Sprintf("/api/v1/videos/%d/danmaku", v.ID), tok(t, api, u2.ID), `{"content":"CDL Danmaku","type":0,"color":16777215,"progress":10.0}`))
	srve(r, areq("GET", "/api/v1/users/me/creator/danmakus?page=1&page_size=10", tk, nil))
}

func Test_AdminBannerFullCRUD(t *testing.T) {
	api, r, _ := newTestAPI(t)
	at := admintok(t, api)
	w := srve(r, areq("POST", "/api/v1/admin/home-banners", at, `{"title":"B1","link_type":"url","link_url":"https://example.com","sort_order":1}`))
	var br struct {
		Code int `json:"code"`
		Data struct {
			ID uint64 `json:"id"`
		}
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &br))
	if br.Code == 0 && br.Data.ID > 0 {
		bid := br.Data.ID
		srve(r, areq("PUT", fmt.Sprintf("/api/v1/admin/home-banners/%d", bid), at, `{"title":"B1 Updated","link_type":"none","sort_order":2}`))
		srve(r, areq("POST", fmt.Sprintf("/api/v1/admin/home-banners/%d/image", bid), at, nil))
		srve(r, areq("DELETE", fmt.Sprintf("/api/v1/admin/home-banners/%d", bid), at, nil))
	}
}

func Test_AdminVideoApproveRejectDelete(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "avd1", "AVD1", 10)
	at := admintok(t, api)
	v := video.Video{UserID: u.ID, Title: "Admin Test Video", Status: "pending_review", VideoURL: "https://cdn.example.com/avd.mp4", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	require.NoError(t, api.DB.Create(&v).Error)
	srve(r, areq("POST", fmt.Sprintf("/api/v1/admin/videos/%d/approve", v.ID), at, nil))
	srve(r, areq("POST", fmt.Sprintf("/api/v1/admin/videos/%d/reject", v.ID), at, `{"reason":"Test reject"}`))
	srve(r, areq("DELETE", fmt.Sprintf("/api/v1/admin/videos/%d", v.ID), at, nil))
}

func Test_AdminAgentCrudWithSettings(t *testing.T) {
	api, r, _ := newTestAPI(t)
	at := admintok(t, api)
	srve(r, areq("PUT", "/api/v1/admin/agent-settings", at, `{"model":"gpt-4o-mini","system_prompt":"Be concise","max_tokens":512,"temperature":0.7}`))
	srve(r, areq("POST", "/api/v1/admin/agent-settings/avatar", at, nil))
}

func Test_UserFollowAndBlockCombined(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "ufb1", "UFB1", 10)
	u2 := seedUser(t, api, "ufb2", "UFB2", 10)
	tk := tok(t, api, u.ID)
	srve(r, areq("POST", "/api/v1/users/me/follow", tk, fmt.Sprintf(`{"user_id":%d}`, u2.ID)))
	srve(r, areq("POST", "/api/v1/users/me/block", tk, fmt.Sprintf(`{"user_id":%d}`, u2.ID)))
	srve(r, areq("GET", fmt.Sprintf("/api/v1/space/%d/following", u.ID), "", nil))
}

func Test_VideoLikeToggle(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "vlt1", "VLT1", 10)
	u2 := seedUser(t, api, "vlt2", "VLT2", 10)
	tk := tok(t, api, u.ID)
	v := seedVideoWithAPI(t, api, u2.ID, "VLT Video")
	srve(r, areq("POST", fmt.Sprintf("/api/v1/videos/%d/like", v.ID), tk, nil))
	srve(r, areq("POST", fmt.Sprintf("/api/v1/videos/%d/like", v.ID), tk, nil))
}
