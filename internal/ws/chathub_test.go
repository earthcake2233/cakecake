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

func TestChatHub_New(t *testing.T) {
	h := NewChatHub()
	require.NotNil(t, h)
}

func TestChatHub_JoinLeave(t *testing.T) {
	h := NewChatHub()
	c := &websocket.Conn{}

	h.Join(42, c)
	h.Join(42, c) // duplicate join should be fine

	h.Leave(42, c)
}

func TestChatHub_JoinZeroUser(t *testing.T) {
	h := NewChatHub()
	h.Join(0, &websocket.Conn{}) // should not panic
}

func TestChatHub_JoinNilConn(t *testing.T) {
	h := NewChatHub()
	h.Join(42, nil) // should not panic
}

func TestChatHub_LeaveNonExistent(t *testing.T) {
	h := NewChatHub()
	h.Leave(42, &websocket.Conn{}) // should not panic
}

func TestChatHub_LeaveZeroUser(t *testing.T) {
	h := NewChatHub()
	h.Leave(0, &websocket.Conn{}) // should not panic
}

func TestChatHub_LeaveNilConn(t *testing.T) {
	h := NewChatHub()
	h.Leave(42, nil) // should not panic
}

func TestChatHub_JoinMultipleUsers(t *testing.T) {
	h := NewChatHub()
	c1 := &websocket.Conn{}
	c2 := &websocket.Conn{}
	c3 := &websocket.Conn{}

	h.Join(1, c1)
	h.Join(1, c2)
	h.Join(2, c3)

	// Leave one user's connection
	h.Leave(1, c1)
	h.Leave(2, c3)

	// Leave remaining
	h.Leave(1, c2)
}

func TestChatHub_PushJSON_InvalidJSON(t *testing.T) {
	h := NewChatHub()
	// PushJSON with a channel should cause json.Marshal to fail silently
	h.PushJSON(42, make(chan int)) // should not panic
}

func TestChatHub_PushRaw_ZeroUser(t *testing.T) {
	h := NewChatHub()
	h.PushRaw(0, []byte("data")) // should not panic
}

func TestChatHub_PushRaw_EmptyData(t *testing.T) {
	h := NewChatHub()
	h.PushRaw(42, nil)      // should not panic
	h.PushRaw(42, []byte{}) // should not panic
}

func TestChatHub_PushRaw_NoConnections(t *testing.T) {
	h := NewChatHub()
	h.PushRaw(99, []byte("hello")) // no connections, should not panic
}

func TestChatHub_ConcurrentJoin(t *testing.T) {
	h := NewChatHub()
	var wg sync.WaitGroup
	n := 50

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c := &websocket.Conn{}
			h.Join(1, c)
		}()
	}
	wg.Wait()

	// All 50 connections should be registered
	// Clean up
	for i := 0; i < n; i++ {
		h.Leave(1, &websocket.Conn{})
	}
}

func TestChatHub_ConcurrentLeave(t *testing.T) {
	h := NewChatHub()
	var wg sync.WaitGroup
	n := 50

	conns := make([]*websocket.Conn, n)
	for i := 0; i < n; i++ {
		conns[i] = &websocket.Conn{}
		h.Join(1, conns[i])
	}

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			h.Leave(1, conns[idx])
		}(i)
	}
	wg.Wait()
}

func TestChatHub_PushJSONWithRealConn(t *testing.T) {
	h := NewChatHub()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		up := websocket.Upgrader{}
		conn, err := up.Upgrade(w, r, nil)
		require.NoError(t, err)
		h.Join(100, conn)
		defer h.Leave(100, conn)
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

	time.Sleep(50 * time.Millisecond)

	payload := map[string]string{"type": "chat", "text": "hello"}
	h.PushJSON(100, payload)

	_, msg, err := conn.ReadMessage()
	require.NoError(t, err)
	var received map[string]string
	require.NoError(t, json.Unmarshal(msg, &received))
	require.Equal(t, "chat", received["type"])
	require.Equal(t, "hello", received["text"])
}

func TestChatHub_PushRawWithRealConn(t *testing.T) {
	h := NewChatHub()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		up := websocket.Upgrader{}
		conn, err := up.Upgrade(w, r, nil)
		require.NoError(t, err)
		h.Join(101, conn)
		defer h.Leave(101, conn)
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

	time.Sleep(50 * time.Millisecond)

	payload := "{\"type\":\"push\",\"data\":42}"
	h.PushRaw(101, []byte(payload))

	_, msg, err := conn.ReadMessage()
	require.NoError(t, err)
	require.JSONEq(t, payload, string(msg))
}

func TestChatHub_PushToMultipleConns(t *testing.T) {
	h := NewChatHub()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		up := websocket.Upgrader{}
		conn, err := up.Upgrade(w, r, nil)
		require.NoError(t, err)
		h.Join(200, conn)
		defer h.Leave(200, conn)
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				break
			}
		}
	}))
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	conn1, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer conn1.Close()

	conn2, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer conn2.Close()

	time.Sleep(50 * time.Millisecond)

	h.PushRaw(200, []byte("multi"))

	_, msg1, err := conn1.ReadMessage()
	require.NoError(t, err)
	require.Equal(t, "multi", string(msg1))

	_, msg2, err := conn2.ReadMessage()
	require.NoError(t, err)
	require.Equal(t, "multi", string(msg2))
}
