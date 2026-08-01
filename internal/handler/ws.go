package handler

import (
	"cakecake/internal/model/user"
	"cakecake/internal/model/video"
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

var wsUpgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// ServeDanmaku upgrades to WebSocket (F6, S-011).
// Published videos: connections allowed without a token for real-time danmaku and watching count;
// non-empty but invalid tokens still return auth_failed.
// Supports current_time param: loads nearby danmaku by playback position; without it, loads the latest 200.
// ServeDanmaku godoc
// @Summary      WebSocket: danmaku stream
// @Description  WebSocket endpoint for real-time danmaku messages
// @Tags         WebSocket
// @Success      101 {string} string "Switching Protocols"
// @Router       /api/v1/ws/danmaku [get]
func (a *API) ServeDanmaku(c *gin.Context) {
	videoID, _ := strconv.ParseUint(c.Query("video_id"), 10, 64)
	if videoID == 0 {
		c.Status(http.StatusBadRequest)
		return
	}
	token := strings.TrimSpace(c.Query("token"))

	// ---- current_time support ----
	var currentTime float64
	if ct := c.Query("current_time"); ct != "" {
		currentTime, _ = strconv.ParseFloat(ct, 64)
	}

	v, errV := a.VideoSvc.GetVideoByID(c.Request.Context(), videoID)
	if errV != nil {
		c.Status(http.StatusNotFound)
		return
	}
	if token != "" {
		uid, _, err := a.JWT.ParseAccess(token)
		if err != nil {
			conn, errUp := wsUpgrader.Upgrade(c.Writer, c.Request, nil)
			if errUp == nil && conn != nil {
				_ = conn.WriteJSON(gin.H{"type": "auth_failed", "msg": "Token 无效或已过期"})
				_ = conn.Close()
			}
			return
		}
		if v.Status != video.StatusPublished && v.UserID != uid {
			c.Status(http.StatusNotFound)
			return
		}
	} else {
		if v.Status != video.StatusPublished {
			c.Status(http.StatusNotFound)
			return
		}
	}

	conn, err := wsUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}

	cl := a.Hub.Join(videoID, conn)
	defer func() {
		a.Hub.Leave(videoID, cl)
		a.pushWatchingCount(videoID)
		_ = conn.Close()
	}()
	a.pushWatchingCount(videoID)

	// ---- history: N+1 fix + current_time support ----
	hist, _ := a.DanmakuSvc.ListHistory(c.Request.Context(), videoID, currentTime, 200)

	// Batch load users (fix N+1: was doing 200 separate queries)
	userIDs := make([]uint64, 0, len(hist))
	seen := make(map[uint64]bool)
	for _, d := range hist {
		if !seen[d.UserID] {
			seen[d.UserID] = true
			userIDs = append(userIDs, d.UserID)
		}
	}
	userMap := make(map[uint64]user.User)
	if len(userIDs) > 0 {
		users := a.UserSvc.BatchGetUsers(c.Request.Context(), userIDs)
		for id, u := range users {
			userMap[id] = *u
		}
	}

	items := make([]gin.H, 0, len(hist))
	for _, d := range hist {
		u := userMap[d.UserID]
		items = append(items, gin.H{
			"id":         d.ID,
			"content":    d.Content,
			"color":      strings.ToUpper(strings.TrimSpace(d.Color)),
			"type":       d.Type,
			"font_size":  danmakuFontSizeField(d),
			"video_time": d.VideoTime,
			"user":       user.DisplayUsername(&u),
			"created_at": d.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	watching := a.Hub.RoomSize(videoID)
	historyPayload := gin.H{"type": "history", "items": items, "watching_count": watching}
	historyBytes, _ := json.Marshal(historyPayload)
	cl.Send(historyBytes)

	for {
		_ = conn.SetReadDeadline(time.Now().Add(120 * time.Second))
		if _, _, err := conn.ReadMessage(); err != nil {
			break
		}
	}
}

func (a *API) pushWatchingCount(videoID uint64) {
	if a.Hub == nil {
		return
	}
	n := a.Hub.RoomSize(videoID)
	payload := gin.H{"type": "watching", "count": n}
	if a.DanmakuRelay != nil {
		if err := a.DanmakuRelay.Publish(context.Background(), videoID, payload); err != nil {
			a.Log.Error("danmaku relay publish watching", zap.Error(err))
		}
		return
	}
	a.Hub.BroadcastJSON(videoID, payload)
}
