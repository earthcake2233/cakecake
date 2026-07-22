//go:build integration

package dailyreward

import (
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"minibili/internal/model"
)

func setupDailyRewardDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.UserDailyTask{}, &model.CoinLedger{}, &model.VideoCoin{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func createDailyUser(t *testing.T, db *gorm.DB, id uint64) {
	t.Helper()
	u := model.User{
		ID:               id,
		Username:         "dailyuser",
		PasswordHash:     "hash",
		CoinBalanceTenths: 230,
		Experience:       0,
	}
	if err := db.Create(&u).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
}

func TestMarkLogin_Integration(t *testing.T) {
	db := setupDailyRewardDB(t)
	createDailyUser(t, db, 1)

	err := MarkLogin(db, 1)
	if err != nil {
		t.Fatalf("MarkLogin: %v", err)
	}

	// Check user daily task
	var task model.UserDailyTask
	if err := db.Where("user_id = ?", 1).First(&task).Error; err != nil {
		t.Fatalf("find task: %v", err)
	}
	if !task.LoginDone {
		t.Error("LoginDone should be true")
	}

	// Check experience added
	var u model.User
	db.First(&u, 1)
	if u.Experience != ExpLogin {
		t.Errorf("Experience = %d, want %d", u.Experience, ExpLogin)
	}

	// Check coin added
	if u.CoinBalanceTenths != 240 {
		t.Errorf("CoinBalanceTenths = %d, want 240", u.CoinBalanceTenths)
	}

	// Check ledger
	var ledgerCount int64
	db.Model(&model.CoinLedger{}).Where("user_id = ?", 1).Count(&ledgerCount)
	if ledgerCount != 1 {
		t.Errorf("expected 1 ledger row, got %d", ledgerCount)
	}
}

func TestMarkLogin_Idempotent(t *testing.T) {
	db := setupDailyRewardDB(t)
	createDailyUser(t, db, 1)

	// First call
	if err := MarkLogin(db, 1); err != nil {
		t.Fatalf("first MarkLogin: %v", err)
	}

	var u1 model.User
	db.First(&u1, 1)
	exp1 := u1.Experience
	coin1 := u1.CoinBalanceTenths

	// Second call should be no-op
	if err := MarkLogin(db, 1); err != nil {
		t.Fatalf("second MarkLogin: %v", err)
	}

	var u2 model.User
	db.First(&u2, 1)
	if u2.Experience != exp1 {
		t.Errorf("experience changed from %d to %d", exp1, u2.Experience)
	}
	if u2.CoinBalanceTenths != coin1 {
		t.Errorf("coins changed from %d to %d", coin1, u2.CoinBalanceTenths)
	}
}

func TestMarkWatch_Integration(t *testing.T) {
	db := setupDailyRewardDB(t)
	createDailyUser(t, db, 1)

	err := MarkWatch(db, 1)
	if err != nil {
		t.Fatalf("MarkWatch: %v", err)
	}

	var task model.UserDailyTask
	if err := db.Where("user_id = ?", 1).First(&task).Error; err != nil {
		t.Fatalf("find task: %v", err)
	}
	if !task.WatchDone {
		t.Error("WatchDone should be true")
	}

	var u model.User
	db.First(&u, 1)
	if u.Experience != ExpWatch {
		t.Errorf("Experience = %d, want %d", u.Experience, ExpWatch)
	}
}

func TestMarkWatch_Idempotent(t *testing.T) {
	db := setupDailyRewardDB(t)
	createDailyUser(t, db, 1)

	if err := MarkWatch(db, 1); err != nil {
		t.Fatalf("first MarkWatch: %v", err)
	}
	var u1 model.User
	db.First(&u1, 1)

	if err := MarkWatch(db, 1); err != nil {
		t.Fatalf("second MarkWatch: %v", err)
	}
	var u2 model.User
	db.First(&u2, 1)
	if u2.Experience != u1.Experience {
		t.Errorf("experience changed from %d to %d", u1.Experience, u2.Experience)
	}
}

func TestBuildSnapshot_Integration(t *testing.T) {
	db := setupDailyRewardDB(t)
	createDailyUser(t, db, 1)

	// Mark login first
	if err := MarkLogin(db, 1); err != nil {
		t.Fatalf("MarkLogin: %v", err)
	}

	snapshot, err := BuildSnapshot(db, 1)
	if err != nil {
		t.Fatalf("BuildSnapshot: %v", err)
	}

	if !snapshot.Login.Done {
		t.Error("Login should be done")
	}
	if snapshot.Login.Exp != ExpLogin {
		t.Errorf("Login.Exp = %d, want %d", snapshot.Login.Exp, ExpLogin)
	}

	if snapshot.Watch.Done {
		t.Error("Watch should not be done yet")
	}

	if snapshot.Coin.Done {
		t.Error("Coin should not be done with 0 progress")
	}
	if snapshot.Coin.Progress != 0 {
		t.Errorf("Coin.Progress = %d, want 0", snapshot.Coin.Progress)
	}
	if snapshot.Coin.Max != ExpCoinMax {
		t.Errorf("Coin.Max = %d, want %d", snapshot.Coin.Max, ExpCoinMax)
	}

	// Share is always not done
	if snapshot.Share.Done {
		t.Error("Share should not be done")
	}
}

func TestBuildSnapshot_FreshUser(t *testing.T) {
	db := setupDailyRewardDB(t)
	createDailyUser(t, db, 1)

	snapshot, err := BuildSnapshot(db, 1)
	if err != nil {
		t.Fatalf("BuildSnapshot: %v", err)
	}

	if snapshot.Login.Done {
		t.Error("Login should not be done for fresh user")
	}
	if snapshot.Watch.Done {
		t.Error("Watch should not be done for fresh user")
	}
	if snapshot.Share.Done {
		t.Error("Share should not be done")
	}
}

func TestBuildSnapshot_WithCoinProgress(t *testing.T) {
	db := setupDailyRewardDB(t)
	createDailyUser(t, db, 1)

	// Add a video coin record for today
	vc := model.VideoCoin{
		UserID:    1,
		VideoID:   100,
		Amount:    2,
		CreatedAt: time.Now(),
	}
	if err := db.Create(&vc).Error; err != nil {
		t.Fatalf("create VideoCoin: %v", err)
	}

	snapshot, err := BuildSnapshot(db, 1)
	if err != nil {
		t.Fatalf("BuildSnapshot: %v", err)
	}

	// 2 coins * ExpPerCoinUnit = 20 progress
	expectedProgress := 2 * ExpPerCoinUnit
	if snapshot.Coin.Progress != expectedProgress {
		t.Errorf("Coin.Progress = %d, want %d", snapshot.Coin.Progress, expectedProgress)
	}
}

func TestCoinProgress_Integration(t *testing.T) {
	db := setupDailyRewardDB(t)
	createDailyUser(t, db, 1)

	// No coins yet
	if p := CoinProgress(db, 1); p != 0 {
		t.Errorf("CoinProgress = %d, want 0", p)
	}

	// Add coins
	vc := model.VideoCoin{
		UserID:    1,
		VideoID:   100,
		Amount:    1,
		CreatedAt: time.Now(),
	}
	db.Create(&vc)

	p := CoinProgress(db, 1)
	if p != ExpPerCoinUnit {
		t.Errorf("CoinProgress = %d, want %d", p, ExpPerCoinUnit)
	}
}

func TestCoinProgress_Capped(t *testing.T) {
	db := setupDailyRewardDB(t)
	createDailyUser(t, db, 1)

	// Add coins that would exceed max
	for i := uint64(1); i <= 10; i++ {
		vc := model.VideoCoin{
			UserID:    1,
			VideoID:   i,
			Amount:    2,
			CreatedAt: time.Now(),
		}
		db.Create(&vc)
	}

	p := CoinProgress(db, 1)
	if p > ExpCoinMax {
		t.Errorf("CoinProgress = %d, should be capped at %d", p, ExpCoinMax)
	}
	if p != ExpCoinMax {
		t.Errorf("CoinProgress = %d, want %d", p, ExpCoinMax)
	}
}
