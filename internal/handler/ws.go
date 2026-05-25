package handler

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"

	"minibili/internal/model"
)

var wsUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// ServeDanmaku upgrades to WebSocket (F6, S-011).
// 已发布稿件：无 token 也可连接，用于实时弹幕与「正在看」计数；非空但非法 token 仍返回 auth_failed。
func (a *API) ServeDanmaku(c *gin.Context) {
	videoID, _ := strconv.ParseUint(c.Query("video_id"), 10, 64)
	if videoID == 0 {
		c.Status(http.StatusBadRequest)
		return
	}
	token := strings.TrimSpace(c.Query("token"))
	var v model.Video
	if err := a.DB.First(&v, videoID).Error; err != nil {
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
		if v.Status != "published" && v.UserID != uid {
			c.Status(http.StatusNotFound)
			return
		}
	} else {
		if v.Status != "published" {
			c.Status(http.StatusNotFound)
			return
		}
	}
	conn, err := wsUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer func() {
		a.Hub.Leave(videoID, conn)
		a.pushWatchingCount(videoID)
		_ = conn.Close()
	}()
	a.Hub.Join(videoID, conn)
	a.pushWatchingCount(videoID)

	var hist []model.Danmaku
	_ = a.DB.Where("video_id = ?", videoID).Order("id DESC").Limit(200).Find(&hist).Error
	items := make([]gin.H, 0, len(hist))
	for i := len(hist) - 1; i >= 0; i-- {
		d := hist[i]
		var u model.User
		_ = a.DB.First(&u, d.UserID).Error
		items = append(items, gin.H{
			"id":         d.ID,
			"content":    d.Content,
			"color":      strings.ToUpper(strings.TrimSpace(d.Color)),
			"type":       d.Type,
			"font_size":  danmakuFontSizeField(d),
			"video_time": d.VideoTime,
			"user":       model.DisplayUsername(&u),
			"created_at": d.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}
	watching := a.Hub.RoomSize(videoID)
	_ = conn.WriteJSON(gin.H{"type": "history", "items": items, "watching_count": watching})

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
