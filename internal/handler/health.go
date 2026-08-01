package handler

import (
	"github.com/gin-gonic/gin"

	"cakecake/internal/pkg/resp"
)

// Health is a liveness probe (no DB dependency).
// Health godoc
// @Summary      Health check
// @Description  Server health check endpoint
// @Tags         System
// @Success      200 {object} map[string]string
// @Router       /health [get]
func (a *API) Health(c *gin.Context) {
	resp.OK(c, gin.H{"status": "ok"})
}
