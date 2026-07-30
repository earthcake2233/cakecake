package logger

import (
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// L is the global logger for non-HTTP contexts (consumers, background jobs).
var L *zap.Logger

// Config allows customising the logger on initialisation.
type Config struct {
	// DisableSensitiveFieldRedaction disables the automatic sensitive-field
	// redaction core.  Intended only for integration tests that need raw values.
	DisableSensitiveFieldRedaction bool
}

// Init initialises production zap logger per Skill S-003.
func Init() {
	InitWithConfig(Config{})
}

// InitWithConfig initialises the zap logger with optional configuration.
func InitWithConfig(cfg Config) {
	config := zap.NewProductionConfig()
	config.EncoderConfig.TimeKey = "timestamp"
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	lg, err := config.Build()
	if err != nil {
		panic(err)
	}
	if !cfg.DisableSensitiveFieldRedaction {
		L = lg.WithOptions(zap.WrapCore(func(c zapcore.Core) zapcore.Core {
			return &redactCore{Core: c}
		}))
	} else {
		L = lg
	}
}

// redactCore wraps a zapcore.Core and redacts fields whose keys match known
// sensitive patterns (passwords, secrets, API keys, etc.).
type redactCore struct {
	zapcore.Core
}

// sensitiveKeys is a list of substrings to match against lower-cased field
// names.  Any field whose name contains any of these patterns will have its
// value replaced with "[REDACTED]".
//
//nolint:gochecknoglobals
var sensitiveKeys = []string{
	"secret", "password", "passwd", "api_key", "apikey",
	"access_key", "accesskey", "token", "jwt",
	"dsn", "mysql_dsn",
	"private_key", "privatekey",
}

// isSensitiveKey returns true when the lower-cased key contains any sensitive
// substring.
func isSensitiveKey(key string) bool {
	lower := key
	needsLower := false
	for i := 0; i < len(lower); i++ {
		if lower[i] >= 'A' && lower[i] <= 'Z' {
			needsLower = true
			break
		}
	}
	if needsLower {
		lower = strings.ToLower(key)
	}
	for _, pat := range sensitiveKeys {
		if strings.Contains(lower, pat) {
			return true
		}
	}
	return false
}

// redactField returns a sanitised copy of the field, hiding the original value
// when the field key matches a sensitive pattern.
func redactField(f zapcore.Field) zapcore.Field {
	if !isSensitiveKey(f.Key) {
		return f
	}
	switch f.Type {
	case zapcore.StringType:
		f.String = "[REDACTED]"
	case zapcore.Int64Type, zapcore.Int32Type, zapcore.Int16Type, zapcore.Int8Type,
		zapcore.Uint64Type, zapcore.Uint32Type, zapcore.Uint16Type, zapcore.Uint8Type,
		zapcore.Float64Type, zapcore.BoolType:
		f.Integer = 0
	default:
		// zapcore.ReflectType, zapcore.AnyType, etc. ? degrade to a string.
		f.Type = zapcore.StringType
		f.String = "[REDACTED]"
	}
	return f
}

// Write implements zapcore.Core.  It redacts sensitive fields then delegates
// to the underlying core.
func (rc *redactCore) Write(entry zapcore.Entry, fields []zapcore.Field) error {
	for i := range fields {
		fields[i] = redactField(fields[i])
	}
	return rc.Core.Write(entry, fields)
}

// Check implements zapcore.Core.
func (rc *redactCore) Check(entry zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if rc.Enabled(entry.Level) {
		return ce.AddCore(entry, rc)
	}
	return ce
}

// GinMiddleware injects *zap.Logger into gin.Context as "logger".
func GinMiddleware(lg *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("logger", lg)
		c.Next()
	}
}

// Redacted returns a zap.String field whose value is unconditionally
// "[REDACTED]" regardless of the provided value.  Callers should use
// this when they intentionally want to log that a sensitive field was
// present without exposing its content.
func Redacted(key string) zap.Field {
	return zap.String(key, "[REDACTED]")
}
