package ws

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const writeBufferLen = 64
const writeWait = 5 * time.Second

type Client struct {
	Conn *websocket.Conn
	send chan []byte
}

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

type Hub struct {
	mu    sync.Mutex
	rooms map[uint64]map[*Client]struct{}
}

func NewHub() *Hub {
	return &Hub{rooms: make(map[uint64]map[*Client]struct{})}
}

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

func (h *Hub) RoomSize(videoID uint64) int {
	h.mu.Lock()
	defer h.mu.Unlock()
	if m, ok := h.rooms[videoID]; ok {
		return len(m)
	}
	return 0
}

func (h *Hub) TotalConnections() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	n := 0
	for _, m := range h.rooms {
		n += len(m)
	}
	return n
}

func (h *Hub) BroadcastJSON(videoID uint64, v interface{}) {
	data, err := json.Marshal(v)
	if err != nil {
		return
	}
	h.BroadcastRaw(videoID, data)
}

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
