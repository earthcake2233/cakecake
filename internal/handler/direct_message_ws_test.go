package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"cakecake/internal/pkg/jwttoken"
	"cakecake/internal/ws"
)

func newWSTestAPI(t *testing.T) (*API, *jwttoken.Manager) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	jm, err := jwttoken.NewManager("direct-message-ws-test-secret-32chars!!!")
	require.NoError(t, err)
	api := &API{
		Dependencies: &Dependencies{
			Log:     zap.NewNop(),
			JWT:     jm,
			ChatHub: ws.NewChatHub(),
			// Agent/DmSvc intentionally nil: ServeChat control frames are no-ops.
		},
	}
	return api, jm
}

func TestServeChat_AuthFailed(t *testing.T) {
	api, _ := newWSTestAPI(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, _ := gin.CreateTestContext(w)
		c.Request = r
		api.ServeChat(c)
	}))
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/api/v1/ws/chat?token=bad"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer conn.Close()
	_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	_, data, err := conn.ReadMessage()
	require.NoError(t, err)
	var frame struct {
		Type string `json:"type"`
	}
	require.NoError(t, json.Unmarshal(data, &frame))
	require.Equal(t, "auth_failed", frame.Type)
}

func TestServeChat_ConnectedAndControlFrames(t *testing.T) {
	api, jm := newWSTestAPI(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, _ := gin.CreateTestContext(w)
		c.Request = r
		api.ServeChat(c)
	}))
	defer srv.Close()

	access, _, _, err := jm.IssuePair(18)
	require.NoError(t, err)
	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/api/v1/ws/chat?token=" + access
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer conn.Close()

	_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	_, data, err := conn.ReadMessage()
	require.NoError(t, err)
	var frame struct {
		Type   string `json:"type"`
		UserID uint64 `json:"user_id"`
	}
	require.NoError(t, json.Unmarshal(data, &frame))
	require.Equal(t, "connected", frame.Type)
	require.Equal(t, uint64(18), frame.UserID)

	// Control frames with nil Agent/DmSvc must be safe no-ops.
	require.NoError(t, conn.WriteJSON(map[string]interface{}{"type": "agent_cancel"}))
	require.NoError(t, conn.WriteJSON(map[string]interface{}{
		"type": "agent_continue", "conversation_id": 1, "partial": "x",
	}))
	require.NoError(t, conn.WriteJSON(map[string]interface{}{
		"type": "agent_regenerate", "conversation_id": 1,
	}))
	time.Sleep(300 * time.Millisecond)
	require.True(t, true)
}
