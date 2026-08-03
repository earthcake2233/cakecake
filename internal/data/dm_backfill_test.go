package data

import (
	"cakecake/internal/model/dm"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestEnsureDmParticipantHiddenAt(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&dm.DmParticipant{}))
	require.NoError(t, ensureDmParticipantHiddenAt(db, nil))
	require.NoError(t, ensureDmParticipantHiddenAt(db, nil)) // idempotent
	require.True(t, db.Migrator().HasColumn(&dm.DmParticipant{}, "HiddenAt"))
}

func TestEnsureDmParticipantHiddenAt_NoTable(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, ensureDmParticipantHiddenAt(db, nil))
}

func TestBackfillDmParticipantPins(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&dm.DmParticipant{}))

	now := time.Now()
	older := now.Add(time.Minute)
	newest := now.Add(2 * time.Minute)
	// User 1 pinned three conversations (oldest first).
	require.NoError(t, db.Create(&dm.DmParticipant{ConversationID: 1, UserID: 1, Pinned: true, PinnedAt: &now}).Error)
	require.NoError(t, db.Create(&dm.DmParticipant{ConversationID: 2, UserID: 1, Pinned: true, PinnedAt: &older}).Error)
	require.NoError(t, db.Create(&dm.DmParticipant{ConversationID: 3, UserID: 1, Pinned: true, PinnedAt: &newest}).Error)
	// User 2 has an unpinned row with a stale pinned_at.
	require.NoError(t, db.Create(&dm.DmParticipant{ConversationID: 4, UserID: 2, Pinned: false, PinnedAt: &now}).Error)

	require.NoError(t, backfillDmParticipantPins(db, nil))

	var pinned int64
	require.NoError(t, db.Model(&dm.DmParticipant{}).Where("pinned = ?", true).Count(&pinned).Error)
	require.Equal(t, int64(1), pinned)
	// Stale pinned_at cleared on unpinned rows.
	var stale int64
	require.NoError(t, db.Model(&dm.DmParticipant{}).Where("pinned = ? AND pinned_at IS NOT NULL", false).Count(&stale).Error)
	require.Zero(t, stale)
}
