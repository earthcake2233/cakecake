// Package queryutil provides shared gorm query templates for service data
// providers. These helpers are the future internal/data query foundation:
// when providers move out of the service layer, this package moves with them.
package queryutil

import (
	"context"

	"gorm.io/gorm"
)

// FetchByIDs loads rows whose primary key is in ids and returns them keyed by
// the value produced by key (usually the row's ID field). Empty ids return an
// empty map without touching the database.
func FetchByIDs[T any](ctx context.Context, db *gorm.DB, ids []uint64, key func(T) uint64) (map[uint64]T, error) {
	result := make(map[uint64]T, len(ids))
	if len(ids) == 0 {
		return result, nil
	}
	var list []T
	if err := db.WithContext(ctx).Where("id IN ?", ids).Find(&list).Error; err != nil {
		return nil, err
	}
	for i := range list {
		result[key(list[i])] = list[i]
	}
	return result, nil
}

// FirstByID loads the row with the given primary key, returning nil on
// gorm.ErrRecordNotFound (which is left to the caller to translate).
func FirstByID[T any](ctx context.Context, db *gorm.DB, id uint64) (*T, error) {
	var out T
	if err := db.WithContext(ctx).First(&out, id).Error; err != nil {
		return nil, err
	}
	return &out, nil
}

// CountByParentID groups rows of model T by their parent_id column and returns
// the reply count per parent id. Errors are non-fatal: on failure an empty map
// is returned (matching the historical best-effort behavior of callers).
func CountByParentID[T any](ctx context.Context, db *gorm.DB, ids []uint64) map[uint64]uint64 {
	out := make(map[uint64]uint64, len(ids))
	if len(ids) == 0 {
		return out
	}
	type row struct {
		ParentID uint64
		C        int64
	}
	var rows []row
	_ = db.WithContext(ctx).Model(new(T)).
		Select("parent_id, COUNT(*) AS c").
		Where("parent_id IN ?", ids).
		Group("parent_id").
		Scan(&rows).Error
	for _, r := range rows {
		if r.C > 0 {
			out[r.ParentID] = uint64(r.C)
		}
	}
	return out
}

// ExistsByOwner reports whether a row of model T exists with the given id and
// user_id owner.
func ExistsByOwner[T any](ctx context.Context, db *gorm.DB, id, ownerID uint64) (bool, error) {
	var out T
	err := db.WithContext(ctx).Where("id = ? AND user_id = ?", id, ownerID).First(&out).Error
	if err == gorm.ErrRecordNotFound {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// Count returns the total row count for a query built by base. base is called
// once per query so the caller can build a fresh gorm statement each time.
func Count(base func() *gorm.DB) (int64, error) {
	var total int64
	if err := base().Count(&total).Error; err != nil {
		return 0, err
	}
	return total, nil
}

// FindPage runs base with the given ORDER BY and LIMIT/OFFSET paging. page is
// 1-based; pageSize is the max rows per page.
func FindPage[T any](base func() *gorm.DB, page, pageSize int, order string, out *[]T) error {
	offset := (page - 1) * pageSize
	if err := base().Order(order).Offset(offset).Limit(pageSize).Find(out).Error; err != nil {
		return err
	}
	return nil
}

// ListPage counts the total rows and fetches one page in a single call,
// returning the total count alongside the page rows.
func ListPage[T any](base func() *gorm.DB, page, pageSize int, order string, out *[]T) (int64, error) {
	total, err := Count(base)
	if err != nil {
		return 0, err
	}
	if err := FindPage(base, page, pageSize, order, out); err != nil {
		return 0, err
	}
	return total, nil
}
