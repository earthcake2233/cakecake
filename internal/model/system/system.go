package system

import "time"

// SystemConfig stores runtime-tunable key-value configuration.
// Keys are admin-managed operational parameters (Agent + RateLimit).
// Values are plain strings; typed accessors live in config.RuntimeConfig.
type SystemConfig struct {
	Key       string `gorm:"primaryKey;size:64"`
	Value     string `gorm:"size:1024;not null"`
	UpdatedAt time.Time
}

// SchemaVersion records which DB migrations have been executed.
type SchemaVersion struct {
	ID          uint64    `gorm:"primaryKey"`
	Version     int       `gorm:"uniqueIndex;not null"`
	Name        string    `gorm:"size:128;not null"`
	Description string    `gorm:"size:512"`
	ExecutedAt  time.Time `gorm:"not null"`
}
