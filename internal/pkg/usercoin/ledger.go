package usercoin

import (
	"time"

	"gorm.io/gorm"

	"minibili/internal/model"
)

const (
	ReasonLoginReward     = "login_reward"
	ReasonVideoTip        = "video_tip"
	ReasonVideoTipIncome  = "video_tip_income"
	ReasonArticleTip      = "article_tip"
	ReasonArticleTipIncome = "article_tip_income"
	ReasonNicknameChange  = "nickname_change"
)

// RecordLedger appends a ledger row (call inside the same transaction as balance change).
func RecordLedger(tx *gorm.DB, uid uint64, deltaTenths int64, reasonType string, videoID uint64) error {
	return RecordLedgerAt(tx, uid, deltaTenths, reasonType, videoID, time.Now())
}

// RecordLedgerAt is like RecordLedger but sets CreatedAt (for backfill).
func RecordLedgerAt(tx *gorm.DB, uid uint64, deltaTenths int64, reasonType string, videoID uint64, at time.Time) error {
	if deltaTenths == 0 {
		return nil
	}
	row := model.CoinLedger{
		UserID:      uid,
		DeltaTenths: deltaTenths,
		ReasonType:  reasonType,
		VideoID:     videoID,
		CreatedAt:   at,
	}
	return tx.Create(&row).Error
}
