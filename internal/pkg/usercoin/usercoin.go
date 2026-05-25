package usercoin

import (
	"errors"

	"gorm.io/gorm"

	"minibili/internal/model"
)

const (
	// DefaultCoinTenths is the initial balance for new users (23.0 coins).
	DefaultCoinTenths int64 = 230
	// TenthsPerCoin: balance is stored in 0.1-coin units (10 tenths = 1 coin).
	TenthsPerCoin int64 = 10
	// DailyLoginCoinTenths is the daily login grant (1 coin).
	DailyLoginCoinTenths int64 = 10
	// NicknameChangeCostCoins is charged when the user changes nickname (personal center).
	NicknameChangeCostCoins = 6
)

// ErrInsufficientCoins is returned when the user cannot afford a coin spend.
var ErrInsufficientCoins = errors.New("insufficient coins")

// BalanceFloat converts stored tenths to API/display coins (e.g. 230 → 23.0).
func BalanceFloat(tenths int64) float64 {
	return float64(tenths) / float64(TenthsPerCoin)
}

// CostTenths returns the balance cost for spending `amount` whole coins (1 or 2).
func CostTenths(amount int) int64 {
	return int64(amount) * TenthsPerCoin
}

// CreatorShareTenths is 10% of coins spent by the viewer (0.1 coin per 1 coined, 0.2 per 2).
func CreatorShareTenths(amount int) int64 {
	return int64(amount)
}

// AddTenths credits coins to a user (daily login, UP share, etc.).
func AddTenths(db *gorm.DB, uid uint64, delta int64) error {
	if delta <= 0 {
		return nil
	}
	return db.Model(&model.User{}).Where("id = ?", uid).
		UpdateColumn("coin_balance_tenths", gorm.Expr("coin_balance_tenths + ?", delta)).Error
}

// GrantDailyLoginCoin adds 1 coin for the first login reward of the day.
func GrantDailyLoginCoin(db *gorm.DB, uid uint64) error {
	return db.Transaction(func(tx *gorm.DB) error {
		if err := AddTenths(tx, uid, DailyLoginCoinTenths); err != nil {
			return err
		}
		return RecordLedger(tx, uid, DailyLoginCoinTenths, ReasonLoginReward, 0)
	})
}

// SpendWholeCoins atomically deducts whole coins from a user (nickname change, etc.).
func SpendWholeCoins(tx *gorm.DB, uid uint64, wholeCoins int) error {
	if wholeCoins <= 0 {
		return nil
	}
	cost := CostTenths(wholeCoins)
	res := tx.Model(&model.User{}).
		Where("id = ? AND coin_balance_tenths >= ?", uid, cost).
		UpdateColumn("coin_balance_tenths", gorm.Expr("coin_balance_tenths - ?", cost))
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrInsufficientCoins
	}
	return RecordLedger(tx, uid, -cost, ReasonNicknameChange, 0)
}

// SpendOnVideoCoin atomically deducts viewer coins and credits the uploader 10% share.
func SpendOnVideoCoin(tx *gorm.DB, viewerID, uploaderID, videoID uint64, amount int) error {
	cost := CostTenths(amount)
	share := CreatorShareTenths(amount)

	res := tx.Model(&model.User{}).
		Where("id = ? AND coin_balance_tenths >= ?", viewerID, cost).
		UpdateColumn("coin_balance_tenths", gorm.Expr("coin_balance_tenths - ?", cost))
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrInsufficientCoins
	}
	if err := RecordLedger(tx, viewerID, -cost, ReasonVideoTip, videoID); err != nil {
		return err
	}
	if share > 0 {
		if err := tx.Model(&model.User{}).Where("id = ?", uploaderID).
			UpdateColumn("coin_balance_tenths", gorm.Expr("coin_balance_tenths + ?", share)).Error; err != nil {
			return err
		}
		return RecordLedger(tx, uploaderID, share, ReasonVideoTipIncome, videoID)
	}
	return nil
}

// SpendOnArticleCoin atomically deducts viewer coins and credits the author 10% share.
func SpendOnArticleCoin(tx *gorm.DB, viewerID, authorID, articleID uint64, amount int) error {
	cost := CostTenths(amount)
	share := CreatorShareTenths(amount)

	res := tx.Model(&model.User{}).
		Where("id = ? AND coin_balance_tenths >= ?", viewerID, cost).
		UpdateColumn("coin_balance_tenths", gorm.Expr("coin_balance_tenths - ?", cost))
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrInsufficientCoins
	}
	if err := RecordLedger(tx, viewerID, -cost, ReasonArticleTip, articleID); err != nil {
		return err
	}
	if share > 0 {
		if err := tx.Model(&model.User{}).Where("id = ?", authorID).
			UpdateColumn("coin_balance_tenths", gorm.Expr("coin_balance_tenths + ?", share)).Error; err != nil {
			return err
		}
		return RecordLedger(tx, authorID, share, ReasonArticleTipIncome, articleID)
	}
	return nil
}
