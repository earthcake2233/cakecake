package service

import (
	"context"
	"minibili/internal/model/admin"
	"strings"

	"gorm.io/gorm"
)

// HotSearchService handles hot search business operations,
// wrapping both DB (HotSearchOp, DisplayLayout) and Redis (SearchHotRecorder).
type HotSearchService struct {
	db  *gorm.DB
	rec *SearchHotRecorder
}

// NewHotSearchService creates a new HotSearchService.
func NewHotSearchService(db *gorm.DB, rec *SearchHotRecorder) *HotSearchService {
	return &HotSearchService{db: db, rec: rec}
}

// ---------------------------------------------------------------------------
// HotSearchOp CRUD
// ---------------------------------------------------------------------------

// ListOps returns all HotSearchOp records ordered by pin_rank and id.
func (s *HotSearchService) ListOps(ctx context.Context) ([]admin.HotSearchOp, error) {
	var rows []admin.HotSearchOp
	if err := s.db.WithContext(ctx).Order("pin_rank ASC, id ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// CreateOp creates a new HotSearchOp.
func (s *HotSearchService) CreateOp(ctx context.Context, op *admin.HotSearchOp) error {
	return s.db.WithContext(ctx).Create(op).Error
}

// GetOp retrieves a HotSearchOp by ID.
func (s *HotSearchService) GetOp(ctx context.Context, id uint64) (*admin.HotSearchOp, error) {
	var op admin.HotSearchOp
	if err := s.db.WithContext(ctx).First(&op, id).Error; err != nil {
		return nil, err
	}
	return &op, nil
}

// UpdateOp updates fields of a HotSearchOp.
func (s *HotSearchService) UpdateOp(ctx context.Context, id uint64, updates map[string]any) error {
	return s.db.WithContext(ctx).Model(&admin.HotSearchOp{}).Where("id = ?", id).Updates(updates).Error
}

// DeleteOp deletes a HotSearchOp by ID.
func (s *HotSearchService) DeleteOp(ctx context.Context, id uint64) error {
	return s.db.WithContext(ctx).Delete(&admin.HotSearchOp{}, id).Error
}

// FindAllOps returns all ops without ordering (for reorder logic).
func (s *HotSearchService) FindAllOps(ctx context.Context) ([]admin.HotSearchOp, error) {
	var rows []admin.HotSearchOp
	if err := s.db.WithContext(ctx).Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// ---------------------------------------------------------------------------
// QuickOp (create or update)
// ---------------------------------------------------------------------------

// QuickOpCreateOrUpdate implements the quick-op logic: find existing by normalized keyword,
// then create or update, syncing layout as needed.
// Returns the op, whether it was newly created, and any error.
func (s *HotSearchService) QuickOpCreateOrUpdate(ctx context.Context, opType, keyword, displayTitle, badge string, pinRank int) (*admin.HotSearchOp, bool, error) {
	kw := strings.TrimSpace(keyword)
	norm := NormalizeSearchKeyword(kw)

	// Find existing op by normalized keyword
	var existing admin.HotSearchOp
	found := false
	if norm != "" {
		var rows []admin.HotSearchOp
		_ = s.db.WithContext(ctx).Find(&rows).Error
		for i := range rows {
			if NormalizeSearchKeyword(rows[i].Keyword) == norm {
				existing = rows[i]
				found = true
				break
			}
		}
	}

	display := strings.TrimSpace(displayTitle)
	if display == "" {
		display = kw
	}
	if opType == "block" {
		pinRank = 0
		badge = ""
	} else if pinRank <= 0 {
		pinRank = 1
	}

	if found {
		updates := map[string]any{
			"op_type":       opType,
			"keyword":       kw,
			"display_title": display,
			"badge":         strings.TrimSpace(badge),
			"pin_rank":      pinRank,
			"enabled":       true,
		}
		if err := s.db.WithContext(ctx).Model(&existing).Updates(updates).Error; err != nil {
			return nil, false, err
		}
		_ = s.db.WithContext(ctx).First(&existing, existing.ID)
		s.syncLayoutAfterOp(ctx, kw, display, opType, pinRank)
		return &existing, false, nil
	}

	op := admin.HotSearchOp{
		OpType:       opType,
		Keyword:      kw,
		DisplayTitle: display,
		Badge:        strings.TrimSpace(badge),
		PinRank:      pinRank,
		Enabled:      true,
	}
	if err := s.db.WithContext(ctx).Create(&op).Error; err != nil {
		return nil, false, err
	}
	s.syncLayoutAfterOp(ctx, kw, display, opType, pinRank)
	return &op, true, nil
}

// ReorderItems persists drag order and updates pin_rank for pin/manual ops.
// ReorderItem represents one entry in a reorder request.
type ReorderItem struct {
	Keyword string
	Title   string
	OpID    uint64
	Source  string
}

func (s *HotSearchService) ReorderItems(ctx context.Context, items []ReorderItem) error {
	allOps, err := s.FindAllOps(ctx)
	if err != nil {
		return err
	}
	opByNorm := make(map[string]*admin.HotSearchOp, len(allOps))
	opByID := make(map[uint64]*admin.HotSearchOp, len(allOps))
	for i := range allOps {
		op := &allOps[i]
		opByID[op.ID] = op
		if norm := NormalizeSearchKeyword(op.Keyword); norm != "" {
			opByNorm[norm] = op
		}
	}

	for i, it := range items {
		kw := strings.TrimSpace(it.Keyword)
		title := strings.TrimSpace(it.Title)
		if kw == "" {
			kw = title
		}
		if kw == "" {
			continue
		}
		items[i].Keyword = kw
		if items[i].Title == "" {
			items[i].Title = kw
		}
		rank := i + 1
		norm := NormalizeSearchKeyword(kw)
		var existing *admin.HotSearchOp
		if it.OpID > 0 {
			existing = opByID[it.OpID]
		}
		if existing == nil && norm != "" {
			existing = opByNorm[norm]
		}
		if existing != nil && (existing.OpType == "pin" || existing.OpType == "manual") {
			if err := s.db.WithContext(ctx).Model(existing).Update("pin_rank", rank).Error; err != nil {
				return err
			}
		}
	}

	layout := make([]HotSearchLayoutEntry, len(items))
	for i, it := range items {
		layout[i] = HotSearchLayoutEntry{Keyword: it.Keyword, Title: it.Title}
	}
	return SaveHotSearchDisplayLayout(s.db, layout)
}

// syncLayoutAfterOp mirrors the original syncHotSearchLayoutAfterOp helper.
func (s *HotSearchService) syncLayoutAfterOp(ctx context.Context, keyword, title, opType string, pinRank int) {
	switch opType {
	case "block":
		_ = RemoveHotSearchLayoutEntry(s.db, keyword)
	case "pin", "manual":
		if HasHotSearchDisplayLayout(s.db) {
			_ = ApplyHotSearchLayoutMove(s.db, keyword, title, pinRank)
			return
		}
		_ = EnsureHotSearchLayoutFromMerged(ctx, s.db, s.rec, 10)
		_ = ApplyHotSearchLayoutMove(s.db, keyword, title, pinRank)
	}
}

// ---------------------------------------------------------------------------
// Merged list (Redis + DB ops)
// ---------------------------------------------------------------------------

// ListMerged returns merged hot search items (DB ops + Redis auto rank).
func (s *HotSearchService) ListMerged(ctx context.Context, limit int) ([]HotSearchItem, error) {
	return ListHotSearchMerged(ctx, s.db, s.rec, limit)
}

// ListMergedDetail returns annotated merged hot search items for admin dashboard.
func (s *HotSearchService) ListMergedDetail(ctx context.Context, limit int) ([]HotSearchMergedDetail, error) {
	return ListHotSearchMergedDetail(ctx, s.db, s.rec, limit)
}

// ActiveOpFlags returns active intervention flags for all keywords.
func (s *HotSearchService) ActiveOpFlags(ctx context.Context) map[string]HotSearchOpFlags {
	return ActiveHotSearchOpFlags(s.db)
}

// ---------------------------------------------------------------------------
// Display Layout
// ---------------------------------------------------------------------------

// HasDisplayLayout reports whether a custom drag order exists.
func (s *HotSearchService) HasDisplayLayout(ctx context.Context) bool {
	return HasHotSearchDisplayLayout(s.db)
}

// SaveDisplayLayout persists drag order.
func (s *HotSearchService) SaveDisplayLayout(ctx context.Context, entries []HotSearchLayoutEntry) error {
	return SaveHotSearchDisplayLayout(s.db, entries)
}

// ClearDisplayLayout removes custom drag order.
func (s *HotSearchService) ClearDisplayLayout(ctx context.Context) error {
	return ClearHotSearchDisplayLayout(s.db)
}

// ApplyLayoutMove reorders one keyword in drag layout.
func (s *HotSearchService) ApplyLayoutMove(ctx context.Context, keyword, title string, targetRank int) error {
	return ApplyHotSearchLayoutMove(s.db, keyword, title, targetRank)
}

// RemoveLayoutEntry drops one keyword from drag layout.
func (s *HotSearchService) RemoveLayoutEntry(ctx context.Context, keyword string) error {
	return RemoveHotSearchLayoutEntry(s.db, keyword)
}

// EnsureLayoutFromMerged seeds layout when absent.
func (s *HotSearchService) EnsureLayoutFromMerged(ctx context.Context, limit int) error {
	return EnsureHotSearchLayoutFromMerged(ctx, s.db, s.rec, limit)
}

// ---------------------------------------------------------------------------
// Redis hot search recorder passthrough
// ---------------------------------------------------------------------------

// TopWithScores returns Redis auto rank with search counts (admin).
func (s *HotSearchService) TopWithScores(ctx context.Context, limit int) ([]HotSearchRedisRow, error) {
	if s.rec == nil {
		return nil, nil
	}
	return s.rec.TopWithScores(ctx, limit)
}

// RemoveKeywordFromRedis deletes a keyword from Redis hot-search rank.
func (s *HotSearchService) RemoveKeywordFromRedis(ctx context.Context, keyword string) error {
	if s.rec == nil {
		return nil
	}
	return s.rec.RemoveKeyword(ctx, keyword)
}

// BoostKeyword increases Redis hot-search score.
func (s *HotSearchService) BoostKeyword(ctx context.Context, keyword string, delta float64) error {
	if s.rec == nil {
		return nil
	}
	return s.rec.BoostKeyword(ctx, keyword, delta)
}

// ---------------------------------------------------------------------------
// Suggest
// ---------------------------------------------------------------------------

// SearchSuggest builds keyword suggestions for autocomplete.
func (s *HotSearchService) SearchSuggest(ctx context.Context, userID uint64, term string, limit int) []SearchSuggestTag {
	return SearchSuggest(ctx, s.db, s.rec, userID, term, limit)
}

// ValidateSuggestTerm validates suggest term length.
func (s *HotSearchService) ValidateSuggestTerm(term string) bool {
	return ValidateSuggestTerm(term)
}

// NormalizeKeyword lowercases and strips spaces for dedup / ZSET member.
func (s *HotSearchService) NormalizeKeyword(keyword string) string {
	return NormalizeSearchKeyword(keyword)
}
