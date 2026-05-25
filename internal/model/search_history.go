package model

import "time"

// UserSearchHistory stores recent search keywords per user (server sync).
// Unique (user_id, keyword_norm) is created in data.migrateUserSearchHistory after backfill/dedup.
type UserSearchHistory struct {
	ID          uint64    `gorm:"primaryKey"`
	UserID      uint64    `gorm:"not null;index:idx_user_search_user"`
	Keyword     string    `gorm:"size:100;not null"`
	KeywordNorm string    `gorm:"size:100;not null"`
	UpdatedAt   time.Time `gorm:"not null;index:idx_user_search_updated"`
}
