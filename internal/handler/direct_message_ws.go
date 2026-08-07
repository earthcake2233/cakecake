package handler

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// ServeChat upgrades to WebSocket for private-message push (token required).
func (a *API) ServeChat(c *gin.Context) {
	token := strings.TrimSpace(c.Query("token"))
	if token == "" {
		c.Status(http.StatusUnauthorized)
		return
	}
	uid, _, err := a.JWT.ParseAccess(token)
	if err != nil || uid == 0 {
		conn, errUp := wsUpgrader.Upgrade(c.Writer, c.Request, nil)
		if errUp == nil && conn != nil {
			_ = conn.WriteJSON(wsErrorFrame{Type: "auth_failed", Msg: "Token 无效或已过期"})
			_ = conn.Close()
		}
		return
	}
	conn, err := wsUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	if a.ChatHub == nil {
		_ = conn.Close()
		return
	}
	defer func() {
		a.ChatHub.Leave(uid, conn)
		_ = conn.Close()
	}()
	a.ChatHub.Join(uid, conn)
	_ = conn.WriteJSON(wsConnectedFrame{Type: "connected", UserID: uid})

	for {
		_ = conn.SetReadDeadline(time.Now().Add(120 * time.Second))
		_, data, err := conn.ReadMessage()
		if err != nil {
			break
		}
		var req struct {
			Type           string `json:"type"`
			ConversationID uint64 `json:"conversation_id"`
			Partial        string `json:"partial"`
		}
		if err := json.Unmarshal(data, &req); err != nil {
			continue
		}
		switch req.Type {
		case "agent_cancel":
			a.pauseAgentReply(uid)
		case "agent_regenerate":
			a.regenerateAgentReply(uid, req.ConversationID)
		case "agent_continue":
			go a.resumeAgentReply(uid, req.ConversationID, req.Partial)
		}
	}
}
