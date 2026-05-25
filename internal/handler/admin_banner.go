package handler

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"minibili/internal/errcode"
	"minibili/internal/model"
	"minibili/internal/pkg/resp"
)

type bannerReq struct {
	Title      string `json:"title"`
	ImageURL   string `json:"image_url"`
	LinkType   string `json:"link_type"`
	LinkTarget string `json:"link_target"`
	SortOrder  int    `json:"sort_order"`
	Enabled    *bool  `json:"enabled"`
	StartAt    *int64 `json:"start_at"` // unix sec, optional
	EndAt      *int64 `json:"end_at"`
}

func parseOptionalUnix(p *int64) *time.Time {
	if p == nil || *p <= 0 {
		return nil
	}
	t := time.Unix(*p, 0)
	return &t
}

func bannerToJSON(b *model.HomeBanner) gin.H {
	return gin.H{
		"id":          b.ID,
		"title":       b.Title,
		"image_url":   b.ImageURL,
		"link_type":   b.LinkType,
		"link_target": b.LinkTarget,
		"sort_order":  b.SortOrder,
		"enabled":     b.Enabled,
		"start_at":    b.StartAt,
		"end_at":      b.EndAt,
		"created_at":  b.CreatedAt,
		"updated_at":  b.UpdatedAt,
	}
}

// AdminListBanners GET /api/v1/admin/home-banners
func (a *API) AdminListBanners(c *gin.Context) {
	var rows []model.HomeBanner
	if err := a.DB.Order("sort_order ASC, id ASC").Find(&rows).Error; err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	out := make([]gin.H, 0, len(rows))
	for i := range rows {
		out = append(out, bannerToJSON(&rows[i]))
	}
	resp.OK(c, gin.H{"items": out})
}

// AdminCreateBanner POST /api/v1/admin/home-banners
func (a *API) AdminCreateBanner(c *gin.Context) {
	var req bannerReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	title := strings.TrimSpace(req.Title)
	img := strings.TrimSpace(req.ImageURL)
	if title == "" || img == "" {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	lt := strings.TrimSpace(req.LinkType)
	if lt == "" {
		lt = "none"
	}
	en := true
	if req.Enabled != nil {
		en = *req.Enabled
	}
	b := model.HomeBanner{
		Title:      title,
		ImageURL:   img,
		LinkType:   lt,
		LinkTarget: strings.TrimSpace(req.LinkTarget),
		SortOrder:  req.SortOrder,
		Enabled:    en,
		StartAt:    parseOptionalUnix(req.StartAt),
		EndAt:      parseOptionalUnix(req.EndAt),
	}
	if err := a.DB.Create(&b).Error; err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	resp.JSON(c, http.StatusCreated, errcode.CodeSuccess, bannerToJSON(&b))
}

// AdminUpdateBanner PUT /api/v1/admin/home-banners/:id
func (a *API) AdminUpdateBanner(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	var b model.HomeBanner
	if err := a.DB.First(&b, id).Error; err != nil {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	oldURL := b.ImageURL
	var req bannerReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	updates := map[string]any{}
	if t := strings.TrimSpace(req.Title); t != "" {
		updates["title"] = t
	}
	if u := strings.TrimSpace(req.ImageURL); u != "" {
		updates["image_url"] = u
	}
	if lt := strings.TrimSpace(req.LinkType); lt != "" {
		updates["link_type"] = lt
	}
	updates["link_target"] = strings.TrimSpace(req.LinkTarget)
	updates["sort_order"] = req.SortOrder
	if req.Enabled != nil {
		updates["enabled"] = *req.Enabled
	}
	if req.StartAt != nil {
		updates["start_at"] = parseOptionalUnix(req.StartAt)
	}
	if req.EndAt != nil {
		updates["end_at"] = parseOptionalUnix(req.EndAt)
	}
	if err := a.DB.Model(&b).Updates(updates).Error; err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	_ = a.DB.First(&b, id)
	if u := strings.TrimSpace(req.ImageURL); u != "" && u != oldURL {
		purgeBannerImageURL(a.Cfg, a.OSS, a.Log, oldURL)
	}
	resp.OK(c, bannerToJSON(&b))
}

// AdminDeleteBanner DELETE /api/v1/admin/home-banners/:id
func (a *API) AdminDeleteBanner(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	var b model.HomeBanner
	if err := a.DB.First(&b, id).Error; err != nil {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	if err := a.DB.Delete(&model.HomeBanner{}, id).Error; err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	purgeBannerOSSObjects(a.Cfg, a.OSS, a.Log, b)
	resp.OK(c, gin.H{"deleted": true})
}
