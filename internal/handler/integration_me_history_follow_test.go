//go:build integration

package handler

import (
	"cakecake/internal/model/article"
	"cakecake/internal/model/extra"
	"cakecake/internal/model/user"
	"cakecake/internal/model/video"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

func TestMe_ProfileEndpoints(t *testing.T) {
	api, r, token := setupHandlerIntegrationDB(t)
	hash, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.MinCost)
	require.NoError(t, err)
	require.NoError(t, api.DB.Create(&user.User{
		ID: 1, Username: "u1", PasswordHash: string(hash), CoinBalanceTenths: 230,
	}).Error)

	// GetMe.
	w := doReq(r, "GET", "/api/v1/users/me", token, "", nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	// Update profile.
	w = doJSON(r, "PUT", "/api/v1/users/me/profile", token, map[string]interface{}{
		"nickname": "nick", "sign": "hi", "gender": "female", "birthday": "2000-01-01",
	})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	// Update username.
	w = doJSON(r, "PUT", "/api/v1/users/me", token, map[string]interface{}{"username": "newname"})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	// Too short.
	w = doJSON(r, "PUT", "/api/v1/users/me", token, map[string]interface{}{"username": "ab"})
	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())

	// Announcement.
	w = doJSON(r, "PUT", "/api/v1/users/me/announcement", token, map[string]interface{}{"announcement": "hello all"})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	// Space privacy.
	w = doReq(r, "GET", "/api/v1/users/me/space-privacy", token, "", nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	w = doJSON(r, "PUT", "/api/v1/users/me/space-privacy", token, map[string]interface{}{"public_fans": true})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	// Password change.
	w = doJSON(r, "PUT", "/api/v1/users/me/password", token, map[string]interface{}{
		"old_password": "password123", "new_password": "newpassword456",
	})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	// Wrong old password.
	w = doJSON(r, "PUT", "/api/v1/users/me/password", token, map[string]interface{}{
		"old_password": "wrong", "new_password": "newpassword456",
	})
	require.Equal(t, http.StatusForbidden, w.Code, w.Body.String())

	// Coin ledger + daily rewards.
	require.NoError(t, api.DB.Create(&user.CoinLedger{UserID: 1, DeltaTenths: 10}).Error)
	w = doReq(r, "GET", "/api/v1/users/me/coin-ledger", token, "", nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	w = doReq(r, "GET", "/api/v1/users/me/daily-rewards", token, "", nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
}

func TestViewHistory_Endpoints(t *testing.T) {
	api, r, token := setupHandlerIntegrationDB(t)
	require.NoError(t, api.DB.Create(&user.User{ID: 1, Username: "u1", PasswordHash: "x", CoinBalanceTenths: 230}).Error)
	require.NoError(t, api.DB.Create(&video.Video{ID: 10, UserID: 2, Title: "v", Status: video.StatusPublished, DurationSec: 100}).Error)

	// Record view history.
	w := doJSON(r, "POST", "/api/v1/videos/10/view-history", token, map[string]interface{}{
		"progress_sec": 20.0, "duration_sec": 100, "device": "mobile",
	})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	// List.
	w = doReq(r, "GET", "/api/v1/users/me/view-history", token, "", nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var n int64
	require.NoError(t, api.DB.Model(&extra.VideoViewHistory{}).Count(&n).Error)
	require.Equal(t, int64(1), n)

	// Settings.
	w = doReq(r, "GET", "/api/v1/users/me/view-history/settings", token, "", nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	w = doJSON(r, "PUT", "/api/v1/users/me/view-history/settings", token, map[string]interface{}{"paused": true})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	// Paused recording returns Recorded:false.
	w = doJSON(r, "POST", "/api/v1/videos/10/view-history", token, map[string]interface{}{"progress_sec": 1})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	// Delete entries.
	var row extra.VideoViewHistory
	require.NoError(t, api.DB.First(&row).Error)
	w = doReq(r, "DELETE", fmt.Sprintf("/api/v1/users/me/view-history/%d", row.VideoID), token, "", nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	w = doReq(r, "DELETE", "/api/v1/users/me/view-history/articles/99", token, "", nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	// Clear.
	require.NoError(t, api.DB.Create(&extra.VideoViewHistory{UserID: 1, VideoID: 10, ViewedAt: time.Now()}).Error)
	w = doReq(r, "DELETE", "/api/v1/users/me/view-history", token, "", nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NoError(t, api.DB.Model(&extra.VideoViewHistory{}).Count(&n).Error)
	require.Zero(t, n)

	// Unpause so the missing-video check reaches the video lookup.
	w = doJSON(r, "PUT", "/api/v1/users/me/view-history/settings", token, map[string]interface{}{"paused": false})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	// Missing video.
	w = doJSON(r, "POST", "/api/v1/videos/999/view-history", token, map[string]interface{}{})
	require.Equal(t, http.StatusNotFound, w.Code, w.Body.String())
}

func TestArticleComment_Endpoints(t *testing.T) {
	api, r, token := setupHandlerIntegrationDB(t)
	require.NoError(t, api.DB.Create(&user.User{ID: 1, Username: "u1", PasswordHash: "x", CoinBalanceTenths: 230}).Error)
	require.NoError(t, api.DB.Create(&article.Article{ID: 20, UserID: 1, Title: "a", BodyMD: "b", Status: article.StatusPublished}).Error)

	w := doJSON(r, "POST", "/api/v1/articles/20/comments", token, map[string]interface{}{"content": "nice", "parent_id": 0})
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	var cr struct {
		Data struct {
			ID uint64 `json:"id"`
		} `json:"data"`
	}
	decodeBody(t, w, &cr)
	acID := cr.Data.ID

	w = doJSON(r, "POST", fmt.Sprintf("/api/v1/article-comments/%d/like", acID), token, nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	w = doJSON(r, "POST", fmt.Sprintf("/api/v1/article-comments/%d/dislike", acID), token, nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	w = doJSON(r, "POST", fmt.Sprintf("/api/v1/article-comments/%d/pin", acID), token, nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	w = doJSON(r, "POST", fmt.Sprintf("/api/v1/article-comments/%d/approve", acID), token, nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	w = doJSON(r, "POST", fmt.Sprintf("/api/v1/article-comments/%d/ignore-curated", acID), token, nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	w = doReq(r, "DELETE", fmt.Sprintf("/api/v1/article-comments/%d", acID), token, "", nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
}

func TestFollowGroups_Endpoints(t *testing.T) {
	api, r, token := setupHandlerIntegrationDB(t)
	require.NoError(t, api.DB.Create(&user.User{ID: 1, Username: "u1", PasswordHash: "x", CoinBalanceTenths: 230}).Error)
	require.NoError(t, api.DB.Create(&user.User{ID: 2, Username: "u2", PasswordHash: "x", CoinBalanceTenths: 230}).Error)
	require.NoError(t, api.DB.Create(&user.UserFollow{FollowerID: 1, FolloweeID: 2}).Error)

	// Create group.
	w := doJSON(r, "POST", "/api/v1/users/me/follow-groups", token, map[string]interface{}{"name": "friends"})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var cr struct {
		Data struct {
			ID uint64 `json:"id"`
		} `json:"data"`
	}
	decodeBody(t, w, &cr)
	groupID := cr.Data.ID

	// List.
	w = doReq(r, "GET", "/api/v1/users/me/follow-groups", token, "", nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	// Update + add member + list member groups.
	w = doJSON(r, "PUT", fmt.Sprintf("/api/v1/users/me/follow-groups/%d", groupID), token, map[string]interface{}{"name": "bffs"})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	w = doJSON(r, "POST", fmt.Sprintf("/api/v1/users/me/follow-groups/%d/members", groupID), token, map[string]interface{}{"followee_id": 2})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	w = doReq(r, "GET", "/api/v1/users/me/following/2/groups", token, "", nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	// Remove member + delete group.
	w = doReq(r, "DELETE", fmt.Sprintf("/api/v1/users/me/follow-groups/%d/members/2", groupID), token, "", nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	w = doReq(r, "DELETE", fmt.Sprintf("/api/v1/users/me/follow-groups/%d", groupID), token, "", nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
}
