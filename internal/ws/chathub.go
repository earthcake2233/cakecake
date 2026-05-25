package ws

import (
	"encoding/json"
	"sync"

	"github.com/gorilla/websocket"
)

// ChatHub tracks WebSocket connections per user for private-message push.
type ChatHub struct {
	mu    sync.RWMutex
	users map[uint64]map[*websocket.Conn]struct{}
}

// NewChatHub creates an empty chat hub.
func NewChatHub() *ChatHub {
	return &ChatHub{users: make(map[uint64]map[*websocket.Conn]struct{})}
}

// Join registers a connection for a user.
func (h *ChatHub) Join(userID uint64, c *websocket.Conn) {
	if userID == 0 || c == nil {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	m := h.users[userID]
	if m == nil {
		m = make(map[*websocket.Conn]struct{})
		h.users[userID] = m
	}
	m[c] = struct{}{}
}

// Leave removes a connection for a user.
func (h *ChatHub) Leave(userID uint64, c *websocket.Conn) {
	if userID == 0 || c == nil {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if m, ok := h.users[userID]; ok {
		delete(m, c)
		if len(m) == 0 {
			delete(h.users, userID)
		}
	}
}

// PushJSON sends a payload to all connections of a user.
func (h *ChatHub) PushJSON(userID uint64, v interface{}) {
	data, err := json.Marshal(v)
	if err != nil {
		return
	}
	h.PushRaw(userID, data)
}

// PushRaw sends raw JSON bytes to all connections of a user.
func (h *ChatHub) PushRaw(userID uint64, data []byte) {
	if userID == 0 || len(data) == 0 {
		return
	}
	h.mu.RLock()
	m := h.users[userID]
	conns := make([]*websocket.Conn, 0, len(m))
	for c := range m {
		conns = append(conns, c)
	}
	h.mu.RUnlock()
	for _, c := range conns {
		if err := c.WriteMessage(websocket.TextMessage, data); err != nil {
			h.Leave(userID, c)
			_ = c.Close()
		}
	}
}
