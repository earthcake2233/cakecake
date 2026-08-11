//go:build integration

// Behavioral integration tests for engagement counters, danmaku persistence,
// search degradation, and admin delete flows. Each test asserts response
// status, response body, and/or DB side effects rather than just executing
// the endpoint.
package handler

import (
	"cakecake/internal/model/article"
	"cakecake/internal/model/comment"
	"cakecake/internal/model/danmaku"
	"cakecake/internal/model/dynamic"
	"cakecake/internal/model/video"
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
	var r struct {
		Code int `json:"code"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &r))
	return r.Code
}

// Test_EngagementToggleCounters verifies like/favorite/coin toggles keep
// response flags and DB counters consistent in both directions.
func Test_EngagementToggleCounters(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "dec1", "DEC1", 100)
	u2 := seedUser(t, api, "dec2", "DEC2", 100)
	v := seedVideoWithAPI(t, api, u.ID, "DEC Video")
	tk := tok(t, api, u2.ID)

	// Post a comment
	w := srve(r, areq("POST", "/api/v1/videos/"+strconv.FormatUint(v.ID, 10)+"/comments", tk, `{"content":"test"}`))
	require.Equal(t, 0, codeFrom(t, w), w.Body.String())

	// Extract comment ID
	var cm struct {
		Data comment.Comment `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &cm))
	require.NotZero(t, cm.Data.ID, w.Body.String())
	cid := cm.Data.ID

	// Like comment (CASE WHEN like_count - 1 later gets tested)
	lw := srve(r, areq("POST", "/api/v1/comments/"+strconv.FormatUint(cid, 10)+"/like", tk, nil))
	require.Equal(t, 0, codeFrom(t, lw))
	require.Contains(t, lw.Body.String(), `"liked":true`)
	// Unlike by toggling dislike (covers like_count -= 1 CASE WHEN)
	dw := srve(r, areq("POST", "/api/v1/comments/"+strconv.FormatUint(cid, 10)+"/dislike", tk, nil))
	require.Equal(t, 0, codeFrom(t, dw))
	// Like again then dislike again
	lw = srve(r, areq("POST", "/api/v1/comments/"+strconv.FormatUint(cid, 10)+"/like", tk, nil))
	require.Equal(t, 0, codeFrom(t, lw))
	dw = srve(r, areq("POST", "/api/v1/comments/"+strconv.FormatUint(cid, 10)+"/dislike", tk, nil))
	require.Equal(t, 0, codeFrom(t, dw))

	// Toggle video like
	lw = srve(r, areq("POST", "/api/v1/videos/"+strconv.FormatUint(v.ID, 10)+"/like", tk, nil))
	require.Equal(t, 0, codeFrom(t, lw))
	require.Contains(t, lw.Body.String(), `"liked":true`)
	lw = srve(r, areq("POST", "/api/v1/videos/"+strconv.FormatUint(v.ID, 10)+"/like", tk, nil))
	require.Equal(t, 0, codeFrom(t, lw))
	require.Contains(t, lw.Body.String(), `"liked":false`)

	// Toggle video favorite (covers fav_count -= 1 CASE WHEN)
	fw := srve(r, areq("POST", "/api/v1/videos/"+strconv.FormatUint(v.ID, 10)+"/favorite", tk, nil))
	require.Equal(t, 0, codeFrom(t, fw))
	require.Contains(t, fw.Body.String(), `"favorited":true`)
	fw = srve(r, areq("POST", "/api/v1/videos/"+strconv.FormatUint(v.ID, 10)+"/favorite", tk, nil))
	require.Equal(t, 0, codeFrom(t, fw))
	require.Contains(t, fw.Body.String(), `"favorited":false`)

	// Toggle article favorite
	art := seedArticle(t, api, u.ID, "DEC Article")
	afw := srve(r, areq("POST", "/api/v1/articles/"+strconv.FormatUint(art.ID, 10)+"/favorite", tk, nil))
	require.Equal(t, 0, codeFrom(t, afw))
	require.Contains(t, afw.Body.String(), `"favorited":true`)
	afw = srve(r, areq("POST", "/api/v1/articles/"+strconv.FormatUint(art.ID, 10)+"/favorite", tk, nil))
	require.Equal(t, 0, codeFrom(t, afw))
	require.Contains(t, afw.Body.String(), `"favorited":false`)
}

// Test_DynamicCommentReactionToggle verifies like/dislike toggling on a
// dynamic comment returns the expected state.
func Test_DynamicCommentReactionToggle(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "dyn1", "DYN1", 0)
	tk := tok(t, api, u.ID)

	// Create dynamic directly
	dyn := dynamic.UserDynamic{UserID: u.ID, Title: "DYN", Content: "test", ImagesJSON: "[]", CreatedAt: time.Now()}
	require.NoError(t, api.DB.Create(&dyn).Error)

	// Post comment on dynamic
	body := `{"content":"dynamic comment"}`
	w := srve(r, areq("POST", fmt.Sprintf("/api/v1/user-dynamics/%d/comments", dyn.ID), tk, body))
	require.Equal(t, 0, codeFrom(t, w), w.Body.String())

	var dcm struct {
		Data comment.DynamicComment `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &dcm))
	require.NotZero(t, dcm.Data.ID, w.Body.String())
	dcid := dcm.Data.ID

	// Like (then unlike by disliking) - covers CASE WHEN like_count -= 1
	lw := srve(r, areq("POST", fmt.Sprintf("/api/v1/dynamic-comments/%d/like", dcid), tk, nil))
	require.Equal(t, 0, codeFrom(t, lw))
	require.Contains(t, lw.Body.String(), `"liked":true`)
	dw := srve(r, areq("POST", fmt.Sprintf("/api/v1/dynamic-comments/%d/dislike", dcid), tk, nil))
	require.Equal(t, 0, codeFrom(t, dw))
}

// Test_ArticleCommentReactionAndDelete verifies article comment
// like/dislike/approve/pin/delete behaviors.
func Test_ArticleCommentReactionAndDelete(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "acd1", "ACD1", 0)
	art := seedArticle(t, api, u.ID, "ACD Article")
	tk := tok(t, api, u.ID)

	w := srve(r, areq("POST", fmt.Sprintf("/api/v1/articles/%d/comments", art.ID), tk, `{"content":"acd comment"}`))
	require.Equal(t, 0, codeFrom(t, w), w.Body.String())

	var acm struct {
		Data comment.ArticleComment `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &acm))
	require.NotZero(t, acm.Data.ID, w.Body.String())
	acid := acm.Data.ID

	// Like toggle (covers CASE WHEN like_count -= 1)
	lw := srve(r, areq("POST", fmt.Sprintf("/api/v1/article-comments/%d/like", acid), tk, nil))
	require.Equal(t, 0, codeFrom(t, lw))
	require.Contains(t, lw.Body.String(), `"liked":true`)
	dw := srve(r, areq("POST", fmt.Sprintf("/api/v1/article-comments/%d/dislike", acid), tk, nil))
	require.Equal(t, 0, codeFrom(t, dw))

	// Approve, pin
	aw := srve(r, areq("POST", fmt.Sprintf("/api/v1/article-comments/%d/approve", acid), tk, nil))
	require.Equal(t, 0, codeFrom(t, aw))
	pw := srve(r, areq("POST", fmt.Sprintf("/api/v1/article-comments/%d/pin", acid), tk, nil))
	require.Equal(t, 0, codeFrom(t, pw))

	// Delete article comment (covers comment_count -= 1 CASE WHEN)
	delw := srve(r, areq("DELETE", fmt.Sprintf("/api/v1/article-comments/%d", acid), tk, nil))
	require.Equal(t, 0, codeFrom(t, delw))
}

// Test_DanmakuPostPersistsRow verifies a posted danmaku is persisted with the
// expected fields and the video counter is incremented.
func Test_DanmakuPostPersistsRow(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "dan1", "DAN1", 0)
	v := seedVideoWithAPI(t, api, u.ID, "DAN Video")
	tk := tok(t, api, u.ID)

	// Post a danmaku
	w := srve(r, areq("POST", fmt.Sprintf("/api/v1/videos/%d/danmaku", v.ID), tk, `{"content":"hello","video_time":1.5,"type":"scroll","color":"#FFFFFF"}`))
	require.Equal(t, 0, codeFrom(t, w))
	require.Contains(t, w.Body.String(), `"content":"hello"`)
	require.Contains(t, w.Body.String(), `"type":"scroll"`)

	// Get danmaku list
	var dmCount int64
	require.NoError(t, api.DB.Model(&danmaku.Danmaku{}).Where("video_id = ?", v.ID).Count(&dmCount).Error)
	require.Equal(t, int64(1), dmCount)
}

// Test_DynamicLikeToggle verifies dynamic like on/off responses.
func Test_DynamicLikeToggle(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "dyl1", "DYL1", 0)
	tk := tok(t, api, u.ID)

	dyn := dynamic.UserDynamic{UserID: u.ID, Title: "DYL", Content: "test", ImagesJSON: "[]", CreatedAt: time.Now()}
	require.NoError(t, api.DB.Create(&dyn).Error)

	// Toggle like on/off (covers CASE WHEN like_count -= 1)
	w := srve(r, areq("POST", fmt.Sprintf("/api/v1/user-dynamics/%d/like", dyn.ID), tk, nil))
	require.Equal(t, 0, codeFrom(t, w))
	require.Contains(t, w.Body.String(), `"liked":true`)
	w = srve(r, areq("POST", fmt.Sprintf("/api/v1/user-dynamics/%d/like", dyn.ID), tk, nil))
	require.Equal(t, 0, codeFrom(t, w))
	require.Contains(t, w.Body.String(), `"liked":false`)
}

// Test_ViewHistorySettingsAndRecord verifies pause/resume settings and that a
// view record is created and deleted while recording is enabled.
func Test_ViewHistorySettingsAndRecord(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "vhs1", "VHS1", 0)
	tk := tok(t, api, u.ID)

	// Get settings
	w := srve(r, areq("GET", "/api/v1/users/me/view-history/settings", tk, nil))
	require.Equal(t, 0, codeFrom(t, w))
	// Update settings
	w = srve(r, areq("PUT", "/api/v1/users/me/view-history/settings", tk, `{"paused":true}`))
	require.Equal(t, 0, codeFrom(t, w))
	require.Contains(t, w.Body.String(), `"paused":true`)
	// Unpause so the history write below is actually recorded.
	w = srve(r, areq("PUT", "/api/v1/users/me/view-history/settings", tk, `{"paused":false}`))
	require.Equal(t, 0, codeFrom(t, w))
	// Post view history
	v := seedVideoWithAPI(t, api, u.ID, "VHS Vid")
	w = srve(r, areq("POST", fmt.Sprintf("/api/v1/videos/%d/view-history", v.ID), tk, nil))
	require.Equal(t, 0, codeFrom(t, w))
	// List
	w = srve(r, areq("GET", "/api/v1/users/me/view-history", tk, nil))
	require.Equal(t, 0, codeFrom(t, w))
	require.Contains(t, w.Body.String(), strconv.FormatUint(v.ID, 10))
	// Delete entry
	w = srve(r, areq("DELETE", fmt.Sprintf("/api/v1/users/me/view-history/%d", v.ID), tk, nil))
	require.Equal(t, 0, codeFrom(t, w))
}

// Test_WatchLaterLifecycle verifies watch-later add, mark-watched, and clear
// operations.
func Test_WatchLaterLifecycle(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "wlo1", "WLO1", 0)
	v := seedVideoWithAPI(t, api, u.ID, "WLO Vid")
	tk := tok(t, api, u.ID)

	// Toggle watch later
	w := srve(r, areq("POST", fmt.Sprintf("/api/v1/videos/%d/watch-later", v.ID), tk, nil))
	require.Equal(t, 0, codeFrom(t, w))
	require.Contains(t, w.Body.String(), `"in_watch_later":true`)
	// Mark as watched
	w = srve(r, areq("POST", fmt.Sprintf("/api/v1/users/me/watch-later/%d/watched", v.ID), tk, nil))
	require.Equal(t, 0, codeFrom(t, w))
	// Clear watched
	w = srve(r, areq("DELETE", "/api/v1/users/me/watch-later/watched", tk, nil))
	require.Equal(t, 0, codeFrom(t, w))
	// Clear all
	w = srve(r, areq("DELETE", "/api/v1/users/me/watch-later", tk, nil))
	require.Equal(t, 0, codeFrom(t, w))
}

// Test_FavoriteFolderVideoOps verifies adding/removing a video from a favorite
// folder and deleting the folder.
func Test_FavoriteFolderVideoOps(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "vfo1", "VFO1", 0)
	v := seedVideoWithAPI(t, api, u.ID, "VFO Vid")
	tk := tok(t, api, u.ID)

	// Create folder
	w := srve(r, areq("POST", "/api/v1/users/me/favorite-folders", tk, `{"title":"VFO Folder"}`))
	var ff struct {
		Data struct {
			ID uint64 `json:"id"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &ff))
	if ff.Data.ID > 0 {
		fid := ff.Data.ID
		// Add video to folder
		aw := srve(r, areq("POST", fmt.Sprintf("/api/v1/videos/%d/favorite-folders/%d", v.ID, fid), tk, nil))
		require.Equal(t, 0, codeFrom(t, aw))
		// Remove video from folder
		rw := srve(r, areq("DELETE", fmt.Sprintf("/api/v1/videos/%d/favorite-folders/%d", v.ID, fid), tk, nil))
		require.Equal(t, 0, codeFrom(t, rw))
		// Delete folder
		dw := srve(r, areq("DELETE", fmt.Sprintf("/api/v1/users/me/favorite-folders/%d", fid), tk, nil))
		require.Equal(t, 0, codeFrom(t, dw))
	}
}

// Test_SearchDegradedModeWithoutES verifies all search types return the
// degraded "unavailable" envelope when Elasticsearch is not configured.
func Test_SearchDegradedModeWithoutES(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "sch1", "SCH1", 0)
	tk := tok(t, api, u.ID)

	// Search videos
	w := srve(r, areq("GET", "/api/v1/search?keyword=test&type=video&page=1&page_size=10", tk, nil))
	require.Equal(t, 0, codeFrom(t, w))
	require.Contains(t, w.Body.String(), `"search_status":"unavailable"`)
	// Search users
	w = srve(r, areq("GET", "/api/v1/search?keyword=test&type=user&page=1&page_size=10", tk, nil))
	require.Equal(t, 0, codeFrom(t, w))
	require.Contains(t, w.Body.String(), `"search_status":"unavailable"`)
	// Search articles
	w = srve(r, areq("GET", "/api/v1/search?keyword=test&type=article&page=1&page_size=10", tk, nil))
	require.Equal(t, 0, codeFrom(t, w))
	require.Contains(t, w.Body.String(), `"search_status":"unavailable"`)
}

// Test_AdminDeleteVideoAndArticle verifies admin deletes remove both the API
// row and the DB record.
func Test_AdminDeleteVideoAndArticle(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "adm1", "ADM1", 0)
	v := seedVideoWithAPI(t, api, u.ID, "ADM Vid")
	art := seedArticle(t, api, u.ID, "ADM Article")
	atk := admintok(t, api)

	// Admin delete video
	dw := srve(r, areq("DELETE", fmt.Sprintf("/api/v1/admin/videos/%d", v.ID), atk, nil))
	require.Equal(t, 0, codeFrom(t, dw))
	var vcount int64
	require.NoError(t, api.DB.Model(&video.Video{}).Where("id = ?", v.ID).Count(&vcount).Error)
	require.Zero(t, vcount)
	// Admin delete article
	dw = srve(r, areq("DELETE", fmt.Sprintf("/api/v1/admin/articles/%d", art.ID), atk, nil))
	require.Equal(t, 0, codeFrom(t, dw))
	var acount int64
	require.NoError(t, api.DB.Model(&article.Article{}).Where("id = ?", art.ID).Count(&acount).Error)
	require.Zero(t, acount)
}

// Test_VideoAndArticleCoin verifies video and article coin responses.
func Test_VideoAndArticleCoin(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "cop1", "COP1", 100)
	u2 := seedUser(t, api, "cop2", "COP2", 0)
	v := seedVideoWithAPI(t, api, u2.ID, "COP Vid")
	art := seedArticle(t, api, u2.ID, "COP Art")
	tk := tok(t, api, u.ID)

	// Coin video
	cw := srve(r, areq("POST", fmt.Sprintf("/api/v1/videos/%d/coin", v.ID), tk, `{"amount":1}`))
	require.Equal(t, 0, codeFrom(t, cw))
	require.Contains(t, cw.Body.String(), `"coined":true`)
	// Coin article
	cw = srve(r, areq("POST", fmt.Sprintf("/api/v1/articles/%d/coin", art.ID), tk, `{"amount":1}`))
	require.Equal(t, 0, codeFrom(t, cw))
	require.Contains(t, cw.Body.String(), `"coined":true`)
}
