package worker

import (
	"context"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"cakecake/internal/search"
)

const (
	searchSyncRetryInterval = 5 * time.Second
	searchSyncRetryBatch    = 50
)

// StartSearchSyncRetry periodically executes pending Elasticsearch sync jobs
// with exponential backoff (outbox-style per-item retry for ES sync).
func StartSearchSyncRetry(ctx context.Context, db *gorm.DB, exec search.SyncExecutor, lg *zap.Logger) {
	if db == nil || exec == nil {
		return
	}
	t := time.NewTicker(searchSyncRetryInterval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			handled, err := search.ProcessDueSyncJobs(
				ctx, db, exec, searchSyncRetryBatch, time.Now(), search.DefaultSyncBackoff)
			if err != nil && lg != nil {
				lg.Warn("search sync retry", zap.Error(err))
			}
			if handled > 0 && lg != nil {
				lg.Debug("search sync retry handled", zap.Int("handled", handled))
			}
		}
	}
}
