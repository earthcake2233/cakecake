package data

import (
	"fmt"

	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// NewDB opens MySQL and runs AutoMigrate (Skill S-002).
// NewDB opens MySQL and runs migrations.
// When autoMigrate is true (default), runs AutoMigrateAll (Go-based V1-V19).
// When autoMigrate is false (production), skips Go-based migrations and runs
// only goose SQL migrations (V20+) from the migrations/ directory.
// For a brand-new production database, run once with autoMigrate=true first.
func NewDB(dsn string, lg *zap.Logger, autoMigrate bool) (*gorm.DB, error) {
	if dsn == "" {
		return nil, fmt.Errorf("MYSQL_DSN is empty")
	}
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Warn),
	})
	if err != nil {
		return nil, err
	}
	if autoMigrate {
		if err := AutoMigrateAll(db, lg); err != nil {
			return nil, err
		}
	} else {
		if lg != nil {
			lg.Info("DB_AUTO_MIGRATE disabled, running goose SQL migrations only")
		}
		sqlDB, err := db.DB()
		if err != nil {
			return nil, fmt.Errorf("get underlying sql.DB: %w", err)
		}
		if err := RunGooseMigrations(sqlDB, "migrations", lg); err != nil {
			return nil, fmt.Errorf("goose migrations: %w", err)
		}
	}
	return db, nil
}
