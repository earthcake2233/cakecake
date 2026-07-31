package admin

import "time"

const StatusActive = "active"

type Admin struct {
	ID           uint64 `gorm:"primaryKey"`
	Username     string `gorm:"size:64;uniqueIndex;not null"`
	PasswordHash string `gorm:"size:128;not null"`
	DisplayName  string `gorm:"size:64"`
	Status       string `gorm:"size:16;not null;default:active;index"` // active | disabled
	LastLoginAt  *time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
type HomeBanner struct {
	ID         uint64 `gorm:"primaryKey"`
	Title      string `gorm:"size:120;not null"`
	ImageURL   string `gorm:"size:1024;not null"`
	LinkType   string `gorm:"size:16;not null;default:none"` // video | url | none
	LinkTarget string `gorm:"size:512"`                      // video id or external URL
	SortOrder  int    `gorm:"not null;default:0;index"`
	Enabled    bool   `gorm:"not null;default:1;index"`
	StartAt    *time.Time
	EndAt      *time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time
}
type HotSearchOp struct {
	ID           uint64 `gorm:"primaryKey"`
	OpType       string `gorm:"size:16;not null;index"`   // pin | block | manual
	Keyword      string `gorm:"size:100;not null"`        // match key (normalized on write)
	DisplayTitle string `gorm:"size:100"`                 //展示文案，空则用 keyword
	Badge        string `gorm:"size:8"`                   // 热 | 新 | 荐 | empty
	PinRank      int    `gorm:"not null;default:0;index"` // 1..N for pin/manual slot
	Enabled      bool   `gorm:"not null;default:1;index"`
	StartAt      *time.Time
	EndAt        *time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
type HotSearchDisplayLayout struct {
	ID        uint64 `gorm:"primaryKey"`
	OrderJSON string `gorm:"type:text;not null"`
	UpdatedAt time.Time
}
