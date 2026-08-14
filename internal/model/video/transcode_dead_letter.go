package video

import "time"

// TranscodeDeadLetter records a transcode job whose retries were exhausted.
// It is the audit/compensation trail behind the RabbitMQ dead-letter queue.
type TranscodeDeadLetter struct {
	ID          uint64 `gorm:"primaryKey"`
	VideoID     uint64 `gorm:"not null;index"`
	Reason      string `gorm:"size:1900;not null;default:''"`
	RetryCount  int    `gorm:"type:bigint;not null;default:0"`
	PayloadJSON string `gorm:"type:text"`
	CreatedAt   time.Time
	// ProcessedAt is set when the dead-letter consumer acks a message.
	ProcessedAt *time.Time
	// RequeuedAt / RequeuedCount track manual compensation via the admin API.
	RequeuedAt    *time.Time
	RequeuedCount int `gorm:"type:bigint;not null;default:0"`
	// AutoRetryCount / LastAutoRetryAt track automatic requeue of transient
	// failures (the dead-letter auto-retry loop), separate from manual replay.
	AutoRetryCount  int `gorm:"not null;default:0"`
	LastAutoRetryAt *time.Time
	// ArchivedAt is set by the retention job instead of physically deleting
	// the audit row: dead letters are archived, never silently destroyed.
	ArchivedAt *time.Time
}
