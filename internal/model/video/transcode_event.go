package video

import "time"

// TranscodeEvent is an append-only audit trail of transcode state changes:
// who/what moved a video from one status to another, when, and why.
type TranscodeEvent struct {
	ID         uint64 `gorm:"primaryKey"`
	VideoID    uint64 `gorm:"not null;index"`
	JobID      string `gorm:"size:64;not null;default:''"`
	FromStatus string `gorm:"size:32;not null;default:''"`
	ToStatus   string `gorm:"size:32;not null;default:''"`
	Reason     string `gorm:"size:1900;not null;default:''"`
	CreatedAt  time.Time
}

// TableName pins the name to the goose migration (migrations/00011).
func (TranscodeEvent) TableName() string {
	return "transcode_events"
}
