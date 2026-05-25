package model

import "time"

// UserDailyTask tracks per-day completion of personal-center daily rewards (Asia/Shanghai calendar day).
type UserDailyTask struct {
	ID        uint64 `gorm:"primaryKey"`
	UserID    uint64 `gorm:"uniqueIndex:uk_user_daily_task,priority:1;not null"`
	TaskDate  string `gorm:"size:10;uniqueIndex:uk_user_daily_task,priority:2;not null"` // YYYY-MM-DD
	LoginDone bool   `gorm:"not null;default:0"`
	WatchDone bool   `gorm:"not null;default:0"`
	CreatedAt time.Time
	UpdatedAt time.Time
}
