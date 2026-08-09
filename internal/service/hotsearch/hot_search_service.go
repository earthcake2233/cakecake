package hotsearch

import (
	"cakecake/internal/model/admin"
	"context"
	"strings"

	"gorm.io/gorm"
)

// HotSearchService handles hot search business operations,
// wrapping both DB (HotSearchOp, DisplayLayout) and Redis (SearchHotRecorder).
type HotSearchService struct {
	store HotSearchProvider
	rec   *SearchHotRecorder
}

// NewHotSearchService creates a new HotSearchService.
func NewHotSearchService(db *gorm.DB, rec *SearchHotRecorder) *HotSearchService {
	return &HotSearchService{store: NewHotSearchProvider(db), rec: rec}
}

// ---------------------------------------------------------------------------
// HotSearchOp CRUD
// ---------------------------------------------------------------------------

// ListOps returns all HotSearchOp records ordered by pin_rank and id.
func (s *HotSearchService) ListOps(ctx context.Context) ([]admin.HotSearchOp, error) {
	return s.store.ListOps(ctx)
}

// CreateOp creates a new HotSearchOp.
func (s *HotSearchService) CreateOp(ctx context.Context, op *admin.HotSearchOp) error {
	return s.store.CreateOp(ctx, op)
}

// GetOp retrieves a HotSearchOp by ID.
func (s *HotSearchService) GetOp(ctx context.Context, id uint64) (*admin.HotSearchOp, error) {
	return s.store.GetOp(ctx, id)
}

// UpdateOp updates fields of a HotSearchOp.
func (s *HotSearchService) UpdateOp(ctx context.Context, id uint64, updates map[string]any) error {
	return s.store.UpdateOp(ctx, id, updates)
}

// DeleteOp deletes a HotSearchOp by ID.
func (s *HotSearchService) DeleteOp(ctx context.Context, id uint64) error {
	return s.store.DeleteOp(ctx, id)
}

// FindAllOps returns all ops without ordering (for reorder logic).
func (s *HotSearchService) FindAllOps(ctx context.Context) ([]admin.HotSearchOp, error) {
	return s.store.FindAllOps(ctx)
}

// ---------------------------------------------------------------------------
// QuickOp (create or update)
// ---------------------------------------------------------------------------

// QuickOpCreateOrUpdate implements the quick-op logic: find existing by normalized keyword,
// then create or update, syncing layout as needed.
// Returns the op, whether it was newly created, and any error.
func (s *HotSearchService) QuickOpCreateOrUpdate(ctx context.Context, opType, keyword, displayTitle, badge string, pinRank int) (*admin.HotSearchOp, bool, error) {
	kw := strings.TrimSpace(keyword)
	norm := normalizeSearchKeyword(kw)

	// Find existing op by normalized keyword
	var existing admin.HotSearchOp
	found := false
	if norm != "" {
		rows, _ := s.store.FindAllOps(ctx)
		for i := range rows {
			if normalizeSearchKeyword(rows[i].Keyword) == norm {
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
		if err := s.store.UpdateOpModel(ctx, &existing, updates); err != nil {
			return nil, false, err
		}
		if reloaded, err := s.store.ReloadOp(ctx, existing.ID); err == nil {
			existing = *reloaded
		}
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
	if err := s.store.CreateOp(ctx, &op); err != nil {
		return nil, false, err
	}
	s.syncLayoutAfterOp(ctx, kw, display, opType, pinRank)
	return &op, true, nil
}

// ReorderItem represents one entry in a reorder request.
type ReorderItem struct {
	Keyword string
	Title   string
	OpID    uint64
	Source  string
}

// ReorderItems persists a manual ordering of hot-search items.
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
		if norm := normalizeSearchKeyword(op.Keyword); norm != "" {
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
		norm := normalizeSearchKeyword(kw)
		var existing *admin.HotSearchOp
		if it.OpID > 0 {
			existing = opByID[it.OpID]
		}
		if existing == nil && norm != "" {
			existing = opByNorm[norm]
		}
		if existing != nil && (existing.OpType == "pin" || existing.OpType == "manual") {
			if err := s.store.UpdateOpPinRank(ctx, existing, rank); err != nil {
				return err
			}
		}
	}

	layout := make([]HotSearchLayoutEntry, len(items))
	for i, it := range items {
		layout[i] = HotSearchLayoutEntry{Keyword: it.Keyword, Title: it.Title}
	}
	return s.store.SaveLayout(ctx, layout)
}

// syncLayoutAfterOp mirrors the original syncHotSearchLayoutAfterOp helper.
func (s *HotSearchService) syncLayoutAfterOp(ctx context.Context, keyword, title, opType string, pinRank int) {
	switch opType {
	case "block":
		_ = s.store.RemoveLayoutEntry(ctx, keyword)
	case "pin", "manual":
		if s.store.HasLayout(ctx) {
			_ = s.store.ApplyLayoutMove(ctx, keyword, title, pinRank)
			return
		}
		_ = s.store.EnsureLayoutFromMerged(ctx, s.rec, 10)
		_ = s.store.ApplyLayoutMove(ctx, keyword, title, pinRank)
	}
}

// ---------------------------------------------------------------------------
// Merged list (Redis + DB ops)
// ---------------------------------------------------------------------------

// ListMerged returns merged hot search items (DB ops + Redis auto rank).
func (s *HotSearchService) ListMerged(ctx context.Context, limit int) ([]HotSearchItem, error) {
	return s.store.ListMerged(ctx, s.rec, limit)
}

// ListMergedDetail returns annotated merged hot search items for admin dashboard.
func (s *HotSearchService) ListMergedDetail(ctx context.Context, limit int) ([]HotSearchMergedDetail, error) {
	return s.store.ListMergedDetail(ctx, s.rec, limit)
}

// ActiveOpFlags returns active intervention flags for all keywords.
func (s *HotSearchService) ActiveOpFlags(ctx context.Context) map[string]HotSearchOpFlags {
	return s.store.ActiveOpFlags(ctx)
}

// ---------------------------------------------------------------------------
// Display Layout
// ---------------------------------------------------------------------------

// HasDisplayLayout reports whether a custom drag order exists.
func (s *HotSearchService) HasDisplayLayout(ctx context.Context) bool {
	return s.store.HasLayout(ctx)
}

// SaveDisplayLayout persists drag order.
func (s *HotSearchService) SaveDisplayLayout(ctx context.Context, entries []HotSearchLayoutEntry) error {
	return s.store.SaveLayout(ctx, entries)
}

// ClearDisplayLayout removes custom drag order.
func (s *HotSearchService) ClearDisplayLayout(ctx context.Context) error {
	return s.store.ClearLayout(ctx)
}

// ApplyLayoutMove reorders one keyword in drag layout.
func (s *HotSearchService) ApplyLayoutMove(ctx context.Context, keyword, title string, targetRank int) error {
	return s.store.ApplyLayoutMove(ctx, keyword, title, targetRank)
}

// RemoveLayoutEntry drops one keyword from drag layout.
func (s *HotSearchService) RemoveLayoutEntry(ctx context.Context, keyword string) error {
	return s.store.RemoveLayoutEntry(ctx, keyword)
}

// EnsureLayoutFromMerged seeds layout when absent.
func (s *HotSearchService) EnsureLayoutFromMerged(ctx context.Context, limit int) error {
	return s.store.EnsureLayoutFromMerged(ctx, s.rec, limit)
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
	return s.store.SearchSuggest(ctx, s.rec, userID, term, limit)
}

// ValidateSuggestTerm validates suggest term length.
func (s *HotSearchService) ValidateSuggestTerm(term string) bool {
	return ValidateSuggestTerm(term)
}

// NormalizeKeyword lowercases and strips spaces for dedup / ZSET member.
func (s *HotSearchService) NormalizeKeyword(keyword string) string {
	return normalizeSearchKeyword(keyword)
}
