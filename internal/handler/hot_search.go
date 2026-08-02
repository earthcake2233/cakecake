package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"cakecake/internal/errcode"
	"cakecake/internal/pkg/resp"
)

// HotSearchList returns hot search keywords aggregated in Redis.
// GET /api/v1/hot-search?limit=10
type hotSearchItem struct {
	Rank  int    `json:"rank"`
	Title string `json:"title"`
	Badge string `json:"badge"`
}

type hotSearchListResponse struct {
	Items []hotSearchItem `json:"items"`
}

// HotSearchList godoc
// HotSearchList godoc
// @Summary      List hot searches
// @Description  Get current hot search list
// @Tags        Search
// @Produce     json
// @Success     200 {object} map[string]interface{}
// @Router      /hot-search [get]
func (a *API) HotSearchList(c *gin.Context) {
	limit := parseLimit(c, 10, 20)
	if a.SearchHot == nil {
		resp.OK(c, hotSearchListResponse{Items: []hotSearchItem{}})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()
	items, err := a.HotSearchSvc.ListMerged(ctx, limit)
	if err != nil {
		a.Log.Error("hot search list", zap.Error(err))
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	out := make([]hotSearchItem, 0, len(items))
	for _, it := range items {
		out = append(out, hotSearchItem{
			Rank:  it.Rank,
			Title: it.Title,
			Badge: it.Badge,
		})
	}
	resp.OK(c, hotSearchListResponse{Items: out})
}
