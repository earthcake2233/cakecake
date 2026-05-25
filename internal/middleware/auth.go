package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"minibili/internal/errcode"
	"minibili/internal/pkg/jwttoken"
	"minibili/internal/pkg/resp"
)

// CtxUserIDKey is the gin context key for JWT user id.
const CtxUserIDKey = "user_id"

// JWTAuth requires a valid access token (Rule R-API-4).
func JWTAuth(j *jwttoken.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.GetHeader("Authorization")
		if h == "" || !strings.HasPrefix(strings.ToLower(h), "bearer ") {
			resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
			c.Abort()
			return
		}
		raw := strings.TrimSpace(h[7:])
		uid, _, err := j.ParseAccess(raw)
		if err != nil {
			resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
			c.Abort()
			return
		}
		c.Set(CtxUserIDKey, uid)
		c.Next()
	}
}

// OptionalJWT parses Bearer token when present without failing.
func OptionalJWT(j *jwttoken.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.GetHeader("Authorization")
		if h != "" && strings.HasPrefix(strings.ToLower(h), "bearer ") {
			raw := strings.TrimSpace(h[7:])
			if uid, _, err := j.ParseAccess(raw); err == nil {
				c.Set(CtxUserIDKey, uid)
			}
		}
		c.Next()
	}
}

// UserID returns the authenticated user id or 0 if unset.
func UserID(c *gin.Context) (uint64, bool) {
	v, ok := c.Get(CtxUserIDKey)
	if !ok {
		return 0, false
	}
	id, ok := v.(uint64)
	return id, ok
}
