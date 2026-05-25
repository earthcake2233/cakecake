package handler

import (
	"github.com/gin-gonic/gin"

	"minibili/internal/model"
	"minibili/internal/pkg/resp"
)

// HomeStats returns homepage sidebar metrics (online viewers + published video count).
func (a *API) HomeStats(c *gin.Context) {
	var published int64
	if a.DB != nil {
		_ = a.DB.Model(&model.Video{}).Where("status = ?", "published").Count(&published).Error
	}
	webOnline := 0
	if a.Hub != nil {
		webOnline = a.Hub.TotalConnections()
	}
	resp.OK(c, gin.H{
		"web_online": webOnline,
		"all_count":  published,
	})
}
