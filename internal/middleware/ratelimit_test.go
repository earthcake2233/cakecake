package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

)

func setupTest(t *testing.T) (*RateLimiter, *gin.Engine) {
	t.Helper()
	s := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: s.Addr()})
	rl := NewRateLimiter(rdb, 10, 5)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(rl.RateLimit())
	r.GET("/api/v1/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	return rl, r
}

func TestRateLimit_Allowed(t *testing.T) {
	_, r := setupTest(t)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/test", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if w.Header().Get("X-RateLimit-Remaining") == "" {
		t.Error("expected X-RateLimit-Remaining header")
	}
}

func TestRateLimit_Blocked(t *testing.T) {
	_, r := setupTest(t)
	for i := 0; i < 5; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/test", nil)
		req.RemoteAddr = "192.168.1.2:12345"
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("request %d should be allowed, got %d", i+1, w.Code)
		}
	}
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/test", nil)
	req.RemoteAddr = "192.168.1.2:12345"
	r.ServeHTTP(w, req)
	if w.Code != http.StatusTooManyRequests {
		t.Errorf("expected 429, got %d", w.Code)
	}
	if w.Header().Get("Retry-After") == "" {
		t.Error("expected Retry-After header")
	}
}

func TestRateLimit_Refill(t *testing.T) {
	_, r := setupTest(t)
	for i := 0; i < 5; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/test", nil)
		req.RemoteAddr = "192.168.1.3:12345"
		r.ServeHTTP(w, req)
	}
	time.Sleep(1100 * time.Millisecond)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/test", nil)
	req.RemoteAddr = "192.168.1.3:12345"
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 after refill, got %d", w.Code)
	}
}

func TestRateLimit_SkipsWebSocket(t *testing.T) {
	_, r := setupTest(t)
	for i := 0; i < 10; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/test", nil)
		req.RemoteAddr = "192.168.1.4:12345"
		r.ServeHTTP(w, req)
	}
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/ws/danmaku?token=abc", nil)
	req.RemoteAddr = "192.168.1.4:12345"
	req.Header.Set("Upgrade", "websocket")
	r.ServeHTTP(w, req)
	if w.Code == http.StatusTooManyRequests {
		t.Error("WebSocket request should not be rate limited")
	}
}
