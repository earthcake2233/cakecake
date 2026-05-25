package logger

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// L is the global logger for non-HTTP contexts (consumers, background jobs).
var L *zap.Logger

// Init initializes production zap logger per Skill S-003.
func Init() {
	config := zap.NewProductionConfig()
	config.EncoderConfig.TimeKey = "timestamp"
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	lg, err := config.Build()
	if err != nil {
		panic(err)
	}
	L = lg
}

// GinMiddleware injects *zap.Logger into gin.Context as "logger".
func GinMiddleware(lg *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("logger", lg)
		c.Next()
	}
}
