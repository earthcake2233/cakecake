package video

import "time"

// TranscodeJobDedup makes at-least-once delivery idempotent per attempt: a
// (job_id, retry_count) pair is inserted before processing starts, so a
// redelivered message for an in-flight or finished attempt is skipped instead
// of re-transcoding.
type TranscodeJobDedup struct {
	JobID      string `gorm:"size:64;not null;primaryKey"`
	RetryCount int    `gorm:"not null;primaryKey"`
	VideoID    uint64 `gorm:"not null;index"`
	CreatedAt  time.Time
}

// TableName pins the name to the goose migration (migrations/00010).
func (TranscodeJobDedup) TableName() string {
	return "transcode_job_dedup"
}
