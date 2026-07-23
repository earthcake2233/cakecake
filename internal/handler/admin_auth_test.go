package handler

import (
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
"minibili/internal/config"
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"minibili/internal/data"
	"minibili/internal/model"
	"minibili/internal/pkg/jwttoken"
	"minibili/internal/ws"
)

func seedAdminUser(t *testing.T, db *gorm.DB, username, password string) *model.Admin {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	require.NoError(t, err)
	adm := &model.Admin{
		Username:     username,
		PasswordHash: string(hash),
		DisplayName:  username,
		Status:       "active",
		LastLoginAt:  nil,
	}
	require.NoError(t, db.Create(adm).Error)
	return adm
}

func newTestAdminAPI(t *testing.T) (*API, *gin.Engine, *jwttoken.Manager) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, data.AutoMigrateAll(db, zap.NewNop()))

	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)

	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})

	jm, err := jwttoken.NewManager("admin-auth-test-secret-32chars!!!")
	require.NoError(t, err)

	cfg := &config.C{
		RedisAddr:     mr.Addr(),
		RedisPassword: "",
		RedisDB:       0,
		RedisDial:     5 * time.Second,
		RedisRead:     3 * time.Second,
		RedisWrite:    3 * time.Second,
		RedisPoolSize: 10,
	}

	api := &API{
		Dependencies: &Dependencies{
			Cfg:   cfg,
			DB:    db,
			Redis: rdb,
			JWT:   jm,
			Hub:   ws.NewHub(),
			Log:   zap.NewNop(),
			Play:  nil,
		},
	}
	r := gin.New()
	RegisterRoutes(r, api, jm)
	return api, r, jm
}

func TestAdminLogin_Success(t *testing.T) {
	api, r, _ := newTestAdminAPI(t)
	seedAdminUser(t, api.DB, "admin", "admin12345")

	body := map[string]string{"username": "admin", "password": "admin12345"}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/auth/login", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var resp struct {
		Code int `json:"code"`
		Data struct {
			AccessToken  string `json:"access_token"`
			RefreshToken string `json:"refresh_token"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)
	require.NotEmpty(t, resp.Data.AccessToken)
	require.NotEmpty(t, resp.Data.RefreshToken)
}

func TestAdminLogin_InvalidPassword(t *testing.T) {
	api, r, _ := newTestAdminAPI(t)
	seedAdminUser(t, api.DB, "admin", "admin12345")

	body := map[string]string{"username": "admin", "password": "wrongpassword"}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/auth/login", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusUnauthorized, w.Code, w.Body.String())
}

func TestAdminLogin_UnknownUser(t *testing.T) {
	api, r, _ := newTestAdminAPI(t)
	seedAdminUser(t, api.DB, "admin", "admin12345")

	body := map[string]string{"username": "nobody", "password": "admin12345"}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/auth/login", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusUnauthorized, w.Code, w.Body.String())
}

func TestAdminLogin_DisabledAccount(t *testing.T) {
	api, r, _ := newTestAdminAPI(t)
	adm := seedAdminUser(t, api.DB, "disabledadm", "admin12345")
	api.DB.Model(adm).Update("status", "disabled")

	body := map[string]string{"username": "disabledadm", "password": "admin12345"}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/auth/login", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusForbidden, w.Code, w.Body.String())
}

func TestAdminLogin_BadRequest(t *testing.T) {
	_, r, _ := newTestAdminAPI(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/auth/login", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
}

func TestAdminMe_Success(t *testing.T) {
	api, r, jm := newTestAdminAPI(t)
	adm := seedAdminUser(t, api.DB, "admin", "admin12345")
	access, _, _, err := jm.IssueAdminPair(adm.ID)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/me", nil)
	req.Header.Set("Authorization", "Bearer "+access)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var resp struct {
		Code int `json:"code"`
		Data struct {
			ID          uint64 `json:"id"`
			Username    string `json:"username"`
			DisplayName string `json:"display_name"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)
	require.Equal(t, adm.ID, resp.Data.ID)
	require.Equal(t, "admin", resp.Data.Username)
	require.Equal(t, "admin", resp.Data.DisplayName)
}

func TestAdminMe_Unauthorized(t *testing.T) {
	_, r, _ := newTestAdminAPI(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/me", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusUnauthorized, w.Code, w.Body.String())
}

func TestAdminMe_InvalidToken(t *testing.T) {
	_, r, _ := newTestAdminAPI(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/me", nil)
	req.Header.Set("Authorization", "Bearer invalidtoken123")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusUnauthorized, w.Code, w.Body.String())
}

func TestAdminRefresh_Success(t *testing.T) {
	api, r, jm := newTestAdminAPI(t)
	adm := seedAdminUser(t, api.DB, "admin", "admin12345")
	_, refresh, _, err := jm.IssueAdminPair(adm.ID)
	require.NoError(t, err)

	body := map[string]string{"refresh_token": refresh}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/auth/refresh", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var resp struct {
		Code int `json:"code"`
		Data struct {
			AccessToken  string `json:"access_token"`
			RefreshToken string `json:"refresh_token"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)
	require.NotEmpty(t, resp.Data.AccessToken)
	require.NotEmpty(t, resp.Data.RefreshToken)
}

func TestAdminRefresh_InvalidToken(t *testing.T) {
	_, r, _ := newTestAdminAPI(t)

	body := map[string]string{"refresh_token": "invalid"}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/auth/refresh", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusUnauthorized, w.Code, w.Body.String())
}
