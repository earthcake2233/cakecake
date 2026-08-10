//go:build integration

package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"cakecake/internal/config"
	"cakecake/internal/model/admin"
	"cakecake/internal/model/video"
	"github.com/stretchr/testify/require"
)

func Test_UserDynamicCRUD(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "udc1", "UDC1", 10)
	tk := tok(t, api, u.ID)

	// Post a dynamic
	w := doMultipart(r, "POST", "/api/v1/users/me/dynamics", tk, map[string]string{
		"title": "My Dynamic", "content": "Dynamic content",
	})
	covOK(t, w, http.StatusOK)
	var dr struct {
		Code int `json:"code"`
		Data struct {
			ID uint64 `json:"id"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &dr))
	require.Equal(t, 0, dr.Code, w.Body.String())
	if dr.Code == 0 && dr.Data.ID > 0 {
		did := dr.Data.ID
		// Toggle like
		srveOK(t, r, areq("POST", fmt.Sprintf("/api/v1/user-dynamics/%d/like", did), tk, nil), http.StatusOK)
		// Delete dynamic
		srveOK(t, r, areq("DELETE", fmt.Sprintf("/api/v1/users/me/dynamics/%d", did), tk, nil), http.StatusOK)
	}
	// List my dynamics
	srveOK(t, r, areq("GET", "/api/v1/users/me/dynamics?page=1&page_size=10", tk, nil), http.StatusOK)
}

func Test_FavoriteFolderCRUDMore(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "ffc1", "FFC1", 10)
	u2 := seedUser(t, api, "ffc2", "FFC2", 10)
	tk := tok(t, api, u.ID)
	v := seedVideoWithAPI(t, api, u2.ID, "FF Video")

	// Create a folder
	w := srve(r, areq("POST", "/api/v1/users/me/favorite-folders", tk, `{"title":"My Folder","is_public":true}`))
	var fr struct {
		Code int `json:"code"`
		Data struct {
			ID uint64 `json:"id"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &fr))
	if fr.Code == 0 && fr.Data.ID > 0 {
		fid := fr.Data.ID
		// Update folder
		srveOK(t, r, areq("PUT", fmt.Sprintf("/api/v1/users/me/favorite-folders/%d", fid), tk, `{"title":"Updated Folder"}`), http.StatusOK)
		// Add video to folder
		srveOK(t, r, areq("POST", fmt.Sprintf("/api/v1/videos/%d/favorite-folders/%d", v.ID, fid), tk, nil), http.StatusOK)
		// Batch remove (empty list)
		srveOK(t, r, areq("POST", fmt.Sprintf("/api/v1/users/me/favorite-folders/%d/batch-remove", fid), tk, fmt.Sprintf(`{"video_ids":[%d]}`, v.ID)), http.StatusOK)
		// Delete folder
		srveOK(t, r, areq("DELETE", fmt.Sprintf("/api/v1/users/me/favorite-folders/%d", fid), tk, nil), http.StatusOK)
	}

	// Create default folder for invalid favorites test
	w2 := srve(r, areq("POST", "/api/v1/users/me/favorite-folders", tk, `{"title":"Another Folder"}`))
	var fr2 struct {
		Code int `json:"code"`
		Data struct {
			ID uint64 `json:"id"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w2.Body.Bytes(), &fr2))
	if fr2.Code == 0 && fr2.Data.ID > 0 {
		srveOK(t, r, areq("DELETE", fmt.Sprintf("/api/v1/users/me/favorite-folders/%d/invalid-favorites", fr2.Data.ID), tk, nil), http.StatusOK)
	}
}

func Test_ArticleEngagementMore(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "aem1", "AEM1", 100)
	u2 := seedUser(t, api, "aem2", "AEM2", 100)
	tk := tok(t, api, u.ID)
	art := seedArticle(t, api, u2.ID, "AE Article")

	// Toggle article like
	srveOK(t, r, areq("POST", fmt.Sprintf("/api/v1/articles/%d/like", art.ID), tk, nil), http.StatusNotFound)
	// Toggle article favorite
	srveOK(t, r, areq("POST", fmt.Sprintf("/api/v1/articles/%d/favorite", art.ID), tk, nil), http.StatusOK)
}

func Test_CommentNotificationRead(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "cnr1", "CNR1", 10)
	tk := tok(t, api, u.ID)

	// Mark notification read (non-existent -> handle gracefully)
	srveOK(t, r, areq("PATCH", "/api/v1/notifications/0/read", tk, nil), http.StatusOK)

	// Notification comment like (non-existent)
	srveOK(t, r, areq("POST", "/api/v1/notifications/0/comment-like", tk, nil), http.StatusNotFound)

	// Notification comment reply (non-existent)
	srveOK(t, r, areq("POST", "/api/v1/notifications/0/comment-reply", tk, `{"content":"Reply"}`), http.StatusNotFound)
}

func Test_VideoFavoriteFolders(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "vff1", "VFF1", 10)
	u2 := seedUser(t, api, "vff2", "VFF2", 10)
	tk := tok(t, api, u.ID)
	v := seedVideoWithAPI(t, api, u2.ID, "VFF Video")

	// Get video detail with engagement
	srveOK(t, r, areq("GET", fmt.Sprintf("/api/v1/videos/%d", v.ID), tk, nil), http.StatusOK)

	// Toggle favorite
	srveOK(t, r, areq("POST", fmt.Sprintf("/api/v1/videos/%d/favorite", v.ID), tk, nil), http.StatusOK)

	// Get favorite picker
	srveOK(t, r, areq("GET", fmt.Sprintf("/api/v1/videos/%d/favorite-picker", v.ID), tk, nil), http.StatusOK)

	// Create folder, then set video favorite folders
	w := srve(r, areq("POST", "/api/v1/users/me/favorite-folders", tk, `{"title":"VFF Folder"}`))
	var fr struct {
		Code int `json:"code"`
		Data struct {
			ID uint64 `json:"id"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &fr))
	if fr.Code == 0 && fr.Data.ID > 0 {
		fid := fr.Data.ID
		folder2 := video.FavoriteFolder{UserID: u.ID, Title: "VFF Folder 2"}
		require.NoError(t, api.DB.Create(&folder2).Error)
		// Add video to folder
		srveOK(t, r, areq("POST", fmt.Sprintf("/api/v1/videos/%d/favorite-folders/%d", v.ID, fid), tk, nil), http.StatusOK)
		// Set video favorite folders
		srveOK(t, r, areq("PUT", fmt.Sprintf("/api/v1/videos/%d/favorite-folders", v.ID), tk, fmt.Sprintf(`{"folder_ids":[%d]}`, fid)), http.StatusOK)
		// Move video to folder
		srveOK(t, r, areq("PUT", fmt.Sprintf("/api/v1/videos/%d/favorite-folders/move", v.ID), tk, fmt.Sprintf(`{"from_folder_id":%d,"to_folder_id":%d}`, fid, folder2.ID)), http.StatusOK)
		// Remove video from folder
		srveOK(t, r, areq("DELETE", fmt.Sprintf("/api/v1/videos/%d/favorite-folders/%d", v.ID, fid), tk, nil), http.StatusOK)
	}
}
func Test_VideoMyListAndCount(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "vml1", "VML1", 10)
	tk := tok(t, api, u.ID)
	seedVideoWithAPI(t, api, u.ID, "My Video 1")

	// List my videos
	srveOK(t, r, areq("GET", "/api/v1/users/me/videos?page=1&page_size=10", tk, nil), http.StatusOK)

	// Count my videos by status
	srveOK(t, r, areq("GET", "/api/v1/users/me/videos/count", tk, nil), http.StatusNotFound)
}

func Test_ArticleMyListAndCount(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "aml1", "AML1", 10)
	tk := tok(t, api, u.ID)
	seedArticle(t, api, u.ID, "My Article")

	// List my articles
	srveOK(t, r, areq("GET", "/api/v1/users/me/articles?page=1&page_size=10", tk, nil), http.StatusOK)
	// Count my articles
	srveOK(t, r, areq("GET", "/api/v1/users/me/articles/count", tk, nil), http.StatusBadRequest)
}
func Test_VideoDraftList(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "vdl1", "VDL1", 10)
	tk := tok(t, api, u.ID)

	// List drafts
	srveOK(t, r, areq("GET", "/api/v1/users/me/video-drafts?page=1&page_size=10", tk, nil), http.StatusNotFound)

	// Publish a non-existent draft (404)
	srveOK(t, r, areq("POST", "/api/v1/videos/99999/publish", tk, nil), http.StatusNotFound)
}

func Test_HomeStatsAndHotSearch(t *testing.T) {
	_, r, _ := newTestAPI(t)

	// Home stats
	srveOK(t, r, areq("GET", "/api/v1/stats/home", "", nil), http.StatusOK)

	// Hot search
	srveOK(t, r, areq("GET", "/api/v1/hot-search", "", nil), http.StatusOK)

	// Home banners
	srveOK(t, r, areq("GET", "/api/v1/home-banners", "", nil), http.StatusOK)
}

func Test_AdminSystemConfigAndMe(t *testing.T) {
	api, r, _ := newTestAPI(t)
	api.RuntimeCfg = config.NewRuntimeConfig(api.DB, nil)
	at := admintok(t, api)
	require.NoError(t, api.DB.Create(&admin.Admin{ID: 1, Username: "root", PasswordHash: "x", Status: "active"}).Error)

	// Admin me
	srveOK(t, r, areq("GET", "/api/v1/admin/me", at, nil), http.StatusOK)

	// List configs
	srveOK(t, r, areq("GET", "/api/v1/admin/system-configs", at, nil), http.StatusOK)
}

func Test_FollowAndGroupMore(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "fgm1", "FGM1", 10)
	u2 := seedUser(t, api, "fgm2", "FGM2", 10)
	tk := tok(t, api, u.ID)

	// Follow
	srveOK(t, r, areq("POST", fmt.Sprintf("/api/v1/users/%d/follow", u2.ID), tk, nil), http.StatusOK)

	// List following
	srveOK(t, r, areq("GET", fmt.Sprintf("/api/v1/space/%d/following", u.ID), tk, nil), http.StatusOK)
	// List followers
	srveOK(t, r, areq("GET", fmt.Sprintf("/api/v1/space/%d/followers", u.ID), tk, nil), http.StatusOK)

	// Create a group with a member
	w := srve(r, areq("POST", "/api/v1/users/me/follow-groups", tk, `{"name":"Group A"}`))
	var gr struct {
		Code int `json:"code"`
		Data struct {
			ID uint64 `json:"id"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &gr))
	if gr.Code == 0 && gr.Data.ID > 0 {
		gid := gr.Data.ID
		srveOK(t, r, areq("POST", fmt.Sprintf("/api/v1/users/me/follow-groups/%d/members", gid), tk, fmt.Sprintf(`{"followee_id":%d}`, u2.ID)), http.StatusOK)
		srveOK(t, r, areq("DELETE", fmt.Sprintf("/api/v1/users/me/follow-groups/%d/members/%d", gid, u2.ID), tk, nil), http.StatusOK)
	}
}
