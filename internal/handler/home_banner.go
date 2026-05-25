package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"minibili/internal/errcode"
	"minibili/internal/model"
	"minibili/internal/pkg/resp"
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
func (a *API) ListHomeBanners(c *gin.Context) {
	now := time.Now()
	var rows []model.HomeBanner
	q := a.DB.Where("enabled = ?", true).
		Where("(start_at IS NULL OR start_at <= ?)", now).
		Where("(end_at IS NULL OR end_at >= ?)", now).
		Order("sort_order ASC, id ASC")
	if err := q.Find(&rows).Error; err != nil {
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
