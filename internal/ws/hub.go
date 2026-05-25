package ws

import (
	"encoding/json"
	"sync"

	"github.com/gorilla/websocket"
)

// Hub tracks WebSocket connections per video room for realtime fan-out.
type Hub struct {
	mu    sync.Mutex
	rooms map[uint64]map[*websocket.Conn]struct{}
}

// NewHub creates an empty hub.
func NewHub() *Hub {
	return &Hub{rooms: make(map[uint64]map[*websocket.Conn]struct{})}
}

// Join adds a connection to a video room.
func (h *Hub) Join(videoID uint64, c *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	m := h.rooms[videoID]
	if m == nil {
		m = make(map[*websocket.Conn]struct{})
		h.rooms[videoID] = m
	}
	m[c] = struct{}{}
}

// Leave removes a connection from a room.
func (h *Hub) Leave(videoID uint64, c *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if m, ok := h.rooms[videoID]; ok {
		delete(m, c)
		if len(m) == 0 {
			delete(h.rooms, videoID)
		}
	}
}

// RoomSize returns the number of open WebSocket connections in a video room.
func (h *Hub) RoomSize(videoID uint64) int {
	h.mu.Lock()
	defer h.mu.Unlock()
	if m, ok := h.rooms[videoID]; ok {
		return len(m)
	}
	return 0
}

// TotalConnections returns open WebSocket connections across all video rooms.
func (h *Hub) TotalConnections() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	n := 0
	for _, m := range h.rooms {
		n += len(m)
	}
	return n
}

// BroadcastJSON sends a JSON message to every connection in a room.
func (h *Hub) BroadcastJSON(videoID uint64, v interface{}) {
	data, err := json.Marshal(v)
	if err != nil {
		return
	}
	h.BroadcastRaw(videoID, data)
}

// BroadcastRaw sends the same JSON payload bytes to every connection in a room
// (used after Redis Pub/Sub relay to avoid double-marshalling).
func (h *Hub) BroadcastRaw(videoID uint64, data []byte) {
	if len(data) == 0 {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	m := h.rooms[videoID]
	for c := range m {
		if err := c.WriteMessage(websocket.TextMessage, data); err != nil {
			_ = c.Close()
			delete(m, c)
		}
	}
}
