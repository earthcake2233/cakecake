package handler

import (
	"github.com/gin-gonic/gin"

	"cakecake/internal/pkg/resp"
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
	type statsResponse struct {
		WebOnline int   `json:"web_online"`
		AllCount  int64 `json:"all_count"`
	}
	var published int64
	if a.VideoSvc != nil {
		published = a.VideoSvc.CountPublishedVideos(c.Request.Context())
	}
	webOnline := 0
	if a.Hub != nil {
		webOnline = a.Hub.TotalConnections()
	}
	resp.OK(c, statsResponse{WebOnline: webOnline, AllCount: published})
}
