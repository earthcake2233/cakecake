package handler

import (
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"cakecake/internal/errcode"
	"cakecake/internal/middleware"
	"cakecake/internal/pkg/resp"
	"cakecake/internal/pkg/searchhist"
	"cakecake/internal/search"
)

const maxUserSearchHistory = 20

func normalizeSearchHistoryKeywords(raw []string) []string {
	seen := make(map[string]struct{})
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		k := strings.TrimSpace(item)
		if k == "" {
			continue
		}
		if utf8.RuneCountInString(k) > 50 {
			continue
		}
		if err := search.ValidateKeyword(k); err != nil {
			continue
		}
		norm := searchhist.Norm(k)
		if norm == "" {
			continue
		}
		if _, ok := seen[norm]; ok {
			continue
		}
		seen[norm] = struct{}{}
		out = append(out, k)
		if len(out) >= maxUserSearchHistory {
			break
		}
	}
	return out
}

// GetMySearchHistory returns the caller's recent search keywords (newest first).
// GET /api/v1/users/me/search-history
// GetMySearchHistory godoc
// @Summary      Get search history
// @Description  Get recent search history for current user
// @Tags         Search
// @Produce      json
// @Success      200 {object} map[string]interface{}
// @Router       /users/me/search-history [get]
func (a *API) GetMySearchHistory(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	keywords, err := a.SearchHistorySvc.ListKeywords(c.Request.Context(), uid)
	if err != nil {
		a.Log.Error("list search history", zap.Error(err))
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	resp.OK(c, gin.H{"keywords": keywords})
}

type searchHistoryPutReq struct {
	Keywords []string `json:"keywords"`
}

// PutMySearchHistory replaces the caller's search history with the given keyword list.
// PUT /api/v1/users/me/search-history
// PutMySearchHistory godoc
// @Summary      Replace search history
// @Description  Replace the entire search history for current user
// @Tags         Search
// @Produce      json
// @Param        body body array true "Search history items"
// @Success      200 {object} map[string]interface{}
// @Router       /users/me/search-history [put]
func (a *API) PutMySearchHistory(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	var req searchHistoryPutReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	keywords := normalizeSearchHistoryKeywords(req.Keywords)
	if err := a.SearchHistorySvc.ReplaceHistory(c.Request.Context(), uid, keywords); err != nil {
		a.Log.Error("put search history", zap.Error(err))
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	resp.OK(c, gin.H{"keywords": keywords})
}

type searchHistoryPostReq struct {
	Keyword string `json:"keyword"`
}

// PostMySearchHistory records one search keyword (moves it to the top).
// POST /api/v1/users/me/search-history
// PostMySearchHistory godoc
// @Summary      Add search history entry
// @Description  Append a keyword to search history
// @Tags         Search
// @Produce      json
// @Param        body body object{keyword=string} true "Search keyword"
// @Success      200 {object} map[string]interface{}
// @Router       /users/me/search-history [post]
func (a *API) PostMySearchHistory(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	var req searchHistoryPostReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	kw := strings.TrimSpace(req.Keyword)
	if err := search.ValidateKeyword(kw); err != nil {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	if err := a.SearchHistorySvc.UpsertKeyword(c.Request.Context(), uid, kw, time.Now()); err != nil {
		a.Log.Error("post search history", zap.Error(err))
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	_ = a.SearchHistorySvc.TrimHistory(c.Request.Context(), uid)
	keywords, err := a.SearchHistorySvc.ListKeywords(c.Request.Context(), uid)
	if err != nil {
		a.Log.Error("list search history after post", zap.Error(err))
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	resp.OK(c, gin.H{"keywords": keywords})
}
