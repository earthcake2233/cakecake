package handler

import (
	"encoding/json"
	"fmt"
	"testing"


)


func Test_UserDynamicCRUD(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "udc1", "UDC1", 10)
	tk := tok(t, api, u.ID)
	
	// Post a dynamic
	w := srve(r, areq("POST", "/api/v1/users/me/dynamics", tk, `{"title":"My Dynamic","content":"Dynamic content"}`))
	var dr struct {
		Code int `json:"code"`
		Data struct {
			ID uint64 `json:"id"`
		} `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &dr)
	if dr.Code == 0 && dr.Data.ID > 0 {
		did := dr.Data.ID
		// Toggle like
		srve(r, areq("POST", fmt.Sprintf("/api/v1/user-dynamics/%d/like", did), tk, nil))
		// Delete dynamic
		srve(r, areq("DELETE", fmt.Sprintf("/api/v1/users/me/dynamics/%d", did), tk, nil))
	}
	// List my dynamics
	srve(r, areq("GET", "/api/v1/users/me/dynamics?page=1&page_size=10", tk, nil))
}

func Test_FavoriteFolderCRUDMore(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "ffc1", "FFC1", 10)
	u2 := seedUser(t, api, "ffc2", "FFC2", 10)
	tk := tok(t, api, u.ID)
	v := seedVideo(t, api, u2.ID, "FF Video")
	
	// Create a folder
	w := srve(r, areq("POST", "/api/v1/users/me/favorite-folders", tk, `{"name":"My Folder","public":true}`))
	var fr struct {
		Code int `json:"code"`
		Data struct {
			ID uint64 `json:"id"`
		} `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &fr)
	if fr.Code == 0 && fr.Data.ID > 0 {
		fid := fr.Data.ID
		// Update folder
		srve(r, areq("PUT", fmt.Sprintf("/api/v1/users/me/favorite-folders/%d", fid), tk, `{"name":"Updated Folder","public":false}`))
		// Add video to folder
		srve(r, areq("POST", fmt.Sprintf("/api/v1/videos/%d/favorite-folders/%d", v.ID, fid), tk, nil))
		// Batch remove (empty list)
		srve(r, areq("POST", fmt.Sprintf("/api/v1/users/me/favorite-folders/%d/batch-remove", fid), tk, `{"video_ids":[]}`))
		// Delete folder
		srve(r, areq("DELETE", fmt.Sprintf("/api/v1/users/me/favorite-folders/%d", fid), tk, nil))
	}
	
	// Create default folder for invalid favorites test
	w2 := srve(r, areq("POST", "/api/v1/users/me/favorite-folders", tk, `{"name":"Another Folder"}`))
	var fr2 struct { Code int `json:"code"`; Data struct { ID uint64 `json:"id"` } `json:"data"` }
	json.Unmarshal(w2.Body.Bytes(), &fr2)
	if fr2.Code == 0 && fr2.Data.ID > 0 {
		srve(r, areq("POST", fmt.Sprintf("/api/v1/users/me/favorite-folders/%d/invalid-favorites", fr2.Data.ID), tk, nil))
	}
}

func Test_ArticleEngagementMore(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "aem1", "AEM1", 100)
	u2 := seedUser(t, api, "aem2", "AEM2", 100)
	tk := tok(t, api, u.ID)
	art := seedArticle(t, api, u2.ID, "AE Article")
	
	// Toggle article like
	srve(r, areq("POST", fmt.Sprintf("/api/v1/articles/%d/like", art.ID), tk, nil))
	// Toggle article favorite
	srve(r, areq("POST", fmt.Sprintf("/api/v1/articles/%d/favorite", art.ID), tk, nil))
}

func Test_CommentNotificationRead(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "cnr1", "CNR1", 10)
	tk := tok(t, api, u.ID)
	
	// Mark notification read (non-existent -> handle gracefully)
	srve(r, areq("PATCH", "/api/v1/notifications/0/read", tk, nil))
	
	// Notification comment like (non-existent)
	srve(r, areq("POST", "/api/v1/notifications/0/comment-like", tk, nil))
	
	// Notification comment reply (non-existent)
	srve(r, areq("POST", "/api/v1/notifications/0/comment-reply", tk, `{"content":"Reply"}`))
}

func Test_VideoFavoriteFolders(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "vff1", "VFF1", 10)
	u2 := seedUser(t, api, "vff2", "VFF2", 10)
	tk := tok(t, api, u.ID)
	v := seedVideo(t, api, u2.ID, "VFF Video")
	
	// Get video detail with engagement
	srve(r, areq("GET", fmt.Sprintf("/api/v1/videos/%d", v.ID), tk, nil))
	
	// Toggle favorite
	srve(r, areq("POST", fmt.Sprintf("/api/v1/videos/%d/favorite", v.ID), tk, nil))
	
	// Get favorite picker
	srve(r, areq("GET", fmt.Sprintf("/api/v1/videos/%d/favorite-picker", v.ID), tk, nil))
	
	// Create folder, then set video favorite folders
	w := srve(r, areq("POST", "/api/v1/users/me/favorite-folders", tk, `{"name":"VFF Folder"}`))
	var fr struct { Code int `json:"code"`; Data struct { ID uint64 `json:"id"` } `json:"data"` }
	json.Unmarshal(w.Body.Bytes(), &fr)
	if fr.Code == 0 && fr.Data.ID > 0 {
		fid := fr.Data.ID
		// Add video to folder
		srve(r, areq("POST", fmt.Sprintf("/api/v1/videos/%d/favorite-folders/%d", v.ID, fid), tk, nil))
		// Set video favorite folders
		srve(r, areq("PUT", fmt.Sprintf("/api/v1/videos/%d/favorite-folders", v.ID), tk, fmt.Sprintf(`{"folder_ids":[%d]}`, fid)))
		// Move video to folder
		srve(r, areq("PUT", fmt.Sprintf("/api/v1/videos/%d/favorite-folders/move", v.ID), tk, fmt.Sprintf(`{"folder_ids":[%d]}`, fid)))
		// Remove video from folder
		srve(r, areq("DELETE", fmt.Sprintf("/api/v1/videos/%d/favorite-folders/%d", v.ID, fid), tk, nil))
	}
}
func Test_VideoMyListAndCount(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "vml1", "VML1", 10)
	tk := tok(t, api, u.ID)
	seedVideo(t, api, u.ID, "My Video 1")
	
	// List my videos
	srve(r, areq("GET", "/api/v1/users/me/videos?page=1&page_size=10", tk, nil))
	
	// Count my videos by status
	srve(r, areq("GET", "/api/v1/users/me/videos/count", tk, nil))
}

func Test_ArticleMyListAndCount(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "aml1", "AML1", 10)
	tk := tok(t, api, u.ID)
	seedArticle(t, api, u.ID, "My Article")
	
	// List my articles
	srve(r, areq("GET", "/api/v1/users/me/articles?page=1&page_size=10", tk, nil))
	// Count my articles
	srve(r, areq("GET", "/api/v1/users/me/articles/count", tk, nil))
}
func Test_VideoDraftList(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "vdl1", "VDL1", 10)
	tk := tok(t, api, u.ID)
	
	// List drafts
	srve(r, areq("GET", "/api/v1/users/me/video-drafts?page=1&page_size=10", tk, nil))
	
	// Publish a non-existent draft (404)
	srve(r, areq("POST", "/api/v1/videos/99999/publish", tk, nil))
}

func Test_HomeStatsAndHotSearch(t *testing.T) {
	_, r, _ := newTestAPI(t)
	
	// Home stats
	srve(r, areq("GET", "/api/v1/stats/home", "", nil))
	
	// Hot search
	srve(r, areq("GET", "/api/v1/hot-search", "", nil))
	
	// Home banners
	srve(r, areq("GET", "/api/v1/home-banners", "", nil))
}

func Test_AdminSystemConfigAndMe(t *testing.T) {
	api, r, _ := newTestAPI(t)
	at := admintok(t, api)
	
	// Admin me
	srve(r, areq("GET", "/api/v1/admin/me", at, nil))
	
	// List configs
	srve(r, areq("GET", "/api/v1/admin/system-configs", at, nil))
}

func Test_FollowAndGroupMore(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "fgm1", "FGM1", 10)
	u2 := seedUser(t, api, "fgm2", "FGM2", 10)
	tk := tok(t, api, u.ID)
	
	// Follow
	srve(r, areq("POST", "/api/v1/users/me/follow", tk, fmt.Sprintf(`{"user_id":%d}`, u2.ID)))
	
	// List following
	srve(r, areq("GET", fmt.Sprintf("/api/v1/space/%d/following", u.ID), "", nil))
	// List followers
	srve(r, areq("GET", fmt.Sprintf("/api/v1/space/%d/followers", u.ID), "", nil))
	
	// Create a group with a member
	w := srve(r, areq("POST", "/api/v1/users/me/follow-groups", tk, `{"name":"Group A"}`))
	var gr struct { Code int `json:"code"`; Data struct { ID uint64 `json:"id"` } `json:"data"` }
	json.Unmarshal(w.Body.Bytes(), &gr)
	if gr.Code == 0 && gr.Data.ID > 0 {
		gid := gr.Data.ID
		srve(r, areq("POST", fmt.Sprintf("/api/v1/users/me/follow-groups/%d/members", gid), tk, fmt.Sprintf(`{"user_id":%d}`, u2.ID)))
		srve(r, areq("DELETE", fmt.Sprintf("/api/v1/users/me/follow-groups/%d/members/%d", gid, u2.ID), tk, nil))
	}
}

