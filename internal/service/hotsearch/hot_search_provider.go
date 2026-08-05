package hotsearch

import (
	"cakecake/internal/model/admin"
	"context"

	"gorm.io/gorm"
)

// HotSearchProvider is the hot-search DB storage boundary.
// Phase 1: *gorm.DB impl. Phase 2+: replaced by gRPC client / per-domain store.
type HotSearchProvider interface {
	ListOps(ctx context.Context) ([]admin.HotSearchOp, error)
	CreateOp(ctx context.Context, op *admin.HotSearchOp) error
	GetOp(ctx context.Context, id uint64) (*admin.HotSearchOp, error)
	UpdateOp(ctx context.Context, id uint64, updates map[string]any) error
	DeleteOp(ctx context.Context, id uint64) error
	FindAllOps(ctx context.Context) ([]admin.HotSearchOp, error)
	// UpdateOpModel applies updates to an already-loaded op.
	UpdateOpModel(ctx context.Context, op *admin.HotSearchOp, updates map[string]any) error
	// ReloadOp re-fetches an op by id.
	ReloadOp(ctx context.Context, id uint64) (*admin.HotSearchOp, error)
	// UpdateOpPinRank sets a single op's pin_rank.
	UpdateOpPinRank(ctx context.Context, op *admin.HotSearchOp, rank int) error

	SaveLayout(ctx context.Context, entries []HotSearchLayoutEntry) error
	ClearLayout(ctx context.Context) error
	HasLayout(ctx context.Context) bool
	ApplyLayoutMove(ctx context.Context, keyword, title string, targetRank int) error
	RemoveLayoutEntry(ctx context.Context, keyword string) error
	EnsureLayoutFromMerged(ctx context.Context, rec *SearchHotRecorder, limit int) error
	ListMerged(ctx context.Context, rec *SearchHotRecorder, limit int) ([]HotSearchItem, error)
	ListMergedDetail(ctx context.Context, rec *SearchHotRecorder, limit int) ([]HotSearchMergedDetail, error)
	ActiveOpFlags(ctx context.Context) map[string]HotSearchOpFlags
	SearchSuggest(ctx context.Context, rec *SearchHotRecorder, userID uint64, term string, limit int) []SearchSuggestTag
}

// HotSearchProviderImpl implements HotSearchProvider using *gorm.DB (Phase 1 monolith).
type HotSearchProviderImpl struct {
	db *gorm.DB
}

// NewHotSearchProvider creates a gorm-backed HotSearchProvider implementation.
func NewHotSearchProvider(db *gorm.DB) *HotSearchProviderImpl {
	return &HotSearchProviderImpl{db: db}
}

// ListOps lists hot-search ops ordered by pin rank.
func (p *HotSearchProviderImpl) ListOps(ctx context.Context) ([]admin.HotSearchOp, error) {
	var rows []admin.HotSearchOp
	if err := p.db.WithContext(ctx).Order("pin_rank ASC, id ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// CreateOp inserts a hot-search op.
func (p *HotSearchProviderImpl) CreateOp(ctx context.Context, op *admin.HotSearchOp) error {
	return p.db.WithContext(ctx).Create(op).Error
}

// GetOp loads a hot-search op by id.
func (p *HotSearchProviderImpl) GetOp(ctx context.Context, id uint64) (*admin.HotSearchOp, error) {
	var op admin.HotSearchOp
	if err := p.db.WithContext(ctx).First(&op, id).Error; err != nil {
		return nil, err
	}
	return &op, nil
}

// UpdateOp applies partial updates to a hot-search op by id.
func (p *HotSearchProviderImpl) UpdateOp(ctx context.Context, id uint64, updates map[string]any) error {
	return p.db.WithContext(ctx).Model(&admin.HotSearchOp{}).Where("id = ?", id).Updates(updates).Error
}

// DeleteOp removes a hot-search op by id.
func (p *HotSearchProviderImpl) DeleteOp(ctx context.Context, id uint64) error {
	return p.db.WithContext(ctx).Delete(&admin.HotSearchOp{}, id).Error
}

// FindAllOps loads every hot-search op.
func (p *HotSearchProviderImpl) FindAllOps(ctx context.Context) ([]admin.HotSearchOp, error) {
	var rows []admin.HotSearchOp
	if err := p.db.WithContext(ctx).Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// UpdateOpModel applies partial updates to a loaded op.
func (p *HotSearchProviderImpl) UpdateOpModel(ctx context.Context, op *admin.HotSearchOp, updates map[string]any) error {
	return p.db.WithContext(ctx).Model(op).Updates(updates).Error
}

// ReloadOp refreshes an op row from the database.
func (p *HotSearchProviderImpl) ReloadOp(ctx context.Context, id uint64) (*admin.HotSearchOp, error) {
	var op admin.HotSearchOp
	if err := p.db.WithContext(ctx).First(&op, id).Error; err != nil {
		return nil, err
	}
	return &op, nil
}

// UpdateOpPinRank sets an op's pin rank.
func (p *HotSearchProviderImpl) UpdateOpPinRank(ctx context.Context, op *admin.HotSearchOp, rank int) error {
	return p.db.WithContext(ctx).Model(op).Update("pin_rank", rank).Error
}

// SaveLayout persists the display layout entries.
func (p *HotSearchProviderImpl) SaveLayout(ctx context.Context, entries []HotSearchLayoutEntry) error {
	return SaveHotSearchDisplayLayout(p.db, entries)
}

// ClearLayout removes the display layout.
func (p *HotSearchProviderImpl) ClearLayout(ctx context.Context) error {
	return ClearHotSearchDisplayLayout(p.db)
}

// HasLayout reports whether a display layout exists.
func (p *HotSearchProviderImpl) HasLayout(ctx context.Context) bool {
	return HasHotSearchDisplayLayout(p.db)
}

// ApplyLayoutMove moves a layout entry to a target rank.
func (p *HotSearchProviderImpl) ApplyLayoutMove(ctx context.Context, keyword, title string, targetRank int) error {
	return ApplyHotSearchLayoutMove(p.db, keyword, title, targetRank)
}

// RemoveLayoutEntry removes a layout entry by keyword.
func (p *HotSearchProviderImpl) RemoveLayoutEntry(ctx context.Context, keyword string) error {
	return RemoveHotSearchLayoutEntry(p.db, keyword)
}

// EnsureLayoutFromMerged seeds the display layout from merged results when absent.
func (p *HotSearchProviderImpl) EnsureLayoutFromMerged(ctx context.Context, rec *SearchHotRecorder, limit int) error {
	return EnsureHotSearchLayoutFromMerged(ctx, p.db, rec, limit)
}

// ListMerged returns merged hot-search items (ops + Redis).
func (p *HotSearchProviderImpl) ListMerged(ctx context.Context, rec *SearchHotRecorder, limit int) ([]HotSearchItem, error) {
	return ListHotSearchMerged(ctx, p.db, rec, limit)
}

// ListMergedDetail returns merged hot-search details with source annotations.
func (p *HotSearchProviderImpl) ListMergedDetail(ctx context.Context, rec *SearchHotRecorder, limit int) ([]HotSearchMergedDetail, error) {
	return ListHotSearchMergedDetail(ctx, p.db, rec, limit)
}

// ActiveOpFlags maps normalized keywords to intervention flags.
func (p *HotSearchProviderImpl) ActiveOpFlags(ctx context.Context) map[string]HotSearchOpFlags {
	return ActiveHotSearchOpFlags(p.db)
}

// SearchSuggest builds suggestion tags from history, ops, and Redis ranks.
func (p *HotSearchProviderImpl) SearchSuggest(ctx context.Context, rec *SearchHotRecorder, userID uint64, term string, limit int) []SearchSuggestTag {
	return SearchSuggest(ctx, p.db, rec, userID, term, limit)
}
