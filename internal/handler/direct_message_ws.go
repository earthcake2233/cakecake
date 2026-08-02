package handler

import (
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
		if _, _, err := conn.ReadMessage(); err != nil {
			break
		}
	}
}
