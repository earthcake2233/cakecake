//go:build integration

package handler

import (
	"cakecake/internal/model/article"
	"cakecake/internal/model/video"
	"encoding/json"
	"fmt"
	"net/http"
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
	w := covReq(t, r, "POST", fmt.Sprintf("/api/v1/videos/%d/like", v.ID), tk, nil)
	covOK(t, w, http.StatusOK)
	require.Contains(t, w.Body.String(), `"liked":true`)

	// Favorite video
	w = covReq(t, r, "POST", fmt.Sprintf("/api/v1/videos/%d/favorite", v.ID), tk, nil)
	covOK(t, w, http.StatusOK)
	require.Contains(t, w.Body.String(), `"favorited":true`)

	// Coin video
	w = covReq(t, r, "POST", fmt.Sprintf("/api/v1/videos/%d/coin", v.ID), tk, map[string]any{"amount": 1})
	covOK(t, w, http.StatusOK)
	require.Contains(t, w.Body.String(), `"coined":true`)

	// Watch later
	w = covReq(t, r, "POST", fmt.Sprintf("/api/v1/videos/%d/watch-later", v.ID), tk, nil)
	covOK(t, w, http.StatusOK)
	require.Contains(t, w.Body.String(), `"in_watch_later":true`)

	// Post danmaku
	w = covReq(t, r, "POST", fmt.Sprintf("/api/v1/videos/%d/danmaku", v.ID), tk, map[string]any{
		"content": "Great vid", "type": "scroll", "color": "#FFFFFF", "video_time": 10.5,
	})
	covOK(t, w, http.StatusOK)
	require.Contains(t, w.Body.String(), `"content":"Great vid"`)

	// Post comment
	body := fmt.Sprintf(`{"content":"Nice video!","video_id":%d}`, v.ID)
	w = covReq(t, r, "POST", "/api/v1/videos/"+fmt.Sprint(v.ID)+"/comments", tk, body)
	covOK(t, w, http.StatusCreated)
	var cr struct {
		Code int `json:"code"`
		Data struct {
			ID uint64 `json:"id"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &cr))
	if cr.Code == 0 && cr.Data.ID > 0 {
		cid := cr.Data.ID
		// Like comment
		cw := covReq(t, r, "POST", fmt.Sprintf("/api/v1/comments/%d/like", cid), tk2, nil)
		covOK(t, cw, http.StatusOK)
		// Dislike comment
		cw = covReq(t, r, "POST", fmt.Sprintf("/api/v1/comments/%d/dislike", cid), tk2, nil)
		covOK(t, cw, http.StatusOK)
		// Pin comment
		cw = covReq(t, r, "POST", fmt.Sprintf("/api/v1/comments/%d/pin", cid), tk2, nil)
		covOK(t, cw, http.StatusOK)
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
	w := covReq(t, r, "POST", fmt.Sprintf("/api/v1/articles/%d/view", art.ID), tk, nil)
	covOK(t, w, http.StatusOK)

	// Toggle article favorite
	w = covReq(t, r, "POST", fmt.Sprintf("/api/v1/articles/%d/favorite", art.ID), tk, nil)
	covOK(t, w, http.StatusOK)
	require.Contains(t, w.Body.String(), `"favorited":true`)

	// Post article comment
	w = covReq(t, r, "POST", fmt.Sprintf("/api/v1/articles/%d/comments", art.ID), tk, fmt.Sprintf(`{"content":"Great article!","article_id":%d}`, art.ID))
	covOK(t, w, http.StatusCreated)
	var acr struct {
		Code int `json:"code"`
		Data struct {
			ID uint64 `json:"id"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &acr))
	if acr.Code == 0 && acr.Data.ID > 0 {
		cid := acr.Data.ID
		// Like article comment
		cw := covReq(t, r, "POST", fmt.Sprintf("/api/v1/article-comments/%d/like", cid), tk, nil)
		covOK(t, cw, http.StatusOK)
		// Pin article comment
		cw = covReq(t, r, "POST", fmt.Sprintf("/api/v1/article-comments/%d/pin", cid), tk, nil)
		covOK(t, cw, http.StatusOK)
		// Approve article comment
		cw = covReq(t, r, "POST", fmt.Sprintf("/api/v1/article-comments/%d/approve", cid), tk, nil)
		covOK(t, cw, http.StatusOK)
	}
}

func Test_UserFollowFullCycle(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "uff1", "UFF1", 10)
	u2 := seedUser(t, api, "uff2", "UFF2", 10)
	tk := tok(t, api, u.ID)
	tk2 := tok(t, api, u2.ID)
	followPath := fmt.Sprintf("/api/v1/users/%d/follow", u2.ID)

	// Follow user
	w := covReq(t, r, "POST", followPath, tk, nil)
	covOK(t, w, http.StatusOK)
	require.Contains(t, w.Body.String(), `"followed":true`)

	// Check following list
	w = covReq(t, r, "GET", fmt.Sprintf("/api/v1/space/%d/following", u.ID), tk, nil)
	covOK(t, w, http.StatusOK)
	require.Contains(t, w.Body.String(), u2.Nickname)

	// Check followers list
	w = covReq(t, r, "GET", fmt.Sprintf("/api/v1/space/%d/followers", u2.ID), tk2, nil)
	covOK(t, w, http.StatusOK)
	require.Contains(t, w.Body.String(), u.Nickname)

	// Toggle again -> unfollow.
	w = covReq(t, r, "POST", followPath, tk, nil)
	covOK(t, w, http.StatusOK)
	require.Contains(t, w.Body.String(), `"followed":false`)
}

func Test_DmConversationFull(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "dmf1", "DMF1", 10)
	u2 := seedUser(t, api, "dmf2", "DMF2", 10)
	tk := tok(t, api, u.ID)
	tk2 := tok(t, api, u2.ID)

	// Create conversation
	w := covReq(t, r, "POST", "/api/v1/dm/conversations", tk, map[string]any{"peer_id": u2.ID})
	covOK(t, w, http.StatusOK)
	var dcr struct {
		Code int `json:"code"`
		Data struct {
			ID uint64 `json:"id"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &dcr))
	if dcr.Code == 0 && dcr.Data.ID > 0 {
		cid := dcr.Data.ID
		// Post message from user 1
		mw := covReq(t, r, "POST", fmt.Sprintf("/api/v1/dm/conversations/%d/messages", cid), tk, map[string]any{"content": "Hello from U1"})
		covOK(t, mw, http.StatusOK)
		// Post message from user 2
		mw = covReq(t, r, "POST", fmt.Sprintf("/api/v1/dm/conversations/%d/messages", cid), tk2, map[string]any{"content": "Reply from U2"})
		covOK(t, mw, http.StatusOK)
		// List messages
		lw := covReq(t, r, "GET", fmt.Sprintf("/api/v1/dm/conversations/%d/messages", cid), tk, nil)
		covOK(t, lw, http.StatusOK)
		require.Contains(t, lw.Body.String(), "Hello from U1")
		require.Contains(t, lw.Body.String(), "Reply from U2")
		// List conversations
		cw := covReq(t, r, "GET", "/api/v1/dm/conversations", tk, nil)
		covOK(t, cw, http.StatusOK)
		// Delete conversation
		dw := covReq(t, r, "DELETE", fmt.Sprintf("/api/v1/dm/conversations/%d", cid), tk, nil)
		covOK(t, dw, http.StatusOK)
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
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &ffr))
	if ffr.Code == 0 && ffr.Data.ID > 0 {
		fid := ffr.Data.ID
		// List my folders
		lw := covReq(t, r, "GET", "/api/v1/users/me/favorite-folders", tk, nil)
		covOK(t, lw, http.StatusOK)
		// Add video to folder
		aw := covReq(t, r, "POST", fmt.Sprintf("/api/v1/videos/%d/favorite-folders/%d", v.ID, fid), tk, nil)
		covOK(t, aw, http.StatusOK)
		// List favorites in folder
		fw := covReq(t, r, "GET", "/api/v1/users/me/favorites?folder_id="+fmt.Sprint(fid), tk, nil)
		covOK(t, fw, http.StatusOK)
		require.Contains(t, fw.Body.String(), v.Title)
		// Update folder
		uw := covReq(t, r, "PUT", fmt.Sprintf("/api/v1/users/me/favorite-folders/%d", fid), tk, map[string]any{"title": "Updated Collection"})
		covOK(t, uw, http.StatusOK)
		// Clear invalid favorites
		cw := covReq(t, r, "DELETE", fmt.Sprintf("/api/v1/users/me/favorite-folders/%d/invalid-favorites", fid), tk, nil)
		covOK(t, cw, http.StatusOK)
		// Delete folder
		dw := covReq(t, r, "DELETE", fmt.Sprintf("/api/v1/users/me/favorite-folders/%d", fid), tk, nil)
		covOK(t, dw, http.StatusOK)
	}
}

func Test_SpaceEndpointsFull(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "sef1", "SEF1", 100)
	u2 := seedUser(t, api, "sef2", "SEF2", 100)
	v := seedVideoWithAPI(t, api, u.ID, "SEF Video")
	_ = seedArticle(t, api, u.ID, "SEF Article")
	tk := tok(t, api, u.ID)

	// Get user space
	w := covReq(t, r, "GET", fmt.Sprintf("/api/v1/space/%d", u.ID), "", nil)
	covOK(t, w, http.StatusOK)
	require.Contains(t, w.Body.String(), u.Nickname)

	// Space videos
	w = covReq(t, r, "GET", fmt.Sprintf("/api/v1/space/%d/videos", u.ID), "", nil)
	covOK(t, w, http.StatusOK)
	require.Contains(t, w.Body.String(), v.Title)

	// Space articles
	w = covReq(t, r, "GET", fmt.Sprintf("/api/v1/space/%d/articles", u.ID), "", nil)
	covOK(t, w, http.StatusOK)
	require.Contains(t, w.Body.String(), "SEF Article")

	// Space favorites
	w = covReq(t, r, "GET", fmt.Sprintf("/api/v1/space/%d/favorites", u.ID), "", nil)
	covOK(t, w, http.StatusOK)

	// Space favorite folders
	w = covReq(t, r, "GET", fmt.Sprintf("/api/v1/space/%d/favorite-folders", u.ID), "", nil)
	covOK(t, w, http.StatusOK)

	// Space article favorites
	w = covReq(t, r, "GET", fmt.Sprintf("/api/v1/space/%d/article-favorites", u.ID), "", nil)
	covOK(t, w, http.StatusOK)

	// Post coin from another user to create coin record
	coinVideo := seedVideoWithAPI(t, api, u2.ID, "SEF Coin Video")
	w = covReq(t, r, "POST", fmt.Sprintf("/api/v1/videos/%d/coin", coinVideo.ID), tk, map[string]any{"amount": 1})
	covOK(t, w, http.StatusOK)
	require.Contains(t, w.Body.String(), `"coined":true`)

	// Space recent coins
	w = covReq(t, r, "GET", fmt.Sprintf("/api/v1/space/%d/recent-coins", u.ID), tk, nil)
	covOK(t, w, http.StatusOK)
	require.Contains(t, w.Body.String(), "SEF Coin Video")
}
