//go:build integration

package usercoin

import (
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"minibili/internal/model"
)

func setupUserCoinDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.CoinLedger{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func createTestUser(t *testing.T, db *gorm.DB, id uint64, tenths int64) {
	t.Helper()
	u := model.User{
		ID:               id,
		Username:         "testuser",
		PasswordHash:     "hash",
		CoinBalanceTenths: tenths,
	}
	if err := db.Create(&u).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
}

func TestRecordLedger_Integration(t *testing.T) {
	db := setupUserCoinDB(t)
	createTestUser(t, db, 1, 100)

	err := RecordLedger(db, 1, 10, ReasonLoginReward, 0)
	if err != nil {
		t.Fatalf("RecordLedger: %v", err)
	}

	var ledger model.CoinLedger
	if err := db.First(&ledger, "user_id = ?", 1).Error; err != nil {
		t.Fatalf("find ledger: %v", err)
	}
	if ledger.DeltaTenths != 10 {
		t.Errorf("DeltaTenths = %d, want 10", ledger.DeltaTenths)
	}
	if ledger.ReasonType != ReasonLoginReward {
		t.Errorf("ReasonType = %q, want %q", ledger.ReasonType, ReasonLoginReward)
	}
	if ledger.UserID != 1 {
		t.Errorf("UserID = %d, want 1", ledger.UserID)
	}
}

func TestRecordLedger_ZeroDelta(t *testing.T) {
	db := setupUserCoinDB(t)
	createTestUser(t, db, 1, 100)
	// zero delta should be no-op
	err := RecordLedger(db, 1, 0, ReasonLoginReward, 0)
	if err != nil {
		t.Fatalf("RecordLedger with zero delta: %v", err)
	}
	var count int64
	db.Model(&model.CoinLedger{}).Where("user_id = ?", 1).Count(&count)
	if count != 0 {
		t.Errorf("expected 0 ledger rows, got %d", count)
	}
}

func TestRecordLedger_NegativeDelta(t *testing.T) {
	db := setupUserCoinDB(t)
	createTestUser(t, db, 1, 100)

	err := RecordLedger(db, 1, -20, ReasonNicknameChange, 0)
	if err != nil {
		t.Fatalf("RecordLedger negative: %v", err)
	}

	var ledger model.CoinLedger
	if err := db.First(&ledger, "user_id = ?", 1).Error; err != nil {
		t.Fatalf("find ledger: %v", err)
	}
	if ledger.DeltaTenths != -20 {
		t.Errorf("DeltaTenths = %d, want -20", ledger.DeltaTenths)
	}
}

func TestAddTenths_Integration(t *testing.T) {
	db := setupUserCoinDB(t)
	createTestUser(t, db, 1, 100)

	err := AddTenths(db, 1, 50)
	if err != nil {
		t.Fatalf("AddTenths: %v", err)
	}

	var u model.User
	if err := db.First(&u, 1).Error; err != nil {
		t.Fatalf("find user: %v", err)
	}
	if u.CoinBalanceTenths != 150 {
		t.Errorf("CoinBalanceTenths = %d, want 150", u.CoinBalanceTenths)
	}
}

func TestAddTenths_ZeroDelta(t *testing.T) {
	db := setupUserCoinDB(t)
	createTestUser(t, db, 1, 100)

	err := AddTenths(db, 1, 0)
	if err != nil {
		t.Fatalf("AddTenths(0): %v", err)
	}
	var u model.User
	db.First(&u, 1)
	if u.CoinBalanceTenths != 100 {
		t.Errorf("should not change: got %d", u.CoinBalanceTenths)
	}
}

func TestAddTenths_NegativeDelta(t *testing.T) {
	db := setupUserCoinDB(t)
	createTestUser(t, db, 1, 100)

	// negative delta should be ignored (delta <= 0)
	err := AddTenths(db, 1, -10)
	if err != nil {
		t.Fatalf("AddTenths(-10): %v", err)
	}
	var u model.User
	db.First(&u, 1)
	if u.CoinBalanceTenths != 100 {
		t.Errorf("should not change for negative: got %d", u.CoinBalanceTenths)
	}
}

func TestSpendWholeCoins_Integration(t *testing.T) {
	db := setupUserCoinDB(t)
	// 23.0 coins = 230 tenths
	createTestUser(t, db, 1, 230)

	err := SpendWholeCoins(db, 1, 6)
	if err != nil {
		t.Fatalf("SpendWholeCoins: %v", err)
	}

	var u model.User
	if err := db.First(&u, 1).Error; err != nil {
		t.Fatalf("find user: %v", err)
	}
	// 230 - 60 = 170 tenths
	if u.CoinBalanceTenths != 170 {
		t.Errorf("CoinBalanceTenths = %d, want 170", u.CoinBalanceTenths)
	}

	// Check ledger
	var ledger model.CoinLedger
	if err := db.First(&ledger, "user_id = ?", 1).Error; err != nil {
		t.Fatalf("find ledger: %v", err)
	}
	if ledger.DeltaTenths != -60 {
		t.Errorf("DeltaTenths = %d, want -60", ledger.DeltaTenths)
	}
}

func TestSpendWholeCoins_Insufficient(t *testing.T) {
	db := setupUserCoinDB(t)
	createTestUser(t, db, 1, 30) // only 3 coins

	err := SpendWholeCoins(db, 1, 6)
	if err != ErrInsufficientCoins {
		t.Fatalf("expected ErrInsufficientCoins, got %v", err)
	}
}

func TestSpendWholeCoins_ZeroAmount(t *testing.T) {
	db := setupUserCoinDB(t)
	createTestUser(t, db, 1, 100)

	err := SpendWholeCoins(db, 1, 0)
	if err != nil {
		t.Fatalf("SpendWholeCoins(0): %v", err)
	}
	var u model.User
	db.First(&u, 1)
	if u.CoinBalanceTenths != 100 {
		t.Errorf("should not change: got %d", u.CoinBalanceTenths)
	}
}

func TestSpendWholeCoins_ExactBalance(t *testing.T) {
	db := setupUserCoinDB(t)
	createTestUser(t, db, 1, 60) // exactly 6 coins

	err := SpendWholeCoins(db, 1, 6)
	if err != nil {
		t.Fatalf("SpendWholeCoins exact: %v", err)
	}
	var u model.User
	db.First(&u, 1)
	if u.CoinBalanceTenths != 0 {
		t.Errorf("CoinBalanceTenths = %d, want 0", u.CoinBalanceTenths)
	}
}
