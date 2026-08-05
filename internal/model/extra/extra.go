package extra

import "time"

// VideoViewHistory records a user's video viewing history.
type VideoViewHistory struct {
	ID          uint64    `gorm:"primaryKey"`
	UserID      uint64    `gorm:"uniqueIndex:uk_view_hist_user_video,priority:1;not null"`
	VideoID     uint64    `gorm:"uniqueIndex:uk_view_hist_user_video,priority:2;not null"`
	ProgressSec float64   `gorm:"not null;default:0"`
	DurationSec float64   `gorm:"not null;default:0"`
	Device      string    `gorm:"size:16;not null;default:web"` // web | mobile
	ViewedAt    time.Time `gorm:"index:idx_view_hist_user_viewed,priority:2"`
	UpdatedAt   time.Time
}

// ArticleViewHistory records a user's article viewing history.
type ArticleViewHistory struct {
	ID        uint64    `gorm:"primaryKey"`
	UserID    uint64    `gorm:"uniqueIndex:uk_view_hist_user_article,priority:1;not null"`
	ArticleID uint64    `gorm:"uniqueIndex:uk_view_hist_user_article,priority:2;not null"`
	Device    string    `gorm:"size:16;not null;default:web"` // web | mobile
	ViewedAt  time.Time `gorm:"index:idx_view_hist_art_user_viewed,priority:2"`
	UpdatedAt time.Time
}

// UserSearchHistory records a user's search keyword history.
type UserSearchHistory struct {
	ID          uint64    `gorm:"primaryKey"`
	UserID      uint64    `gorm:"not null;index:idx_user_search_user"`
	Keyword     string    `gorm:"size:100;not null"`
	KeywordNorm string    `gorm:"size:100;not null"`
	UpdatedAt   time.Time `gorm:"not null;index:idx_user_search_updated"`
}

// UserDailyTask tracks a user's daily reward task progress.
type UserDailyTask struct {
	ID        uint64 `gorm:"primaryKey"`
	UserID    uint64 `gorm:"uniqueIndex:uk_user_daily_task,priority:1;not null"`
	TaskDate  string `gorm:"size:10;uniqueIndex:uk_user_daily_task,priority:2;not null"` // YYYY-MM-DD
	LoginDone bool   `gorm:"not null;default:0"`
	WatchDone bool   `gorm:"not null;default:0"`
	CreatedAt time.Time
	UpdatedAt time.Time
}
