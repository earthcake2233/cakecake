// Package banner owns home-page banner business logic (admin/HomeBanner).
package banner

import (
	"cakecake/internal/model/admin"
	"context"
	"time"

	"gorm.io/gorm"
)

// BannerService handles home banner CRUD.
type BannerService struct {
	store BannerStore
}

// NewBannerService creates a BannerService backed by the gorm store.
func NewBannerService(db *gorm.DB) *BannerService {
	return &BannerService{store: NewBannerStore(db)}
}

// BannerStore is the banner storage boundary (Phase 1: *gorm.DB impl).
type BannerStore interface {
	ListActiveBanners(ctx context.Context) ([]admin.HomeBanner, error)
	ListBanners(ctx context.Context) ([]admin.HomeBanner, error)
	CreateBanner(ctx context.Context, b *admin.HomeBanner) error
	GetBanner(ctx context.Context, id uint64) (*admin.HomeBanner, error)
	UpdateBanner(ctx context.Context, id uint64, updates map[string]interface{}) error
	DeleteBanner(ctx context.Context, id uint64) error
}

// BannerStoreImpl implements BannerStore using *gorm.DB (Phase 1 monolith).
type BannerStoreImpl struct {
	db *gorm.DB
}

var _ BannerStore = (*BannerStoreImpl)(nil)

// NewBannerStore creates a gorm-backed BannerStore implementation.
func NewBannerStore(db *gorm.DB) *BannerStoreImpl {
	return &BannerStoreImpl{db: db}
}

// ListActiveBanners lists banners enabled for public display within their time window.
func (p *BannerStoreImpl) ListActiveBanners(ctx context.Context) ([]admin.HomeBanner, error) {
	now := time.Now()
	var rows []admin.HomeBanner
	q := p.db.WithContext(ctx).Where("enabled = ?", true).
		Where("(start_at IS NULL OR start_at <= ?)", now).
		Where("(end_at IS NULL OR end_at >= ?)", now).
		Order("sort_order ASC, id ASC")
	if err := q.Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// ListBanners lists all banners for the admin panel.
func (p *BannerStoreImpl) ListBanners(ctx context.Context) ([]admin.HomeBanner, error) {
	var rows []admin.HomeBanner
	if err := p.db.WithContext(ctx).Order("sort_order ASC, id ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// CreateBanner inserts a home banner.
func (p *BannerStoreImpl) CreateBanner(ctx context.Context, b *admin.HomeBanner) error {
	return p.db.WithContext(ctx).Create(b).Error
}

// GetBanner loads a banner by id.
func (p *BannerStoreImpl) GetBanner(ctx context.Context, id uint64) (*admin.HomeBanner, error) {
	var b admin.HomeBanner
	if err := p.db.WithContext(ctx).First(&b, id).Error; err != nil {
		return nil, err
	}
	return &b, nil
}

// UpdateBanner applies partial updates to a banner.
func (p *BannerStoreImpl) UpdateBanner(ctx context.Context, id uint64, updates map[string]interface{}) error {
	return p.db.WithContext(ctx).Model(&admin.HomeBanner{}).Where("id = ?", id).Updates(updates).Error
}

// DeleteBanner removes a banner by id.
func (p *BannerStoreImpl) DeleteBanner(ctx context.Context, id uint64) error {
	return p.db.WithContext(ctx).Delete(&admin.HomeBanner{}, id).Error
}

// ---- service delegates ----

// ListActiveBanners returns active home page banners.
func (s *BannerService) ListActiveBanners(ctx context.Context) ([]admin.HomeBanner, error) {
	return s.store.ListActiveBanners(ctx)
}

// ListBanners returns all home banners ordered by sort_order and id.
func (s *BannerService) ListBanners(ctx context.Context) ([]admin.HomeBanner, error) {
	return s.store.ListBanners(ctx)
}

// CreateBanner creates a new home banner.
func (s *BannerService) CreateBanner(ctx context.Context, b *admin.HomeBanner) error {
	return s.store.CreateBanner(ctx, b)
}

// GetBanner returns a home banner by ID.
func (s *BannerService) GetBanner(ctx context.Context, id uint64) (*admin.HomeBanner, error) {
	return s.store.GetBanner(ctx, id)
}

// UpdateBanner updates fields of a home banner.
func (s *BannerService) UpdateBanner(ctx context.Context, id uint64, updates map[string]interface{}) error {
	return s.store.UpdateBanner(ctx, id, updates)
}

// DeleteBanner deletes a home banner by ID.
func (s *BannerService) DeleteBanner(ctx context.Context, id uint64) error {
	return s.store.DeleteBanner(ctx, id)
}
