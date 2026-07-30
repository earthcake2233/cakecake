package logger

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
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

func TestRedactCore_StringField(t *testing.T) {
	core, logs := observer.New(zap.DebugLevel)
	rc := &redactCore{Core: core}

	_ = rc.Write(zapcore.Entry{}, []zapcore.Field{
		zap.String("jwt_secret", "super-secret-value"),
		zap.String("safe_field", "visible-value"),
	})

	allLogs := logs.TakeAll()
	if len(allLogs) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(allLogs))
	}

	entry := allLogs[0]
	for _, f := range entry.Context {
		if f.Key == "jwt_secret" {
			if f.String != "[REDACTED]" {
				t.Errorf("jwt_secret should be redacted, got %q", f.String)
			}
		} else if f.Key == "safe_field" {
			if f.String != "visible-value" {
				t.Errorf("safe_field should be unchanged, got %q", f.String)
			}
		}
	}
}

func TestRedactCore_AnyField(t *testing.T) {
	core, logs := observer.New(zap.DebugLevel)
	rc := &redactCore{Core: core}

	_ = rc.Write(zapcore.Entry{}, []zapcore.Field{
		zap.Any("api_key", map[string]string{"key": "value"}),
		zap.Any("config_data", map[string]int{"port": 8080}),
	})

	allLogs := logs.TakeAll()
	if len(allLogs) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(allLogs))
	}

	entry := allLogs[0]
	for _, f := range entry.Context {
		if f.Key == "api_key" {
			if f.Type != zapcore.StringType || f.String != "[REDACTED]" {
				t.Errorf("api_key should be redacted to string, got type=%v val=%v", f.Type, f.String)
			}
		} else if f.Key == "config_data" {
			if f.Type == zapcore.StringType && f.String == "[REDACTED]" {
				t.Errorf("config_data should NOT be redacted")
			}
		}
	}
}

func TestIsSensitiveKey(t *testing.T) {
	tests := []struct {
		key  string
		want bool
	}{
		{"jwt_secret", true},
		{"JWT_SECRET", true},
		{"password", true},
		{"mysql_dsn", true},
		{"safe_field", false},
		{"username", false},
		{"api_key_openai", true},
		{"video_id", false},
	}
	for _, tt := range tests {
		got := isSensitiveKey(tt.key)
		if got != tt.want {
			t.Errorf("isSensitiveKey(%q) = %v, want %v", tt.key, got, tt.want)
		}
	}
}

func TestRedactedHelper(t *testing.T) {
	f := Redacted("some_key")
	if f.Key != "some_key" {
		t.Errorf("expected key 'some_key', got %q", f.Key)
	}
	if f.Type != zapcore.StringType {
		t.Errorf("expected StringType, got %v", f.Type)
	}
	if !strings.Contains(f.String, "REDACTED") {
		t.Errorf("value should contain REDACTED, got %q", f.String)
	}
}

func TestInitWithConfig_DisableRedaction(t *testing.T) {
	L = nil
	InitWithConfig(Config{DisableSensitiveFieldRedaction: true})
	if L == nil {
		t.Fatal("InitWithConfig should set L")
	}
}

func TestInitWithConfig_DefaultRedaction(t *testing.T) {
	L = nil
	InitWithConfig(Config{})
	if L == nil {
		t.Fatal("InitWithConfig should set L")
	}
}
