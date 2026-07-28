package model

import "time"

// SchemaVersion records which DB migrations have been executed.
type SchemaVersion struct {
	ID          uint64    `gorm:"primaryKey"`
	Version     int       `gorm:"uniqueIndex;not null"`
	Name        string    `gorm:"size:128;not null"`
	Description string    `gorm:"size:512"`
	ExecutedAt  time.Time `gorm:"not null"`
}
