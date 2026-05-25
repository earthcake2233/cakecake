package model

import "time"

// VideoViewHistory stores a user's watch progress for the account history page.
type VideoViewHistory struct {
	ID          uint64  `gorm:"primaryKey"`
	UserID      uint64  `gorm:"uniqueIndex:uk_view_hist_user_video,priority:1;not null"`
	VideoID     uint64  `gorm:"uniqueIndex:uk_view_hist_user_video,priority:2;not null"`
	ProgressSec float64 `gorm:"not null;default:0"`
	DurationSec float64 `gorm:"not null;default:0"`
	Device      string  `gorm:"size:16;not null;default:web"` // web | mobile
	ViewedAt    time.Time `gorm:"index:idx_view_hist_user_viewed,priority:2"`
	UpdatedAt   time.Time
}

// ArticleViewHistory stores a user's read history for columns (专栏).
type ArticleViewHistory struct {
	ID        uint64 `gorm:"primaryKey"`
	UserID    uint64 `gorm:"uniqueIndex:uk_view_hist_user_article,priority:1;not null"`
	ArticleID uint64 `gorm:"uniqueIndex:uk_view_hist_user_article,priority:2;not null"`
	Device    string `gorm:"size:16;not null;default:web"` // web | mobile
	ViewedAt  time.Time `gorm:"index:idx_view_hist_art_user_viewed,priority:2"`
	UpdatedAt time.Time
}
