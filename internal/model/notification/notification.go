package notification

import "time"

// Notification is an in-app notification row.
type Notification struct {
	ID              uint64 `gorm:"primaryKey"`
	RecipientID     uint64 `gorm:"index:idx_notif_recipient;not null"`
	Type            string `gorm:"size:48;index;not null"`
	RelatedID       uint64 `gorm:"index"`
	SenderNamesJSON string `gorm:"type:text"`
	TotalLikes      int    `gorm:"default:0"`
	CommentPreview  string `gorm:"size:32"`
	// PayloadJSON holds type-specific fields (e.g. reply_received: sender, reply body, video_id).
	PayloadJSON string `gorm:"type:text"`
	IsRead      bool   `gorm:"index"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// LikeNotifMute silences like notifications for a comment.
type LikeNotifMute struct {
	RecipientID uint64 `gorm:"uniqueIndex:idx_like_notif_mute_pair;not null"`
	CommentID   uint64 `gorm:"uniqueIndex:idx_like_notif_mute_pair;not null"`
	CreatedAt   time.Time
}

// TableName returns the like_notif_mutes table name.
func (LikeNotifMute) TableName() string { return "like_notif_mutes" }
