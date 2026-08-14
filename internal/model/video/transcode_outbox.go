package video

import "time"

// TranscodeOutbox is the local message table that makes "create video row +
// publish transcode job" atomic: the job is written in the same DB
// transaction as the business change, and a relay publishes it to RabbitMQ
// with publisher confirm, marking it sent only after confirmation.
type TranscodeOutbox struct {
	ID       uint64 `gorm:"primaryKey"`
	JobID    string `gorm:"size:64;not null;uniqueIndex"`
	VideoID  uint64 `gorm:"not null;index"`
	Payload  string `gorm:"type:text;not null"`
	Status   string `gorm:"size:16;not null;default:'pending';index"`
	Attempts int    `gorm:"not null;default:0"`
	// NextRetryAt is NULL for a fresh row; MySQL strict mode rejects the zero
	// datetime, so a pointer keeps new pending rows insertable.
	NextRetryAt *time.Time
	CreatedAt   time.Time
	SentAt      *time.Time
}

// TableName pins the name to the goose migration (migrations/00009) instead
// of GORM's default pluralization, keeping both migration tracks identical.
func (TranscodeOutbox) TableName() string {
	return "transcode_outbox"
}

// Outbox statuses.
const (
	OutboxStatusPending = "pending"
	OutboxStatusSent    = "sent"
)
