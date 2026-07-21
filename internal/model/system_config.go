package model

import "time"

// SystemConfig stores runtime-tunable key-value configuration.
// Keys are admin-managed operational parameters (Agent + RateLimit).
// Values are plain strings; typed accessors live in config.RuntimeConfig.
type SystemConfig struct {
	Key       string    `gorm:"primaryKey;size:64"`
	Value     string    `gorm:"size:1024;not null"`
	UpdatedAt time.Time
}
