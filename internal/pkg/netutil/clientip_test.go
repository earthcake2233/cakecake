package netutil

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestClientIP_XFF(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/", nil)
	c.Request.Header.Set("X-Forwarded-For", "203.0.113.1, 10.0.0.1")

	if got := ClientIP(c); got != "203.0.113.1" {
		t.Fatalf("ClientIP = %q, want %q", got, "203.0.113.1")
	}
}

func TestClientIP_XRealIP(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/", nil)
	c.Request.Header.Set("X-Real-IP", "10.0.0.5")

	if got := ClientIP(c); got != "10.0.0.5" {
		t.Fatalf("ClientIP = %q, want %q", got, "10.0.0.5")
	}
}

func TestClientIP_XFFPrecedesXRealIP(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/", nil)
	c.Request.Header.Set("X-Forwarded-For", "1.2.3.4")
	c.Request.Header.Set("X-Real-IP", "10.0.0.5")

	if got := ClientIP(c); got != "1.2.3.4" {
		t.Fatalf("ClientIP = %q, want %q", got, "1.2.3.4")
	}
}

func TestClientIP_NilContext(t *testing.T) {
	if got := ClientIP(nil); got != "" {
		t.Fatalf("ClientIP(nil) = %q, want empty", got)
	}
}

func TestClientIP_FallbackToRemoteAddr(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/", nil)
	c.Request.RemoteAddr = "192.0.2.42:54321"

	got := ClientIP(c)
	if got == "" {
		t.Fatal("ClientIP should fallback to a non-empty value from RemoteAddr")
	}
}

func TestIsLoopbackOrPrivate_Loopback(t *testing.T) {
	if !IsLoopbackOrPrivate("127.0.0.1") {
		t.Fatal("127.0.0.1 should be loopback")
	}
}

func TestIsLoopbackOrPrivate_Private10(t *testing.T) {
	if !IsLoopbackOrPrivate("10.0.0.1") {
		t.Fatal("10.0.0.1 should be private")
	}
}

func TestIsLoopbackOrPrivate_Private192(t *testing.T) {
	if !IsLoopbackOrPrivate("192.168.1.1") {
		t.Fatal("192.168.1.1 should be private")
	}
}

func TestIsLoopbackOrPrivate_LinkLocal(t *testing.T) {
	if !IsLoopbackOrPrivate("169.254.1.1") {
		t.Fatal("169.254.1.1 should be link-local")
	}
}

func TestIsLoopbackOrPrivate_PublicIP(t *testing.T) {
	if IsLoopbackOrPrivate("8.8.8.8") {
		t.Fatal("8.8.8.8 should not be loopback or private")
	}
}

func TestIsLoopbackOrPrivate_Invalid(t *testing.T) {
	if IsLoopbackOrPrivate("not-an-ip") {
		t.Fatal("invalid IP should return false")
	}
}
