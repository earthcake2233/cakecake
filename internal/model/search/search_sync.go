package search

import "time"

// SearchSyncJob is the outbox-style pending table for Elasticsearch sync.
// Business writes enqueue a row; a background worker executes it with
// exponential backoff, so per-item failures are retried instead of relying
// only on startup ReindexAll.
type SearchSyncJob struct {
	ID          uint64     `gorm:"primaryKey"`
	EntityType  string     `gorm:"size:16;not null;index"` // video | article | user
	EntityID    uint64     `gorm:"not null;index"`
	Action      string     `gorm:"size:8;not null"`                        // upsert | delete
	Status      string     `gorm:"size:16;not null;default:pending;index"` // pending | done
	Attempts    int        `gorm:"not null;default:0"`
	NextRetryAt *time.Time `gorm:"index"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// TableName pins the name to the goose migration (migrations/00014).
func (SearchSyncJob) TableName() string {
	return "search_sync_outbox"
}
