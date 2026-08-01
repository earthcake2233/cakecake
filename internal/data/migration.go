package data

import (
	"cakecake/internal/model/system"
	"fmt"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Migration is a single versioned schema/data migration.
type Migration struct {
	Version     int
	Name        string
	Description string
	Func        func(*gorm.DB, *zap.Logger) error
}

// RunVersionedMigrations creates the schema_versions table, then runs each
// registered migration in order, skipping versions that have already been
// recorded as executed. It is safe to call on every startup.
func RunVersionedMigrations(db *gorm.DB, lg *zap.Logger, migrations []Migration) error {
	if err := db.AutoMigrate(&system.SchemaVersion{}); err != nil {
		return fmt.Errorf("auto-migrate schema_versions: %w", err)
	}

	var done map[int]bool
	{
		var rows []system.SchemaVersion
		if err := db.Find(&rows).Error; err != nil {
			return fmt.Errorf("query schema_versions: %w", err)
		}
		done = make(map[int]bool, len(rows))
		for _, r := range rows {
			done[r.Version] = true
		}
	}

	for _, m := range migrations {
		if done[m.Version] {
			if lg != nil {
				lg.Debug("migration already applied, skip",
					zap.Int("version", m.Version),
					zap.String("name", m.Name),
				)
			}
			continue
		}

		if lg != nil {
			lg.Info("running migration",
				zap.Int("version", m.Version),
				zap.String("name", m.Name),
			)
		}

		if err := m.Func(db, lg); err != nil {
			return fmt.Errorf("migration v%d %s: %w", m.Version, m.Name, err)
		}

		rec := system.SchemaVersion{
			Version:     m.Version,
			Name:        m.Name,
			Description: m.Description,
			ExecutedAt:  time.Now(),
		}
		if err := db.Create(&rec).Error; err != nil {
			return fmt.Errorf("record migration v%d: %w", m.Version, err)
		}

		if lg != nil {
			lg.Info("migration applied",
				zap.Int("version", m.Version),
				zap.String("name", m.Name),
			)
		}
	}

	if lg != nil {
		lg.Info("versioned migrations complete")
	}
	return nil
}
