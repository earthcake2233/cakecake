package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"

	"cakecake/internal/pkg/traceid"
)

// TraceIDHeader is the response header (and accepted request header) carrying
// the trace ID.
const TraceIDHeader = "X-Trace-Id"

// TraceID injects a request-scoped trace ID into the context and response
// header, so upload handlers can pass it through to the transcode pipeline.
func TraceID() gin.HandlerFunc {
	return func(c *gin.Context) {
		tid := strings.TrimSpace(c.GetHeader(TraceIDHeader))
		if tid == "" {
			tid = traceid.New()
		}
		c.Writer.Header().Set(TraceIDHeader, tid)
		c.Request = c.Request.WithContext(traceid.WithContext(c.Request.Context(), tid))
		c.Next()
	}
}
