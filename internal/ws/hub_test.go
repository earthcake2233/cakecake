package ws

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
)

func TestHubJoinLeave(t *testing.T) {
	h := NewHub()
	require.Equal(t, 0, h.TotalConnections())

	conn := &websocket.Conn{}
	h.Join(100, conn)
	require.Equal(t, 1, h.TotalConnections())
	require.Equal(t, 1, h.RoomSize(100))
	require.Equal(t, 0, h.RoomSize(200))

	h.Leave(100, conn)
	require.Equal(t, 0, h.TotalConnections())
	require.Equal(t, 0, h.RoomSize(100))
}

func TestHubMultipleRooms(t *testing.T) {
	h := NewHub()
	c1, c2, c3 := &websocket.Conn{}, &websocket.Conn{}, &websocket.Conn{}

	h.Join(1, c1)
	h.Join(1, c2)
	h.Join(2, c3)
	require.Equal(t, 3, h.TotalConnections())
	require.Equal(t, 2, h.RoomSize(1))
	require.Equal(t, 1, h.RoomSize(2))

	h.Leave(1, c1)
	require.Equal(t, 2, h.TotalConnections())
	require.Equal(t, 1, h.RoomSize(1))

	// Leaving the last connection should remove the room
	h.Leave(2, c3)
	require.Equal(t, 1, h.TotalConnections())
	require.Equal(t, 1, h.RoomSize(1))
	require.Equal(t, 0, h.RoomSize(2))
}

// waitRoom waits up to 3s for room to reach the expected size.
func waitRoom(h *Hub, room uint64, expect int) bool {
	for i := 0; i < 300; i++ {
		if h.RoomSize(room) == expect {
			return true
		}
		time.Sleep(10 * time.Millisecond)
	}
	return false
}

func TestHubBroadcastJSON(t *testing.T) {
	h := NewHub()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		up := websocket.Upgrader{}
		conn, err := up.Upgrade(w, r, nil)
		require.NoError(t, err)
		h.Join(42, conn)
		defer h.Leave(42, conn)
		// Read until close
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				break
			}
		}
	}))
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer conn.Close()

	require.True(t, waitRoom(h, 42, 1), "connection should be in room 42")

	payload := map[string]string{"type": "danmaku", "text": "hello"}
	h.BroadcastJSON(42, payload)

	_, msg, err := conn.ReadMessage()
	require.NoError(t, err)
	var received map[string]string
	require.NoError(t, json.Unmarshal(msg, &received))
	require.Equal(t, "danmaku", received["type"])
	require.Equal(t, "hello", received["text"])
}

func TestHubBroadcastRaw(t *testing.T) {
	h := NewHub()
	payload := `{"type":"test","data":123}`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		up := websocket.Upgrader{}
		conn, _ := up.Upgrade(w, r, nil)
		h.Join(7, conn)
		defer h.Leave(7, conn)
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				break
			}
		}
	}))
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer conn.Close()

	require.True(t, waitRoom(h, 7, 1), "connection should be in room 7")

	h.BroadcastRaw(7, []byte(payload))
	_, msg, err := conn.ReadMessage()
	require.NoError(t, err)
	require.JSONEq(t, payload, string(msg))
}

func TestHubConcurrentJoin(t *testing.T) {
	h := NewHub()
	var wg sync.WaitGroup
	n := 100

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c := &websocket.Conn{}
			h.Join(1, c)
		}()
	}
	wg.Wait()
	require.Equal(t, n, h.RoomSize(1))
	require.Equal(t, n, h.TotalConnections())
}
