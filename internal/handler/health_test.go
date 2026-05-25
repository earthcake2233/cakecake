package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"minibili/internal/pkg/jwttoken"
	"minibili/internal/ws"
)

func TestHealth(t *testing.T) {
	gin.SetMode(gin.TestMode)
	jm, err := jwttoken.NewManager("test-secret-key-for-health-only-32chars")
	require.NoError(t, err)
	api := &API{
		Dependencies: &Dependencies{
			Log: zap.NewNop(),
			Hub: ws.NewHub(),
			JWT: jm,
		},
	}
	r := gin.New()
	r.GET("/api/v1/health", api.Health)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), `"code":0`)
	require.Contains(t, w.Body.String(), `"status":"ok"`)
}
