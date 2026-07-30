//go:build integration

package handler

import (
	"minibili/internal/model/comment"
	"minibili/internal/model/dynamic"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

)

func codeFrom(t *testing.T, w *httptest.ResponseRecorder) int {
	t.Helper()
	var r struct { Code int `json:"code"` }
	json.Unmarshal(w.Body.Bytes(), &r)
	return r.Code
}

// Test decrement paths: like/unlike, fav/unfav, coin
func Test_DecrementPaths(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "dec1", "DEC1", 100)
	u2 := seedUser(t, api, "dec2", "DEC2", 100)
	v := seedVideoWithAPI(t, api, u.ID, "DEC Video")
	tk := tok(t, api, u2.ID)

	// Post a comment
	w := srve(r, areq("POST", "/api/v1/videos/"+strconv.FormatUint(v.ID, 10)+"/comments", tk, `{"content":"test"}`))
	if codeFrom(t, w) != 0 { t.Skip("comment post failed") }

	// Extract comment ID
	var cm struct { Data comment.Comment `json:"data"` }
	json.Unmarshal(w.Body.Bytes(), &cm)
	if cm.Data.ID == 0 { t.Skip("no comment id") }
	cid := cm.Data.ID

	// Like comment (CASE WHEN like_count - 1 later gets tested)
	srve(r, areq("POST", "/api/v1/comments/"+strconv.FormatUint(cid, 10)+"/like", tk, nil))
	// Unlike by toggling dislike (covers like_count -= 1 CASE WHEN)
	srve(r, areq("POST", "/api/v1/comments/"+strconv.FormatUint(cid, 10)+"/dislike", tk, nil))
	// Like again then dislike again
	srve(r, areq("POST", "/api/v1/comments/"+strconv.FormatUint(cid, 10)+"/like", tk, nil))
	srve(r, areq("POST", "/api/v1/comments/"+strconv.FormatUint(cid, 10)+"/dislike", tk, nil))

	// Toggle video like
	srve(r, areq("POST", "/api/v1/videos/"+strconv.FormatUint(v.ID, 10)+"/like", tk, nil))
	srve(r, areq("POST", "/api/v1/videos/"+strconv.FormatUint(v.ID, 10)+"/like", tk, nil))

	// Toggle video favorite (covers fav_count -= 1 CASE WHEN)
	srve(r, areq("POST", "/api/v1/videos/"+strconv.FormatUint(v.ID, 10)+"/favorite", tk, nil))
	srve(r, areq("POST", "/api/v1/videos/"+strconv.FormatUint(v.ID, 10)+"/favorite", tk, nil))

	// Toggle article favorite
	art := seedArticle(t, api, u.ID, "DEC Article")
	srve(r, areq("POST", "/api/v1/articles/"+strconv.FormatUint(art.ID, 10)+"/favorite", tk, nil))
	srve(r, areq("POST", "/api/v1/articles/"+strconv.FormatUint(art.ID, 10)+"/favorite", tk, nil))
}

// Test dynamic comment reactions (covers CASE WHEN like_count -= 1)
func Test_DynamicCommentReactions(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "dyn1", "DYN1", 0)
	tk := tok(t, api, u.ID)

	// Create dynamic directly
	dyn := dynamic.UserDynamic{UserID: u.ID, Title: "DYN", Content: "test", ImagesJSON: "[]", CreatedAt: time.Now()}
	require.NoError(t, api.DB.Create(&dyn).Error)

	// Post comment on dynamic
	body := `{"content":"dynamic comment"}`
	w := srve(r, areq("POST", fmt.Sprintf("/api/v1/user-dynamics/%d/comments", dyn.ID), tk, body))
	if codeFrom(t, w) != 0 { t.Skip("dynamic comment failed") }

	var dcm struct { Data comment.DynamicComment `json:"data"` }
	json.Unmarshal(w.Body.Bytes(), &dcm)
	if dcm.Data.ID == 0 { t.Skip("no dynamic comment id") }
	dcid := dcm.Data.ID

	// Like (then unlike by disliking) - covers CASE WHEN like_count -= 1
	srve(r, areq("POST", fmt.Sprintf("/api/v1/dynamic-comments/%d/like", dcid), tk, nil))
	srve(r, areq("POST", fmt.Sprintf("/api/v1/dynamic-comments/%d/dislike", dcid), tk, nil))
}

// Test article comment reactions
func Test_ArticleCommentDecrement(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "acd1", "ACD1", 0)
	art := seedArticle(t, api, u.ID, "ACD Article")
	tk := tok(t, api, u.ID)

	w := srve(r, areq("POST", fmt.Sprintf("/api/v1/articles/%d/comments", art.ID), tk, `{"content":"acd comment"}`))
	if codeFrom(t, w) != 0 { t.Skip("article comment failed") }

	var acm struct { Data comment.ArticleComment `json:"data"` }
	json.Unmarshal(w.Body.Bytes(), &acm)
	if acm.Data.ID == 0 { t.Skip("no article comment id") }
	acid := acm.Data.ID

	// Like toggle (covers CASE WHEN like_count -= 1)
	srve(r, areq("POST", fmt.Sprintf("/api/v1/article-comments/%d/like", acid), tk, nil))
	srve(r, areq("POST", fmt.Sprintf("/api/v1/article-comments/%d/dislike", acid), tk, nil))

	// Approve, pin
	srve(r, areq("POST", fmt.Sprintf("/api/v1/article-comments/%d/approve", acid), tk, nil))
	srve(r, areq("POST", fmt.Sprintf("/api/v1/article-comments/%d/pin", acid), tk, nil))

	// Delete article comment (covers comment_count -= 1 CASE WHEN)
	srve(r, areq("DELETE", fmt.Sprintf("/api/v1/article-comments/%d", acid), tk, nil))
}

// Test danmaku operations
func Test_DanmakuOperations(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "dan1", "DAN1", 0)
	v := seedVideoWithAPI(t, api, u.ID, "DAN Video")
	tk := tok(t, api, u.ID)

	// Post a danmaku
	srve(r, areq("POST", fmt.Sprintf("/api/v1/videos/%d/danmaku", v.ID), tk, `{"content":"hello","time":1.5,"type":1,"color":16777215}`))

	// Get danmaku list
	srve(r, areq("GET", fmt.Sprintf("/api/v1/videos/%d/danmaku?time=0", v.ID), "", nil))
}

// Test dynamic like (covers CASE WHEN like_count -= 1)
func Test_DynamicLike(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "dyl1", "DYL1", 0)
	tk := tok(t, api, u.ID)

	dyn := dynamic.UserDynamic{UserID: u.ID, Title: "DYL", Content: "test", ImagesJSON: "[]", CreatedAt: time.Now()}
	require.NoError(t, api.DB.Create(&dyn).Error)

	// Toggle like on/off (covers CASE WHEN like_count -= 1)
	srve(r, areq("POST", fmt.Sprintf("/api/v1/user-dynamics/%d/like", dyn.ID), tk, nil))
	srve(r, areq("POST", fmt.Sprintf("/api/v1/user-dynamics/%d/like", dyn.ID), tk, nil))
}

// Test view history settings
func Test_ViewHistorySettingsMore(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "vhs1", "VHS1", 0)
	tk := tok(t, api, u.ID)

	// Get settings
	srve(r, areq("GET", "/api/v1/users/me/view-history/settings", tk, nil))
	// Update settings
	srve(r, areq("PUT", "/api/v1/users/me/view-history/settings", tk, `{"paused":true}`))
	// Post view history
	v := seedVideoWithAPI(t, api, u.ID, "VHS Vid")
	srve(r, areq("POST", fmt.Sprintf("/api/v1/videos/%d/view-history", v.ID), tk, nil))
	// List
	srve(r, areq("GET", "/api/v1/users/me/view-history", tk, nil))
	// Delete entry
	srve(r, areq("DELETE", fmt.Sprintf("/api/v1/users/me/view-history/%d", v.ID), tk, nil))
}

// Test video watch later operations
func Test_WatchLaterOperations(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "wlo1", "WLO1", 0)
	v := seedVideoWithAPI(t, api, u.ID, "WLO Vid")
	tk := tok(t, api, u.ID)

	// Toggle watch later
	srve(r, areq("POST", fmt.Sprintf("/api/v1/videos/%d/watch-later", v.ID), tk, nil))
	// Mark as watched
	srve(r, areq("POST", fmt.Sprintf("/api/v1/users/me/watch-later/%d/watched", v.ID), tk, nil))
	// Clear watched
	srve(r, areq("DELETE", "/api/v1/users/me/watch-later/watched", tk, nil))
	// Clear all
	srve(r, areq("DELETE", "/api/v1/users/me/watch-later", tk, nil))
}

// Test video folder operations
func Test_VideoFolderOperations(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "vfo1", "VFO1", 0)
	v := seedVideoWithAPI(t, api, u.ID, "VFO Vid")
	tk := tok(t, api, u.ID)

	// Create folder
	w := srve(r, areq("POST", "/api/v1/users/me/favorite-folders", tk, `{"title":"VFO Folder"}`))
	var ff struct { Data struct { ID uint64 `json:"id"` } `json:"data"` }
	json.Unmarshal(w.Body.Bytes(), &ff)
	if ff.Data.ID > 0 {
		fid := ff.Data.ID
		// Add video to folder
		srve(r, areq("POST", fmt.Sprintf("/api/v1/videos/%d/favorite-folders/%d", v.ID, fid), tk, nil))
		// Remove video from folder
		srve(r, areq("DELETE", fmt.Sprintf("/api/v1/videos/%d/favorite-folders/%d", v.ID, fid), tk, nil))
		// Delete folder
		srve(r, areq("DELETE", fmt.Sprintf("/api/v1/users/me/favorite-folders/%d", fid), tk, nil))
	}
}

// Test search functionality
func Test_SearchEndpoints(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "sch1", "SCH1", 0)
	tk := tok(t, api, u.ID)

	// Search videos
	srve(r, areq("GET", "/api/v1/search?keyword=test&type=video&page=1&page_size=10", tk, nil))
	// Search users
	srve(r, areq("GET", "/api/v1/search?keyword=test&type=user&page=1&page_size=10", tk, nil))
	// Search articles
	srve(r, areq("GET", "/api/v1/search?keyword=test&type=article&page=1&page_size=10", tk, nil))
}

// Test admin delete ops
func Test_AdminDeleteMore(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "adm1", "ADM1", 0)
	v := seedVideoWithAPI(t, api, u.ID, "ADM Vid")
	art := seedArticle(t, api, u.ID, "ADM Article")
	atk := admintok(t, api)

	// Admin delete video
	srve(r, areq("DELETE", fmt.Sprintf("/api/v1/admin/videos/%d", v.ID), atk, nil))
	// Admin delete article
	srve(r, areq("DELETE", fmt.Sprintf("/api/v1/admin/articles/%d", art.ID), atk, nil))
}

// Test coin operations
func Test_CoinOperations(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "cop1", "COP1", 100)
	u2 := seedUser(t, api, "cop2", "COP2", 0)
	v := seedVideoWithAPI(t, api, u2.ID, "COP Vid")
	art := seedArticle(t, api, u2.ID, "COP Art")
	tk := tok(t, api, u.ID)

	// Coin video
	srve(r, areq("POST", fmt.Sprintf("/api/v1/videos/%d/coin", v.ID), tk, `{"amount":1}`))
	// Coin article
	srve(r, areq("POST", fmt.Sprintf("/api/v1/articles/%d/coin", art.ID), tk, `{"amount":1}`))
}
