package handler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"minibili/internal/errcode"
	"minibili/internal/model"
	"minibili/internal/pkg/resp"
	"minibili/internal/service"
)

type hotSearchOpReq struct {
	OpType       string `json:"op_type"`
	Keyword      string `json:"keyword"`
	DisplayTitle string `json:"display_title"`
	Badge        string `json:"badge"`
	PinRank      int    `json:"pin_rank"`
	Enabled      *bool  `json:"enabled"`
	StartAt      *int64 `json:"start_at"`
	EndAt        *int64 `json:"end_at"`
}

func hotSearchOpToJSON(op *model.HotSearchOp) gin.H {
	return gin.H{
		"id":            op.ID,
		"op_type":       op.OpType,
		"keyword":       op.Keyword,
		"display_title": op.DisplayTitle,
		"badge":         op.Badge,
		"pin_rank":      op.PinRank,
		"enabled":       op.Enabled,
		"start_at":      op.StartAt,
		"end_at":        op.EndAt,
		"created_at":    op.CreatedAt,
		"updated_at":    op.UpdatedAt,
	}
}

// AdminListHotSearchOps GET /api/v1/admin/hot-search/ops
func (a *API) AdminListHotSearchOps(c *gin.Context) {
	var rows []model.HotSearchOp
	if err := a.DB.Order("pin_rank ASC, id ASC").Find(&rows).Error; err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	out := make([]gin.H, 0, len(rows))
	for i := range rows {
		out = append(out, hotSearchOpToJSON(&rows[i]))
	}
	resp.OK(c, gin.H{"items": out})
}

// AdminCreateHotSearchOp POST /api/v1/admin/hot-search/ops
func (a *API) AdminCreateHotSearchOp(c *gin.Context) {
	var req hotSearchOpReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	kw := strings.TrimSpace(req.Keyword)
	ot := strings.TrimSpace(req.OpType)
	if kw == "" || (ot != "pin" && ot != "block" && ot != "manual") {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	en := true
	if req.Enabled != nil {
		en = *req.Enabled
	}
	op := model.HotSearchOp{
		OpType:       ot,
		Keyword:      kw,
		DisplayTitle: strings.TrimSpace(req.DisplayTitle),
		Badge:        strings.TrimSpace(req.Badge),
		PinRank:      req.PinRank,
		Enabled:      en,
		StartAt:      parseOptionalUnix(req.StartAt),
		EndAt:        parseOptionalUnix(req.EndAt),
	}
	if err := a.DB.Create(&op).Error; err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	resp.JSON(c, http.StatusCreated, errcode.CodeSuccess, hotSearchOpToJSON(&op))
}

// AdminUpdateHotSearchOp PUT /api/v1/admin/hot-search/ops/:id
func (a *API) AdminUpdateHotSearchOp(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	var op model.HotSearchOp
	if err := a.DB.First(&op, id).Error; err != nil {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	var req hotSearchOpReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	updates := map[string]any{}
	if t := strings.TrimSpace(req.OpType); t != "" {
		updates["op_type"] = t
	}
	if k := strings.TrimSpace(req.Keyword); k != "" {
		updates["keyword"] = k
	}
	updates["display_title"] = strings.TrimSpace(req.DisplayTitle)
	updates["badge"] = strings.TrimSpace(req.Badge)
	updates["pin_rank"] = req.PinRank
	if req.Enabled != nil {
		updates["enabled"] = *req.Enabled
	}
	if req.StartAt != nil {
		updates["start_at"] = parseOptionalUnix(req.StartAt)
	}
	if req.EndAt != nil {
		updates["end_at"] = parseOptionalUnix(req.EndAt)
	}
	if err := a.DB.Model(&op).Updates(updates).Error; err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	_ = a.DB.First(&op, id)
	resp.OK(c, hotSearchOpToJSON(&op))
}

// AdminDeleteHotSearchOp DELETE /api/v1/admin/hot-search/ops/:id
func (a *API) AdminDeleteHotSearchOp(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	if err := a.DB.Delete(&model.HotSearchOp{}, id).Error; err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	resp.OK(c, gin.H{"deleted": true})
}

// AdminPreviewHotSearch GET /api/v1/admin/hot-search/preview?limit=10
func (a *API) AdminPreviewHotSearch(c *gin.Context) {
	limit := 10
	if s := c.Query("limit"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 && n <= 20 {
			limit = n
		}
	}
	items, err := service.ListHotSearchMerged(c.Request.Context(), a.DB, a.SearchHot, limit)
	if err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	out := make([]gin.H, 0, len(items))
	for _, it := range items {
		out = append(out, gin.H{
			"rank":  it.Rank,
			"title": it.Title,
			"badge": it.Badge,
		})
	}
	resp.OK(c, gin.H{"items": out})
}
