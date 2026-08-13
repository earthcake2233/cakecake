package video

import "time"

// DirectUploadClaim binds an OSS raw key to the user and video row created
// from it. The unique raw_key makes direct-upload submits idempotent: the
// same object cannot be submitted twice to create a second video.
type DirectUploadClaim struct {
	ID        uint64 `gorm:"primaryKey"`
	RawKey    string `gorm:"size:255;not null;uniqueIndex"`
	UserID    uint64 `gorm:"not null;index"`
	VideoID   uint64 `gorm:"not null;default:0"`
	CreatedAt time.Time
}
