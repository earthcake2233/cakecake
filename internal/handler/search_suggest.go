package handler

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"cakecake/internal/errcode"
	"cakecake/internal/middleware"
	"cakecake/internal/pkg/resp"
	"cakecake/internal/service"
	"cakecake/internal/service/hotsearch"
)

// SearchSuggest GET /api/v1/search/suggest?term=xxx&limit=10
// SearchSuggest godoc
// @Summary      Search suggestions
// @Description  Get auto-complete suggestions for search queries
// @Tags         Search
// @Produce      json
// @Param        q query string true "Search prefix"
// @Success      200 {object} map[string]interface{}
// @Router       /search/suggest [get]
func (a *API) SearchSuggest(c *gin.Context) {
	type searchSuggestResponse struct {
		Tag []hotsearch.SearchSuggestTag `json:"tag"`
	}
	term := strings.TrimSpace(c.Query("term"))
	if term == "" {
		term = strings.TrimSpace(c.Query("q"))
	}
	if term != "" {
		if err := service.ValidateKeyword(term); err != nil {
			resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
			return
		}
		if !a.HotSearchSvc.ValidateSuggestTerm(term) {
			resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
			return
		}
	}
	limit := parseLimit(c, 10, 0)
	var uid uint64
	if id, ok := middleware.UserID(c); ok {
		uid = id
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 800*time.Millisecond)
	defer cancel()
	tags := a.HotSearchSvc.SearchSuggest(ctx, uid, term, limit)
	resp.OK(c, searchSuggestResponse{Tag: tags})
}
