package logger

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func TestInit_SetsL(t *testing.T) {
	L = nil
	Init()
	if L == nil {
		t.Fatal("Init() should set L to non-nil")
	}
}

func TestGinMiddleware_ReturnsNonNil(t *testing.T) {
	lg, err := zap.NewProduction()
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}
	handler := GinMiddleware(lg)
	if handler == nil {
		t.Fatal("GinMiddleware returned nil handler")
	}
}

func TestGinMiddleware_SetsLoggerInContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	lg, err := zap.NewProduction()
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/", nil)

	handler := GinMiddleware(lg)
	handler(c)

	val, exists := c.Get("logger")
	if !exists {
		t.Fatal("\"logger\" key not set in context")
	}
	if val == nil {
		t.Fatal("\"logger\" value is nil")
	}
	if _, ok := val.(*zap.Logger); !ok {
		t.Fatalf("\"logger\" value is not *zap.Logger, got %T", val)
	}
}
