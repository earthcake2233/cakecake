package handler

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"minibili/internal/errcode"
	"minibili/internal/pkg/resp"
	"minibili/internal/service"
)

// HotSearchList returns hot search keywords aggregated in Redis.
// GET /api/v1/hot-search?limit=10
func (a *API) HotSearchList(c *gin.Context) {
	limit := 10
	if s := c.Query("limit"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 && n <= 20 {
			limit = n
		}
	}
	if a.SearchHot == nil {
		resp.OK(c, gin.H{"items": []any{}})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()
	items, err := service.ListHotSearchMerged(ctx, a.DB, a.SearchHot, limit)
	if err != nil {
		a.Log.Error("hot search list", zap.Error(err))
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
