package ws

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const writeBufferLen = 64
const writeWait = 5 * time.Second

// Client is a single WebSocket connection with a buffered write queue.
type Client struct {
	Conn *websocket.Conn
	send chan []byte
}

// Send queues a message to the client's write pump.
func (c *Client) Send(data []byte) bool {
	if data == nil || c.send == nil {
		return false
	}
	select {
	case c.send <- data:
		return true
	default:
		return false
	}
}

func (c *Client) writePump() {
	defer func() {
		if c.Conn != nil {
			defer func() { _ = recover() }()
			_ = c.Conn.Close()
		}
	}()
	for msg := range c.send {
		_ = c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
		if err := c.Conn.WriteMessage(websocket.TextMessage, msg); err != nil {
			break
		}
	}
}

// Hub tracks live WebSocket clients per video room.
type Hub struct {
	mu    sync.Mutex
	rooms map[uint64]map[*Client]struct{}
}

// NewHub creates an empty Hub.
func NewHub() *Hub {
	return &Hub{rooms: make(map[uint64]map[*Client]struct{})}
}

// Join registers a WebSocket connection in a video room.
func (h *Hub) Join(videoID uint64, conn *websocket.Conn) *Client {
	cl := &Client{Conn: conn, send: make(chan []byte, writeBufferLen)}
	go cl.writePump()
	h.mu.Lock()
	defer h.mu.Unlock()
	m := h.rooms[videoID]
	if m == nil {
		m = make(map[*Client]struct{})
		h.rooms[videoID] = m
	}
	m[cl] = struct{}{}
	return cl
}

// Leave removes a client from a video room.
func (h *Hub) Leave(videoID uint64, cl *Client) {
	if cl == nil {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if m, ok := h.rooms[videoID]; ok {
		delete(m, cl)
		if cl.send != nil {
			close(cl.send)
		}
		if len(m) == 0 {
			delete(h.rooms, videoID)
		}
	}
}

// RoomSize returns the number of live clients in a video room.
func (h *Hub) RoomSize(videoID uint64) int {
	h.mu.Lock()
	defer h.mu.Unlock()
	if m, ok := h.rooms[videoID]; ok {
		return len(m)
	}
	return 0
}

// TotalConnections returns the total number of live WebSocket clients.
func (h *Hub) TotalConnections() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	n := 0
	for _, m := range h.rooms {
		n += len(m)
	}
	return n
}

// BroadcastJSON JSON-encodes data and broadcasts it to a video room.
func (h *Hub) BroadcastJSON(videoID uint64, v interface{}) {
	data, err := json.Marshal(v)
	if err != nil {
		return
	}
	h.BroadcastRaw(videoID, data)
}

// BroadcastRaw broadcasts raw bytes to a video room without re-encoding.
func (h *Hub) BroadcastRaw(videoID uint64, data []byte) {
	if len(data) == 0 {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	m := h.rooms[videoID]
	if len(m) == 0 {
		return
	}
	for cl := range m {
		select {
		case cl.send <- data:
		default:
		}
	}
}
