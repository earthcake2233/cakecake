package handler

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"minibili/internal/errcode"
	"minibili/internal/middleware"
	"minibili/internal/pkg/resp"
	"minibili/internal/search"
	"minibili/internal/service"
)

// SearchSuggest GET /api/v1/search/suggest?term=xxx&limit=10
func (a *API) SearchSuggest(c *gin.Context) {
	term := strings.TrimSpace(c.Query("term"))
	if term == "" {
		term = strings.TrimSpace(c.Query("q"))
	}
	if term != "" {
		if err := search.ValidateKeyword(term); err != nil {
			resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
			return
		}
		if !service.ValidateSuggestTerm(term) {
			resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
			return
		}
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	var uid uint64
	if id, ok := middleware.UserID(c); ok {
		uid = id
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 800*time.Millisecond)
	defer cancel()
	tags := service.SearchSuggest(ctx, a.DB, a.SearchHot, uid, term, limit)
	resp.OK(c, gin.H{"tag": tags})
}
