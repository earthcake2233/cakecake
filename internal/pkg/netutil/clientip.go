package netutil

import (
	"net"
	"strings"

	"github.com/gin-gonic/gin"
)

// ClientIP returns the best-effort client address behind reverse proxies.
func ClientIP(c *gin.Context) string {
	if c == nil || c.Request == nil {
		return ""
	}
	if xff := strings.TrimSpace(c.GetHeader("X-Forwarded-For")); xff != "" {
		parts := strings.Split(xff, ",")
		if ip := strings.TrimSpace(parts[0]); ip != "" {
			return ip
		}
	}
	if xri := strings.TrimSpace(c.GetHeader("X-Real-IP")); xri != "" {
		return xri
	}
	return strings.TrimSpace(c.ClientIP())
}

// IsLoopbackOrPrivate reports whether ip is local-only (no public geolocation).
func IsLoopbackOrPrivate(ip string) bool {
	parsed := net.ParseIP(strings.TrimSpace(ip))
	if parsed == nil {
		return false
	}
	return parsed.IsLoopback() || parsed.IsPrivate() || parsed.IsLinkLocalUnicast()
}
