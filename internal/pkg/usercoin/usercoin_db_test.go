package usercoin

import (
	"fmt"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"minibili/internal/model"
)

func setupUserCoinDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.CoinLedger{}))
	return db
}

func createTestUser(t *testing.T, db *gorm.DB, id uint64, tenths int64) {
	t.Helper()
	u := model.User{
		ID:               id,
		Username:         fmt.Sprintf("user_%d", id),
		PasswordHash:     "hash",
		CoinBalanceTenths: tenths,
	}
	require.NoError(t, db.Create(&u).Error)
	// GORM default:230 overrides zero values; re-set explicitly
	db.Model(&model.User{}).Where("id = ?", id).Update("coin_balance_tenths", tenths)
}

func TestAddTenths(t *testing.T) {
	db := setupUserCoinDB(t)
	createTestUser(t, db, 1, 100)

	err := AddTenths(db, 1, 50)
	require.NoError(t, err)

	var u model.User
	require.NoError(t, db.First(&u, 1).Error)
	require.Equal(t, int64(150), u.CoinBalanceTenths)
}

func TestAddTenths_ZeroDelta(t *testing.T) {
	db := setupUserCoinDB(t)
	createTestUser(t, db, 1, 100)

	err := AddTenths(db, 1, 0)
	require.NoError(t, err)
	var u model.User
	db.First(&u, 1)
	require.Equal(t, int64(100), u.CoinBalanceTenths)
}

func TestAddTenths_NegativeDelta(t *testing.T) {
	db := setupUserCoinDB(t)
	createTestUser(t, db, 1, 100)

	err := AddTenths(db, 1, -10)
	require.NoError(t, err)
	var u model.User
	db.First(&u, 1)
	require.Equal(t, int64(100), u.CoinBalanceTenths)
}

func TestSpendWholeCoins(t *testing.T) {
	db := setupUserCoinDB(t)
	createTestUser(t, db, 1, 230)

	err := SpendWholeCoins(db, 1, 6)
	require.NoError(t, err)

	var u model.User
	require.NoError(t, db.First(&u, 1).Error)
	require.Equal(t, int64(170), u.CoinBalanceTenths)

	var ledger model.CoinLedger
	require.NoError(t, db.First(&ledger, "user_id = ?", 1).Error)
	require.Equal(t, int64(-60), ledger.DeltaTenths)
}

func TestSpendWholeCoins_Insufficient(t *testing.T) {
	db := setupUserCoinDB(t)
	createTestUser(t, db, 1, 30)

	err := SpendWholeCoins(db, 1, 6)
	require.ErrorIs(t, err, ErrInsufficientCoins)
}

func TestSpendWholeCoins_ZeroAmount(t *testing.T) {
	db := setupUserCoinDB(t)
	createTestUser(t, db, 1, 100)

	err := SpendWholeCoins(db, 1, 0)
	require.NoError(t, err)
	var u model.User
	db.First(&u, 1)
	require.Equal(t, int64(100), u.CoinBalanceTenths)
}

func TestSpendWholeCoins_ExactBalance(t *testing.T) {
	db := setupUserCoinDB(t)
	createTestUser(t, db, 1, 60)

	err := SpendWholeCoins(db, 1, 6)
	require.NoError(t, err)
	var u model.User
	db.First(&u, 1)
	require.Equal(t, int64(0), u.CoinBalanceTenths)
}

func TestGrantDailyLoginCoin(t *testing.T) {
	db := setupUserCoinDB(t)
	createTestUser(t, db, 1, 100)

	err := GrantDailyLoginCoin(db, 1)
	require.NoError(t, err)

	var u model.User
	require.NoError(t, db.First(&u, 1).Error)
	require.Equal(t, int64(110), u.CoinBalanceTenths)

	var ledger model.CoinLedger
	require.NoError(t, db.First(&ledger, "user_id = ? AND reason_type = ?", 1, ReasonLoginReward).Error)
	require.Equal(t, int64(10), ledger.DeltaTenths)
}

func TestSpendOnVideoCoin(t *testing.T) {
	db := setupUserCoinDB(t)
	createTestUser(t, db, 1, 100)
	createTestUser(t, db, 2, 0)

	err := SpendOnVideoCoin(db, 1, 2, 42, 1)
	require.NoError(t, err)

	var viewer model.User
	db.First(&viewer, 1)
	require.Equal(t, int64(90), viewer.CoinBalanceTenths)

	var uploader model.User
	db.First(&uploader, 2)
	require.Equal(t, int64(1), uploader.CoinBalanceTenths)
}

func TestSpendOnVideoCoin_Insufficient(t *testing.T) {
	db := setupUserCoinDB(t)
	createTestUser(t, db, 1, 5)
	createTestUser(t, db, 2, 0)

	err := SpendOnVideoCoin(db, 1, 2, 42, 1)
	require.ErrorIs(t, err, ErrInsufficientCoins)
}

func TestSpendOnVideoCoin_ZeroAmount(t *testing.T) {
	db := setupUserCoinDB(t)
	createTestUser(t, db, 1, 100)
	createTestUser(t, db, 2, 0)

	err := SpendOnVideoCoin(db, 1, 2, 42, 0)
	require.NoError(t, err)
}

func TestSpendOnArticleCoin(t *testing.T) {
	db := setupUserCoinDB(t)
	createTestUser(t, db, 1, 100)
	createTestUser(t, db, 2, 0)

	err := SpendOnArticleCoin(db, 1, 2, 99, 1)
	require.NoError(t, err)

	var viewer model.User
	db.First(&viewer, 1)
	require.Equal(t, int64(90), viewer.CoinBalanceTenths)

	var author model.User
	db.First(&author, 2)
	require.Equal(t, int64(1), author.CoinBalanceTenths)
}

func TestSpendOnArticleCoin_Insufficient(t *testing.T) {
	db := setupUserCoinDB(t)
	createTestUser(t, db, 1, 5)
	createTestUser(t, db, 2, 0)

	err := SpendOnArticleCoin(db, 1, 2, 99, 1)
	require.ErrorIs(t, err, ErrInsufficientCoins)
}

func TestSpendOnArticleCoin_ZeroAmount(t *testing.T) {
	db := setupUserCoinDB(t)
	createTestUser(t, db, 1, 100)
	createTestUser(t, db, 2, 0)

	err := SpendOnArticleCoin(db, 1, 2, 99, 0)
	require.NoError(t, err)
}
