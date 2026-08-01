package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"cakecake/internal/errcode"
	"cakecake/internal/pkg/resp"
)

func bannerSlideURL(linkType, linkTarget string) string {
	switch strings.ToLower(strings.TrimSpace(linkType)) {
	case "video":
		id, _ := strconv.ParseUint(strings.TrimSpace(linkTarget), 10, 64)
		if id > 0 {
			return fmt.Sprintf("/#/video/BV%d", id)
		}
	case "url":
		u := strings.TrimSpace(linkTarget)
		if u != "" {
			return u
		}
	}
	return "/"
}

// ListHomeBanners GET /api/v1/home-banners — public carousel for homepage.
// ListHomeBanners godoc
// @Summary      List home banners
// @Description  Get active home page banners
// @Tags        Home
// @Produce     json
// @Success     200 {object} map[string]interface{}
// @Router      /home-banners [get]
func (a *API) ListHomeBanners(c *gin.Context) {
	rows, err := a.VideoSvc.ListActiveBanners(c.Request.Context())
	if err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	items := make([]gin.H, 0, len(rows))
	for _, b := range rows {
		items = append(items, gin.H{
			"id":   b.ID,
			"name": b.Title,
			"pic":  b.ImageURL,
			"url":  bannerSlideURL(b.LinkType, b.LinkTarget),
		})
	}
	resp.OK(c, gin.H{"items": items})
}
