//go:build integration

package handler

import (
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// Black-box check against a running server (optional CI / local).
// Run: MINIBILI_TEST_BASE_URL=http://127.0.0.1:8080 go test -tags=integration ./internal/handler/...
func TestLiveHealthEndpoint(t *testing.T) {
	base := strings.TrimSuffix(os.Getenv("MINIBILI_TEST_BASE_URL"), "/")
	if base == "" {
		t.Skip("set MINIBILI_TEST_BASE_URL to run live integration test")
	}
	c := &http.Client{Timeout: 5 * time.Second}
	resp, err := c.Get(base + "/api/v1/health")
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
}
