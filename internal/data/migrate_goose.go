package data

import (
	"context"
	"database/sql"
	"fmt"
	"os"

	"github.com/pressly/goose/v3"
	"go.uber.org/zap"
)

// RunGooseMigrations applies SQL-based migrations (V20+) from the given
// directory using goose. The baseline (V1-V19) must already be applied
// via AutoMigrateAll / RunVersionedMigrations.
//
// Set GOOSE_DRIVER to "mysql" or "sqlite3" depending on the database.
func RunGooseMigrations(db *sql.DB, dir string, lg *zap.Logger) error {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if lg != nil {
			lg.Warn("goose migrations directory not found, skipping", zap.String("dir", dir))
		}
		return nil
	}

	provider, err := goose.NewProvider(
		goose.DialectMySQL,
		db,
		os.DirFS(dir),
	)
	if err != nil {
		return fmt.Errorf("goose provider: %w", err)
	}

	results, err := provider.Up(context.Background())
	if err != nil {
		return fmt.Errorf("goose up: %w", err)
	}

	if lg != nil {
		for _, r := range results {
			lg.Info("goose migration applied",
				zap.Int64("version", r.Source.Version),
				zap.String("path", r.Source.Path),
				zap.Duration("duration", r.Duration),
			)
		}
		lg.Info("goose migrations complete", zap.Int("applied", len(results)))
	}
	return nil
}
