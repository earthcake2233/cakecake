package usercoin

import (
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"minibili/internal/model"
)

func setupLedgerDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.CoinLedger{}))
	return db
}

func TestRecordLedger_CreatesRow(t *testing.T) {
	db := setupLedgerDB(t)
	err := RecordLedger(db, 1, 100, ReasonLoginReward, 0)
	require.NoError(t, err)

	var rows []model.CoinLedger
	require.NoError(t, db.Find(&rows).Error)
	require.Len(t, rows, 1)
	require.Equal(t, uint64(1), rows[0].UserID)
	require.Equal(t, int64(100), rows[0].DeltaTenths)
	require.Equal(t, ReasonLoginReward, rows[0].ReasonType)
}

func TestRecordLedger_ZeroDelta(t *testing.T) {
	err := RecordLedger(nil, 1, 0, "test", 0)
	require.NoError(t, err) // should skip without error
}

func TestRecordLedger_NegativeDelta(t *testing.T) {
	db := setupLedgerDB(t)
	err := RecordLedger(db, 1, -50, ReasonNicknameChange, 0)
	require.NoError(t, err)

	var rows []model.CoinLedger
	require.NoError(t, db.Find(&rows).Error)
	require.Len(t, rows, 1)
	require.Equal(t, int64(-50), rows[0].DeltaTenths)
}

func TestRecordLedger_WithVideoID(t *testing.T) {
	db := setupLedgerDB(t)
	err := RecordLedger(db, 2, 10, ReasonVideoTip, 42)
	require.NoError(t, err)

	var row model.CoinLedger
	require.NoError(t, db.First(&row).Error)
	require.Equal(t, uint64(42), row.VideoID)
}

func TestRecordLedgerAt_SpecificTime(t *testing.T) {
	db := setupLedgerDB(t)
	at := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	err := RecordLedgerAt(db, 3, 200, ReasonArticleTipIncome, 99, at)
	require.NoError(t, err)

	var row model.CoinLedger
	require.NoError(t, db.First(&row).Error)
	require.True(t, row.CreatedAt.Equal(at), "expected %v, got %v", at, row.CreatedAt)
}

func TestRecordLedgerAt_ZeroDelta(t *testing.T) {
	err := RecordLedgerAt(nil, 1, 0, "test", 0, time.Now())
	require.NoError(t, err)
}

func TestRecordLedger_MultipleReasons(t *testing.T) {
	db := setupLedgerDB(t)
	reasons := []string{ReasonLoginReward, ReasonVideoTip, ReasonVideoTipIncome, ReasonArticleTip, ReasonArticleTipIncome, ReasonNicknameChange}
	for i, r := range reasons {
		require.NoError(t, RecordLedger(db, uint64(i+1), int64((i+1)*10), r, uint64(i)))
	}

	var count int64
	db.Model(&model.CoinLedger{}).Count(&count)
	require.Equal(t, int64(len(reasons)), count)
}
