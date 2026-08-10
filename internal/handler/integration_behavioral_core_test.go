//go:build integration

package handler

import (
	"cakecake/internal/model/danmaku"
	"cakecake/internal/model/user"
	"cakecake/internal/model/video"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestBehavioral_Auth_RegisterLoginRefresh verifies the auth chain with real
// response bodies and DB side effects, including refresh-token rotation.
func TestBehavioral_Auth_RegisterLoginRefresh(t *testing.T) {
	api, r, _ := newTestAPI(t)

	// Register -> 201 + code 0 + a real user row with a bcrypt hash.
	w := covReq(t, r, "POST", "/api/v1/users", "", map[string]string{"username": "behavioral_alice", "password": "password12"})
	covOK(t, w, http.StatusCreated)
	var reg struct {
		Code int `json:"code"`
		Data struct {
			UserID uint64 `json:"user_id"`
			CakeID string `json:"cake_id"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &reg))
	require.Equal(t, 0, reg.Code)
	require.NotZero(t, reg.Data.UserID)
	require.Equal(t, "cake_00000000001", reg.Data.CakeID)

	var dbUser user.User
	require.NoError(t, api.DB.First(&dbUser, reg.Data.UserID).Error)
	require.Equal(t, "behavioral_alice", dbUser.Username)
	require.Contains(t, dbUser.PasswordHash, "$2")

	// Duplicate registration is rejected with the username-exists code.
	w = covReq(t, r, "POST", "/api/v1/users", "", map[string]string{"username": "behavioral_alice", "password": "password12"})
	covOK(t, w, http.StatusBadRequest)
	var dup struct {
		Code int `json:"code"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &dup))
	require.Equal(t, 40006, dup.Code)

	// Wrong password -> 401.
	w = covReq(t, r, "POST", "/api/v1/auth/login", "", map[string]string{"username": "behavioral_alice", "password": "wrongpass1"})
	covOK(t, w, http.StatusUnauthorized)

	// Correct login -> token pair.
	w = covReq(t, r, "POST", "/api/v1/auth/login", "", map[string]string{"username": "behavioral_alice", "password": "password12"})
	covOK(t, w, http.StatusOK)
	var login struct {
		Code int `json:"code"`
		Data struct {
			AccessToken  string `json:"access_token"`
			RefreshToken string `json:"refresh_token"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &login))
	require.Equal(t, 0, login.Code)
	require.NotEmpty(t, login.Data.AccessToken)
	require.NotEmpty(t, login.Data.RefreshToken)

	// Access token works on /users/me.
	w = covReq(t, r, "GET", "/api/v1/users/me", login.Data.AccessToken, nil)
	covOK(t, w, http.StatusOK)
	var me struct {
		Code int `json:"code"`
		Data struct {
			UserID   uint64 `json:"user_id"`
			Username string `json:"username"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &me))
	require.Equal(t, reg.Data.UserID, me.Data.UserID)
	require.Equal(t, "behavioral_alice", me.Data.Username)

	// Refresh rotates the token; the old refresh token is invalidated.
	w = covReq(t, r, "POST", "/api/v1/auth/refresh", "", map[string]string{"refresh_token": login.Data.RefreshToken})
	covOK(t, w, http.StatusOK)
	var ref struct {
		Code int `json:"code"`
		Data struct {
			AccessToken  string `json:"access_token"`
			RefreshToken string `json:"refresh_token"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &ref))
	require.Equal(t, 0, ref.Code)
	require.NotEmpty(t, ref.Data.AccessToken)
	require.NotEmpty(t, ref.Data.RefreshToken)
	require.NotEqual(t, login.Data.RefreshToken, ref.Data.RefreshToken)

	w = covReq(t, r, "POST", "/api/v1/auth/refresh", "", map[string]string{"refresh_token": login.Data.RefreshToken})
	covOK(t, w, http.StatusUnauthorized)
}

// TestBehavioral_Danmaku_PostCooldownSensitiveDB verifies the danmaku write
// path: response body, DB row, counter, cooldown, sensitive-word rejection.
func TestBehavioral_Danmaku_PostCooldownSensitiveDB(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "beh_dm", "行为弹幕", 0)
	owner := seedUser(t, api, "beh_dm_owner", "弹幕UP", 0)
	v := seedVideoWithAPI(t, api, owner.ID, "Behavior Danmaku Video")
	tk := tok(t, api, u.ID)
	postPath := fmt.Sprintf("/api/v1/videos/%d/danmaku", v.ID)

	// Valid danmaku -> full response body + persisted row + counter increment.
	w := covReq(t, r, "POST", postPath, tk, map[string]any{
		"content": "hello 弹幕", "type": "scroll", "color": "#ABCDEF", "video_time": 1.5,
	})
	covOK(t, w, http.StatusOK)
	var posted struct {
		Code int `json:"code"`
		Data struct {
			ID        uint64  `json:"id"`
			Content   string  `json:"content"`
			Color     string  `json:"color"`
			Type      string  `json:"type"`
			FontSize  string  `json:"font_size"`
			VideoTime float64 `json:"video_time"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &posted))
	require.Equal(t, 0, posted.Code)
	require.NotZero(t, posted.Data.ID)
	require.Equal(t, "hello 弹幕", posted.Data.Content)
	require.Equal(t, "#ABCDEF", posted.Data.Color)
	require.Equal(t, "scroll", posted.Data.Type)
	require.Equal(t, "md", posted.Data.FontSize)
	require.InDelta(t, 1.5, posted.Data.VideoTime, 0.0001)

	var dbDm danmaku.Danmaku
	require.NoError(t, api.DB.First(&dbDm, posted.Data.ID).Error)
	require.Equal(t, u.ID, dbDm.UserID)
	require.Equal(t, v.ID, dbDm.VideoID)
	require.Equal(t, "hello 弹幕", dbDm.Content)
	require.Equal(t, "#ABCDEF", dbDm.Color)
	var dbVideo video.Video
	require.NoError(t, api.DB.First(&dbVideo, v.ID).Error)
	require.Equal(t, uint64(1), dbVideo.DanmakuCount)

	// Same user within TTL -> cooldown error code (40004).
	w = covReq(t, r, "POST", postPath, tk, map[string]any{"content": "again", "type": "scroll"})
	covOK(t, w, http.StatusBadRequest)
	var cd struct {
		Code int `json:"code"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &cd))
	require.Equal(t, 40004, cd.Code)

	// Clear cooldown, then sensitive word -> rejected with 40005 and no row.
	ctx := context.Background()
	require.NoError(t, api.Redis.Del(ctx, fmt.Sprintf("danmaku:cooldown:%d:%d", u.ID, v.ID)).Err())
	w = covReq(t, r, "POST", postPath, tk, map[string]any{"content": "contains badword", "type": "scroll"})
	covOK(t, w, http.StatusBadRequest)
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &cd))
	require.Equal(t, 40005, cd.Code)
	var dmCount int64
	require.NoError(t, api.DB.Model(&danmaku.Danmaku{}).Where("video_id = ?", v.ID).Count(&dmCount).Error)
	require.Equal(t, int64(1), dmCount)

	// Validation errors and auth failures.
	w = covReq(t, r, "POST", postPath, tk, map[string]any{"content": "x", "type": "top", "color": "not-a-color"})
	covOK(t, w, http.StatusBadRequest)
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &cd))
	require.Equal(t, 40014, cd.Code)

	w = covReq(t, r, "POST", postPath, "", map[string]any{"content": "x", "type": "scroll"})
	covOK(t, w, http.StatusUnauthorized)

	w = covReq(t, r, "POST", "/api/v1/videos/999999/danmaku", tk, map[string]any{"content": "x", "type": "scroll"})
	covOK(t, w, http.StatusNotFound)
}

// TestBehavioral_VideoLike_ToggleIdempotentDB verifies the like chain:
// response body, DB like row, like_count, and repeated-toggle idempotency.
func TestBehavioral_VideoLike_ToggleIdempotentDB(t *testing.T) {
	api, r, _ := newTestAPI(t)
	viewer := seedUser(t, api, "beh_like", "点赞观众", 0)
	owner := seedUser(t, api, "beh_like_owner", "点赞UP", 0)
	v := seedVideoWithAPI(t, api, owner.ID, "Behavior Like Video")
	tk := tok(t, api, viewer.ID)
	likePath := fmt.Sprintf("/api/v1/videos/%d/like", v.ID)

	like := func() (bool, int) {
		t.Helper()
		w := covReq(t, r, "POST", likePath, tk, nil)
		covOK(t, w, http.StatusOK)
		var resp struct {
			Code int `json:"code"`
			Data struct {
				Liked bool `json:"liked"`
			} `json:"data"`
		}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		require.Equal(t, 0, resp.Code)
		var vv video.Video
		require.NoError(t, api.DB.First(&vv, v.ID).Error)
		return resp.Data.Liked, int(vv.LikeCount)
	}

	liked, count := like()
	require.True(t, liked)
	require.Equal(t, 1, count)
	var likeRows int64
	require.NoError(t, api.DB.Model(&video.VideoLike{}).Where("user_id = ? AND video_id = ?", viewer.ID, v.ID).Count(&likeRows).Error)
	require.Equal(t, int64(1), likeRows)

	// Toggle off -> both the row and the counter disappear.
	liked, count = like()
	require.False(t, liked)
	require.Zero(t, count)
	require.NoError(t, api.DB.Model(&video.VideoLike{}).Where("user_id = ? AND video_id = ?", viewer.ID, v.ID).Count(&likeRows).Error)
	require.Zero(t, likeRows)

	// On again -> single row, counter 1 (idempotent from the user's perspective).
	liked, count = like()
	require.True(t, liked)
	require.Equal(t, 1, count)
	require.NoError(t, api.DB.Model(&video.VideoLike{}).Where("user_id = ? AND video_id = ?", viewer.ID, v.ID).Count(&likeRows).Error)
	require.Equal(t, int64(1), likeRows)

	// Error paths.
	w := covReq(t, r, "POST", likePath, "", nil)
	covOK(t, w, http.StatusUnauthorized)
	w = covReq(t, r, "POST", "/api/v1/videos/999999/like", tk, nil)
	covOK(t, w, http.StatusNotFound)
}

// TestBehavioral_SearchHistory_UpsertDedupTrim verifies the search-history
// chain with response bodies and DB side effects (dedup + newest-first).
func TestBehavioral_SearchHistory_UpsertDedupTrim(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "beh_search", "搜索用户", 0)
	tk := tok(t, api, u.ID)
	postPath := "/api/v1/users/me/search-history"

	// First keyword -> one row.
	w := covReq(t, r, "POST", postPath, tk, map[string]string{"keyword": "golang"})
	covOK(t, w, http.StatusOK)
	var resp struct {
		Code int `json:"code"`
		Data struct {
			Keywords []string `json:"keywords"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, []string{"golang"}, resp.Data.Keywords)

	// Duplicate keyword -> still one row (idempotent upsert), moved to top.
	w = covReq(t, r, "POST", postPath, tk, map[string]string{"keyword": "golang"})
	covOK(t, w, http.StatusOK)
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, []string{"golang"}, resp.Data.Keywords)

	// Second keyword -> newest first.
	w = covReq(t, r, "POST", postPath, tk, map[string]string{"keyword": "rust"})
	covOK(t, w, http.StatusOK)
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, []string{"rust", "golang"}, resp.Data.Keywords)

	// Re-posting the old keyword moves it to the top without duplication.
	time.Sleep(2 * time.Millisecond)
	w = covReq(t, r, "POST", postPath, tk, map[string]string{"keyword": "golang"})
	covOK(t, w, http.StatusOK)
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, []string{"golang", "rust"}, resp.Data.Keywords)

	// DB has exactly two rows.
	var n int64
	require.NoError(t, api.DB.Table("user_search_histories").Where("user_id = ?", u.ID).Count(&n).Error)
	require.Equal(t, int64(2), n)

	// Unauthorized.
	w = covReq(t, r, "GET", postPath, "", nil)
	covOK(t, w, http.StatusUnauthorized)
}
