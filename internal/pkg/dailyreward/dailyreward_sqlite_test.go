package dailyreward

import (
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"minibili/internal/model"
)

func setupDailyRewardDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.UserDailyTask{}, &model.User{}, &model.CoinLedger{}, &model.VideoCoin{}))
	return db
}

func createUser(t *testing.T, db *gorm.DB, id uint64) {
	require.NoError(t, db.Create(&model.User{ID: id, Username: "u" + uitoa(id), PasswordHash: "hash", CoinBalanceTenths: 200}).Error)
}

func uitoa(n uint64) string {
	if n == 0 {
		return "0"
	}
	s := ""
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	return s
}

func TestMarkLogin_FirstTime(t *testing.T) {
	db := setupDailyRewardDB(t)
	createUser(t, db, 1)

	err := MarkLogin(db, 1)
	require.NoError(t, err)

	// should create a row and mark login_done
	var row model.UserDailyTask
	require.NoError(t, db.Where("user_id = ? AND task_date = ?", 1, TodayDate()).First(&row).Error)
	require.True(t, row.LoginDone)
	require.False(t, row.WatchDone)
}

func TestMarkLogin_Idempotent(t *testing.T) {
	db := setupDailyRewardDB(t)
	createUser(t, db, 2)

	err := MarkLogin(db, 2)
	require.NoError(t, err)

	// second call should be no-op
	err = MarkLogin(db, 2)
	require.NoError(t, err)

	var rows []model.UserDailyTask
	require.NoError(t, db.Find(&rows).Error)
	require.Len(t, rows, 1)
}

func TestMarkWatch_FirstTime(t *testing.T) {
	db := setupDailyRewardDB(t)
	createUser(t, db, 3)

	err := MarkWatch(db, 3)
	require.NoError(t, err)

	var row model.UserDailyTask
	require.NoError(t, db.Where("user_id = ? AND task_date = ?", 3, TodayDate()).First(&row).Error)
	require.True(t, row.WatchDone)
	require.False(t, row.LoginDone)
}

func TestMarkWatch_Idempotent(t *testing.T) {
	db := setupDailyRewardDB(t)
	createUser(t, db, 4)

	require.NoError(t, MarkWatch(db, 4))
	require.NoError(t, MarkWatch(db, 4))

	var rows []model.UserDailyTask
	require.NoError(t, db.Find(&rows).Error)
	require.Len(t, rows, 1)
}

func TestGrantCoinExp_PositiveDelta(t *testing.T) {
	db := setupDailyRewardDB(t)
	createUser(t, db, 5)

	err := GrantCoinExp(db, 5, 0, 10)
	require.NoError(t, err)

	var u model.User
	require.NoError(t, db.First(&u, 5).Error)
	require.Equal(t, uint64(10), u.Experience)
}

func TestGrantCoinExp_ZeroDelta(t *testing.T) {
	db := setupDailyRewardDB(t)
	createUser(t, db, 6)

	err := GrantCoinExp(db, 6, 10, 10)
	require.NoError(t, err)

	var u model.User
	require.NoError(t, db.First(&u, 6).Error)
	require.Equal(t, uint64(0), u.Experience)
}

func TestGrantCoinExp_NegativeDelta(t *testing.T) {
	db := setupDailyRewardDB(t)
	createUser(t, db, 7)

	err := GrantCoinExp(db, 7, 10, 5)
	require.NoError(t, err) // delta <= 0, no-op

	var u model.User
	require.NoError(t, db.First(&u, 7).Error)
	require.Equal(t, uint64(0), u.Experience)
}

func TestCoinProgress_Zero(t *testing.T) {
	db := setupDailyRewardDB(t)
	createUser(t, db, 8)

	cp := CoinProgress(db, 8)
	require.Equal(t, 0, cp)
}

func TestBuildSnapshot_FreshUser(t *testing.T) {
	db := setupDailyRewardDB(t)
	createUser(t, db, 9)

	snap, err := BuildSnapshot(db, 9)
	require.NoError(t, err)
	require.Equal(t, ExpLogin, snap.Login.Exp)
	require.False(t, snap.Login.Done)
	require.Equal(t, ExpWatch, snap.Watch.Exp)
	require.False(t, snap.Watch.Done)
	require.Equal(t, ExpCoinMax, snap.Coin.Exp)
	require.False(t, snap.Coin.Done)
	require.Equal(t, 0, snap.Coin.Progress)
	require.Equal(t, ExpCoinMax, snap.Coin.Max)
}

func TestBuildSnapshot_AfterLogin(t *testing.T) {
	db := setupDailyRewardDB(t)
	createUser(t, db, 10)

	require.NoError(t, MarkLogin(db, 10))

	snap, err := BuildSnapshot(db, 10)
	require.NoError(t, err)
	require.True(t, snap.Login.Done)
}

func TestBuildSnapshot_AfterWatch(t *testing.T) {
	db := setupDailyRewardDB(t)
	createUser(t, db, 11)

	require.NoError(t, MarkWatch(db, 11))

	snap, err := BuildSnapshot(db, 11)
	require.NoError(t, err)
	require.True(t, snap.Watch.Done)
}

func TestTodayDate_NotEmpty(t *testing.T) {
	date := TodayDate()
	require.NotEmpty(t, date)
	require.Len(t, date, 10) // YYYY-MM-DD
}

func TestEnsureRow_CreatesAndFetches(t *testing.T) {
	db := setupDailyRewardDB(t)
	createUser(t, db, 12)

	row, err := ensureRow(db, 12)
	require.NoError(t, err)
	require.NotNil(t, row)
	require.Equal(t, uint64(12), row.UserID)
	require.Equal(t, TodayDate(), row.TaskDate)
}

func TestAddUserExp_ZeroDelta(t *testing.T) {
	db := setupDailyRewardDB(t)
	createUser(t, db, 13)

	err := addUserExp(db, 13, 0)
	require.NoError(t, err)

	var u model.User
	require.NoError(t, db.First(&u, 13).Error)
	require.Equal(t, uint64(0), u.Experience)
}

func TestAddUserExp_Positive(t *testing.T) {
	db := setupDailyRewardDB(t)
	createUser(t, db, 14)

	err := addUserExp(db, 14, 50)
	require.NoError(t, err)

	var u model.User
	require.NoError(t, db.First(&u, 14).Error)
	require.Equal(t, uint64(50), u.Experience)
}

func TestGrantCoinExp_Additive(t *testing.T) {
	db := setupDailyRewardDB(t)
	createUser(t, db, 15)

	require.NoError(t, GrantCoinExp(db, 15, 0, 10))
	require.NoError(t, GrantCoinExp(db, 15, 10, 25))

	var u model.User
	require.NoError(t, db.First(&u, 15).Error)
	require.Equal(t, uint64(25), u.Experience) // 10 + 15
}

func TestDayBounds_Consistent(t *testing.T) {
	start, end := dayBounds()
	require.True(t, start.Before(end))
	require.Equal(t, 24*time.Hour, end.Sub(start))
}
