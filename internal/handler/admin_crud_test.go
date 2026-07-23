package handler

import (
	"encoding/json"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"minibili/internal/config"
	"minibili/internal/data"
	"minibili/internal/pkg/jwttoken"
	"minibili/internal/ws"
)
func newAdminCRUDAPI(t *testing.T) (*API, *gin.Engine, *jwttoken.Manager) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, data.AutoMigrateAll(db, zap.NewNop()))
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	jm, err := jwttoken.NewManager("admin-crud-test-secret-32chars!!!!!")
	require.NoError(t, err)
	cfg := &config.C{RedisAddr: mr.Addr(), RedisPassword: "", RedisDB: 0, RedisDial: 5 * time.Second, RedisRead: 3 * time.Second, RedisWrite: 3 * time.Second, RedisPoolSize: 10}
	api := &API{Dependencies: &Dependencies{Cfg: cfg, DB: db, Redis: rdb, JWT: jm, Hub: ws.NewHub(), Log: zap.NewNop(), Play: nil}}
	r := gin.New()
	RegisterRoutes(r, api, jm)
	return api, r, jm
}

func TestAdminBannerCRUD_Full(t *testing.T) {
	_, r, jm := newAdminCRUDAPI(t)
	access, _, _, err := jm.IssueAdminPair(1)
	require.NoError(t, err)
	req := httptest.NewRequest("POST", "/api/v1/admin/home-banners", strings.NewReader(`{"title":"Banner 1","image_url":"https://ex.com/img.jpg","link_type":"url","link_target":"https://ex.com","sort_order":1,"enabled":true}`))
	req.Header.Set("Authorization", "Bearer " + access)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, 201, w.Code, w.Body.String())
	var cr struct {
		Code int `json:"code"`
		Data gin.H `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &cr)
	require.Equal(t, 0, cr.Code)
	id := int(cr.Data["id"].(float64))

	// Test DELETE
	req4 := httptest.NewRequest("DELETE", "/api/v1/admin/home-banners/"+strconv.Itoa(id), nil)
	req4.Header.Set("Authorization", "Bearer " + access)
	w4 := httptest.NewRecorder()
	r.ServeHTTP(w4, req4)
	require.Equal(t, 200, w4.Code)
}

func TestAdminBanner_RequiresAuth(t *testing.T) {
	_, r, _ := newAdminCRUDAPI(t)
	req := httptest.NewRequest("GET", "/api/v1/admin/home-banners", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, 401, w.Code)
}
