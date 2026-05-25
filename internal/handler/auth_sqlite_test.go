package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"minibili/internal/config"
	"minibili/internal/data"
	"minibili/internal/pkg/jwttoken"
	"minibili/internal/pkg/sensitive"
	"minibili/internal/service"
	"minibili/internal/ws"
)

type noopMQ struct{}

func (noopMQ) PublishTranscode(ctx context.Context, body []byte) error { return nil }

func newTestAPI(t *testing.T) (*API, *gin.Engine, *jwttoken.Manager) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, data.AutoMigrateAll(db, zap.NewNop()))

	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)

	tmp := t.TempDir()
	wordFile := filepath.Join(tmp, "words.txt")
	require.NoError(t, os.WriteFile(wordFile, []byte("badword\n"), 0o600))

	cfg := &config.C{
		RedisAddr:          mr.Addr(),
		RedisPassword:      "",
		RedisDB:            0,
		RedisDial:          5 * time.Second,
		RedisRead:          3 * time.Second,
		RedisWrite:         3 * time.Second,
		RedisPoolSize:      10,
		TempUploadDir:      tmp,
		SensitiveWordsFile: wordFile,
	}
	rdb, err := data.NewRedis(cfg)
	require.NoError(t, err)

	jm, err := jwttoken.NewManager("integration-test-jwt-secret-32chars!!")
	require.NoError(t, err)

	log := zap.NewNop()
	sens := sensitive.NewFilter(wordFile, log)
	require.NoError(t, sens.Reload())

	pc := &service.PlayCounter{Rdb: rdb, DB: db}
	hub := ws.NewHub()
	relay := service.NewDanmakuRelay(rdb, hub, log)
	ctx, cancel := context.WithCancel(context.Background())
	go relay.RunSubscriber(ctx)
	t.Cleanup(cancel)
	api := &API{
		Dependencies: &Dependencies{
			Cfg:          cfg,
			DB:           db,
			Redis:        rdb,
			Log:          log,
			Hub:          hub,
			JWT:          jm,
			Sens:         sens,
			OSS:          nil,
			MQ:           noopMQ{},
			Play:         pc,
			DanmakuRelay: relay,
		},
	}
	r := gin.New()
	RegisterRoutes(r, api, jm)
	return api, r, jm
}

func TestRegisterLogin_SQLite(t *testing.T) {
	_, r, _ := newTestAPI(t)

	body := map[string]string{"username": "testuser", "password": "password12"}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())

	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/users", bytes.NewReader(b))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	require.Equal(t, http.StatusBadRequest, w2.Code)

	loginBody := map[string]string{"username": "testuser", "password": "password12"}
	lb, _ := json.Marshal(loginBody)
	req3 := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(lb))
	req3.Header.Set("Content-Type", "application/json")
	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, req3)
	require.Equal(t, http.StatusOK, w3.Code, w3.Body.String())
	var out struct {
		Code int `json:"code"`
		Data struct {
			AccessToken  string `json:"access_token"`
			RefreshToken string `json:"refresh_token"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w3.Body.Bytes(), &out))
	require.Equal(t, 0, out.Code)
	require.NotEmpty(t, out.Data.AccessToken)
}

func TestGetMe_SQLite(t *testing.T) {
	_, r, _ := newTestAPI(t)
	body := map[string]string{"username": "meuser", "password": "password12"}
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
	var login struct {
		Code int `json:"code"`
		Data struct {
			AccessToken string `json:"access_token"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w2.Body.Bytes(), &login))
	tok := login.Data.AccessToken

	req3 := httptest.NewRequest(http.MethodGet, "/api/v1/users/me", nil)
	req3.Header.Set("Authorization", "Bearer "+tok)
	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, req3)
	require.Equal(t, http.StatusOK, w3.Code, w3.Body.String())
	var me struct {
		Code int `json:"code"`
		Data struct {
			UserID   uint64 `json:"user_id"`
			Username string `json:"username"`
			CakeID   string `json:"cake_id"`
			Nickname string `json:"nickname"`
			Gender   string `json:"gender"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w3.Body.Bytes(), &me))
	require.Equal(t, 0, me.Code)
	require.Equal(t, "meuser", me.Data.Username)
	require.Equal(t, "cake_00000000001", me.Data.CakeID)
	require.Equal(t, "secret", me.Data.Gender)
}

func TestUpdateMeProfile_SQLite(t *testing.T) {
	_, r, _ := newTestAPI(t)
	body := map[string]string{"username": "profuser", "password": "password12"}
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
	var login struct {
		Code int `json:"code"`
		Data struct {
			AccessToken string `json:"access_token"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w2.Body.Bytes(), &login))
	tok := login.Data.AccessToken

	pb, _ := json.Marshal(map[string]string{
		"nickname": "alice",
		"sign":     "hello",
		"gender":   "female",
		"birthday": "2006-03-04",
	})
	req3 := httptest.NewRequest(http.MethodPut, "/api/v1/users/me/profile", bytes.NewReader(pb))
	req3.Header.Set("Authorization", "Bearer "+tok)
	req3.Header.Set("Content-Type", "application/json")
	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, req3)
	require.Equal(t, http.StatusOK, w3.Code, w3.Body.String())
	var out struct {
		Code int `json:"code"`
		Data struct {
			Nickname string `json:"nickname"`
			Sign     string `json:"sign"`
			Gender   string `json:"gender"`
			Birthday string `json:"birthday"`
			CakeID   string `json:"cake_id"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w3.Body.Bytes(), &out))
	require.Equal(t, 0, out.Code)
	require.Equal(t, "alice", out.Data.Nickname)
	require.Equal(t, "hello", out.Data.Sign)
	require.Equal(t, "female", out.Data.Gender)
	require.Equal(t, "2006-03-04", out.Data.Birthday)
	require.NotEmpty(t, out.Data.CakeID)
}

func TestUpdateMeAnnouncement_SQLite(t *testing.T) {
	_, r, _ := newTestAPI(t)
	body := map[string]string{"username": "annuser", "password": "password12"}
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
	var login struct {
		Code int `json:"code"`
		Data struct {
			AccessToken string `json:"access_token"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w2.Body.Bytes(), &login))
	tok := login.Data.AccessToken

	ab, _ := json.Marshal(map[string]string{"announcement": "空间公告一条"})
	req3 := httptest.NewRequest(http.MethodPut, "/api/v1/users/me/announcement", bytes.NewReader(ab))
	req3.Header.Set("Authorization", "Bearer "+tok)
	req3.Header.Set("Content-Type", "application/json")
	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, req3)
	require.Equal(t, http.StatusOK, w3.Code, w3.Body.String())
	var out struct {
		Code int `json:"code"`
		Data struct {
			Announcement string `json:"announcement"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w3.Body.Bytes(), &out))
	require.Equal(t, 0, out.Code)
	require.Equal(t, "空间公告一条", out.Data.Announcement)
}
