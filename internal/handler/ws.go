package handler

import (
    "context"
    "net/http"
	"encoding/json"
    "strconv"
    "strings"
    "time"

    "github.com/gin-gonic/gin"
    "github.com/gorilla/websocket"
    "go.uber.org/zap"

    "minibili/internal/model"
)

var wsUpgrader = websocket.Upgrader{
    ReadBufferSize:  4096,
    WriteBufferSize: 4096,
    CheckOrigin: func(r *http.Request) bool {
        return true
    },
}

// ServeDanmaku upgrades to WebSocket (F6, S-011).
// 已发布稿件：无 token 也可连接，用于实时弹幕与「正在看」计数；
// 非空但非法 token 仍返回 auth_failed。
// 支持 current_time 参数：按播放位置加载附近弹幕，不传则加载最新 200 条。
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

    cl := a.Hub.Join(videoID, conn)
    defer func() {
        a.Hub.Leave(videoID, cl)
        a.pushWatchingCount(videoID)
        _ = conn.Close()
    }()
    a.pushWatchingCount(videoID)

    // ---- history: N+1 修复 + current_time 支持 ----
    query := a.DB.Where("video_id = ?", videoID)
    if currentTime > 0 {
        query = query.Where("video_time BETWEEN ? AND ?", currentTime-10, currentTime+2)
    }
    query = query.Order("video_time ASC").Limit(200)
    var hist []model.Danmaku
    _ = query.Find(&hist).Error

    // Batch load users (fix N+1: was doing 200 separate queries)
    userIDs := make([]uint64, 0, len(hist))
    seen := make(map[uint64]bool)
    for _, d := range hist {
        if !seen[d.UserID] {
            seen[d.UserID] = true
            userIDs = append(userIDs, d.UserID)
        }
    }
    userMap := make(map[uint64]model.User)
    if len(userIDs) > 0 {
        var users []model.User
        _ = a.DB.Where("id IN ?", userIDs).Find(&users).Error
        for i := range users {
            userMap[users[i].ID] = users[i]
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
            "user":       model.DisplayUsername(&u),
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
