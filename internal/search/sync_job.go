package search

import (
	"context"
	"time"

	searchmodel "cakecake/internal/model/search"
	"gorm.io/gorm"
)

const (
	SyncJobEntityVideo   = "video"
	SyncJobEntityArticle = "article"
	SyncJobEntityUser    = "user"

	SyncJobActionUpsert = "upsert"
	SyncJobActionDelete = "delete"

	SyncJobStatusPending = "pending"
	SyncJobStatusDone    = "done"
)

// EnqueueSyncJob records a pending ES sync job. An existing pending row for
// the same entity+action is reset (attempts=0, due immediately) instead of
// duplicating it, so frequent updates collapse into one retryable job.
func EnqueueSyncJob(ctx context.Context, db *gorm.DB, entityType string, entityID uint64, action string) error {
	if db == nil {
		return nil
	}
	now := time.Now()
	var existing searchmodel.SearchSyncJob
	err := db.WithContext(ctx).
		Where("entity_type = ? AND entity_id = ? AND action = ? AND status = ?",
			entityType, entityID, action, SyncJobStatusPending).
		First(&existing).Error
	if err == nil {
		return db.Model(&existing).Updates(map[string]any{
			"attempts":      0,
			"next_retry_at": now,
			"updated_at":    now,
		}).Error
	}
	if err != gorm.ErrRecordNotFound {
		return err
	}
	return db.WithContext(ctx).Create(&searchmodel.SearchSyncJob{
		EntityType: entityType,
		EntityID:   entityID,
		Action:     action,
		Status:     SyncJobStatusPending,
	}).Error
}

// SyncExecutor executes one queued sync job.
type SyncExecutor interface {
	Exec(ctx context.Context, entityType string, entityID uint64, action string) error
}

// ESClientExecutor adapts the ES client + DB to SyncExecutor.
type ESClientExecutor struct {
	Client *Client
	DB     *gorm.DB
}

func (e *ESClientExecutor) Exec(ctx context.Context, entityType string, entityID uint64, action string) error {
	if e == nil || e.Client == nil || !e.Client.Enabled() {
		return nil
	}
	switch entityType {
	case SyncJobEntityVideo:
		if action == SyncJobActionDelete {
			return e.Client.DeleteVideo(ctx, entityID)
		}
		return e.Client.IndexVideoFromDB(ctx, e.DB, entityID)
	case SyncJobEntityArticle:
		if action == SyncJobActionDelete {
			return e.Client.DeleteArticle(ctx, entityID)
		}
		return e.Client.IndexArticleFromDB(ctx, e.DB, entityID)
	case SyncJobEntityUser:
		// IndexUserFromDB removes docs for anonymized users, so a user
		// "delete" is handled by upserting the anonymized row.
		return e.Client.IndexUserFromDB(ctx, e.DB, entityID)
	}
	return nil
}

// ProcessDueSyncJobs runs one batch of due pending jobs: successes are marked
// done, failures keep the row pending with exponential backoff. Returns the
// number of rows handled.
func ProcessDueSyncJobs(ctx context.Context, db *gorm.DB, exec SyncExecutor, batch int, now time.Time, backoff func(attempts int) time.Duration) (int, error) {
	if db == nil || exec == nil {
		return 0, nil
	}
	var rows []searchmodel.SearchSyncJob
	if err := db.WithContext(ctx).
		Where("status = ? AND (next_retry_at IS NULL OR next_retry_at <= ?)", SyncJobStatusPending, now).
		Order("id ASC").
		Limit(batch).
		Find(&rows).Error; err != nil {
		return 0, err
	}
	handled := 0
	for i := range rows {
		row := rows[i]
		handled++
		if err := exec.Exec(ctx, row.EntityType, row.EntityID, row.Action); err != nil {
			attempts := row.Attempts + 1
			next := now.Add(backoff(attempts))
			_ = db.Model(&searchmodel.SearchSyncJob{}).
				Where("id = ? AND status = ?", row.ID, SyncJobStatusPending).
				Updates(map[string]any{"attempts": attempts, "next_retry_at": next}).Error
			continue
		}
		_ = db.Model(&searchmodel.SearchSyncJob{}).
			Where("id = ? AND status = ?", row.ID, SyncJobStatusPending).
			Update("status", SyncJobStatusDone).Error
	}
	return handled, nil
}

// DefaultSyncBackoff is 10s doubling, capped at 5 minutes.
func DefaultSyncBackoff(attempts int) time.Duration {
	b := 10 * time.Second
	for i := 1; i < attempts; i++ {
		b *= 2
		if b >= 5*time.Minute {
			return 5 * time.Minute
		}
	}
	return b
}
