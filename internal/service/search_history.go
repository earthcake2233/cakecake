package service

import (
	"cakecake/internal/model/extra"
	"context"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"cakecake/internal/pkg/searchhist"
)

const maxUserSearchHistory = 20

// SearchHistoryService handles user search history operations.
type SearchHistoryService struct {
	db  *gorm.DB
	log *zap.Logger
}

// NewSearchHistoryService creates a SearchHistoryService.
func NewSearchHistoryService(db *gorm.DB, log *zap.Logger) *SearchHistoryService {
	return &SearchHistoryService{db: db, log: log}
}

// ListKeywords returns the caller's recent search keywords (newest first).
func (s *SearchHistoryService) ListKeywords(ctx context.Context, uid uint64) ([]string, error) {
	var rows []extra.UserSearchHistory
	if err := s.db.WithContext(ctx).Where("user_id = ?", uid).
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

// UpsertKeyword adds or updates a search keyword for the user.
func (s *SearchHistoryService) UpsertKeyword(ctx context.Context, uid uint64, kw string, at time.Time) error {
	norm := searchhist.Norm(kw)
	if norm == "" {
		return nil
	}
	var rows []extra.UserSearchHistory
	if err := s.db.WithContext(ctx).Where("user_id = ? AND keyword_norm = ?", uid, norm).Find(&rows).Error; err != nil {
		return err
	}
	if len(rows) > 0 {
		keep := rows[0]
		for i := 1; i < len(rows); i++ {
			_ = s.db.WithContext(ctx).Delete(&rows[i]).Error
		}
		return s.db.WithContext(ctx).Model(&keep).Updates(map[string]interface{}{
			"keyword":    kw,
			"updated_at": at,
		}).Error
	}
	return s.db.WithContext(ctx).Create(&extra.UserSearchHistory{
		UserID:      uid,
		Keyword:     kw,
		KeywordNorm: norm,
		UpdatedAt:   at,
	}).Error
}

// TrimHistory removes excess entries beyond the limit.
func (s *SearchHistoryService) TrimHistory(ctx context.Context, uid uint64) error {
	var rows []extra.UserSearchHistory
	if err := s.db.WithContext(ctx).Where("user_id = ?", uid).Order("updated_at DESC, id DESC").Find(&rows).Error; err != nil {
		return err
	}
	if len(rows) <= maxUserSearchHistory {
		return nil
	}
	ids := make([]uint64, 0, len(rows)-maxUserSearchHistory)
	for i := maxUserSearchHistory; i < len(rows); i++ {
		ids = append(ids, rows[i].ID)
	}
	return s.db.WithContext(ctx).Where("id IN ?", ids).Delete(&extra.UserSearchHistory{}).Error
}

// ReplaceHistory replaces the entire search history for a user.
func (s *SearchHistoryService) ReplaceHistory(ctx context.Context, uid uint64, keywords []string) error {
	now := time.Now()
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.WithContext(ctx).Where("user_id = ?", uid).Delete(&extra.UserSearchHistory{}).Error; err != nil {
			return err
		}
		for i, kw := range keywords {
			norm := searchhist.Norm(kw)
			if norm == "" {
				continue
			}
			row := extra.UserSearchHistory{
				UserID:      uid,
				Keyword:     kw,
				KeywordNorm: norm,
				UpdatedAt:   now.Add(-time.Duration(i) * time.Millisecond),
			}
			if err := tx.WithContext(ctx).Create(&row).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
