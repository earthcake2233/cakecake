package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"minibili/internal/errcode"
	"minibili/internal/pkg/jwttoken"
	"minibili/internal/pkg/resp"
)

// CtxAdminIDKey is the gin context key for admin JWT id.
const CtxAdminIDKey = "admin_id"

// AdminJWTAuth requires a valid admin access token.
func AdminJWTAuth(j *jwttoken.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.GetHeader("Authorization")
		if h == "" || !strings.HasPrefix(strings.ToLower(h), "bearer ") {
			resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
			c.Abort()
			return
		}
		raw := strings.TrimSpace(h[7:])
		aid, _, err := j.ParseAdminAccess(raw)
		if err != nil {
			resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
			c.Abort()
			return
		}
		c.Set(CtxAdminIDKey, aid)
		c.Next()
	}
}

// AdminID returns authenticated admin id.
func AdminID(c *gin.Context) (uint64, bool) {
	v, ok := c.Get(CtxAdminIDKey)
	if !ok {
		return 0, false
	}
	id, ok := v.(uint64)
	return id, ok
}
