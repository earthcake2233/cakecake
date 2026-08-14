package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"cakecake/internal/pkg/traceid"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestTraceIDMiddleware_GeneratesAndCarries(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(TraceID())
	r.GET("/x", func(c *gin.Context) {
		c.String(http.StatusOK, traceid.FromContext(c.Request.Context()))
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	r.ServeHTTP(w, req)
	require.NotEmpty(t, w.Body.String(), "context should carry a generated trace ID")
	require.Equal(t, w.Body.String(), w.Header().Get(TraceIDHeader))

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set(TraceIDHeader, "client-trace")
	r.ServeHTTP(w, req)
	require.Equal(t, "client-trace", w.Body.String())
}
