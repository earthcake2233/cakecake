package model

import "time"

// CoinLedger records a single coin balance change for the personal-center ledger.
type CoinLedger struct {
	ID          uint64 `gorm:"primaryKey"`
	UserID      uint64 `gorm:"index:idx_coin_ledger_user_created,priority:1;not null"`
	DeltaTenths int64  `gorm:"not null"`
	ReasonType  string `gorm:"size:32;not null;index"`
	VideoID     uint64 `gorm:"index;not null;default:0"`
	CreatedAt   time.Time `gorm:"index:idx_coin_ledger_user_created,priority:2"`
}
