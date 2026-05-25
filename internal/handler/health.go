package handler

import (
	"github.com/gin-gonic/gin"

	"minibili/internal/pkg/resp"
)

// Health is a liveness probe (no DB dependency).
func (a *API) Health(c *gin.Context) {
	resp.OK(c, gin.H{"status": "ok"})
}
