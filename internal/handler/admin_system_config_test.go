package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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

func newTestAPIWithRuntimeCfg(t *testing.T) (*API, *gin.Engine, *jwttoken.Manager) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, data.AutoMigrateAll(db, zap.NewNop()))

	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)

	tmp := t.TempDir()

	cfg := &config.C{
		RedisAddr:          mr.Addr(),
		RedisPassword:      "",
		RedisDB:            0,
		RedisDial:          5 * time.Second,
		RedisRead:          3 * time.Second,
		RedisWrite:         3 * time.Second,
		RedisPoolSize:      10,
		TempUploadDir:      tmp,
		AgentEnabled:       true,
		AgentDailyQuota:    80,
		RateLimitEnabled:   false,
		RateLimitRate:      20,
		RateLimitBurst:     50,
	}
	rdb, err := data.NewRedis(cfg)
	require.NoError(t, err)

	jm, err := jwttoken.NewManager("integration-test-jwt-secret-32chars!!")
	require.NoError(t, err)

	log := zap.NewNop()
	sens := sensitive.NewFilter(tmp, log)

	pc := &service.PlayCounter{Rdb: rdb, DB: db}
	hub := ws.NewHub()
	relay := service.NewDanmakuRelay(rdb, hub, log)
	ctx, cancel := context.WithCancel(context.Background())
	go relay.RunSubscriber(ctx)
	t.Cleanup(cancel)

	runtimeCfg := config.NewRuntimeConfig(db, map[string]string{
		"agent_enabled":     "true",
		"agent_daily_quota": "80",
		"rate_limit_rate":   "20",
		"rate_limit_burst":  "50",
	})
	runtimeCfg.Start(ctx)
	t.Cleanup(runtimeCfg.Stop)

	api := &API{
		Dependencies: &Dependencies{
			Cfg:         cfg,
			DB:          db,
			Redis:       rdb,
			Log:         log,
			Hub:         hub,
			JWT:         jm,
			Sens:        sens,
			OSS:         nil,
			MQ:          noopMQ{},
			Play:        pc,
			DanmakuRelay: relay,
			RuntimeCfg:  runtimeCfg,
		},
	}
	r := gin.New()
	RegisterRoutes(r, api, jm)
	return api, r, jm
}

func TestAdminSystemConfigs_List(t *testing.T) {
	api, r, jm := newTestAPIWithRuntimeCfg(t)
	_ = api

	access, _, _, err := jm.IssueAdminPair(1)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/admin/system-configs", nil)
	req.Header.Set("Authorization", "Bearer "+access)
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Equal(t, float64(0), body["code"])

	data, ok := body["data"].(map[string]interface{})
	require.True(t, ok, "data should be a map")
	require.Equal(t, "true", data["agent_enabled"])
	require.Equal(t, "80", data["agent_daily_quota"])
}

func TestAdminSystemConfigs_Update(t *testing.T) {
	api, r, jm := newTestAPIWithRuntimeCfg(t)
	_ = api

	access, _, _, err := jm.IssueAdminPair(1)
	require.NoError(t, err)

	payload := map[string]interface{}{
		"configs": map[string]string{
			"agent_enabled":     "false",
			"agent_daily_quota": "50",
		},
	}
	raw, _ := json.Marshal(payload)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/v1/admin/system-configs", bytes.NewReader(raw))
	req.Header.Set("Authorization", "Bearer "+access)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var respBody map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &respBody))
	require.Equal(t, float64(0), respBody["code"])

	data, ok := respBody["data"].(map[string]interface{})
	require.True(t, ok)
	require.Equal(t, "false", data["agent_enabled"])
	require.Equal(t, "50", data["agent_daily_quota"])
}

func TestAdminSystemConfigs_RejectsInvalidKey(t *testing.T) {
	api, r, jm := newTestAPIWithRuntimeCfg(t)
	_ = api

	access, _, _, err := jm.IssueAdminPair(1)
	require.NoError(t, err)

	payload := map[string]interface{}{
		"configs": map[string]string{
			"invalid_key": "value",
		},
	}
	raw, _ := json.Marshal(payload)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/v1/admin/system-configs", bytes.NewReader(raw))
	req.Header.Set("Authorization", "Bearer "+access)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAdminSystemConfigs_RequiresAuth(t *testing.T) {
	_, r, _ := newTestAPIWithRuntimeCfg(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/admin/system-configs", nil)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusUnauthorized, w.Code)
}
