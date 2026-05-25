package handler

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"minibili/internal/errcode"
	"minibili/internal/model"
	"minibili/internal/pkg/resp"
	"minibili/internal/service"
)

func adminHotSearchLimit(c *gin.Context, def, max int) int {
	limit := def
	if s := c.Query("limit"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 {
			limit = n
		}
	}
	if limit > max {
		limit = max
	}
	if limit <= 0 {
		limit = def
	}
	return limit
}

// AdminHotSearchDashboard GET /api/v1/admin/hot-search/dashboard
func (a *API) AdminHotSearchDashboard(c *gin.Context) {
	mergedLimit := adminHotSearchLimit(c, 10, 20)
	redisLimit := 30
	if q := c.Query("redis_limit"); q != "" {
		if n, err := strconv.Atoi(q); err == nil && n > 0 && n <= 50 {
			redisLimit = n
		}
	}

	var ops []model.HotSearchOp
	if err := a.DB.Order("pin_rank ASC, id ASC").Find(&ops).Error; err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	opItems := make([]gin.H, 0, len(ops))
	for i := range ops {
		opItems = append(opItems, hotSearchOpToJSON(&ops[i]))
	}

	flags := service.ActiveHotSearchOpFlags(a.DB)

	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	merged := make([]gin.H, 0)
	if a.SearchHot != nil {
		items, err := service.ListHotSearchMergedDetail(ctx, a.DB, a.SearchHot, mergedLimit)
		if err != nil {
			a.Log.Error("hot search dashboard merged", zap.Error(err))
			resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
			return
		}
		for _, it := range items {
			merged = append(merged, gin.H{
				"rank":    it.Rank,
				"title":   it.Title,
				"badge":   it.Badge,
				"source":  it.Source,
				"keyword": it.Keyword,
				"op_id":   it.OpID,
			})
		}
	}

	redisRows := make([]gin.H, 0)
	if a.SearchHot != nil {
		rows, err := a.SearchHot.TopWithScores(ctx, redisLimit)
		if err != nil {
			a.Log.Error("hot search dashboard redis", zap.Error(err))
			resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
			return
		}
		for _, row := range rows {
			f := flags[row.Keyword]
			redisRows = append(redisRows, gin.H{
				"rank":    row.Rank,
				"title":   row.Title,
				"keyword": row.Keyword,
				"score":   row.Score,
				"badge":   row.Badge,
				"blocked": f.Blocked,
				"pinned":  f.Pin,
				"manual":  f.Manual,
				"op_id":   f.OpID,
				"op_type": f.OpType,
			})
		}
	}

	resp.OK(c, gin.H{
		"merged":        merged,
		"redis":         redisRows,
		"ops":           opItems,
		"custom_order":  service.HasHotSearchDisplayLayout(a.DB),
	})
}

type hotSearchKeywordReq struct {
	Keyword string  `json:"keyword"`
	Delta   float64 `json:"delta"`
}

// AdminRemoveHotSearchRedis POST /api/v1/admin/hot-search/redis/remove
func (a *API) AdminRemoveHotSearchRedis(c *gin.Context) {
	if a.SearchHot == nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	var req hotSearchKeywordReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	if strings.TrimSpace(req.Keyword) == "" {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()
	if err := a.SearchHot.RemoveKeyword(ctx, req.Keyword); err != nil {
		a.Log.Error("hot search redis remove", zap.Error(err))
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	resp.OK(c, gin.H{"ok": true})
}

// AdminBoostHotSearchRedis POST /api/v1/admin/hot-search/redis/boost
func (a *API) AdminBoostHotSearchRedis(c *gin.Context) {
	if a.SearchHot == nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	var req hotSearchKeywordReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	if strings.TrimSpace(req.Keyword) == "" {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	delta := req.Delta
	if delta <= 0 {
		delta = 5
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()
	if err := a.SearchHot.BoostKeyword(ctx, req.Keyword, delta); err != nil {
		a.Log.Error("hot search redis boost", zap.Error(err))
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	resp.OK(c, gin.H{"ok": true, "delta": delta})
}

type hotSearchQuickOpReq struct {
	Keyword      string `json:"keyword"`
	OpType       string `json:"op_type"`
	DisplayTitle string `json:"display_title"`
	Badge        string `json:"badge"`
	PinRank      int    `json:"pin_rank"`
}

// AdminQuickHotSearchOp POST /api/v1/admin/hot-search/quick-op — pin/block/manual from Redis row.
func (a *API) AdminQuickHotSearchOp(c *gin.Context) {
	var req hotSearchQuickOpReq
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
	norm := service.NormalizeSearchKeyword(kw)
	var existing model.HotSearchOp
	found := false
	if norm != "" {
		var rows []model.HotSearchOp
		_ = a.DB.Find(&rows).Error
		for i := range rows {
			if service.NormalizeSearchKeyword(rows[i].Keyword) == norm {
				existing = rows[i]
				found = true
				break
			}
		}
	}
	display := strings.TrimSpace(req.DisplayTitle)
	if display == "" {
		display = kw
	}
	pinRank := req.PinRank
	if ot == "block" {
		pinRank = 0
		req.Badge = ""
	} else if pinRank <= 0 {
		pinRank = 1
	}
	if found {
		updates := map[string]any{
			"op_type":       ot,
			"keyword":       kw,
			"display_title": display,
			"badge":         strings.TrimSpace(req.Badge),
			"pin_rank":      pinRank,
			"enabled":       true,
		}
		if err := a.DB.Model(&existing).Updates(updates).Error; err != nil {
			resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
			return
		}
		_ = a.DB.First(&existing, existing.ID)
		a.syncHotSearchLayoutAfterOp(c, kw, display, ot, pinRank)
		resp.OK(c, hotSearchOpToJSON(&existing))
		return
	}
	op := model.HotSearchOp{
		OpType:       ot,
		Keyword:      kw,
		DisplayTitle: display,
		Badge:        strings.TrimSpace(req.Badge),
		PinRank:      pinRank,
		Enabled:      true,
	}
	if err := a.DB.Create(&op).Error; err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	a.syncHotSearchLayoutAfterOp(c, kw, display, ot, pinRank)
	resp.JSON(c, http.StatusCreated, errcode.CodeSuccess, hotSearchOpToJSON(&op))
}

func (a *API) syncHotSearchLayoutAfterOp(c *gin.Context, keyword, title, opType string, pinRank int) {
	if a.DB == nil {
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()
	switch opType {
	case "block":
		_ = service.RemoveHotSearchLayoutEntry(a.DB, keyword)
	case "pin", "manual":
		if service.HasHotSearchDisplayLayout(a.DB) {
			_ = service.ApplyHotSearchLayoutMove(a.DB, keyword, title, pinRank)
			return
		}
		_ = service.EnsureHotSearchLayoutFromMerged(ctx, a.DB, a.SearchHot, 10)
		_ = service.ApplyHotSearchLayoutMove(a.DB, keyword, title, pinRank)
	}
}

type hotSearchReorderItem struct {
	Keyword string `json:"keyword"`
	Title   string `json:"title"`
	OpID    uint64 `json:"op_id"`
	Source  string `json:"source"`
}

type hotSearchReorderReq struct {
	Items []hotSearchReorderItem `json:"items"`
}

// AdminReorderHotSearch POST /api/v1/admin/hot-search/reorder — save drag order without pinning auto items.
func (a *API) AdminReorderHotSearch(c *gin.Context) {
	var req hotSearchReorderReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	if len(req.Items) == 0 || len(req.Items) > 20 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	layout := make([]service.HotSearchLayoutEntry, 0, len(req.Items))
	var allOps []model.HotSearchOp
	_ = a.DB.Find(&allOps).Error
	opByNorm := make(map[string]*model.HotSearchOp, len(allOps))
	opByID := make(map[uint64]*model.HotSearchOp, len(allOps))
	for i := range allOps {
		op := &allOps[i]
		opByID[op.ID] = op
		if norm := service.NormalizeSearchKeyword(op.Keyword); norm != "" {
			opByNorm[norm] = op
		}
	}
	for i, it := range req.Items {
		kw := strings.TrimSpace(it.Keyword)
		title := strings.TrimSpace(it.Title)
		if kw == "" {
			kw = title
		}
		if kw == "" {
			resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
			return
		}
		layout = append(layout, service.HotSearchLayoutEntry{Keyword: kw, Title: title})
		rank := i + 1
		norm := service.NormalizeSearchKeyword(kw)
		var existing *model.HotSearchOp
		if it.OpID > 0 {
			existing = opByID[it.OpID]
		}
		if existing == nil && norm != "" {
			existing = opByNorm[norm]
		}
		if existing != nil && (existing.OpType == "pin" || existing.OpType == "manual") {
			if err := a.DB.Model(existing).Update("pin_rank", rank).Error; err != nil {
				resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
				return
			}
		}
	}
	if err := service.SaveHotSearchDisplayLayout(a.DB, layout); err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	resp.OK(c, gin.H{"ok": true, "custom_order": true})
}

// AdminResetHotSearchDisplayOrder POST /api/v1/admin/hot-search/display-order/reset
func (a *API) AdminResetHotSearchDisplayOrder(c *gin.Context) {
	if err := service.ClearHotSearchDisplayLayout(a.DB); err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	resp.OK(c, gin.H{"ok": true, "custom_order": false})
}

func hotSearchDisplayTitle(op *model.HotSearchOp) string {
	if op == nil {
		return ""
	}
	if t := strings.TrimSpace(op.DisplayTitle); t != "" {
		return t
	}
	return strings.TrimSpace(op.Keyword)
}
