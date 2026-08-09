package dynamic

import (
	"cakecake/internal/model/comment"
	"cakecake/internal/model/dynamic"
	"cakecake/internal/pkg/dbtx"
	"context"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// DynamicService handles user dynamic business logic.
type DynamicService struct {
	store DynamicStore
	rdb   *redis.Client
	log   *zap.Logger
}

// NewDynamicService creates a DynamicService with storage, cache, and logger.
func NewDynamicService(db *gorm.DB, rdb *redis.Client, log *zap.Logger) *DynamicService {
	return &DynamicService{store: NewDynamicStore(db), rdb: rdb, log: log}
}

// DynamicStore is the dynamic-domain storage boundary (Phase 1: *gorm.DB impl).
type DynamicStore interface {
	GetDynamicByID(ctx context.Context, id uint64) (*dynamic.UserDynamic, error)
	CreateDynamic(ctx context.Context, d *dynamic.UserDynamic) error
	DeleteDynamic(ctx context.Context, id uint64) error
	CountUserDynamics(ctx context.Context, userID uint64) (int64, error)
	ListUserDynamicsPaginated(ctx context.Context, userID uint64, page, pageSize int, status string) ([]dynamic.UserDynamic, int64, error)
	UpdateDynamic(ctx context.Context, id uint64, updates map[string]interface{}) error
	ToggleDynamicLike(ctx context.Context, userID, dynamicID uint64) (bool, error)
	BatchCheckLiked(ctx context.Context, viewerID uint64, dynamicIDs []uint64) map[uint64]bool
	ListDynamics(ctx context.Context, userID uint64, page, pageSize int) ([]dynamic.UserDynamic, int64, error)
	ListUserDynamicsCursor(ctx context.Context, userID uint64, cursorID uint64, limit int) ([]dynamic.UserDynamic, error)
	ListMyDynamicsAdvanced(ctx context.Context, f MyDynamicFilter) (*MyDynamicPageResult, error)
	AdminListDynamics(ctx context.Context, q string, page, pageSize int) (*AdminListDynamicsResult, error)
	AdminDeleteDynamicCascade(ctx context.Context, id uint64, fn func(tx dbtx.Tx) error) error
}

// DynamicStoreImpl implements DynamicStore using *gorm.DB (Phase 1 monolith).
type DynamicStoreImpl struct {
	db *gorm.DB
}

var _ DynamicStore = (*DynamicStoreImpl)(nil)

// NewDynamicStore creates a gorm-backed DynamicStore implementation.
func NewDynamicStore(db *gorm.DB) *DynamicStoreImpl {
	return &DynamicStoreImpl{db: db}
}

// GetDynamicByID loads a dynamic by id.
func (p *DynamicStoreImpl) GetDynamicByID(ctx context.Context, id uint64) (*dynamic.UserDynamic, error) {
	var d dynamic.UserDynamic
	if err := p.db.WithContext(ctx).First(&d, id).Error; err != nil {
		return nil, err
	}
	return &d, nil
}

// CreateDynamic inserts a dynamic row.
func (p *DynamicStoreImpl) CreateDynamic(ctx context.Context, d *dynamic.UserDynamic) error {
	return p.db.WithContext(ctx).Create(d).Error
}

// DeleteDynamic removes a dynamic and its cascade rows atomically.
func (p *DynamicStoreImpl) DeleteDynamic(ctx context.Context, id uint64) error {
	return p.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		_ = tx.Where("dynamic_id = ?", id).Delete(&comment.UserDynamicLike{}).Error
		var cmIDs []uint64
		tx.Model(&comment.DynamicComment{}).Where("dynamic_id = ?", id).Pluck("id", &cmIDs)
		if len(cmIDs) > 0 {
			_ = tx.Where("comment_id IN ?", cmIDs).Delete(&comment.DynamicCommentLike{}).Error
			_ = tx.Where("comment_id IN ?", cmIDs).Delete(&comment.DynamicCommentDislike{}).Error
		}
		_ = tx.Where("dynamic_id = ?", id).Delete(&comment.DynamicComment{}).Error
		return tx.Delete(&dynamic.UserDynamic{}, id).Error
	})
}

// CountUserDynamics returns the count of dynamics for a user.
func (p *DynamicStoreImpl) CountUserDynamics(ctx context.Context, userID uint64) (int64, error) {
	var n int64
	err := p.db.WithContext(ctx).Model(&dynamic.UserDynamic{}).Where("user_id = ?", userID).Count(&n).Error
	return n, err
}

// ListUserDynamicsPaginated returns dynamics for a user with pagination.
func (p *DynamicStoreImpl) ListUserDynamicsPaginated(ctx context.Context, userID uint64, page, pageSize int, status string) ([]dynamic.UserDynamic, int64, error) {
	q := p.db.WithContext(ctx).Model(&dynamic.UserDynamic{}).Where("user_id = ?", userID)
	if status != "" {
		q = q.Where("status = ?", status)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	offset := (page - 1) * pageSize
	var list []dynamic.UserDynamic
	if err := q.Order("id DESC").Offset(offset).Limit(pageSize).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

// UpdateDynamic updates dynamic fields.
func (p *DynamicStoreImpl) UpdateDynamic(ctx context.Context, id uint64, updates map[string]interface{}) error {
	return p.db.WithContext(ctx).Model(&dynamic.UserDynamic{}).Where("id = ?", id).Updates(updates).Error
}

// ToggleDynamicLike toggles like on a dynamic.
func (p *DynamicStoreImpl) ToggleDynamicLike(ctx context.Context, userID, dynamicID uint64) (bool, error) {
	var like comment.UserDynamicLike
	res := p.db.WithContext(ctx).Where("user_id = ? AND dynamic_id = ?", userID, dynamicID).Limit(1).Find(&like)
	if res.Error != nil {
		return false, res.Error
	}
	if like.ID > 0 {
		if err := p.db.WithContext(ctx).Delete(&like).Error; err != nil {
			return false, err
		}
		_ = p.db.WithContext(ctx).Model(&dynamic.UserDynamic{}).Where("id = ?", dynamicID).
			UpdateColumn("like_count", gorm.Expr("CASE WHEN like_count - 1 < 0 THEN 0 ELSE like_count - 1 END"))
		return false, nil
	}
	if err := p.db.WithContext(ctx).Create(&comment.UserDynamicLike{UserID: userID, DynamicID: dynamicID}).Error; err != nil {
		return false, err
	}
	_ = p.db.WithContext(ctx).Model(&dynamic.UserDynamic{}).Where("id = ?", dynamicID).
		UpdateColumn("like_count", gorm.Expr("like_count + 1"))
	return true, nil
}

// ListDynamics returns a user's dynamics with ordering.

// BatchCheckLiked returns a map of dynamicID->liked for the given viewer and dynamics.
func (p *DynamicStoreImpl) BatchCheckLiked(ctx context.Context, viewerID uint64, dynamicIDs []uint64) map[uint64]bool {
	out := make(map[uint64]bool, len(dynamicIDs))
	if viewerID == 0 || len(dynamicIDs) == 0 {
		return out
	}
	var rows []comment.UserDynamicLike
	p.db.WithContext(ctx).Where("user_id = ? AND dynamic_id IN ?", viewerID, dynamicIDs).Find(&rows)
	for _, r := range rows {
		out[r.DynamicID] = true
	}
	return out
}

// ListDynamics pages a user's dynamics (newest first).
func (p *DynamicStoreImpl) ListDynamics(ctx context.Context, userID uint64, page, pageSize int) ([]dynamic.UserDynamic, int64, error) {
	q := p.db.WithContext(ctx).Model(&dynamic.UserDynamic{}).Where("user_id = ?", userID)
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	offset := (page - 1) * pageSize
	var list []dynamic.UserDynamic
	if err := q.Order("id DESC").Offset(offset).Limit(pageSize).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

// ListUserDynamicsCursor lists user dynamics with cursor-based pagination.
func (p *DynamicStoreImpl) ListUserDynamicsCursor(ctx context.Context, userID uint64, cursorID uint64, limit int) ([]dynamic.UserDynamic, error) {
	q := p.db.WithContext(ctx).Model(&dynamic.UserDynamic{}).Where("user_id = ?", userID)
	if cursorID > 0 {
		q = q.Where("id < ?", cursorID)
	}
	var list []dynamic.UserDynamic
	if err := q.Order("id DESC").Limit(limit).Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

// MyDynamicFilter filters dynamics by owner, title and sort key.
type MyDynamicFilter struct {
	UserID   uint64
	TitleQ   string
	SortKey  string
	Page     int
	PageSize int
}

// MyDynamicPageResult is a paginated dynamic list for the creator panel.
type MyDynamicPageResult struct {
	Dynamics   []dynamic.UserDynamic
	Total      int64
	TotalPages int
}

// ListMyDynamicsAdvanced pages a user's dynamics with advanced filtering.
func (p *DynamicStoreImpl) ListMyDynamicsAdvanced(ctx context.Context, f MyDynamicFilter) (*MyDynamicPageResult, error) {
	q := p.db.WithContext(ctx).Model(&dynamic.UserDynamic{}).Where("user_id = ?", f.UserID)
	if f.TitleQ != "" {
		q = q.Where("title LIKE ? OR content LIKE ?", "%"+f.TitleQ+"%", "%"+f.TitleQ+"%")
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, err
	}
	totalPages := int((total + int64(f.PageSize) - 1) / int64(f.PageSize))
	if totalPages < 1 {
		totalPages = 1
	}
	if f.Page > totalPages {
		f.Page = totalPages
	}
	offset := (f.Page - 1) * f.PageSize
	var orderClause string
	switch f.SortKey {
	case "reply":
		orderClause = "comment_count DESC, id DESC"
	case "like":
		orderClause = "like_count DESC, id DESC"
	default:
		orderClause = "id DESC"
	}
	var list []dynamic.UserDynamic
	if err := q.Order(orderClause).Offset(offset).Limit(f.PageSize).Find(&list).Error; err != nil {
		return nil, err
	}
	return &MyDynamicPageResult{Dynamics: list, Total: total, TotalPages: totalPages}, nil
}

// AdminListDynamicsResult holds paginated admin dynamic list results.
type AdminListDynamicsResult struct {
	Total int64
	Rows  []dynamic.UserDynamic
}

// AdminListDynamics returns paginated dynamics with search filter for admin panel.
func (p *DynamicStoreImpl) AdminListDynamics(ctx context.Context, q string, page, pageSize int) (*AdminListDynamicsResult, error) {
	dbq := p.db.WithContext(ctx).Model(&dynamic.UserDynamic{})
	if q != "" {
		dbq = dbq.Where("title LIKE ? OR content LIKE ?", "%"+q+"%", "%"+q+"%")
	}
	var total int64
	if err := dbq.Count(&total).Error; err != nil {
		return nil, err
	}
	offset := (page - 1) * pageSize
	var rows []dynamic.UserDynamic
	if err := dbq.Order("created_at DESC, id DESC").Offset(offset).Limit(pageSize).Find(&rows).Error; err != nil {
		return nil, err
	}
	return &AdminListDynamicsResult{Total: total, Rows: rows}, nil
}

// AdminDeleteDynamicCascade deletes a dynamic within a transaction with a custom function.
func (p *DynamicStoreImpl) AdminDeleteDynamicCascade(ctx context.Context, id uint64, fn func(tx dbtx.Tx) error) error {
	return p.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(tx)
	})
}

// GetDynamicByID returns a dynamic by ID.
func (s *DynamicService) GetDynamicByID(ctx context.Context, id uint64) (*dynamic.UserDynamic, error) {
	return s.store.GetDynamicByID(ctx, id)
}

// CreateDynamic creates a new dynamic.
func (s *DynamicService) CreateDynamic(ctx context.Context, d *dynamic.UserDynamic) error {
	return s.store.CreateDynamic(ctx, d)
}

// DeleteDynamic deletes a dynamic and its related likes/comments.
func (s *DynamicService) DeleteDynamic(ctx context.Context, id uint64) error {
	return s.store.DeleteDynamic(ctx, id)
}

// CountUserDynamics returns the count of dynamics for a user.
func (s *DynamicService) CountUserDynamics(ctx context.Context, userID uint64) (int64, error) {
	return s.store.CountUserDynamics(ctx, userID)
}

// ListUserDynamicsPaginated returns dynamics for a user with pagination.
func (s *DynamicService) ListUserDynamicsPaginated(ctx context.Context, userID uint64, page, pageSize int, status string) ([]dynamic.UserDynamic, int64, error) {
	return s.store.ListUserDynamicsPaginated(ctx, userID, page, pageSize, status)
}

// UpdateDynamic updates dynamic fields.
func (s *DynamicService) UpdateDynamic(ctx context.Context, id uint64, updates map[string]interface{}) error {
	return s.store.UpdateDynamic(ctx, id, updates)
}

// ToggleDynamicLike toggles like on a dynamic.
func (s *DynamicService) ToggleDynamicLike(ctx context.Context, userID, dynamicID uint64) (bool, error) {
	return s.store.ToggleDynamicLike(ctx, userID, dynamicID)
}

// BatchCheckLiked returns a map of dynamicID->liked for the given viewer and dynamics.
func (s *DynamicService) BatchCheckLiked(ctx context.Context, viewerID uint64, dynamicIDs []uint64) map[uint64]bool {
	return s.store.BatchCheckLiked(ctx, viewerID, dynamicIDs)
}

// ListDynamics returns a user's dynamics with pagination.
func (s *DynamicService) ListDynamics(ctx context.Context, userID uint64, page, pageSize int) ([]dynamic.UserDynamic, int64, error) {
	return s.store.ListDynamics(ctx, userID, page, pageSize)
}

// ListUserDynamicsCursor lists user dynamics with cursor-based pagination.
func (s *DynamicService) ListUserDynamicsCursor(ctx context.Context, userID uint64, cursorID uint64, limit int) ([]dynamic.UserDynamic, error) {
	return s.store.ListUserDynamicsCursor(ctx, userID, cursorID, limit)
}

// ListMyDynamicsAdvanced lists user dynamics with advanced filtering.
func (s *DynamicService) ListMyDynamicsAdvanced(ctx context.Context, f MyDynamicFilter) (*MyDynamicPageResult, error) {
	return s.store.ListMyDynamicsAdvanced(ctx, f)
}

// AdminListDynamics returns paginated dynamics with search filter for admin panel.
func (s *DynamicService) AdminListDynamics(ctx context.Context, q string, page, pageSize int) (*AdminListDynamicsResult, error) {
	return s.store.AdminListDynamics(ctx, q, page, pageSize)
}

// AdminDeleteDynamicCascade deletes a dynamic within a transaction with a custom function.
func (s *DynamicService) AdminDeleteDynamicCascade(ctx context.Context, id uint64, fn func(tx dbtx.Tx) error) error {
	return s.store.AdminDeleteDynamicCascade(ctx, id, fn)
}
