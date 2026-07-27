package handler

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"minibili/internal/model"
)

func Test_ArticleEngagementFavoriteAndCoin(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "aef1", "AEF1", 100)
	u2 := seedUser(t, api, "aef2", "AEF2", 100)
	tk := tok(t, api, u.ID)
	art := seedArticle(t, api, u2.ID, "AEF Article")
	srve(r, areq("POST", fmt.Sprintf("/api/v1/articles/%d/favorite", art.ID), tk, nil))
	srve(r, areq("POST", fmt.Sprintf("/api/v1/articles/%d/coin", art.ID), tk, `{"amount":1}`))
	srve(r, areq("GET", fmt.Sprintf("/api/v1/space/%d/article-favorites", u2.ID), "", nil))
	srve(r, areq("GET", "/api/v1/users/me/article-favorites", tk, nil))
}

func Test_DynamicCommentFullFlow(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "dcf1", "DCF1", 10)
	u2 := seedUser(t, api, "dcf2", "DCF2", 10)
	tk := tok(t, api, u.ID)
	tk2 := tok(t, api, u2.ID)
	dyn := model.UserDynamic{UserID: u.ID, Title: "DCF Dynamic", Content: "Test content for comments", ImagesJSON: "[]", CreatedAt: time.Now()}
	require.NoError(t, api.DB.Create(&dyn).Error)
	body := fmt.Sprintf(`{"content":"Nice dynamic!","dynamic_id":%d}`, dyn.ID)
	w := srve(r, areq("POST", fmt.Sprintf("/api/v1/user-dynamics/%d/comments", dyn.ID), tk2, body))
	var dcr struct { Code int `json:"code"`; Data struct { ID uint64 `json:"id"`} `json:"data"` }
	json.Unmarshal(w.Body.Bytes(), &dcr)
	if dcr.Code == 0 && dcr.Data.ID > 0 {
		cid := dcr.Data.ID
		srve(r, areq("POST", fmt.Sprintf("/api/v1/user-dynamics/%d/comments/%d/approve", dyn.ID, cid), tk, nil))
		srve(r, areq("POST", fmt.Sprintf("/api/v1/user-dynamics/%d/comments/%d/like", dyn.ID, cid), tk2, nil))
		srve(r, areq("POST", fmt.Sprintf("/api/v1/user-dynamics/%d/comments/%d/dislike", dyn.ID, cid), tk, nil))
		srve(r, areq("DELETE", fmt.Sprintf("/api/v1/user-dynamics/%d/comments/%d", dyn.ID, cid), tk2, nil))
	}
	srve(r, areq("GET", fmt.Sprintf("/api/v1/user-dynamics/%d/comments", dyn.ID), "", nil))
}

func Test_UserDynamicUpdateAndList(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "udl1", "UDL1", 10)
	tk := tok(t, api, u.ID)
	u2 := seedUser(t, api, "udl2", "UDL2", 10)
	w := srve(r, areq("POST", "/api/v1/users/me/dynamics", tk, `{"title":"Update Test","content":"Original content"}`))
	var dr struct { Code int `json:"code"`; Data struct { ID uint64 `json:"id"`} `json:"data"` }
	json.Unmarshal(w.Body.Bytes(), &dr)
	if dr.Code == 0 && dr.Data.ID > 0 {
		did := dr.Data.ID
		srve(r, areq("PUT", fmt.Sprintf("/api/v1/users/me/dynamics/%d", did), tk, `{"title":"Updated Title","content":"Updated content"}`))
		srve(r, areq("PATCH", fmt.Sprintf("/api/v1/users/me/dynamics/%d/playback", did), tk, `{"current_time":30.0}`))
	}
	srve(r, areq("GET", fmt.Sprintf("/api/v1/space/%d/dynamics", u.ID), "", nil))
	dyn2 := model.UserDynamic{UserID: u2.ID, Title: "Other Dynamic", Content: "Other", ImagesJSON: "[]", CreatedAt: time.Now()}
	require.NoError(t, api.DB.Create(&dyn2).Error)
	srve(r, areq("GET", fmt.Sprintf("/api/v1/user-dynamics/%d", dyn2.ID), "", nil))
}

func Test_VideoEngagementFolderOps(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "vef1", "VEF1", 10)
	u2 := seedUser(t, api, "vef2", "VEF2", 10)
	tk := tok(t, api, u.ID)
	v := seedVideo(t, api, u2.ID, "VEF Video")
	w := srve(r, areq("POST", "/api/v1/users/me/favorite-folders", tk, `{"title":"VEF Folder"}`))
	var fr struct { Code int `json:"code"`; Data struct { ID uint64 `json:"id"`} `json:"data"` }
	json.Unmarshal(w.Body.Bytes(), &fr)
	if fr.Code == 0 && fr.Data.ID > 0 {
		fid := fr.Data.ID
		srve(r, areq("POST", fmt.Sprintf("/api/v1/videos/%d/favorite-folders/%d", v.ID, fid), tk, nil))
		srve(r, areq("PUT", fmt.Sprintf("/api/v1/videos/%d/favorite-folders", v.ID), tk, fmt.Sprintf(`{"folder_ids":[%d]}`, fid)))
		srve(r, areq("PUT", fmt.Sprintf("/api/v1/videos/%d/favorite-folders/move", v.ID), tk, fmt.Sprintf(`{"folder_ids":[%d]}`, fid)))
		srve(r, areq("DELETE", fmt.Sprintf("/api/v1/videos/%d/favorite-folders/%d", v.ID, fid), tk, nil))
	}
}

func Test_AdminAgentProfileCRUD(t *testing.T) {
	api, r, _ := newTestAPI(t)
	at := admintok(t, api)
	w := srve(r, areq("POST", "/api/v1/admin/agent-profiles", at, `{"slug":"test-bot","display_name":"Test Bot","welcome_message":"Hello"}`))
	var apr struct { Code int `json:"code"`; Data struct { ID uint64 `json:"id"`} `json:"data"` }
	json.Unmarshal(w.Body.Bytes(), &apr)
	if apr.Code == 0 && apr.Data.ID > 0 {
		pid := apr.Data.ID
		srve(r, areq("PUT", fmt.Sprintf("/api/v1/admin/agent-profiles/%d", pid), at, `{"display_name":"Updated Bot","welcome_message":"Hi there"}`))
		srve(r, areq("DELETE", fmt.Sprintf("/api/v1/admin/agent-profiles/%d", pid), at, nil))
	}
}

func Test_AccountDeletionRequest(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "adr1", "ADR1", 10)
	tk := tok(t, api, u.ID)
	srve(r, areq("POST", "/api/v1/users/me/delete-account", tk, nil))
	srve(r, areq("POST", "/api/v1/users/me/cancel-deletion", tk, nil))
}

func Test_AdminHotSearchDashboardDetail(t *testing.T) {
	api, r, _ := newTestAPI(t)
	at := admintok(t, api)
	srve(r, areq("GET", "/api/v1/admin/hot-search/dashboard", at, nil))
	srve(r, areq("POST", "/api/v1/admin/hot-search/quick-op", at, `{"keyword":"test","score":100}`))
}

func Test_AdminBannerUploadByID_NilOSS(t *testing.T) {
	api, r, _ := newTestAPI(t)
	at := admintok(t, api)
	w := srve(r, areq("POST", "/api/v1/admin/home-banners", at, `{"title":"Banner Upload Test","link_type":"none","sort_order":1}`))
	var br struct { Code int `json:"code"`; Data struct { ID uint64 `json:"id"`} `json:"data"` }
	json.Unmarshal(w.Body.Bytes(), &br)
	if br.Code == 0 && br.Data.ID > 0 {
		srve(r, areq("POST", fmt.Sprintf("/api/v1/admin/home-banners/%d/image", br.Data.ID), at, nil))
	}
	srve(r, areq("POST", "/api/v1/admin/home-banners/99999/image", at, nil))
}

