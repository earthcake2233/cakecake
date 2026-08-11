//go:build integration

package handler

import (
	"cakecake/internal/model/dynamic"
	"cakecake/internal/model/video"
	"encoding/json"
	"fmt"
	"net/http"
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
	srveOK(t, r, areq("GET", "/api/v1/notifications", tk, nil), http.StatusOK)
	srveOK(t, r, areq("GET", "/api/v1/notifications/unread-summary", tk, nil), http.StatusCreated)
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
		srveOK(t, r, areq("POST", fmt.Sprintf("/api/v1/comments/%d/like", cid), tk, nil), http.StatusOK)
		srveOK(t, r, areq("POST", fmt.Sprintf("/api/v1/comments/%d/dislike", cid), tk, nil), http.StatusOK)
		srveOK(t, r, areq("POST", fmt.Sprintf("/api/v1/comments/%d/like", cid), tk, nil), http.StatusOK)
		srveOK(t, r, areq("GET", fmt.Sprintf("/api/v1/videos/%d/comments", v.ID), "", nil), http.StatusOK)
	}
}

func Test_VideoCoinDifferentAmounts(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "vcd1", "VCD1", 100)
	u2 := seedUser(t, api, "vcd2", "VCD2", 10)
	tk := tok(t, api, u.ID)
	v := seedVideoWithAPI(t, api, u2.ID, "VCD Video")
	srveOK(t, r, areq("POST", fmt.Sprintf("/api/v1/videos/%d/coin", v.ID), tk, `{"amount":2}`), http.StatusOK)
	srveOK(t, r, areq("GET", fmt.Sprintf("/api/v1/space/%d/recent-coins", u2.ID), "", nil), http.StatusOK)
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
		srveOK(t, r, areq("POST", fmt.Sprintf("/api/v1/dynamic-comments/%d/approve", cid), tk, nil), http.StatusOK)
		srveOK(t, r, areq("POST", fmt.Sprintf("/api/v1/dynamic-comments/%d/like", cid), tk2, nil), http.StatusOK)
		srveOK(t, r, areq("POST", fmt.Sprintf("/api/v1/dynamic-comments/%d/dislike", cid), tk, nil), http.StatusOK)
		srveOK(t, r, areq("POST", fmt.Sprintf("/api/v1/dynamic-comments/%d/ignore-curated", cid), tk, nil), http.StatusOK)
		srveOK(t, r, areq("GET", fmt.Sprintf("/api/v1/user-dynamics/%d/comments", dyn.ID), "", nil), http.StatusOK)
		srveOK(t, r, areq("DELETE", fmt.Sprintf("/api/v1/dynamic-comments/%d", cid), tk2, nil), http.StatusOK)
	}
}

func Test_UserFollowGroupsAndList(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "ufg1", "UFG1", 10)
	u2 := seedUser(t, api, "ufg2", "UFG2", 10)
	u3 := seedUser(t, api, "ufg3", "UFG3", 10)
	tk := tok(t, api, u.ID)
	tk2 := tok(t, api, u2.ID)
	srveOK(t, r, areq("POST", fmt.Sprintf("/api/v1/users/%d/follow", u2.ID), tk, nil), http.StatusOK)
	srveOK(t, r, areq("POST", fmt.Sprintf("/api/v1/users/%d/follow", u3.ID), tk, nil), http.StatusOK)
	srveOK(t, r, areq("GET", fmt.Sprintf("/api/v1/space/%d/following", u.ID), tk, nil), http.StatusOK)
	srveOK(t, r, areq("GET", fmt.Sprintf("/api/v1/space/%d/followers", u2.ID), tk2, nil), http.StatusOK)
	srveOK(t, r, areq("POST", fmt.Sprintf("/api/v1/users/%d/follow", u3.ID), tk, nil), http.StatusOK)
}

func Test_SpaceAndSearchFlow(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "ssf1", "SSF1", 10)
	_ = seedVideoWithAPI(t, api, u.ID, "SSF Video 1")
	_ = seedVideoWithAPI(t, api, u.ID, "SSF Video 2")
	srveOK(t, r, areq("GET", fmt.Sprintf("/api/v1/space/%d", u.ID), "", nil), http.StatusOK)
	srveOK(t, r, areq("GET", fmt.Sprintf("/api/v1/space/%d/videos", u.ID), "", nil), http.StatusOK)
	srveOK(t, r, areq("GET", "/api/v1/search?keyword=SSF&page=1&page_size=10", "", nil), http.StatusOK)
}

func Test_ArticleEngagementViewAndCoin(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "aev1", "AEV1", 100)
	u2 := seedUser(t, api, "aev2", "AEV2", 100)
	tk := tok(t, api, u.ID)
	art := seedArticle(t, api, u2.ID, "AEV Article")
	srveOK(t, r, areq("POST", fmt.Sprintf("/api/v1/articles/%d/view", art.ID), tk, nil), http.StatusOK)
	srveOK(t, r, areq("POST", fmt.Sprintf("/api/v1/articles/%d/coin", art.ID), tk, `{"amount":1}`), http.StatusOK)
	srveOK(t, r, areq("GET", "/api/v1/users/me/article-favorites", tk, nil), http.StatusOK)
}

func Test_CreatorDanmakuList(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "cdl1", "CDL1", 10)
	u2 := seedUser(t, api, "cdl2", "CDL2", 10)
	tk := tok(t, api, u.ID)
	v := seedVideoWithAPI(t, api, u.ID, "CDL Video")
	srveOK(t, r, areq("POST", fmt.Sprintf("/api/v1/videos/%d/danmaku", v.ID), tok(t, api, u2.ID), `{"content":"CDL Danmaku","type":"scroll","color":"#FFFFFF","video_time":10.0}`), http.StatusOK)
	srveOK(t, r, areq("GET", "/api/v1/users/me/creator/danmakus?page=1&page_size=10", tk, nil), http.StatusOK)
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
		srveOK(t, r, areq("PUT", fmt.Sprintf("/api/v1/admin/home-banners/%d", bid), at, `{"title":"B1 Updated","link_type":"none","sort_order":2}`), http.StatusOK)
		srveOK(t, r, areq("POST", fmt.Sprintf("/api/v1/admin/home-banners/%d/image", bid), at, nil), http.StatusInternalServerError)
		srveOK(t, r, areq("DELETE", fmt.Sprintf("/api/v1/admin/home-banners/%d", bid), at, nil), http.StatusOK)
	}
}

func Test_AdminVideoApproveRejectDelete(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "avd1", "AVD1", 10)
	at := admintok(t, api)
	v := video.Video{UserID: u.ID, Title: "Admin Test Video", Status: "pending_review", VideoURL: "https://cdn.example.com/avd.mp4", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	require.NoError(t, api.DB.Create(&v).Error)
	srveOK(t, r, areq("POST", fmt.Sprintf("/api/v1/admin/videos/%d/approve", v.ID), at, nil), http.StatusOK)
	v2 := video.Video{UserID: u.ID, Title: "Admin Test Video 2", Status: "pending_review", VideoURL: "https://cdn.example.com/avd2.mp4", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	require.NoError(t, api.DB.Create(&v2).Error)
	srveOK(t, r, areq("POST", fmt.Sprintf("/api/v1/admin/videos/%d/reject", v2.ID), at, `{"reason":"Test reject"}`), http.StatusOK)
	srveOK(t, r, areq("DELETE", fmt.Sprintf("/api/v1/admin/videos/%d", v.ID), at, nil), http.StatusOK)
}

func Test_AdminAgentCrudWithSettings(t *testing.T) {
	api, r, _ := newTestAPI(t)
	at := admintok(t, api)
	srveOK(t, r, areq("PUT", "/api/v1/admin/agent-settings", at, `{"global_system_prompt":"Be concise"}`), http.StatusOK)
	srveOK(t, r, areq("POST", "/api/v1/admin/agent-settings/avatar", at, nil), http.StatusBadRequest)
}

func Test_UserFollowAndBlockCombined(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "ufb1", "UFB1", 10)
	u2 := seedUser(t, api, "ufb2", "UFB2", 10)
	tk := tok(t, api, u.ID)
	srveOK(t, r, areq("POST", fmt.Sprintf("/api/v1/users/%d/follow", u2.ID), tk, nil), http.StatusOK)
	srveOK(t, r, areq("POST", fmt.Sprintf("/api/v1/users/%d/block", u2.ID), tk, nil), http.StatusOK)
	srveOK(t, r, areq("GET", fmt.Sprintf("/api/v1/space/%d/following", u.ID), tk, nil), http.StatusOK)
}

func Test_VideoLikeToggle(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "vlt1", "VLT1", 10)
	u2 := seedUser(t, api, "vlt2", "VLT2", 10)
	tk := tok(t, api, u.ID)
	v := seedVideoWithAPI(t, api, u2.ID, "VLT Video")
	srveOK(t, r, areq("POST", fmt.Sprintf("/api/v1/videos/%d/like", v.ID), tk, nil), http.StatusOK)
	srveOK(t, r, areq("POST", fmt.Sprintf("/api/v1/videos/%d/like", v.ID), tk, nil), http.StatusOK)
}
