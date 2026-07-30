//go:build integration

package handler

import (
	"minibili/internal/model/article"
	"minibili/internal/model/dynamic"
	"minibili/internal/model/user"
	"minibili/internal/model/video"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

)

// helpers
func seedUser(t *testing.T, api *API, username, nickname string, coin int) user.User {
	t.Helper()
	u := user.User{Username: username, PasswordHash: "hash", Nickname: nickname, CoinBalanceTenths: int64(coin * 10)}
	require.NoError(t, api.DB.Create(&u).Error)
	ensureUserCakeID(api.DB, &u)
	return u
}

func seedVideoWithAPI(t *testing.T, api *API, uid uint64, title string) video.Video {
	t.Helper()
	v := video.Video{UserID: uid, Title: title, Status: "published", VideoURL: "https://cdn.example.com/v.mp4", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	require.NoError(t, api.DB.Create(&v).Error)
	return v
}

func seedArticle(t *testing.T, api *API, uid uint64, title string) article.Article {
	t.Helper()
	a := article.Article{UserID: uid, Title: title, Status: "published", CreatedAt: time.Now()}
	require.NoError(t, api.DB.Create(&a).Error)
	return a
}

func tok(t *testing.T, api *API, uid uint64) string {
	t.Helper()
	tk, _, _, _ := api.JWT.IssuePair(uid)
	return tk
}

func admintok(t *testing.T, api *API) string {
	t.Helper()
	tk, _, _, _ := api.JWT.IssueAdminPair(1)
	return tk
}

func areq(method, url, token string, body interface{}) *http.Request {
	var r io.Reader
	if s, ok := body.(string); ok {
		r = strings.NewReader(s)
	} else if rd, ok := body.(io.Reader); ok {
		r = rd
	}
	req := httptest.NewRequest(method, url, r)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if r != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return req
}

func srve(r *gin.Engine, req *http.Request) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// ==================== video_engagement.go ====================

func Test_VideoEngagement(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "ve1", "VEng1", 100)
	u2 := seedUser(t, api, "ve2", "VEng2", 100)
	v := seedVideoWithAPI(t, api, u2.ID, "VE Video")
	tk := tok(t, api, u.ID)
	srve(r, areq("POST", fmt.Sprintf("/api/v1/videos/%d/like", v.ID), tk, nil))
	srve(r, areq("POST", fmt.Sprintf("/api/v1/videos/%d/like", v.ID), tk, nil))
	srve(r, areq("POST", fmt.Sprintf("/api/v1/videos/%d/favorite", v.ID), tk, nil))
	srve(r, areq("POST", fmt.Sprintf("/api/v1/videos/%d/coin", v.ID), tk, `{"coins":1}`))
	srve(r, areq("POST", fmt.Sprintf("/api/v1/videos/%d/watch-later", v.ID), tk, nil))
}

func Test_CommentCRUD(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "c1", "C1", 10)
	v := seedVideoWithAPI(t, api, u.ID, "C Video")
	tk := tok(t, api, u.ID)
	srve(r, areq("POST", fmt.Sprintf("/api/v1/videos/%d/comments", v.ID), tk, `{"content":"hello"}`))
	srve(r, areq("GET", fmt.Sprintf("/api/v1/videos/%d/comments", v.ID), "", nil))
	srve(r, areq("POST", fmt.Sprintf("/api/v1/videos/%d/comments", v.ID), tk, `{"content":"second"}`))
}

func Test_ArticleFlow(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u2 := seedUser(t, api, "af2", "AF2", 100)
	a := seedArticle(t, api, u2.ID, "Test Article")
	u := seedUser(t, api, "af1", "AF1", 100)
	tk := tok(t, api, u.ID)
	srve(r, areq("GET", fmt.Sprintf("/api/v1/articles/%d", a.ID), "", nil))
	srve(r, areq("POST", fmt.Sprintf("/api/v1/articles/%d/favorite", a.ID), tk, nil))
	srve(r, areq("POST", fmt.Sprintf("/api/v1/articles/%d/comments", a.ID), tk, `{"content":"ac"}`))
	srve(r, areq("GET", fmt.Sprintf("/api/v1/articles/%d/comments", a.ID), tk, nil))
	srve(r, areq("POST", fmt.Sprintf("/api/v1/articles/%d/coin", a.ID), tk, `{"coins":1}`))
	srve(r, areq("POST", fmt.Sprintf("/api/v1/articles/%d/favorite", a.ID), tk, nil))
}

func Test_ViewHistory(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "vh1", "VH1", 10)
	v := seedVideoWithAPI(t, api, u.ID, "VH Video")
	tk := tok(t, api, u.ID)
	srve(r, areq("POST", fmt.Sprintf("/api/v1/videos/%d/view-history", v.ID), tk, nil))
	srve(r, areq("GET", "/api/v1/users/me/view-history", tk, nil))
	srve(r, areq("GET", "/api/v1/users/me/view-history/settings", tk, nil))
	srve(r, areq("DELETE", fmt.Sprintf("/api/v1/users/me/view-history/%d", v.ID), tk, nil))
}

func Test_UserDynamic(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "dyn1", "Dyn1", 10)
	dyn := dynamic.UserDynamic{UserID: u.ID, Title: "My Dyn", Content: "Content", ImagesJSON: "[]", CreatedAt: time.Now()}
	require.NoError(t, api.DB.Create(&dyn).Error)
	srve(r, areq("GET", fmt.Sprintf("/api/v1/space/%d/dynamics", u.ID), "", nil))
	srve(r, areq("GET", fmt.Sprintf("/api/v1/user-dynamics/%d", dyn.ID), "", nil))
}

func Test_DmConversation(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u1 := seedUser(t, api, "dm1", "DM1", 10)
	u2 := seedUser(t, api, "dm2", "DM2", 10)
	tk1 := tok(t, api, u1.ID)
	tk2 := tok(t, api, u2.ID)
	w := srve(r, areq("POST", "/api/v1/dm/conversations", tk1, fmt.Sprintf(`{"peer_id":%d}`, u2.ID)))
	var cr struct { Code int `json:"code"`; Data struct{ ID uint64 `json:"id"`} `json:"data"` }
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &cr))
	require.Equal(t, 0, cr.Code)
	cid := cr.Data.ID
	srve(r, areq("GET", "/api/v1/dm/conversations", tk1, nil))
	srve(r, areq("POST", fmt.Sprintf("/api/v1/dm/conversations/%d/messages", cid), tk2, `{"content":"hi"}`))
	srve(r, areq("GET", fmt.Sprintf("/api/v1/dm/conversations/%d/messages", cid), tk1, nil))
}

func Test_FavoriteFolder(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "ff1", "FF1", 10)
	v := seedVideoWithAPI(t, api, u.ID, "FF Video")
	tk := tok(t, api, u.ID)
	df := video.FavoriteFolder{UserID: u.ID, Title: "default", IsDefault: true}
	require.NoError(t, api.DB.Create(&df).Error)
	srve(r, areq("POST", fmt.Sprintf("/api/v1/videos/%d/favorite", v.ID), tk, nil))
	w := srve(r, areq("POST", "/api/v1/users/me/favorite-folders", tk, `{"title":"My Folder"}`))
	var fr struct { Code int `json:"code"`; Data struct{ ID uint64 `json:"id"`} `json:"data"` }
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &fr))
	fid := fr.Data.ID
	srve(r, areq("GET", "/api/v1/users/me/favorite-folders", tk, nil))
	srve(r, areq("POST", fmt.Sprintf("/api/v1/videos/%d/favorite-folders/%d", v.ID, fid), tk, nil))
}

func Test_VideoDetailAndListing(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "vd1", "VD1", 10)
	v := seedVideoWithAPI(t, api, u.ID, "VD Video")
	srve(r, areq("GET", fmt.Sprintf("/api/v1/videos/%d", v.ID), "", nil))
	srve(r, areq("GET", "/api/v1/videos?page=1&page_size=10", "", nil))
	srve(r, areq("GET", fmt.Sprintf("/api/v1/space/%d/videos", u.ID), "", nil))
}

func Test_SearchHistory(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "sh1", "SH1", 10)
	tk := tok(t, api, u.ID)
	srve(r, areq("GET", "/api/v1/users/me/search-history", tk, nil))
	srve(r, areq("POST", "/api/v1/users/me/search-history", tk, `{"keyword":"golang"}`))
}

// ==================== admin + OSS + more ====================

func Test_AdminAgentSettings(t *testing.T) {
	api, r, _ := newTestAPI(t)
	at := admintok(t, api)
	srve(r, areq("GET", "/api/v1/admin/agent-settings", at, nil))
	srve(r, areq("PUT", "/api/v1/admin/agent-settings", at, `{"display_name":"AI","welcome_message":"Hello"}`))
}

func Test_AdminAgentProfiles(t *testing.T) {
	api, r, _ := newTestAPI(t)
	at := admintok(t, api)
	srve(r, areq("GET", "/api/v1/admin/agent-profiles", at, nil))
}

func Test_AdminBannerCRUD(t *testing.T) {
	api, r, _ := newTestAPI(t)
	at := admintok(t, api)
	srve(r, areq("GET", "/api/v1/admin/home-banners", at, nil))
	w := srve(r, areq("POST", "/api/v1/admin/home-banners", at, `{"title":"B1","link_type":"none","sort_order":1}`))
	var br struct { Code int `json:"code"`; Data struct{ ID uint64 `json:"id"`} `json:"data"` }
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &br))
	if br.Code == 0 && br.Data.ID > 0 {
		srve(r, areq("PUT", fmt.Sprintf("/api/v1/admin/home-banners/%d", br.Data.ID), at, `{"title":"Updated"}`))
		srve(r, areq("DELETE", fmt.Sprintf("/api/v1/admin/home-banners/%d", br.Data.ID), at, nil))
	}
}

func Test_AdminHotSearch(t *testing.T) {
	api, r, _ := newTestAPI(t)
	at := admintok(t, api)
	srve(r, areq("GET", "/api/v1/admin/hot-search/ops", at, nil))
	srve(r, areq("POST", "/api/v1/admin/hot-search/ops", at, `{"keyword":"test","title":"Test","display_order":1}`))
	srve(r, areq("GET", "/api/v1/admin/hot-search/dashboard", at, nil))
}

func Test_UserFollow(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "uf1", "UF1", 10)
	u2 := seedUser(t, api, "uf2", "UF2", 10)
	tk := tok(t, api, u.ID)
	srve(r, areq("POST", fmt.Sprintf("/api/v1/users/%d/follow", u2.ID), tk, nil))
	srve(r, areq("GET", fmt.Sprintf("/api/v1/space/%d/following", u.ID), "", nil))
	srve(r, areq("GET", fmt.Sprintf("/api/v1/space/%d/followers", u.ID), "", nil))
}

func Test_UserMe(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "um1", "UM1", 50)
	tk := tok(t, api, u.ID)
	srve(r, areq("GET", "/api/v1/users/me", tk, nil))
	srve(r, areq("PUT", "/api/v1/users/me/profile", tk, `{"nickname":"NewName","sign":"Hi","gender":"male"}`))
}

func Test_SpacePrivacy(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "sp1", "SP1", 10)
	tk := tok(t, api, u.ID)
	srve(r, areq("GET", "/api/v1/users/me/space-privacy", tk, nil))
	srve(r, areq("PUT", "/api/v1/users/me/space-privacy", tk, `{"public_favorites":true}`))
}

func Test_DynamicCommentList(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "dc1", "DC1", 10)
	dyn := dynamic.UserDynamic{UserID: u.ID, Title: "D", Content: "C", ImagesJSON: "[]", CreatedAt: time.Now()}
	require.NoError(t, api.DB.Create(&dyn).Error)
	srve(r, areq("GET", fmt.Sprintf("/api/v1/user-dynamics/%d/comments", dyn.ID), "", nil))
}

func Test_SearchAll(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "sa1", "SA1", 10)
	seedVideoWithAPI(t, api, u.ID, "Searchable Video")
	srve(r, areq("GET", "/api/v1/search?keyword=test&page=1&page_size=10", "", nil))
}

func Test_UserMeFavorites(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "ufv1", "UFV1", 10)
	v := seedVideoWithAPI(t, api, u.ID, "Fav Video")
	tk := tok(t, api, u.ID)
	srve(r, areq("POST", fmt.Sprintf("/api/v1/videos/%d/favorite", v.ID), tk, nil))
	srve(r, areq("GET", "/api/v1/users/me/favorites", tk, nil))
}

func Test_DailyReward(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "dr1", "DR1", 10)
	tk := tok(t, api, u.ID)
	srve(r, areq("POST", "/api/v1/users/me/daily-reward", tk, nil))
}

func Test_SearchSuggest(t *testing.T) {
	_, r, _ := newTestAPI(t)
	srve(r, areq("GET", "/api/v1/search/suggest?term=test", "", nil))
}

// ==================== admin_banner_upload (nil OSS path) ====================

func Test_AdminBannerUpload_NilOSS(t *testing.T) {
	api, r, _ := newTestAPI(t)
	at := admintok(t, api)
	w := srve(r, areq("POST", "/api/v1/admin/home-banners/upload-image", at, nil))
	require.Equal(t, 400, w.Code, w.Body.String())
}

func Test_AdminUploadAgentAvatar_NilOSS(t *testing.T) {
	api, r, _ := newTestAPI(t)
	at := admintok(t, api)
	// First call GET agent-settings to trigger EnsureAgentProfiles (creates default profile)
	srve(r, areq("GET", "/api/v1/admin/agent-settings", at, nil))
	// Now upload with nil body; AdminUploadAgentAvatar finds profiles, then fails at ParseMultipartForm -> 400
	w := srve(r, areq("POST", "/api/v1/admin/agent-settings/avatar", at, nil))
	require.Equal(t, 400, w.Code, w.Body.String())
}
func Test_SpaceUserArticles(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "sa2", "SA2", 10)
	seedArticle(t, api, u.ID, "Space Article")
	srve(r, areq("GET", fmt.Sprintf("/api/v1/space/%d/articles", u.ID), "", nil))
}

func Test_SpaceUserFavorites(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "sf1", "SF1", 10)
	srve(r, areq("GET", fmt.Sprintf("/api/v1/space/%d/favorites", u.ID), "", nil))
	srve(r, areq("GET", fmt.Sprintf("/api/v1/space/%d/favorite-folders", u.ID), "", nil))
}

func Test_FollowGroups(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "fg1", "FG1", 10)
	tk := tok(t, api, u.ID)
	srve(r, areq("GET", "/api/v1/users/me/follow-groups", tk, nil))
	srve(r, areq("POST", "/api/v1/users/me/follow-groups", tk, `{"name":"My Group"}`))
}

func Test_DailyRewardStatus(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "dr2", "DR2", 10)
	tk := tok(t, api, u.ID)
	srve(r, areq("GET", "/api/v1/users/me/daily-reward/status", tk, nil))
}

func Test_CoinLedger(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "cl1", "CL1", 100)
	tk := tok(t, api, u.ID)
	srve(r, areq("GET", "/api/v1/users/me/coin-ledger?page=1&page_size=10", tk, nil))
}

func Test_WatchLaterList(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "wl1", "WL1", 10)
	tk := tok(t, api, u.ID)
	srve(r, areq("GET", "/api/v1/users/me/watch-later", tk, nil))
}
