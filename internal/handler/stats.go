package handler

import (
	"github.com/gin-gonic/gin"

	"minibili/internal/pkg/resp"
)

// HomeStats returns homepage sidebar metrics (online viewers + published video count).
// HomeStats godoc
// @Summary      Home page stats
// @Description  Get home page statistics
// @Tags        Home
// @Produce     json
// @Success     200 {object} map[string]interface{}
// @Router      /stats/home [get]
func (a *API) HomeStats(c *gin.Context) {
	var published int64
	if a.VideoSvc != nil {
		published = a.VideoSvc.CountPublishedVideos(c.Request.Context())
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
