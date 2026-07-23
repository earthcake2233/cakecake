package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"minibili/internal/config"
	"minibili/internal/data"
	"minibili/internal/model"
	"minibili/internal/pkg/jwttoken"
	"minibili/internal/service"
	"minibili/internal/ws"
)

func setupHandlerIntegrationDB(t *testing.T) (*API, *gin.Engine, string) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, data.AutoMigrateAll(db, zap.NewNop()))

	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)

	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})

	jm, err := jwttoken.NewManager("handler-integration-secret-32chars!!!")
	require.NoError(t, err)

	cfg := &config.C{
		RedisAddr: mr.Addr(),
		RedisDB:   0,
	}

	api := &API{
		Dependencies: &Dependencies{
			Cfg:   cfg,
			DB:    db,
			Redis: rdb,
			JWT:   jm,
			Hub:   ws.NewHub(),
			Log:   zap.NewNop(),
			Play:  &service.PlayCounter{Rdb: rdb, DB: db},
		},
	}

	r := gin.New()
	RegisterRoutes(r, api, jm)

	access, _, _, _ := jm.IssuePair(1)

	return api, r, access
}

func TestIntegration_HomeBanners(t *testing.T) {
	api, r, _ := setupHandlerIntegrationDB(t)

	now := time.Now()
	b := model.HomeBanner{
		Title:      "Test Banner",
		ImageURL:   "https://ex.com/banner.jpg",
		LinkType:   "url",
		LinkTarget: "https://example.com",
		Enabled:    true,
		SortOrder:  1,
		StartAt:    &now,
	}
	require.NoError(t, api.DB.Create(&b).Error)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/home-banners", nil)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Code int `json:"code"`
		Data struct {
			Items []gin.H `json:"items"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)
	require.Len(t, resp.Data.Items, 1)
}

func TestIntegration_Health(t *testing.T) {
	_, r, _ := setupHandlerIntegrationDB(t)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/health", nil)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
}

func TestIntegration_Register(t *testing.T) {
	_, r, _ := setupHandlerIntegrationDB(t)

	body := `{"username":"newuser","password":"password123"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/users", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)
}

func TestIntegration_Login(t *testing.T) {
	api, r, _ := setupHandlerIntegrationDB(t)

	hash, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	api.DB.Create(&model.User{Username: "logintest", PasswordHash: string(hash)})

	body := `{"username":"logintest","password":"password123"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Data struct {
			AccessToken string `json:"access_token"`
		} `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)
	require.NotEmpty(t, resp.Data.AccessToken)
}

func TestIntegration_LoginFailed(t *testing.T) {
	_, r, _ := setupHandlerIntegrationDB(t)

	body := `{"username":"nouser","password":"wrongpass"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestIntegration_MeEndpoint(t *testing.T) {
	api, r, token := setupHandlerIntegrationDB(t)

	api.DB.Create(&model.User{ID: 1, Username: "metest", Nickname: "MeTest"})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/users/me", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
}

func TestIntegration_SearchHistory_NoAuth(t *testing.T) {
	_, r, _ := setupHandlerIntegrationDB(t)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/users/me/search-history", nil)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestMore_VideoList(t *testing.T) {
	api, r, _ := setupHandlerIntegrationDB(t)
	api.DB.Create(&model.User{ID: 1, Username: "u1", Nickname: "u1"})
	api.DB.Create(&model.Video{ID: 1, UserID: 1, Title: "V1", Status: "published", Zone: "Life-Daily"})
	api.Play = &service.PlayCounter{Rdb: api.Dependencies.Redis, DB: api.DB}

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/api/v1/videos?zone=Life-Daily", nil))
	require.Equal(t, http.StatusOK, w.Code)
}

func TestMore_SpaceUser(t *testing.T) {
	api, r, _ := setupHandlerIntegrationDB(t)
	api.DB.Create(&model.User{ID: 1, Username: "sp", Nickname: "sp"})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/api/v1/space/1", nil))
	require.Equal(t, http.StatusOK, w.Code)
}

func TestMore_SpaceUserNotFound(t *testing.T) {
	_, r, _ := setupHandlerIntegrationDB(t)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/api/v1/space/999", nil))
	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestMore_SpaceVideos(t *testing.T) {
	api, r, _ := setupHandlerIntegrationDB(t)
	api.DB.Create(&model.User{ID: 1, Username: "u1"})
	api.DB.Create(&model.Video{ID: 1, UserID: 1, Title: "V", Status: "published", Zone: "Life"})
	api.Play = &service.PlayCounter{Rdb: api.Dependencies.Redis, DB: api.DB}

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/api/v1/space/1/videos", nil))
	require.Equal(t, http.StatusOK, w.Code)
}

func TestMore_GetVideo(t *testing.T) {
	api, r, _ := setupHandlerIntegrationDB(t)
	api.DB.Create(&model.User{ID: 1, Username: "u1"})
	api.DB.Create(&model.Video{ID: 1, UserID: 1, Title: "D", Status: "published", Zone: "Life"})
	api.Play = &service.PlayCounter{Rdb: api.Dependencies.Redis, DB: api.DB}

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/api/v1/videos/1", nil))
	require.Equal(t, http.StatusOK, w.Code)
}

func TestMore_GetVideoNotFound(t *testing.T) {
	_, r, _ := setupHandlerIntegrationDB(t)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/api/v1/videos/999", nil))
	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestMore_Comments(t *testing.T) {
	api, r, _ := setupHandlerIntegrationDB(t)
	api.DB.Create(&model.User{ID: 2, Username: "u2"})
	api.DB.Create(&model.Video{ID: 1, UserID: 2, Title: "C", Status: "published", Zone: "Life"})
	api.DB.Create(&model.Comment{VideoID: 1, UserID: 2, Content: "Nice!", Approved: true})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/api/v1/videos/1/comments", nil))
	require.Equal(t, http.StatusOK, w.Code)
}

func TestMore_GetArticle(t *testing.T) {
	api, r, _ := setupHandlerIntegrationDB(t)
	api.DB.Create(&model.User{ID: 1, Username: "au"})
	api.DB.Create(&model.Article{ID: 1, UserID: 1, Title: "A", BodyMD: "body", Status: "published"})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/api/v1/articles/1", nil))
	require.Equal(t, http.StatusOK, w.Code)
}

func TestMore_Search(t *testing.T) {
	_, r, _ := setupHandlerIntegrationDB(t)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/api/v1/search?keyword=test", nil))
	require.Equal(t, http.StatusOK, w.Code)
}

func TestMore_SearchSuggest(t *testing.T) {
	_, r, _ := setupHandlerIntegrationDB(t)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/api/v1/search/suggest?keyword=test", nil))
	require.Equal(t, http.StatusOK, w.Code)
}

func TestMore_HomeStats(t *testing.T) {
	_, r, _ := setupHandlerIntegrationDB(t)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/api/v1/stats/home", nil))
	require.Equal(t, http.StatusOK, w.Code)
}

func TestMore_Authd_Me(t *testing.T) {
	api, r, token := setupHandlerIntegrationDB(t)
	api.DB.Create(&model.User{ID: 1, Username: "cu"})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/users/me", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
}

func TestMore_Authd_DailyRewards(t *testing.T) {
	api, r, token := setupHandlerIntegrationDB(t)
	api.DB.Create(&model.User{ID: 1, Username: "u1"})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/users/me/daily-rewards", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
}

func TestMore_Authd_CoinLedger(t *testing.T) {
	api, r, token := setupHandlerIntegrationDB(t)
	api.DB.Create(&model.User{ID: 1, Username: "u1"})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/users/me/coin-ledger", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
}

func TestMore_Authd_FollowGroups(t *testing.T) {
	api, r, token := setupHandlerIntegrationDB(t)
	api.DB.Create(&model.User{ID: 1, Username: "u1"})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/users/me/follow-groups", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
}

func TestMore_Authd_WatchLater(t *testing.T) {
	api, r, token := setupHandlerIntegrationDB(t)
	api.DB.Create(&model.User{ID: 1, Username: "u1"})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/users/me/watch-later", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
}

func TestMore_Authd_ToggleLike(t *testing.T) {
	api, r, token := setupHandlerIntegrationDB(t)
	api.DB.Create(&model.User{ID: 1, Username: "u1"})
	api.DB.Create(&model.Video{ID: 1, UserID: 1, Title: "L", Status: "published", Zone: "Life"})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/videos/1/like", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
}

func TestMore_FavoriteFolders(t *testing.T) {
	api, r, token := setupHandlerIntegrationDB(t)
	api.DB.Create(&model.User{ID: 1, Username: "u1"})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/users/me/favorite-folders", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
}

func TestMore_MyVideos(t *testing.T) {
	api, r, token := setupHandlerIntegrationDB(t)
	api.DB.Create(&model.User{ID: 1, Username: "u1"})
	api.DB.Create(&model.Video{ID: 1, UserID: 1, Title: "MV", Status: "published", Zone: "Life"})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/users/me/videos", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
}

func TestMore_SpaceFollowing(t *testing.T) {
	api, r, token := setupHandlerIntegrationDB(t)
	api.DB.Create(&model.User{ID: 1, Username: "u1"})
	api.DB.Create(&model.User{ID: 2, Username: "u2"})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/space/1/following", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
}

func TestMore_SpaceFollowers(t *testing.T) {
	api, r, token := setupHandlerIntegrationDB(t)
	api.DB.Create(&model.User{ID: 1, Username: "u1"})
	api.DB.Create(&model.User{ID: 2, Username: "u2"})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/space/1/followers", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
}

func TestMore_SpaceArticles(t *testing.T) {
	api, r, _ := setupHandlerIntegrationDB(t)
	api.DB.Create(&model.User{ID: 1, Username: "au"})
	api.DB.Create(&model.Article{ID: 1, UserID: 1, Title: "A1", BodyMD: "body", Status: "published"})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/api/v1/space/1/articles", nil))
	require.Equal(t, http.StatusOK, w.Code)
}

func TestMore_SpaceDynamics(t *testing.T) {
	api, r, _ := setupHandlerIntegrationDB(t)
	api.DB.Create(&model.User{ID: 1, Username: "u1"})
	api.DB.Create(&model.UserDynamic{ID: 1, UserID: 1, Content: "Hello!"})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/api/v1/space/1/dynamics", nil))
	require.Equal(t, http.StatusOK, w.Code)
}

func TestMore_GetDynamic(t *testing.T) {
	api, r, _ := setupHandlerIntegrationDB(t)
	api.DB.Create(&model.User{ID: 1, Username: "u1"})
	api.DB.Create(&model.UserDynamic{ID: 1, UserID: 1, Content: "Test Dynamic"})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/api/v1/user-dynamics/1", nil))
	require.Equal(t, http.StatusOK, w.Code)
}

func TestMore_DynamicComments(t *testing.T) {
	api, r, _ := setupHandlerIntegrationDB(t)
	api.DB.Create(&model.User{ID: 2, Username: "u2"})
	api.DB.Create(&model.UserDynamic{ID: 1, UserID: 1, Content: "Dynamic"})
	api.DB.Create(&model.DynamicComment{DynamicID: 1, UserID: 2, Content: "Nice!", Approved: true})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/api/v1/user-dynamics/1/comments", nil))
	require.Equal(t, http.StatusOK, w.Code)
}

func TestMore_SpaceFavorites(t *testing.T) {
	api, r, _ := setupHandlerIntegrationDB(t)
	api.DB.Create(&model.User{ID: 1, Username: "u1"})
	api.DB.Create(&model.Video{ID: 1, UserID: 1, Title: "Fav", Status: "published", Zone: "Life"})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/api/v1/space/1/favorites", nil))
	require.Equal(t, http.StatusOK, w.Code)
}

func TestMore_FavoritePicker(t *testing.T) {
	api, r, token := setupHandlerIntegrationDB(t)
	api.DB.Create(&model.User{ID: 1, Username: "u1"})
	api.DB.Create(&model.Video{ID: 1, UserID: 1, Title: "Pick", Status: "published", Zone: "Life"})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/videos/1/favorite-picker", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
}

func TestMore_DmConversations(t *testing.T) {
	api, r, token := setupHandlerIntegrationDB(t)
	api.DB.Create(&model.User{ID: 1, Username: "u1"})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/dm/conversations", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
}

func TestMore_ArticleComments(t *testing.T) {
	api, r, _ := setupHandlerIntegrationDB(t)
	api.DB.Create(&model.User{ID: 2, Username: "u2"})
	api.DB.Create(&model.Article{ID: 1, UserID: 1, Title: "Art", BodyMD: "body", Status: "published"})
	api.DB.Create(&model.ArticleComment{ArticleID: 1, UserID: 2, Content: "Great!", Approved: true})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/api/v1/articles/1/comments", nil))
	require.Equal(t, http.StatusOK, w.Code)
}
