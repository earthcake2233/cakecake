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
}
