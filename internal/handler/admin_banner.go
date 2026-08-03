package handler

import (
	"cakecake/internal/model/admin"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"cakecake/internal/errcode"
	"cakecake/internal/pkg/resp"
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

type bannerItem struct {
	ID         uint64     `json:"id"`
	Title      string     `json:"title"`
	ImageURL   string     `json:"image_url"`
	LinkType   string     `json:"link_type"`
	LinkTarget string     `json:"link_target"`
	SortOrder  int        `json:"sort_order"`
	Enabled    bool       `json:"enabled"`
	StartAt    *time.Time `json:"start_at"`
	EndAt      *time.Time `json:"end_at"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

type bannerListResponse struct {
	Items []bannerItem `json:"items"`
}

func bannerToJSON(b *admin.HomeBanner) bannerItem {
	return bannerItem{
		ID:         b.ID,
		Title:      b.Title,
		ImageURL:   b.ImageURL,
		LinkType:   b.LinkType,
		LinkTarget: b.LinkTarget,
		SortOrder:  b.SortOrder,
		Enabled:    b.Enabled,
		StartAt:    b.StartAt,
		EndAt:      b.EndAt,
		CreatedAt:  b.CreatedAt,
		UpdatedAt:  b.UpdatedAt,
	}
}

// AdminListBanners GET /api/v1/admin/home-banners
func (a *API) AdminListBanners(c *gin.Context) {
	rows, err := a.VideoSvc.ListBanners(c.Request.Context())
	if err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	out := make([]bannerItem, 0, len(rows))
	for i := range rows {
		out = append(out, bannerToJSON(&rows[i]))
	}
	resp.OK(c, bannerListResponse{Items: out})
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
	var b admin.HomeBanner
	b = admin.HomeBanner{
		Title:      title,
		ImageURL:   img,
		LinkType:   lt,
		LinkTarget: strings.TrimSpace(req.LinkTarget),
		SortOrder:  req.SortOrder,
		Enabled:    en,
		StartAt:    parseOptionalUnix(req.StartAt),
		EndAt:      parseOptionalUnix(req.EndAt),
	}
	if err := a.VideoSvc.CreateBanner(c.Request.Context(), &b); err != nil {
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
	b, err := a.VideoSvc.GetBanner(c.Request.Context(), id)
	if err != nil {
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
	if err := a.VideoSvc.UpdateBanner(c.Request.Context(), id, updates); err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	b, _ = a.VideoSvc.GetBanner(c.Request.Context(), id)
	if u := strings.TrimSpace(req.ImageURL); u != "" && u != oldURL {
		a.StorageSvc.PurgeBannerImageURL(oldURL)
	}
	resp.OK(c, bannerToJSON(b))
}

// AdminDeleteBanner DELETE /api/v1/admin/home-banners/:id
func (a *API) AdminDeleteBanner(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	b, err := a.VideoSvc.GetBanner(c.Request.Context(), id)
	if err != nil {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	if err := a.VideoSvc.DeleteBanner(c.Request.Context(), id); err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	a.StorageSvc.PurgeBanner(*b)
	resp.OK(c, deletedResponse{Deleted: true})
}
