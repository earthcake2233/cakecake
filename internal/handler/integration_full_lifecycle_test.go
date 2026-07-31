//go:build integration

package handler

import (
	"encoding/json"
	"fmt"
	"minibili/internal/model/article"
	"minibili/internal/model/video"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func Test_FullVideoLifecycle(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "fvl1", "FVL1", 100)
	u2 := seedUser(t, api, "fvl2", "FVL2", 100)
	tk := tok(t, api, u.ID)
	tk2 := tok(t, api, u2.ID)

	// Publish video via DB (simulates upload+approve workflow)
	v := video.Video{UserID: u2.ID, Title: "Lifecycle Video", Status: "published", VideoURL: "https://cdn.example.com/lc.mp4", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	require.NoError(t, api.DB.Create(&v).Error)

	// Like video
	srve(r, areq("POST", fmt.Sprintf("/api/v1/videos/%d/like", v.ID), tk, nil))

	// Favorite video
	srve(r, areq("POST", fmt.Sprintf("/api/v1/videos/%d/favorite", v.ID), tk, nil))

	// Coin video
	srve(r, areq("POST", fmt.Sprintf("/api/v1/videos/%d/coin", v.ID), tk, `{"amount":1}`))

	// Watch later
	srve(r, areq("POST", fmt.Sprintf("/api/v1/videos/%d/watch-later", v.ID), tk, nil))

	// Post danmaku
	srve(r, areq("POST", fmt.Sprintf("/api/v1/videos/%d/danmaku", v.ID), tk, `{"content":"Great vid","type":0,"color":16777215,"progress":10.5}`))

	// Post comment
	body := fmt.Sprintf(`{"content":"Nice video!","video_id":%d}`, v.ID)
	w := srve(r, areq("POST", "/api/v1/videos/"+fmt.Sprint(v.ID)+"/comments", tk, body))
	var cr struct {
		Code int `json:"code"`
		Data struct {
			ID uint64 `json:"id"`
		} `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &cr)
	if cr.Code == 0 && cr.Data.ID > 0 {
		cid := cr.Data.ID
		// Like comment
		srve(r, areq("POST", fmt.Sprintf("/api/v1/comments/%d/like", cid), tk2, nil))
		// Dislike comment
		srve(r, areq("POST", fmt.Sprintf("/api/v1/comments/%d/dislike", cid), tk2, nil))
		// Pin comment
		srve(r, areq("POST", fmt.Sprintf("/api/v1/comments/%d/pin", cid), tk, nil))
	}
}

func Test_FullArticleLifecycle(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "fal1", "FAL1", 100)
	tk := tok(t, api, u.ID)

	// Publish article
	art := article.Article{UserID: u.ID, Title: "Full Lifecycle Article", BodyMD: "# Hello World", Status: "published", CreatedAt: time.Now()}
	require.NoError(t, api.DB.Create(&art).Error)

	// Post view
	srve(r, areq("POST", fmt.Sprintf("/api/v1/articles/%d/view", art.ID), tk, nil))

	// Toggle article like
	srve(r, areq("POST", fmt.Sprintf("/api/v1/articles/%d/like", art.ID), tk, nil))

	// Toggle article favorite
	srve(r, areq("POST", fmt.Sprintf("/api/v1/articles/%d/favorite", art.ID), tk, nil))

	// Post article comment
	w := srve(r, areq("POST", fmt.Sprintf("/api/v1/articles/%d/comments", art.ID), tk, fmt.Sprintf(`{"content":"Great article!","article_id":%d}`, art.ID)))
	var acr struct {
		Code int `json:"code"`
		Data struct {
			ID uint64 `json:"id"`
		} `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &acr)
	if acr.Code == 0 && acr.Data.ID > 0 {
		cid := acr.Data.ID
		// Like article comment
		srve(r, areq("POST", fmt.Sprintf("/api/v1/article-comments/%d/like", cid), tk, nil))
		// Pin article comment
		srve(r, areq("POST", fmt.Sprintf("/api/v1/article-comments/%d/pin", cid), tk, nil))
		// Approve article comment
		srve(r, areq("POST", fmt.Sprintf("/api/v1/article-comments/%d/approve", cid), tk, nil))
	}
}

func Test_UserFollowFullCycle(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "uff1", "UFF1", 10)
	u2 := seedUser(t, api, "uff2", "UFF2", 10)
	tk := tok(t, api, u.ID)

	// Follow user
	srve(r, areq("POST", "/api/v1/users/me/follow", tk, fmt.Sprintf(`{"user_id":%d}`, u2.ID)))

	// Check following list
	srve(r, areq("GET", fmt.Sprintf("/api/v1/space/%d/following", u.ID), "", nil))

	// Check followers list
	srve(r, areq("GET", fmt.Sprintf("/api/v1/space/%d/followers", u2.ID), "", nil))

	// Unfollow
	srve(r, areq("POST", "/api/v1/users/me/unfollow", tk, fmt.Sprintf(`{"user_id":%d}`, u2.ID)))
}

func Test_DmConversationFull(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "dmf1", "DMF1", 10)
	u2 := seedUser(t, api, "dmf2", "DMF2", 10)
	tk := tok(t, api, u.ID)
	tk2 := tok(t, api, u2.ID)

	// Create conversation
	w := srve(r, areq("POST", "/api/v1/dm/conversations", tk, fmt.Sprintf(`{"user_id":%d}`, u2.ID)))
	var dcr struct {
		Code int `json:"code"`
		Data struct {
			ID uint64 `json:"id"`
		} `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &dcr)
	if dcr.Code == 0 && dcr.Data.ID > 0 {
		cid := dcr.Data.ID
		// Post message from user 1
		srve(r, areq("POST", fmt.Sprintf("/api/v1/dm/conversations/%d/messages", cid), tk, `{"content":"Hello from U1"}`))
		// Post message from user 2
		srve(r, areq("POST", fmt.Sprintf("/api/v1/dm/conversations/%d/messages", cid), tk2, `{"content":"Reply from U2"}`))
		// List messages
		srve(r, areq("GET", fmt.Sprintf("/api/v1/dm/conversations/%d/messages", cid), tk, nil))
		// List conversations
		srve(r, areq("GET", "/api/v1/dm/conversations", tk, nil))
		// Delete conversation
		srve(r, areq("DELETE", fmt.Sprintf("/api/v1/dm/conversations/%d", cid), tk, nil))
	}
}

func Test_FavoriteFolderLifecycle(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "ffl1", "FFL1", 10)
	u2 := seedUser(t, api, "ffl2", "FFL2", 10)
	tk := tok(t, api, u.ID)
	v := seedVideoWithAPI(t, api, u2.ID, "FFL Video")

	// Create folder
	w := srve(r, areq("POST", "/api/v1/users/me/favorite-folders", tk, `{"title":"My Collection"}`))
	var ffr struct {
		Code int `json:"code"`
		Data struct {
			ID uint64 `json:"id"`
		} `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &ffr)
	if ffr.Code == 0 && ffr.Data.ID > 0 {
		fid := ffr.Data.ID
		// List my folders
		srve(r, areq("GET", "/api/v1/users/me/favorite-folders", tk, nil))
		// Add video to folder
		srve(r, areq("POST", fmt.Sprintf("/api/v1/videos/%d/favorite-folders/%d", v.ID, fid), tk, nil))
		// List favorites in folder
		srve(r, areq("GET", "/api/v1/users/me/favorites?folder_id="+fmt.Sprint(fid), tk, nil))
		// Update folder
		srve(r, areq("PUT", fmt.Sprintf("/api/v1/users/me/favorite-folders/%d", fid), tk, `{"title":"Updated Collection"}`))
		// Clear invalid favorites
		srve(r, areq("POST", fmt.Sprintf("/api/v1/users/me/favorite-folders/%d/invalid-favorites", fid), tk, nil))
		// Delete folder
		srve(r, areq("DELETE", fmt.Sprintf("/api/v1/users/me/favorite-folders/%d", fid), tk, nil))
	}
}

func Test_SpaceEndpointsFull(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "sef1", "SEF1", 100)
	u2 := seedUser(t, api, "sef2", "SEF2", 100)
	v := seedVideoWithAPI(t, api, u.ID, "SEF Video")
	_ = seedArticle(t, api, u.ID, "SEF Article")

	// Get user space
	srve(r, areq("GET", fmt.Sprintf("/api/v1/space/%d", u.ID), "", nil))

	// Space videos
	srve(r, areq("GET", fmt.Sprintf("/api/v1/space/%d/videos", u.ID), "", nil))

	// Space articles
	srve(r, areq("GET", fmt.Sprintf("/api/v1/space/%d/articles", u.ID), "", nil))

	// Space favorites
	srve(r, areq("GET", fmt.Sprintf("/api/v1/space/%d/favorites", u.ID), "", nil))

	// Space favorite folders
	srve(r, areq("GET", fmt.Sprintf("/api/v1/space/%d/favorite-folders", u.ID), "", nil))

	// Space article favorites
	srve(r, areq("GET", fmt.Sprintf("/api/v1/space/%d/article-favorites", u.ID), "", nil))

	// Post coin from another user to create coin record
	tk2 := tok(t, api, u2.ID)
	srve(r, areq("POST", fmt.Sprintf("/api/v1/videos/%d/coin", v.ID), tk2, `{"amount":1}`))

	// Space recent coins
	srve(r, areq("GET", fmt.Sprintf("/api/v1/space/%d/recent-coins", u.ID), "", nil))
}
