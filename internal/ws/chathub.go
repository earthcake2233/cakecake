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

	// connLocks protects concurrent writes to each connection.
	// gorilla/websocket does not allow concurrent writes.
	connLocks sync.Map
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
	h.connLocks.Store(c, &sync.Mutex{})
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
		h.connLocks.Delete(c)
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
		// Serialize writes per-connection to avoid concurrent write panic.
		l, _ := h.connLocks.LoadOrStore(c, &sync.Mutex{})
		l.(*sync.Mutex).Lock()
		if err := c.WriteMessage(websocket.TextMessage, data); err != nil {
			h.Leave(userID, c)
			_ = c.Close()
			l.(*sync.Mutex).Unlock()
		} else {
			l.(*sync.Mutex).Unlock()
		}
	}
}
