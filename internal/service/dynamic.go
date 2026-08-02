package service

import (
	"cakecake/internal/model/comment"
	"cakecake/internal/model/dynamic"
	"context"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// DynamicService handles user dynamic business logic.
type DynamicService struct {
	db  *gorm.DB
	rdb *redis.Client
	log *zap.Logger
}

func NewDynamicService(db *gorm.DB, rdb *redis.Client, log *zap.Logger) *DynamicService {
	return &DynamicService{db: db, rdb: rdb, log: log}
}

func (s *DynamicService) GetDynamicByID(ctx context.Context, id uint64) (*dynamic.UserDynamic, error) {
	var d dynamic.UserDynamic
	if err := s.db.WithContext(ctx).First(&d, id).Error; err != nil {
		return nil, err
	}
	return &d, nil
}

func (s *DynamicService) CreateDynamic(ctx context.Context, d *dynamic.UserDynamic) error {
	return s.db.WithContext(ctx).Create(d).Error
}

func (s *DynamicService) DeleteDynamic(ctx context.Context, id uint64) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
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
func (s *DynamicService) CountUserDynamics(ctx context.Context, userID uint64) (int64, error) {
	var n int64
	err := s.db.WithContext(ctx).Model(&dynamic.UserDynamic{}).Where("user_id = ?", userID).Count(&n).Error
	return n, err
}

// ListUserDynamicsPaginated returns dynamics for a user with pagination.
func (s *DynamicService) ListUserDynamicsPaginated(ctx context.Context, userID uint64, page, pageSize int, status string) ([]dynamic.UserDynamic, int64, error) {
	q := s.db.WithContext(ctx).Model(&dynamic.UserDynamic{}).Where("user_id = ?", userID)
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
func (s *DynamicService) UpdateDynamic(ctx context.Context, id uint64, updates map[string]interface{}) error {
	return s.db.WithContext(ctx).Model(&dynamic.UserDynamic{}).Where("id = ?", id).Updates(updates).Error
}

// ToggleDynamicLike toggles like on a dynamic.
func (s *DynamicService) ToggleDynamicLike(ctx context.Context, userID, dynamicID uint64) (bool, error) {
	var like comment.UserDynamicLike
	res := s.db.WithContext(ctx).Where("user_id = ? AND dynamic_id = ?", userID, dynamicID).Limit(1).Find(&like)
	if res.Error != nil {
		return false, res.Error
	}
	if like.ID > 0 {
		if err := s.db.WithContext(ctx).Delete(&like).Error; err != nil {
			return false, err
		}
		_ = s.db.WithContext(ctx).Model(&dynamic.UserDynamic{}).Where("id = ?", dynamicID).
			UpdateColumn("like_count", gorm.Expr("CASE WHEN like_count - 1 < 0 THEN 0 ELSE like_count - 1 END"))
		return false, nil
	}
	if err := s.db.WithContext(ctx).Create(&comment.UserDynamicLike{UserID: userID, DynamicID: dynamicID}).Error; err != nil {
		return false, err
	}
	_ = s.db.WithContext(ctx).Model(&dynamic.UserDynamic{}).Where("id = ?", dynamicID).
		UpdateColumn("like_count", gorm.Expr("like_count + 1"))
	return true, nil
}

// ListDynamics returns a user's dynamics with ordering.

// BatchCheckLiked returns a map of dynamicID->liked for the given viewer and dynamics.
func (s *DynamicService) BatchCheckLiked(ctx context.Context, viewerID uint64, dynamicIDs []uint64) map[uint64]bool {
	out := make(map[uint64]bool, len(dynamicIDs))
	if viewerID == 0 || len(dynamicIDs) == 0 {
		return out
	}
	var rows []comment.UserDynamicLike
	s.db.WithContext(ctx).Where("user_id = ? AND dynamic_id IN ?", viewerID, dynamicIDs).Find(&rows)
	for _, r := range rows {
		out[r.DynamicID] = true
	}
	return out
}

func (s *DynamicService) ListDynamics(ctx context.Context, userID uint64, page, pageSize int) ([]dynamic.UserDynamic, int64, error) {
	q := s.db.WithContext(ctx).Model(&dynamic.UserDynamic{}).Where("user_id = ?", userID)
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
func (s *DynamicService) ListUserDynamicsCursor(ctx context.Context, userID uint64, cursorID uint64, limit int) ([]dynamic.UserDynamic, error) {
	q := s.db.WithContext(ctx).Model(&dynamic.UserDynamic{}).Where("user_id = ?", userID)
	if cursorID > 0 {
		q = q.Where("id < ?", cursorID)
	}
	var list []dynamic.UserDynamic
	if err := q.Order("id DESC").Limit(limit).Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

// ListMyDynamicsAdvanced lists user dynamics with advanced filtering (title search, custom sort).
type MyDynamicFilter struct {
	UserID   uint64
	TitleQ   string
	SortKey  string
	Page     int
	PageSize int
}

type MyDynamicPageResult struct {
	Dynamics   []dynamic.UserDynamic
	Total      int64
	TotalPages int
}

func (s *DynamicService) ListMyDynamicsAdvanced(ctx context.Context, f MyDynamicFilter) (*MyDynamicPageResult, error) {
	q := s.db.WithContext(ctx).Model(&dynamic.UserDynamic{}).Where("user_id = ?", f.UserID)
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
func (s *DynamicService) AdminListDynamics(ctx context.Context, q string, page, pageSize int) (*AdminListDynamicsResult, error) {
	dbq := s.db.WithContext(ctx).Model(&dynamic.UserDynamic{})
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
func (s *DynamicService) AdminDeleteDynamicCascade(ctx context.Context, id uint64, fn func(tx *gorm.DB) error) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(tx)
	})
}
