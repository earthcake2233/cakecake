package handler

import (
	"context"
	"minibili/internal/model/admin"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"minibili/internal/errcode"
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

	ops, err := a.HotSearchSvc.ListOps(c.Request.Context())
	if err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	opItems := make([]gin.H, 0, len(ops))
	for i := range ops {
		opItems = append(opItems, hotSearchOpToJSON(&ops[i]))
	}

	flags := a.HotSearchSvc.ActiveOpFlags(c.Request.Context())

	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	merged := make([]gin.H, 0)
	if a.SearchHot != nil {
		items, err := a.HotSearchSvc.ListMergedDetail(ctx, mergedLimit)
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
		rows, err := a.HotSearchSvc.TopWithScores(ctx, redisLimit)
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
		"merged":       merged,
		"redis":        redisRows,
		"ops":          opItems,
		"custom_order": a.HotSearchSvc.HasDisplayLayout(c.Request.Context()),
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
	if err := a.HotSearchSvc.RemoveKeywordFromRedis(ctx, req.Keyword); err != nil {
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
	if err := a.HotSearchSvc.BoostKeyword(ctx, req.Keyword, delta); err != nil {
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

// AdminQuickHotSearchOp POST /api/v1/admin/hot-search/quick-op
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
	op, _, err := a.HotSearchSvc.QuickOpCreateOrUpdate(c.Request.Context(), ot, kw, req.DisplayTitle, req.Badge, req.PinRank)
	if err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	resp.JSON(c, http.StatusCreated, errcode.CodeSuccess, hotSearchOpToJSON(op))
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

// AdminReorderHotSearch POST /api/v1/admin/hot-search/reorder
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
	items := make([]service.ReorderItem, 0, len(req.Items))
	for _, it := range req.Items {
		items = append(items, service.ReorderItem{
			Keyword: it.Keyword,
			Title:   it.Title,
			OpID:    it.OpID,
			Source:  it.Source,
		})
	}
	if err := a.HotSearchSvc.ReorderItems(c.Request.Context(), items); err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	resp.OK(c, gin.H{"ok": true, "custom_order": true})
}

// AdminResetHotSearchDisplayOrder POST /api/v1/admin/hot-search/display-order/reset
func (a *API) AdminResetHotSearchDisplayOrder(c *gin.Context) {
	if err := a.HotSearchSvc.ClearDisplayLayout(c.Request.Context()); err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	resp.OK(c, gin.H{"ok": true, "custom_order": false})
}
func hotSearchDisplayTitle(op *admin.HotSearchOp) string {
	if op == nil {
		return ""
	}
	if t := strings.TrimSpace(op.DisplayTitle); t != "" {
		return t
	}
	return strings.TrimSpace(op.Keyword)
}
