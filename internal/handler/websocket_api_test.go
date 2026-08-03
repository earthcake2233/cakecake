package handler

import (
	"cakecake/internal/model/video"
	"encoding/json"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestWebSocketDanmaku(t *testing.T) {
	api, r, _ := newTestAPI(t)
	tokenA, uidA := covRegister(t, r, "cowsw", "password12")
	vid := covSeedVideo(t, api, uidA, "ws video", video.StatusPublished)

	srv := httptest.NewServer(r)
	defer srv.Close()
	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/api/v1/ws/danmaku?video_id=" + u64s(vid)
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer conn.Close()

	var hist struct {
		Type string `json:"type"`
	}
	for {
		_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
		_, msg, err := conn.ReadMessage()
		require.NoError(t, err)
		require.NoError(t, json.Unmarshal(msg, &hist))
		if hist.Type == "history" {
			break
		}
	}
	require.Equal(t, "history", hist.Type)

	// Broadcast a danmaku over HTTP; the WS room should receive it.
	covOK(t, covReq(t, r, "POST", "/api/v1/videos/"+u64s(vid)+"/danmaku", tokenA, map[string]any{"content": "hi", "video_time": 1, "type": "scroll"}), http.StatusOK)
	_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	_, msg2, err := conn.ReadMessage()
	require.NoError(t, err)
	var frame struct {
		Type string `json:"type"`
	}
	require.NoError(t, json.Unmarshal(msg2, &frame))
	require.Equal(t, "danmaku", frame.Type)
}

func TestWebSocketErrorPaths(t *testing.T) {
	api, r, _ := newTestAPI(t)
	_, uidA := covRegister(t, r, "cowse", "password12")
	vid := covSeedVideo(t, api, uidA, "ws err video", video.StatusPublished)
	srv := httptest.NewServer(r)
	defer srv.Close()

	// Invalid video id -> 400 before upgrade
	resp, err := http.Get(srv.URL + "/api/v1/ws/danmaku?video_id=0")
	require.NoError(t, err)
	resp.Body.Close()
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)

	// Missing video -> 404
	resp2, err := http.Get(srv.URL + "/api/v1/ws/danmaku?video_id=99999")
	require.NoError(t, err)
	resp2.Body.Close()
	require.Equal(t, http.StatusNotFound, resp2.StatusCode)

	// Bad token -> upgrade + auth_failed frame
	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/api/v1/ws/danmaku?video_id=" + u64s(vid) + "&token=bad-token"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	_, msg, err := conn.ReadMessage()
	require.NoError(t, err)
	var frame struct {
		Type string `json:"type"`
	}
	require.NoError(t, json.Unmarshal(msg, &frame))
	require.Equal(t, "auth_failed", frame.Type)
	conn.Close()
}
