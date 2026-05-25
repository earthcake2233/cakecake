package data

import (
	"fmt"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"minibili/internal/model"
	"minibili/internal/pkg/searchhist"
)

// migrateUserSearchHistory applies schema, dedupes rows, then adds the unique (user_id, keyword_norm) index.
func migrateUserSearchHistory(db *gorm.DB, lg *zap.Logger) error {
	m := db.Migrator()
	if m.HasTable(&model.UserSearchHistory{}) {
		for _, name := range []string{"idx_user_search_kw", "idx_user_search_norm"} {
			if !m.HasIndex(&model.UserSearchHistory{}, name) {
				continue
			}
			if err := dropSearchHistoryIndex(db, m, name); err != nil {
				return err
			}
			if lg != nil {
				lg.Info("dropped search history index before migrate", zap.String("index", name))
			}
		}
	}
	if err := db.AutoMigrate(&model.UserSearchHistory{}); err != nil {
		return err
	}
	for _, name := range []string{"idx_user_search_kw", "idx_user_search_norm"} {
		if !m.HasIndex(&model.UserSearchHistory{}, name) {
			continue
		}
		if err := dropSearchHistoryIndex(db, m, name); err != nil {
			return err
		}
		if lg != nil {
			lg.Info("dropped search history index before rebuild", zap.String("index", name))
		}
	}
	if err := CleanupUserSearchHistory(db, lg); err != nil {
		return err
	}
	if m.HasIndex(&model.UserSearchHistory{}, "idx_user_search_norm") {
		return nil
	}
	switch db.Dialector.Name() {
	case "mysql":
		return db.Exec(
			"CREATE UNIQUE INDEX idx_user_search_norm ON user_search_histories (user_id, keyword_norm)",
		).Error
	case "sqlite":
		return db.Exec(
			"CREATE UNIQUE INDEX IF NOT EXISTS idx_user_search_norm ON user_search_histories (user_id, keyword_norm)",
		).Error
	default:
		return db.Exec(
			"CREATE UNIQUE INDEX idx_user_search_norm ON user_search_histories (user_id, keyword_norm)",
		).Error
	}
}

func dropSearchHistoryIndex(db *gorm.DB, m gorm.Migrator, name string) error {
	if err := m.DropIndex(&model.UserSearchHistory{}, name); err != nil {
		if db.Dialector.Name() == "mysql" {
			return db.Exec("ALTER TABLE user_search_histories DROP INDEX " + name).Error
		}
		return err
	}
	return nil
}

// CleanupUserSearchHistory backfills keyword_norm and removes duplicate/invalid rows per user.
func CleanupUserSearchHistory(db *gorm.DB, lg *zap.Logger) error {
	var rows []model.UserSearchHistory
	if err := db.Order("updated_at DESC, id DESC").Find(&rows).Error; err != nil {
		return err
	}
	seen := make(map[string]uint64)
	var deleteIDs []uint64
	for _, r := range rows {
		norm := searchhist.Norm(r.Keyword)
		if norm == "" {
			deleteIDs = append(deleteIDs, r.ID)
			continue
		}
		if r.KeywordNorm != norm {
			_ = db.Model(&r).Update("keyword_norm", norm).Error
		}
		key := fmt.Sprintf("%d:%s", r.UserID, norm)
		if _, dup := seen[key]; dup {
			deleteIDs = append(deleteIDs, r.ID)
			continue
		}
		seen[key] = r.ID
	}
	if len(deleteIDs) > 0 {
		if err := db.Where("id IN ?", deleteIDs).Delete(&model.UserSearchHistory{}).Error; err != nil {
			return err
		}
		if lg != nil {
			lg.Info("removed duplicate search history rows", zap.Int("count", len(deleteIDs)))
		}
	}
	return nil
}
