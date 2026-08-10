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
	"golang.org/x/crypto/bcrypt"
)

func Test_ArticleEngagementFavoriteAndCoin(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "aef1", "AEF1", 100)
	u2 := seedUser(t, api, "aef2", "AEF2", 100)
	tk := tok(t, api, u.ID)
	art := seedArticle(t, api, u2.ID, "AEF Article")
	w := covReq(t, r, "POST", fmt.Sprintf("/api/v1/articles/%d/favorite", art.ID), tk, nil)
	covOK(t, w, http.StatusOK)
	require.Contains(t, w.Body.String(), `"favorited":true`)
	w = covReq(t, r, "POST", fmt.Sprintf("/api/v1/articles/%d/coin", art.ID), tk, map[string]any{"amount": 1})
	covOK(t, w, http.StatusOK)
	require.Contains(t, w.Body.String(), `"coined":true`)
	w = covReq(t, r, "GET", fmt.Sprintf("/api/v1/space/%d/article-favorites", u.ID), tk, nil)
	covOK(t, w, http.StatusOK)
	require.Contains(t, w.Body.String(), "AEF Article")
	w = covReq(t, r, "GET", "/api/v1/users/me/article-favorites", tk, nil)
	covOK(t, w, http.StatusOK)
	require.Contains(t, w.Body.String(), "AEF Article")
}

func Test_DynamicCommentFullFlow(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "dcf1", "DCF1", 10)
	u2 := seedUser(t, api, "dcf2", "DCF2", 10)
	tk := tok(t, api, u.ID)
	tk2 := tok(t, api, u2.ID)
	dyn := dynamic.UserDynamic{UserID: u.ID, Title: "DCF Dynamic", Content: "Test content for comments", ImagesJSON: "[]", CreatedAt: time.Now()}
	require.NoError(t, api.DB.Create(&dyn).Error)
	body := fmt.Sprintf(`{"content":"Nice dynamic!","dynamic_id":%d}`, dyn.ID)
	w := srve(r, areq("POST", fmt.Sprintf("/api/v1/user-dynamics/%d/comments", dyn.ID), tk2, body))
	var dcr struct {
		Code int `json:"code"`
		Data struct {
			ID uint64 `json:"id"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &dcr))
	if dcr.Code == 0 && dcr.Data.ID > 0 {
		cid := dcr.Data.ID
		aw := covReq(t, r, "POST", fmt.Sprintf("/api/v1/dynamic-comments/%d/approve", cid), tk, nil)
		covOK(t, aw, http.StatusOK)
		lw := covReq(t, r, "POST", fmt.Sprintf("/api/v1/dynamic-comments/%d/like", cid), tk2, nil)
		covOK(t, lw, http.StatusOK)
		require.Contains(t, lw.Body.String(), `"liked":true`)
		dw := covReq(t, r, "POST", fmt.Sprintf("/api/v1/dynamic-comments/%d/dislike", cid), tk, nil)
		covOK(t, dw, http.StatusOK)
		delw := covReq(t, r, "DELETE", fmt.Sprintf("/api/v1/dynamic-comments/%d", cid), tk2, nil)
		covOK(t, delw, http.StatusOK)
	}
	w = covReq(t, r, "GET", fmt.Sprintf("/api/v1/user-dynamics/%d/comments", dyn.ID), "", nil)
	covOK(t, w, http.StatusOK)
}

func Test_UserDynamicUpdateAndList(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "udl1", "UDL1", 10)
	tk := tok(t, api, u.ID)
	u2 := seedUser(t, api, "udl2", "UDL2", 10)
	w := doMultipart(r, "POST", "/api/v1/users/me/dynamics", tk, map[string]string{
		"title": "Update Test", "content": "Original content",
	})
	covOK(t, w, http.StatusOK)
	var dr struct {
		Code int `json:"code"`
		Data struct {
			ID uint64 `json:"id"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &dr))
	if dr.Code == 0 && dr.Data.ID > 0 {
		did := dr.Data.ID
		uw := doMultipart(r, "PUT", fmt.Sprintf("/api/v1/users/me/dynamics/%d", did), tk, map[string]string{
			"title": "Updated Title", "content": "Updated content",
		})
		covOK(t, uw, http.StatusOK)
		pw := covReq(t, r, "PATCH", fmt.Sprintf("/api/v1/users/me/dynamics/%d/playback", did), tk, map[string]any{"comments_closed": true})
		covOK(t, pw, http.StatusOK)
		require.Contains(t, pw.Body.String(), `"comments_closed":true`)
	}
	w = covReq(t, r, "GET", fmt.Sprintf("/api/v1/space/%d/dynamics", u.ID), tk, nil)
	covOK(t, w, http.StatusOK)
	require.Contains(t, w.Body.String(), "Updated Title")
	dyn2 := dynamic.UserDynamic{UserID: u2.ID, Title: "Other Dynamic", Content: "Other", ImagesJSON: "[]", CreatedAt: time.Now()}
	require.NoError(t, api.DB.Create(&dyn2).Error)
	w = covReq(t, r, "GET", fmt.Sprintf("/api/v1/user-dynamics/%d", dyn2.ID), "", nil)
	covOK(t, w, http.StatusOK)
	require.Contains(t, w.Body.String(), "Other Dynamic")
}

func Test_VideoEngagementFolderOps(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "vef1", "VEF1", 10)
	u2 := seedUser(t, api, "vef2", "VEF2", 10)
	tk := tok(t, api, u.ID)
	v := seedVideoWithAPI(t, api, u2.ID, "VEF Video")
	w := srve(r, areq("POST", "/api/v1/users/me/favorite-folders", tk, `{"title":"VEF Folder"}`))
	var fr struct {
		Code int `json:"code"`
		Data struct {
			ID uint64 `json:"id"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &fr))
	if fr.Code == 0 && fr.Data.ID > 0 {
		fid := fr.Data.ID
		aw := covReq(t, r, "POST", fmt.Sprintf("/api/v1/videos/%d/favorite-folders/%d", v.ID, fid), tk, nil)
		covOK(t, aw, http.StatusOK)
		folder2 := video.FavoriteFolder{UserID: u.ID, Title: "VEF Folder 2"}
		require.NoError(t, api.DB.Create(&folder2).Error)
		sw := covReq(t, r, "PUT", fmt.Sprintf("/api/v1/videos/%d/favorite-folders", v.ID), tk, map[string]any{"folder_ids": []uint64{fid}})
		covOK(t, sw, http.StatusOK)
		mw := covReq(t, r, "PUT", fmt.Sprintf("/api/v1/videos/%d/favorite-folders/move", v.ID), tk, map[string]any{
			"from_folder_id": fid, "to_folder_id": folder2.ID,
		})
		covOK(t, mw, http.StatusOK)
		dw := covReq(t, r, "DELETE", fmt.Sprintf("/api/v1/videos/%d/favorite-folders/%d", v.ID, fid), tk, nil)
		covOK(t, dw, http.StatusOK)
	}
}

func Test_AdminAgentProfileCRUD(t *testing.T) {
	api, r, _ := newTestAPI(t)
	at := admintok(t, api)
	w := srve(r, areq("POST", "/api/v1/admin/agent-profiles", at, `{"slug":"test-bot","display_name":"Test Bot","welcome_message":"Hello"}`))
	var apr struct {
		Code int `json:"code"`
		Data struct {
			ID uint64 `json:"id"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &apr))
	if apr.Code == 0 && apr.Data.ID > 0 {
		pid := apr.Data.ID
		uw := covReq(t, r, "PUT", fmt.Sprintf("/api/v1/admin/agent-profiles/%d", pid), at, map[string]any{"display_name": "Updated Bot", "welcome_message": "Hi there"})
		covOK(t, uw, http.StatusOK)
		dw := covReq(t, r, "DELETE", fmt.Sprintf("/api/v1/admin/agent-profiles/%d", pid), at, nil)
		covOK(t, dw, http.StatusOK)
	}
}

func Test_AccountDeletionRequest(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "adr1", "ADR1", 10)
	hash, err := bcrypt.GenerateFromPassword([]byte("password12"), bcrypt.MinCost)
	require.NoError(t, err)
	require.NoError(t, api.DB.Model(&u).Update("password_hash", string(hash)).Error)
	tk := tok(t, api, u.ID)
	w := covReq(t, r, "POST", "/api/v1/users/me/deletion/request", tk, map[string]any{"password": "password12"})
	covOK(t, w, http.StatusOK)
	require.Contains(t, w.Body.String(), `"pending":true`)
	w = covReq(t, r, "POST", "/api/v1/users/me/deletion/revoke", tk, nil)
	covOK(t, w, http.StatusOK)
	require.Contains(t, w.Body.String(), `"ok":true`)
}

func Test_AdminHotSearchDashboardDetail(t *testing.T) {
	api, r, _ := newTestAPI(t)
	at := admintok(t, api)
	w := covReq(t, r, "GET", "/api/v1/admin/hot-search/dashboard", at, nil)
	covOK(t, w, http.StatusOK)
	w = covReq(t, r, "POST", "/api/v1/admin/hot-search/quick-op", at, map[string]any{
		"keyword": "test", "op_type": "manual", "display_title": "测试",
	})
	covOK(t, w, http.StatusCreated)
}

func Test_AdminBannerUploadByID_NilOSS(t *testing.T) {
	api, r, _ := newTestAPI(t)
	at := admintok(t, api)
	w := srve(r, areq("POST", "/api/v1/admin/home-banners", at, `{"title":"Banner Upload Test","link_type":"none","sort_order":1}`))
	var br struct {
		Code int `json:"code"`
		Data struct {
			ID uint64 `json:"id"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &br))
	if br.Code == 0 && br.Data.ID > 0 {
		iw := covReq(t, r, "POST", fmt.Sprintf("/api/v1/admin/home-banners/%d/image", br.Data.ID), at, nil)
		covOK(t, iw, http.StatusInternalServerError)
		var iresp struct {
			Code int `json:"code"`
		}
		require.NoError(t, json.Unmarshal(iw.Body.Bytes(), &iresp))
		require.Equal(t, 50000, iresp.Code)
	}
	w = covReq(t, r, "POST", "/api/v1/admin/home-banners/99999/image", at, nil)
	covOK(t, w, http.StatusNotFound)
}
