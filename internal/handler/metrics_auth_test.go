package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"cakecake/internal/aigateway"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestMetricsEndpointAuth(t *testing.T) {
	api, _, jm := newTestAPI(t)
	api.Cfg.MetricsToken = "scrape-secret"
	r := gin.New()
	RegisterRoutes(r, api, jm, "test")
	// Emit one observation so the counter family is present in the exposition.
	aigateway.RecordLLMRequest("ok")

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusUnauthorized, w.Code)

	req = httptest.NewRequest(http.MethodGet, "/metrics", nil)
	req.Header.Set("Authorization", "Bearer wrong")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusUnauthorized, w.Code)

	req = httptest.NewRequest(http.MethodGet, "/metrics", nil)
	req.Header.Set("Authorization", "Bearer scrape-secret")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), "cakecake_llm_requests_total")
}
