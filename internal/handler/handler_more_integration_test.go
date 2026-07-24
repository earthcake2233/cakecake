//go:build integration

package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"minibili/internal/model"
)

// ---------------------------------------------------------------------------
// 1. User Public Profile & Listings
// ---------------------------------------------------------------------------

func TestGetUserPublic(t *testing.T) {
	api, r, _ := newTestAPI(t)

	owner := model.User{Username: "spaceowner", PasswordHash: "hash", Nickname: "SpaceOwner",
		Sign: "hello world", Gender: "male", Birthday: "2000-01-01", CoinBalanceTenths: 230}
	require.NoError(t, api.DB.Create(&owner).Error)
	api.DB.Model(&owner).Update("cake_id", model.FormatCakeID(owner.ID))

	vid := model.Video{UserID: owner.ID, Title: "Space Video", Status: "published",
		VideoURL: "https://cdn.example.com/v.mp4", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	require.NoError(t, api.DB.Create(&vid).Error)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/space/%d", owner.ID), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var resp struct {
		Code int   `json:"code"`
		Data gin.H `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)
	require.Equal(t, float64(owner.ID), resp.Data["user_id"])
	require.Equal(t, "SpaceOwner", resp.Data["nickname"])
	require.Equal(t, false, resp.Data["is_owner"])

	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/space/99999", nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	require.Equal(t, http.StatusNotFound, w2.Code, w2.Body.String())

	req3 := httptest.NewRequest(http.MethodGet, "/api/v1/space/abc", nil)
	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, req3)
	require.Equal(t, http.StatusBadRequest, w3.Code, w3.Body.String())
}

func TestListUserPublishedVideos(t *testing.T) {
	api, r, _ := newTestAPI(t)

	owner := model.User{Username: "vidowner", PasswordHash: "hash", CoinBalanceTenths: 230}
	require.NoError(t, api.DB.Create(&owner).Error)

	for i := 0; i < 3; i++ {
		v := model.Video{UserID: owner.ID, Title: fmt.Sprintf("Pub %d", i), Status: "published",
			VideoURL: "https://cdn.example.com/v.mp4", CreatedAt: time.Now(), UpdatedAt: time.Now()}
		require.NoError(t, api.DB.Create(&v).Error)
	}
	draft := model.Video{UserID: owner.ID, Title: "Draft", Status: "draft",
		CreatedAt: time.Now(), UpdatedAt: time.Now()}
	require.NoError(t, api.DB.Create(&draft).Error)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/space/%d/videos", owner.ID), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var resp struct {
		Code int `json:"code"`
		Data struct {
			Items []gin.H `json:"items"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)
	require.Len(t, resp.Data.Items, 3)
}
func TestListUserFavoriteFolders(t *testing.T) {
	api, r, _ := newTestAPI(t)

	owner := model.User{Username: "favfolderuser", PasswordHash: "hash", CoinBalanceTenths: 230}
	require.NoError(t, api.DB.Create(&owner).Error)

	folder := model.FavoriteFolder{UserID: owner.ID, Title: "My Favs", IsPublic: true}
	require.NoError(t, api.DB.Create(&folder).Error)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/space/%d/favorite-folders", owner.ID), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var resp struct {
		Code int   `json:"code"`
		Data gin.H `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)

	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/space/0/favorite-folders", nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	require.Equal(t, http.StatusBadRequest, w2.Code, w2.Body.String())
}

func TestListUserFavorites(t *testing.T) {
	api, r, _ := newTestAPI(t)

	owner := model.User{Username: "favuser", PasswordHash: "hash", PrivacyPublicFavorites: true, CoinBalanceTenths: 230}
	require.NoError(t, api.DB.Create(&owner).Error)

	uploader := model.User{Username: "uploader1", PasswordHash: "hash", CoinBalanceTenths: 230}
	require.NoError(t, api.DB.Create(&uploader).Error)

	v := model.Video{UserID: uploader.ID, Title: "Fav Video", Status: "published",
		VideoURL: "https://cdn.example.com/v.mp4", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	require.NoError(t, api.DB.Create(&v).Error)

	folder := model.FavoriteFolder{UserID: owner.ID, Title: "Default", IsPublic: true, IsDefault: true}
	require.NoError(t, api.DB.Create(&folder).Error)

	fav := model.VideoFavorite{UserID: owner.ID, VideoID: v.ID, FolderID: folder.ID}
	require.NoError(t, api.DB.Create(&fav).Error)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/space/%d/favorites", owner.ID), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var resp struct {
		Code int   `json:"code"`
		Data gin.H `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)
	items, ok := resp.Data["items"].([]interface{})
	require.True(t, ok)
	require.GreaterOrEqual(t, len(items), 1)
}

func TestListUserRecentCoinVideos(t *testing.T) {
	api, r, jm := newTestAPI(t)

	owner := model.User{Username: "coinuser1", PasswordHash: "hash", PrivacyPublicRecentCoins: true, CoinBalanceTenths: 230}
	require.NoError(t, api.DB.Create(&owner).Error)
	access, _, _, err := jm.IssuePair(owner.ID)
	require.NoError(t, err)

	uploader := model.User{Username: "coinup1", PasswordHash: "hash", CoinBalanceTenths: 2300}
	require.NoError(t, api.DB.Create(&uploader).Error)

	v := model.Video{UserID: uploader.ID, Title: "Coined Video", Status: "published",
		VideoURL: "https://cdn.example.com/v.mp4", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	require.NoError(t, api.DB.Create(&v).Error)

	coin := model.VideoCoin{UserID: owner.ID, VideoID: v.ID, Amount: 1}
	require.NoError(t, api.DB.Create(&coin).Error)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/space/%d/recent-coins", owner.ID), nil)
	req.Header.Set("Authorization", "Bearer "+access)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var resp struct {
		Code int   `json:"code"`
		Data gin.H `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)
	items, ok := resp.Data["items"].([]interface{})
	require.True(t, ok)
	require.GreaterOrEqual(t, len(items), 1)
}

func TestListUserFollowing(t *testing.T) {
	api, r, _ := newTestAPI(t)

	owner := model.User{Username: "followowner1", PasswordHash: "hash", PrivacyPublicFollowing: true, CoinBalanceTenths: 230}
	require.NoError(t, api.DB.Create(&owner).Error)

	target := model.User{Username: "followtarget1", PasswordHash: "hash", CoinBalanceTenths: 230}
	require.NoError(t, api.DB.Create(&target).Error)

	follow := model.UserFollow{FollowerID: owner.ID, FolloweeID: target.ID}
	require.NoError(t, api.DB.Create(&follow).Error)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/space/%d/following", owner.ID), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var resp struct {
		Code int   `json:"code"`
		Data gin.H `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)
	items, ok := resp.Data["items"].([]interface{})
	require.True(t, ok)
	require.GreaterOrEqual(t, len(items), 1)
}

func TestListUserFollowers(t *testing.T) {
	api, r, _ := newTestAPI(t)

	owner := model.User{Username: "followowner2", PasswordHash: "hash", PrivacyPublicFans: true, CoinBalanceTenths: 230}
	require.NoError(t, api.DB.Create(&owner).Error)

	follower := model.User{Username: "follower1", PasswordHash: "hash", CoinBalanceTenths: 230}
	require.NoError(t, api.DB.Create(&follower).Error)

	follow := model.UserFollow{FollowerID: follower.ID, FolloweeID: owner.ID}
	require.NoError(t, api.DB.Create(&follow).Error)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/space/%d/followers", owner.ID), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var resp struct {
		Code int   `json:"code"`
		Data gin.H `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)
}

// ---------------------------------------------------------------------------
// 2. Video Details & Engagement
// ---------------------------------------------------------------------------

func TestGetVideoDetail(t *testing.T) {
	api, r, jm := newTestAPI(t)

	owner := model.User{Username: "videowner1", PasswordHash: "hash", CoinBalanceTenths: 230}
	require.NoError(t, api.DB.Create(&owner).Error)

	v := model.Video{UserID: owner.ID, Title: "Detail Video", Status: "published",
		VideoURL: "https://cdn.example.com/v.mp4", DurationSec: 120,
		CreatedAt: time.Now(), UpdatedAt: time.Now()}
	require.NoError(t, api.DB.Create(&v).Error)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/videos/%d", v.ID), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var resp struct {
		Code int   `json:"code"`
		Data gin.H `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)
	require.Equal(t, float64(v.ID), resp.Data["id"])
	require.Equal(t, "Detail Video", resp.Data["title"])
	require.Equal(t, "published", resp.Data["status"])

	viewer := model.User{Username: "viewer1", PasswordHash: "hash", CoinBalanceTenths: 230}
	require.NoError(t, api.DB.Create(&viewer).Error)
	access, _, _, err := jm.IssuePair(viewer.ID)
	require.NoError(t, err)

	req2 := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/videos/%d", v.ID), nil)
	req2.Header.Set("Authorization", "Bearer "+access)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	require.Equal(t, http.StatusOK, w2.Code, w2.Body.String())

	var resp2 struct {
		Code int   `json:"code"`
		Data gin.H `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w2.Body.Bytes(), &resp2))
	require.Equal(t, 0, resp2.Code)
	require.Equal(t, false, resp2.Data["liked_by_me"])
	require.Equal(t, false, resp2.Data["in_watch_later"])
}

func TestToggleVideoLike(t *testing.T) {
	api, r, jm := newTestAPI(t)

	owner := model.User{Username: "vlikeowner", PasswordHash: "hash", CoinBalanceTenths: 230}
	require.NoError(t, api.DB.Create(&owner).Error)

	v := model.Video{UserID: owner.ID, Title: "Like Video", Status: "published",
		VideoURL: "https://cdn.example.com/v.mp4", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	require.NoError(t, api.DB.Create(&v).Error)

	viewer := model.User{Username: "vlikeviewer", PasswordHash: "hash", CoinBalanceTenths: 230}
	require.NoError(t, api.DB.Create(&viewer).Error)
	access, _, _, err := jm.IssuePair(viewer.ID)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/videos/%d/like", v.ID), nil)
	req.Header.Set("Authorization", "Bearer "+access)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var resp struct {
		Code int   `json:"code"`
		Data gin.H `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)
	require.Equal(t, true, resp.Data["liked"])

	var like model.VideoLike
	err = api.DB.Where("user_id = ? AND video_id = ?", viewer.ID, v.ID).First(&like).Error
	require.NoError(t, err)

	req2 := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/videos/%d/like", v.ID), nil)
	req2.Header.Set("Authorization", "Bearer "+access)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	require.Equal(t, http.StatusOK, w2.Code, w2.Body.String())

	var resp2 struct {
		Code int   `json:"code"`
		Data gin.H `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w2.Body.Bytes(), &resp2))
	require.Equal(t, 0, resp2.Code)
	require.Equal(t, false, resp2.Data["liked"])
}

func TestPostVideoCoin(t *testing.T) {
	api, r, jm := newTestAPI(t)

	owner := model.User{Username: "vcoinowner", PasswordHash: "hash", CoinBalanceTenths: 2300}
	require.NoError(t, api.DB.Create(&owner).Error)

	v := model.Video{UserID: owner.ID, Title: "Coin Video", Status: "published",
		VideoURL: "https://cdn.example.com/v.mp4", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	require.NoError(t, api.DB.Create(&v).Error)

	viewer := model.User{Username: "vcoinviewer", PasswordHash: "hash", CoinBalanceTenths: 2300}
	require.NoError(t, api.DB.Create(&viewer).Error)
	access, _, _, err := jm.IssuePair(viewer.ID)
	require.NoError(t, err)

	cb, _ := json.Marshal(map[string]int{"amount": 1})
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/videos/%d/coin", v.ID), bytes.NewReader(cb))
	req.Header.Set("Authorization", "Bearer "+access)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var resp struct {
		Code int   `json:"code"`
		Data gin.H `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)
	require.Equal(t, true, resp.Data["coined"])

	var coin model.VideoCoin
	err = api.DB.Where("user_id = ? AND video_id = ?", viewer.ID, v.ID).First(&coin).Error
	require.NoError(t, err)
	require.Equal(t, 1, coin.Amount)

	ownAccess, _, _, err := jm.IssuePair(owner.ID)
	require.NoError(t, err)
	req2 := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/videos/%d/coin", v.ID), bytes.NewReader(cb))
	req2.Header.Set("Authorization", "Bearer "+ownAccess)
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	require.Equal(t, http.StatusBadRequest, w2.Code, w2.Body.String())
}

func TestToggleWatchLater(t *testing.T) {
	api, r, jm := newTestAPI(t)

	owner := model.User{Username: "wlowner", PasswordHash: "hash", CoinBalanceTenths: 230}
	require.NoError(t, api.DB.Create(&owner).Error)

	v := model.Video{UserID: owner.ID, Title: "WL Video", Status: "published",
		VideoURL: "https://cdn.example.com/v.mp4", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	require.NoError(t, api.DB.Create(&v).Error)

	viewer := model.User{Username: "wlviewer", PasswordHash: "hash", CoinBalanceTenths: 230}
	require.NoError(t, api.DB.Create(&viewer).Error)
	access, _, _, err := jm.IssuePair(viewer.ID)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/videos/%d/watch-later", v.ID), nil)
	req.Header.Set("Authorization", "Bearer "+access)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var resp struct {
		Code int   `json:"code"`
		Data gin.H `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)
	require.Equal(t, true, resp.Data["in_watch_later"])

	req2 := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/videos/%d/watch-later", v.ID), nil)
	req2.Header.Set("Authorization", "Bearer "+access)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	require.Equal(t, http.StatusOK, w2.Code, w2.Body.String())

	var resp2 struct {
		Code int   `json:"code"`
		Data gin.H `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w2.Body.Bytes(), &resp2))
	require.Equal(t, 0, resp2.Code)
	require.Equal(t, false, resp2.Data["in_watch_later"])
}

func TestToggleVideoFavorite(t *testing.T) {
	api, r, jm := newTestAPI(t)

	owner := model.User{Username: "vfavowner", PasswordHash: "hash", CoinBalanceTenths: 230}
	require.NoError(t, api.DB.Create(&owner).Error)

	v := model.Video{UserID: owner.ID, Title: "Fav Video", Status: "published",
		VideoURL: "https://cdn.example.com/v.mp4", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	require.NoError(t, api.DB.Create(&v).Error)

	viewer := model.User{Username: "vfavviewer", PasswordHash: "hash", CoinBalanceTenths: 230}
	require.NoError(t, api.DB.Create(&viewer).Error)
	access, _, _, err := jm.IssuePair(viewer.ID)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/videos/%d/favorite", v.ID), nil)
	req.Header.Set("Authorization", "Bearer "+access)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var resp struct {
		Code int   `json:"code"`
		Data gin.H `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)
	require.Equal(t, true, resp.Data["favorited"])

	req2 := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/videos/%d/favorite", v.ID), nil)
	req2.Header.Set("Authorization", "Bearer "+access)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	require.Equal(t, http.StatusOK, w2.Code, w2.Body.String())

	var resp2 struct {
		Code int   `json:"code"`
		Data gin.H `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w2.Body.Bytes(), &resp2))
	require.Equal(t, 0, resp2.Code)
	require.Equal(t, false, resp2.Data["favorited"])
}

func TestSetVideoFavoriteFolders(t *testing.T) {
	api, r, jm := newTestAPI(t)

	owner := model.User{Username: "sffowner", PasswordHash: "hash", CoinBalanceTenths: 230}
	require.NoError(t, api.DB.Create(&owner).Error)

	v := model.Video{UserID: owner.ID, Title: "SFav Video", Status: "published",
		VideoURL: "https://cdn.example.com/v.mp4", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	require.NoError(t, api.DB.Create(&v).Error)

	viewer := model.User{Username: "sffviewer", PasswordHash: "hash", CoinBalanceTenths: 230}
	require.NoError(t, api.DB.Create(&viewer).Error)
	access, _, _, err := jm.IssuePair(viewer.ID)
	require.NoError(t, err)

	folder := model.FavoriteFolder{UserID: viewer.ID, Title: "My Collection", IsPublic: true}
	require.NoError(t, api.DB.Create(&folder).Error)

	fb, _ := json.Marshal(map[string][]uint64{"folder_ids": {folder.ID}})
	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/videos/%d/favorite-folders", v.ID), bytes.NewReader(fb))
	req.Header.Set("Authorization", "Bearer "+access)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var resp struct {
		Code int   `json:"code"`
		Data gin.H `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)

	var fav model.VideoFavorite
	err = api.DB.Where("user_id = ? AND video_id = ? AND folder_id = ?", viewer.ID, v.ID, folder.ID).First(&fav).Error
	require.NoError(t, err)
}
// ---------------------------------------------------------------------------
// 3. Comments
// ---------------------------------------------------------------------------

func TestListCommentsMoreVariants(t *testing.T) {
	api, r, _ := newTestAPI(t)

	owner := model.User{Username: "cmtown", PasswordHash: "hash", CoinBalanceTenths: 230}
	require.NoError(t, api.DB.Create(&owner).Error)

	v := model.Video{UserID: owner.ID, Title: "CM Video", Status: "published",
		VideoURL: "https://cdn.example.com/v.mp4", CreatedAt: time.Now(), UpdatedAt: time.Now(),
		CommentsCurated: true}
	require.NoError(t, api.DB.Create(&v).Error)

	cmt := model.Comment{VideoID: v.ID, UserID: owner.ID, Content: "Pending", Approved: false, Level: 1}
	require.NoError(t, api.DB.Create(&cmt).Error)
	cmt2 := model.Comment{VideoID: v.ID, UserID: owner.ID, Content: "Approved!", Approved: true, Level: 1}
	require.NoError(t, api.DB.Create(&cmt2).Error)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/videos/%d/comments", v.ID), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var resp struct {
		Code int   `json:"code"`
		Data gin.H `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)
	items, ok := resp.Data["items"].([]interface{})
	require.True(t, ok)
	require.Len(t, items, 1)

	v2 := model.Video{UserID: owner.ID, Title: "Closed CM", Status: "published",
		VideoURL: "https://cdn.example.com/v2.mp4", CommentsClosed: true,
		CreatedAt: time.Now(), UpdatedAt: time.Now()}
	require.NoError(t, api.DB.Create(&v2).Error)

	req2 := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/videos/%d/comments", v2.ID), nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	require.Equal(t, http.StatusOK, w2.Code, w2.Body.String())

	var resp2 struct {
		Code int   `json:"code"`
		Data gin.H `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w2.Body.Bytes(), &resp2))
	require.Equal(t, 0, resp2.Code)
	require.Equal(t, true, resp2.Data["comments_closed"])
}

func TestPostCommentExceptions(t *testing.T) {
	api, r, jm := newTestAPI(t)

	owner := model.User{Username: "cmtpostown", PasswordHash: "hash", CoinBalanceTenths: 230}
	require.NoError(t, api.DB.Create(&owner).Error)

	v := model.Video{UserID: owner.ID, Title: "Post CM", Status: "published",
		VideoURL: "https://cdn.example.com/v.mp4", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	require.NoError(t, api.DB.Create(&v).Error)

	user := model.User{Username: "cmtpostuser", PasswordHash: "hash", CoinBalanceTenths: 230}
	require.NoError(t, api.DB.Create(&user).Error)
	access, _, _, err := jm.IssuePair(user.ID)
	require.NoError(t, err)

	cb, _ := json.Marshal(map[string]string{"content": ""})
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/videos/%d/comments", v.ID), bytes.NewReader(cb))
	req.Header.Set("Authorization", "Bearer "+access)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())

	v2 := model.Video{UserID: owner.ID, Title: "Closed Post", Status: "published",
		VideoURL: "https://cdn.example.com/v3.mp4", CommentsClosed: true,
		CreatedAt: time.Now(), UpdatedAt: time.Now()}
	require.NoError(t, api.DB.Create(&v2).Error)

	cb2, _ := json.Marshal(map[string]string{"content": "Nice video!"})
	req3 := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/videos/%d/comments", v2.ID), bytes.NewReader(cb2))
	req3.Header.Set("Authorization", "Bearer "+access)
	req3.Header.Set("Content-Type", "application/json")
	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, req3)
	require.Equal(t, http.StatusForbidden, w3.Code, w3.Body.String())
}

func TestPinComment(t *testing.T) {
	api, r, jm := newTestAPI(t)

	owner := model.User{Username: "pinowner", PasswordHash: "hash", CoinBalanceTenths: 230}
	require.NoError(t, api.DB.Create(&owner).Error)
	ownerAccess, _, _, err := jm.IssuePair(owner.ID)
	require.NoError(t, err)

	v := model.Video{UserID: owner.ID, Title: "Pin Video", Status: "published",
		VideoURL: "https://cdn.example.com/v.mp4", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	require.NoError(t, api.DB.Create(&v).Error)

	cmt := model.Comment{VideoID: v.ID, UserID: owner.ID, Content: "Pin me!", Approved: true, Level: 1}
	require.NoError(t, api.DB.Create(&cmt).Error)

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/comments/%d/pin", cmt.ID), nil)
	req.Header.Set("Authorization", "Bearer "+ownerAccess)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var resp struct {
		Code int   `json:"code"`
		Data gin.H `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)
	require.Equal(t, true, resp.Data["pinned"])

	req2 := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/comments/%d/pin", cmt.ID), nil)
	req2.Header.Set("Authorization", "Bearer "+ownerAccess)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	require.Equal(t, http.StatusOK, w2.Code, w2.Body.String())

	var resp2 struct {
		Code int   `json:"code"`
		Data gin.H `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w2.Body.Bytes(), &resp2))
	require.Equal(t, 0, resp2.Code)
	require.Equal(t, false, resp2.Data["pinned"])

	other := model.User{Username: "pinother", PasswordHash: "hash", CoinBalanceTenths: 230}
	require.NoError(t, api.DB.Create(&other).Error)
	otherAccess, _, _, err := jm.IssuePair(other.ID)
	require.NoError(t, err)

	req3 := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/comments/%d/pin", cmt.ID), nil)
	req3.Header.Set("Authorization", "Bearer "+otherAccess)
	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, req3)
	require.Equal(t, http.StatusForbidden, w3.Code, w3.Body.String())
}


func TestLikeCommentOperations(t *testing.T) {
	api, r, jm := newTestAPI(t)

	owner := model.User{Username: "likecmtown", PasswordHash: "hash", CoinBalanceTenths: 230}
	require.NoError(t, api.DB.Create(&owner).Error)

	v := model.Video{UserID: owner.ID, Title: "Like CM", Status: "published",
		VideoURL: "https://cdn.example.com/v.mp4", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	require.NoError(t, api.DB.Create(&v).Error)

	cmt := model.Comment{VideoID: v.ID, UserID: owner.ID, Content: "Like target", Approved: true, Level: 1}
	require.NoError(t, api.DB.Create(&cmt).Error)

	user := model.User{Username: "likeuser", PasswordHash: "hash", CoinBalanceTenths: 230}
	require.NoError(t, api.DB.Create(&user).Error)
	access, _, _, err := jm.IssuePair(user.ID)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/comments/%d/like", cmt.ID), nil)
	req.Header.Set("Authorization", "Bearer "+access)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	req2 := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/comments/%d/dislike", cmt.ID), nil)
	req2.Header.Set("Authorization", "Bearer "+access)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	require.Equal(t, http.StatusOK, w2.Code, w2.Body.String())

	var likeCount int64
	api.DB.Model(&model.CommentLike{}).Where("user_id = ? AND comment_id = ?", user.ID, cmt.ID).Count(&likeCount)
	require.Equal(t, int64(0), likeCount)
}

// ---------------------------------------------------------------------------
// 4. Article Operations
// ---------------------------------------------------------------------------

func TestGetArticle(t *testing.T) {
	api, r, jm := newTestAPI(t)

	author := model.User{Username: "artauthor", PasswordHash: "hash", CoinBalanceTenths: 230}
	require.NoError(t, api.DB.Create(&author).Error)

	art := model.Article{UserID: author.ID, Title: "Test Article", BodyMD: "# Hello\nWorld",
		Status: "published", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	require.NoError(t, api.DB.Create(&art).Error)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/articles/%d", art.ID), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var resp struct {
		Code int   `json:"code"`
		Data gin.H `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)
	require.Equal(t, float64(art.ID), resp.Data["id"])
	require.Equal(t, "Test Article", resp.Data["title"])

	viewer := model.User{Username: "artviewer", PasswordHash: "hash", CoinBalanceTenths: 230}
	require.NoError(t, api.DB.Create(&viewer).Error)
	access, _, _, err := jm.IssuePair(viewer.ID)
	require.NoError(t, err)

	req2 := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/articles/%d", art.ID), nil)
	req2.Header.Set("Authorization", "Bearer "+access)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	require.Equal(t, http.StatusOK, w2.Code, w2.Body.String())

	var resp2 struct {
		Code int   `json:"code"`
		Data gin.H `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w2.Body.Bytes(), &resp2))
	require.Equal(t, 0, resp2.Code)
	require.Equal(t, false, resp2.Data["favorited_by_me"])

	req3 := httptest.NewRequest(http.MethodGet, "/api/v1/articles/99999", nil)
	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, req3)
	require.Equal(t, http.StatusNotFound, w3.Code, w3.Body.String())
}

func TestPostArticleView(t *testing.T) {
	api, r, jm := newTestAPI(t)

	author := model.User{Username: "artviewauth", PasswordHash: "hash", CoinBalanceTenths: 230}
	require.NoError(t, api.DB.Create(&author).Error)

	art := model.Article{UserID: author.ID, Title: "View Article", BodyMD: "# V",
		Status: "published", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	require.NoError(t, api.DB.Create(&art).Error)

	viewer := model.User{Username: "aviewer", PasswordHash: "hash", CoinBalanceTenths: 230}
	require.NoError(t, api.DB.Create(&viewer).Error)
	access, _, _, err := jm.IssuePair(viewer.ID)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/articles/%d/view", art.ID), nil)
	req.Header.Set("Authorization", "Bearer "+access)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var resp struct {
		Code int   `json:"code"`
		Data gin.H `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)
	vc, ok := resp.Data["view_count"].(float64)
	require.True(t, ok)
	require.GreaterOrEqual(t, vc, float64(1))

	req2 := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/articles/%d/view", art.ID), nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	require.Equal(t, http.StatusUnauthorized, w2.Code, w2.Body.String())
}

func TestToggleArticleFavorite(t *testing.T) {
	api, r, jm := newTestAPI(t)

	author := model.User{Username: "artfavauth", PasswordHash: "hash", CoinBalanceTenths: 230}
	require.NoError(t, api.DB.Create(&author).Error)

	art := model.Article{UserID: author.ID, Title: "Fav Article", BodyMD: "# F",
		Status: "published", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	require.NoError(t, api.DB.Create(&art).Error)

	user := model.User{Username: "favuserart", PasswordHash: "hash", CoinBalanceTenths: 230}
	require.NoError(t, api.DB.Create(&user).Error)
	access, _, _, err := jm.IssuePair(user.ID)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/articles/%d/favorite", art.ID), nil)
	req.Header.Set("Authorization", "Bearer "+access)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var resp struct {
		Code int   `json:"code"`
		Data gin.H `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)
	require.Equal(t, true, resp.Data["favorited"])

	req2 := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/articles/%d/favorite", art.ID), nil)
	req2.Header.Set("Authorization", "Bearer "+access)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	require.Equal(t, http.StatusOK, w2.Code, w2.Body.String())

	var resp2 struct {
		Code int   `json:"code"`
		Data gin.H `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w2.Body.Bytes(), &resp2))
	require.Equal(t, 0, resp2.Code)
	require.Equal(t, false, resp2.Data["favorited"])
}

func TestPostArticleCoin(t *testing.T) {
	api, r, jm := newTestAPI(t)

	author := model.User{Username: "artcoinauth", PasswordHash: "hash", CoinBalanceTenths: 2300}
	require.NoError(t, api.DB.Create(&author).Error)

	art := model.Article{UserID: author.ID, Title: "Coin Article", BodyMD: "# C",
		Status: "published", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	require.NoError(t, api.DB.Create(&art).Error)

	user := model.User{Username: "coinuserart", PasswordHash: "hash", CoinBalanceTenths: 2300}
	require.NoError(t, api.DB.Create(&user).Error)
	access, _, _, err := jm.IssuePair(user.ID)
	require.NoError(t, err)

	cb, _ := json.Marshal(map[string]int{"amount": 1})
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/articles/%d/coin", art.ID), bytes.NewReader(cb))
	req.Header.Set("Authorization", "Bearer "+access)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var resp struct {
		Code int   `json:"code"`
		Data gin.H `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)
	require.Equal(t, true, resp.Data["coined"])

	var coin model.ArticleCoin
	err = api.DB.Where("user_id = ? AND article_id = ?", user.ID, art.ID).First(&coin).Error
	require.NoError(t, err)
	require.Equal(t, 1, coin.Amount)

	ownAccess, _, _, err := jm.IssuePair(author.ID)
	require.NoError(t, err)
	req2 := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/articles/%d/coin", art.ID), bytes.NewReader(cb))
	req2.Header.Set("Authorization", "Bearer "+ownAccess)
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	require.Equal(t, http.StatusBadRequest, w2.Code, w2.Body.String())
}
// ---------------------------------------------------------------------------
// 5. Search
// ---------------------------------------------------------------------------

func TestSearchAllWithParams(t *testing.T) {
	_, r, _ := newTestAPI(t)

	tests := []struct {
		name  string
		query string
		want  int
	}{
		{"keyword only", "keyword=golang", 200},
		{"order param", "keyword=go&order=pubdate", 200},
		{"duration param", "keyword=go&duration=1", 200},
		{"zone param", "keyword=go&zone=动画", 200},
		{"page param", "keyword=go&page=1&page_size=10", 200},
		{"type param", "keyword=go&type=video", 200},
		{"highlight param", "keyword=go&highlight=1", 200},
		{"sort param", "keyword=go&sort=totalrank", 200},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/search?"+tc.query, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			require.Equal(t, tc.want, w.Code, w.Body.String())
		})
	}
}

func TestSearchSuggestWithParams(t *testing.T) {
	_, r, _ := newTestAPI(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/search/suggest?term=hello", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var resp struct {
		Code int `json:"code"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)

	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/search/suggest", nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	require.Equal(t, http.StatusOK, w2.Code, w2.Body.String())
}

// ---------------------------------------------------------------------------
// 6. User Operations (authenticated)
// ---------------------------------------------------------------------------

func TestGetMyProfile(t *testing.T) {
	_, r, _ := newTestAPI(t)

	body := map[string]string{"username": "myprofile", "password": "password12"}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	lb, _ := json.Marshal(body)
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(lb))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	require.Equal(t, http.StatusOK, w2.Code)

	var loginResp struct {
		Code int `json:"code"`
		Data struct {
			AccessToken string `json:"access_token"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w2.Body.Bytes(), &loginResp))
	tok := loginResp.Data.AccessToken

	req3 := httptest.NewRequest(http.MethodGet, "/api/v1/users/me", nil)
	req3.Header.Set("Authorization", "Bearer "+tok)
	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, req3)
	require.Equal(t, http.StatusOK, w3.Code, w3.Body.String())

	var meResp struct {
		Code int   `json:"code"`
		Data gin.H `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w3.Body.Bytes(), &meResp))
	require.Equal(t, 0, meResp.Code)
	require.Equal(t, "myprofile", meResp.Data["username"])
	_, hasSP := meResp.Data["space_privacy"]
	require.True(t, hasSP)
	_, hasLI := meResp.Data["level_info"]
	require.True(t, hasLI)
	_, hasCB := meResp.Data["coin_balance"]
	require.True(t, hasCB)
}

func TestPutMyProfile(t *testing.T) {
	_, r, _ := newTestAPI(t)

	body := map[string]string{"username": "putprofile", "password": "password12"}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	lb, _ := json.Marshal(body)
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(lb))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	require.Equal(t, http.StatusOK, w2.Code)

	var loginResp struct {
		Code int `json:"code"`
		Data struct {
			AccessToken string `json:"access_token"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w2.Body.Bytes(), &loginResp))
	tok := loginResp.Data.AccessToken

	pb, _ := json.Marshal(map[string]string{
		"nickname": "UpdatedProfile",
		"sign":     "Updated sign",
		"gender":   "female",
		"birthday": "2000-01-01",
	})
	req3 := httptest.NewRequest(http.MethodPut, "/api/v1/users/me/profile", bytes.NewReader(pb))
	req3.Header.Set("Authorization", "Bearer "+tok)
	req3.Header.Set("Content-Type", "application/json")
	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, req3)
	require.Equal(t, http.StatusOK, w3.Code, w3.Body.String())

	var profileResp struct {
		Code int   `json:"code"`
		Data gin.H `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w3.Body.Bytes(), &profileResp))
	require.Equal(t, 0, profileResp.Code)
	require.Equal(t, "UpdatedProfile", profileResp.Data["nickname"])
	require.Equal(t, "female", profileResp.Data["gender"])
}

func TestChangePassword(t *testing.T) {
	_, r, _ := newTestAPI(t)

	body := map[string]string{"username": "changepw", "password": "password12"}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	lb, _ := json.Marshal(body)
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(lb))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	require.Equal(t, http.StatusOK, w2.Code)

	var loginResp struct {
		Code int `json:"code"`
		Data struct {
			AccessToken string `json:"access_token"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w2.Body.Bytes(), &loginResp))
	tok := loginResp.Data.AccessToken

	pb, _ := json.Marshal(map[string]string{
		"old_password": "password12",
		"new_password": "newpassword12",
	})
	req3 := httptest.NewRequest(http.MethodPut, "/api/v1/users/me/password", bytes.NewReader(pb))
	req3.Header.Set("Authorization", "Bearer "+tok)
	req3.Header.Set("Content-Type", "application/json")
	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, req3)
	require.Equal(t, http.StatusOK, w3.Code, w3.Body.String())

	var pwResp struct {
		Code int   `json:"code"`
		Data gin.H `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w3.Body.Bytes(), &pwResp))
	require.Equal(t, 0, pwResp.Code)

	lb2, _ := json.Marshal(map[string]string{"username": "changepw", "password": "newpassword12"})
	req4 := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(lb2))
	req4.Header.Set("Content-Type", "application/json")
	w4 := httptest.NewRecorder()
	r.ServeHTTP(w4, req4)
	require.Equal(t, http.StatusOK, w4.Code, w4.Body.String())

	pb2, _ := json.Marshal(map[string]string{
		"old_password": "wrongpass",
		"new_password": "anotherpw12",
	})
	req5 := httptest.NewRequest(http.MethodPut, "/api/v1/users/me/password", bytes.NewReader(pb2))
	req5.Header.Set("Authorization", "Bearer "+tok)
	req5.Header.Set("Content-Type", "application/json")
	w5 := httptest.NewRecorder()
	r.ServeHTTP(w5, req5)
	require.Equal(t, http.StatusForbidden, w5.Code, w5.Body.String())
}

// ---------------------------------------------------------------------------
// 7. User Blocks & Follows
// ---------------------------------------------------------------------------

func TestBlockUser(t *testing.T) {
	api, r, jm := newTestAPI(t)

	blocker := model.User{Username: "blocker", PasswordHash: "hash", CoinBalanceTenths: 230}
	require.NoError(t, api.DB.Create(&blocker).Error)
	access, _, _, err := jm.IssuePair(blocker.ID)
	require.NoError(t, err)

	target := model.User{Username: "blocktarget", PasswordHash: "hash", CoinBalanceTenths: 230}
	require.NoError(t, api.DB.Create(&target).Error)

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/users/%d/block", target.ID), nil)
	req.Header.Set("Authorization", "Bearer "+access)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var resp struct {
		Code int   `json:"code"`
		Data gin.H `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)
	require.Equal(t, true, resp.Data["blocked"])

	var block model.UserBlock
	err = api.DB.Where("blocker_id = ? AND blocked_id = ?", blocker.ID, target.ID).First(&block).Error
	require.NoError(t, err)

	req2 := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/users/%d/block", target.ID), nil)
	req2.Header.Set("Authorization", "Bearer "+access)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	require.Equal(t, http.StatusOK, w2.Code, w2.Body.String())

	req3 := httptest.NewRequest(http.MethodPost, "/api/v1/users/99999/block", nil)
	req3.Header.Set("Authorization", "Bearer "+access)
	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, req3)
	require.Equal(t, http.StatusNotFound, w3.Code, w3.Body.String())

	req4 := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/users/%d/block", blocker.ID), nil)
	req4.Header.Set("Authorization", "Bearer "+access)
	w4 := httptest.NewRecorder()
	r.ServeHTTP(w4, req4)
	require.Equal(t, http.StatusBadRequest, w4.Code, w4.Body.String())
}

func TestFollowUser(t *testing.T) {
	api, r, jm := newTestAPI(t)

	follower := model.User{Username: "follower", PasswordHash: "hash", CoinBalanceTenths: 230}
	require.NoError(t, api.DB.Create(&follower).Error)
	access, _, _, err := jm.IssuePair(follower.ID)
	require.NoError(t, err)

	target := model.User{Username: "followtarget", PasswordHash: "hash", CoinBalanceTenths: 230}
	require.NoError(t, api.DB.Create(&target).Error)

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/users/%d/follow", target.ID), nil)
	req.Header.Set("Authorization", "Bearer "+access)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var resp struct {
		Code int   `json:"code"`
		Data gin.H `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)
	require.Equal(t, true, resp.Data["followed"])

	var follow model.UserFollow
	err = api.DB.Where("follower_id = ? AND followee_id = ?", follower.ID, target.ID).First(&follow).Error
	require.NoError(t, err)

	req2 := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/users/%d/follow", target.ID), nil)
	req2.Header.Set("Authorization", "Bearer "+access)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	require.Equal(t, http.StatusOK, w2.Code, w2.Body.String())

	var resp2 struct {
		Code int   `json:"code"`
		Data gin.H `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w2.Body.Bytes(), &resp2))
	require.Equal(t, 0, resp2.Code)
	require.Equal(t, false, resp2.Data["followed"])

	req3 := httptest.NewRequest(http.MethodPost, "/api/v1/users/99999/follow", nil)
	req3.Header.Set("Authorization", "Bearer "+access)
	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, req3)
	require.Equal(t, http.StatusNotFound, w3.Code, w3.Body.String())

	req4 := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/users/%d/follow", follower.ID), nil)
	req4.Header.Set("Authorization", "Bearer "+access)
	w4 := httptest.NewRecorder()
	r.ServeHTTP(w4, req4)
	require.Equal(t, http.StatusBadRequest, w4.Code, w4.Body.String())
}

// ---------------------------------------------------------------------------
// 8. Creator Comment Management
// ---------------------------------------------------------------------------

func TestListCreatorComments(t *testing.T) {
	api, r, jm := newTestAPI(t)

	owner := model.User{Username: "creatorcm", PasswordHash: "hash", CoinBalanceTenths: 230}
	require.NoError(t, api.DB.Create(&owner).Error)
	ownerAccess, _, _, err := jm.IssuePair(owner.ID)
	require.NoError(t, err)

	v := model.Video{UserID: owner.ID, Title: "Creator CM Video", Status: "published",
		VideoURL: "https://cdn.example.com/v.mp4", CommentsCurated: true,
		CreatedAt: time.Now(), UpdatedAt: time.Now()}
	require.NoError(t, api.DB.Create(&v).Error)

	cmt := model.Comment{VideoID: v.ID, UserID: owner.ID, Content: "Creator comment",
		Approved: false, Level: 1}
	require.NoError(t, api.DB.Create(&cmt).Error)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/me/creator/comments", nil)
	req.Header.Set("Authorization", "Bearer "+ownerAccess)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var resp struct {
		Code int   `json:"code"`
		Data gin.H `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)

	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/users/me/creator/comments?pending=1&page=1&page_size=10", nil)
	req2.Header.Set("Authorization", "Bearer "+ownerAccess)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	require.Equal(t, http.StatusOK, w2.Code, w2.Body.String())
}

func TestApproveAndIgnoreCuratedComment(t *testing.T) {
	api, r, jm := newTestAPI(t)

	owner := model.User{Username: "cmapprove", PasswordHash: "hash", CoinBalanceTenths: 230}
	require.NoError(t, api.DB.Create(&owner).Error)
	ownerAccess, _, _, err := jm.IssuePair(owner.ID)
	require.NoError(t, err)

	v := model.Video{UserID: owner.ID, Title: "Approve CM Video", Status: "published",
		VideoURL: "https://cdn.example.com/v.mp4", CommentsCurated: true,
		CreatedAt: time.Now(), UpdatedAt: time.Now()}
	require.NoError(t, api.DB.Create(&v).Error)

	cmt := model.Comment{VideoID: v.ID, UserID: owner.ID, Content: "Pending approval",
		Approved: false, Level: 1}
	require.NoError(t, api.DB.Create(&cmt).Error)

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/comments/%d/approve", cmt.ID), nil)
	req.Header.Set("Authorization", "Bearer "+ownerAccess)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var resp struct {
		Code int   `json:"code"`
		Data gin.H `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)

	var approved model.Comment
	api.DB.First(&approved, cmt.ID)
	require.True(t, approved.Approved)

	cmt2 := model.Comment{VideoID: v.ID, UserID: owner.ID, Content: "To ignore",
		Approved: false, Level: 1}
	require.NoError(t, api.DB.Create(&cmt2).Error)

	req2 := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/comments/%d/ignore-curated", cmt2.ID), nil)
	req2.Header.Set("Authorization", "Bearer "+ownerAccess)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	require.Equal(t, http.StatusOK, w2.Code, w2.Body.String())

	var resp2 struct {
		Code int   `json:"code"`
		Data gin.H `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w2.Body.Bytes(), &resp2))
	require.Equal(t, 0, resp2.Code)
	require.Equal(t, true, resp2.Data["curated_ignored"])

	var ignored model.Comment
	api.DB.First(&ignored, cmt2.ID)
	require.True(t, ignored.CuratedIgnored)
}

// ---------------------------------------------------------------------------
// 9. Health & Extra
// ---------------------------------------------------------------------------

func TestHealthAdditionalCases(t *testing.T) {
	_, r, _ := newTestAPI(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Code int   `json:"code"`
		Data gin.H `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)
	require.Equal(t, "ok", resp.Data["status"])
}

func TestListUserPublishedArticles(t *testing.T) {
	api, r, _ := newTestAPI(t)

	owner := model.User{Username: "pubartowner", PasswordHash: "hash", CoinBalanceTenths: 230}
	require.NoError(t, api.DB.Create(&owner).Error)

	for i := 0; i < 2; i++ {
		art := model.Article{UserID: owner.ID, Title: fmt.Sprintf("Pub Art %d", i), BodyMD: "# A",
			Status: "published", CreatedAt: time.Now(), UpdatedAt: time.Now()}
		require.NoError(t, api.DB.Create(&art).Error)
	}
	draft := model.Article{UserID: owner.ID, Title: "Draft Art", BodyMD: "# D",
		Status: "draft", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	require.NoError(t, api.DB.Create(&draft).Error)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/space/%d/articles", owner.ID), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var resp struct {
		Code int   `json:"code"`
		Data gin.H `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)
	items, ok := resp.Data["items"].([]interface{})
	require.True(t, ok)
	require.Len(t, items, 2)
}


