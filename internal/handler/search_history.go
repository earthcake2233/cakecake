package handler

import (
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"minibili/internal/errcode"
	"minibili/internal/middleware"
	"minibili/internal/model"
	"minibili/internal/pkg/resp"
	"minibili/internal/pkg/searchhist"
	"minibili/internal/search"
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

func (a *API) listSearchHistoryKeywords(uid uint64) ([]string, error) {
	var rows []model.UserSearchHistory
	if err := a.DB.Where("user_id = ?", uid).
		Order("updated_at DESC, id DESC").
		Limit(maxUserSearchHistory * 2).
		Find(&rows).Error; err != nil {
		return nil, err
	}
	seen := make(map[string]struct{})
	out := make([]string, 0, maxUserSearchHistory)
	for _, r := range rows {
		norm := searchhist.Norm(r.Keyword)
		if norm == "" {
			continue
		}
		if _, ok := seen[norm]; ok {
			continue
		}
		seen[norm] = struct{}{}
		out = append(out, r.Keyword)
		if len(out) >= maxUserSearchHistory {
			break
		}
	}
	return out, nil
}

func (a *API) upsertSearchHistoryKeyword(uid uint64, kw string, at time.Time) error {
	norm := searchhist.Norm(kw)
	if norm == "" {
		return nil
	}
	var rows []model.UserSearchHistory
	if err := a.DB.Where("user_id = ? AND keyword_norm = ?", uid, norm).Find(&rows).Error; err != nil {
		return err
	}
	if len(rows) > 0 {
		keep := rows[0]
		for i := 1; i < len(rows); i++ {
			_ = a.DB.Delete(&rows[i]).Error
		}
		return a.DB.Model(&keep).Updates(map[string]interface{}{
			"keyword":    kw,
			"updated_at": at,
		}).Error
	}
	return a.DB.Create(&model.UserSearchHistory{
		UserID:      uid,
		Keyword:     kw,
		KeywordNorm: norm,
		UpdatedAt:   at,
	}).Error
}

func (a *API) trimSearchHistory(uid uint64) error {
	var rows []model.UserSearchHistory
	if err := a.DB.Where("user_id = ?", uid).Order("updated_at DESC, id DESC").Find(&rows).Error; err != nil {
		return err
	}
	if len(rows) <= maxUserSearchHistory {
		return nil
	}
	ids := make([]uint64, 0, len(rows)-maxUserSearchHistory)
	for i := maxUserSearchHistory; i < len(rows); i++ {
		ids = append(ids, rows[i].ID)
	}
	return a.DB.Where("id IN ?", ids).Delete(&model.UserSearchHistory{}).Error
}

// GetMySearchHistory returns the caller's recent search keywords (newest first).
// GET /api/v1/users/me/search-history
func (a *API) GetMySearchHistory(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	keywords, err := a.listSearchHistoryKeywords(uid)
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
	now := time.Now()
	if err := a.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("user_id = ?", uid).Delete(&model.UserSearchHistory{}).Error; err != nil {
			return err
		}
		for i, kw := range keywords {
			norm := searchhist.Norm(kw)
			if norm == "" {
				continue
			}
			row := model.UserSearchHistory{
				UserID:      uid,
				Keyword:     kw,
				KeywordNorm: norm,
				UpdatedAt:   now.Add(-time.Duration(i) * time.Millisecond),
			}
			if err := tx.Create(&row).Error; err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
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
	now := time.Now()
	if err := a.upsertSearchHistoryKeyword(uid, kw, now); err != nil {
		a.Log.Error("post search history", zap.Error(err))
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	_ = a.trimSearchHistory(uid)
	keywords, err := a.listSearchHistoryKeywords(uid)
	if err != nil {
		a.Log.Error("list search history after post", zap.Error(err))
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	resp.OK(c, gin.H{"keywords": keywords})
}
