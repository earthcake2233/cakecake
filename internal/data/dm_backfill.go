package data

import (
	"go.uber.org/zap"
	"gorm.io/gorm"

	"minibili/internal/model"
)

func ensureDmParticipantHiddenAt(db *gorm.DB, lg *zap.Logger) error {
	m := db.Migrator()
	if !m.HasTable(&model.DmParticipant{}) {
		return nil
	}
	if !m.HasColumn(&model.DmParticipant{}, "HiddenAt") {
		if err := m.AddColumn(&model.DmParticipant{}, "HiddenAt"); err != nil {
			return err
		}
		if lg != nil {
			lg.Info("added dm_participants.hidden_at column")
		}
	}
	return nil
}

// backfillDmParticipantPins repairs legacy rows where one user had multiple pinned DM threads.
// Keeps the newest pin (pinned_at, then id); clears pinned_at when pinned=false.
func backfillDmParticipantPins(db *gorm.DB, lg *zap.Logger) error {
	// 未置顶会话不应有 pinned_at（含历史写入的 0000-00-00）
	_ = db.Model(&model.DmParticipant{}).
		Where("pinned = ?", false).
		Update("pinned_at", nil).Error

	type userPinCount struct {
		UserID uint64
		Cnt    int64
	}
	var multi []userPinCount
	if err := db.Model(&model.DmParticipant{}).
		Select("user_id, COUNT(*) as cnt").
		Where("pinned = ?", true).
		Group("user_id").
		Having("COUNT(*) > 1").
		Scan(&multi).Error; err != nil {
		return err
	}

	var unpinnedExtra int64
	for _, u := range multi {
		var parts []model.DmParticipant
		if err := db.Where("user_id = ? AND pinned = ?", u.UserID, true).
			Order("pinned_at DESC, id DESC").
			Find(&parts).Error; err != nil {
			return err
		}
		if len(parts) <= 1 {
			continue
		}
		keepID := parts[0].ID
		res := db.Model(&model.DmParticipant{}).
			Where("user_id = ? AND pinned = ? AND id != ?", u.UserID, true, keepID).
			Updates(map[string]interface{}{
				"pinned":    false,
				"pinned_at": nil,
			})
		if res.Error != nil {
			return res.Error
		}
		unpinnedExtra += res.RowsAffected
	}

	res := db.Model(&model.DmParticipant{}).
		Where("pinned = ? AND pinned_at IS NOT NULL", false).
		Update("pinned_at", nil)
	if res.Error != nil {
		return res.Error
	}

	if lg != nil && (unpinnedExtra > 0 || res.RowsAffected > 0) {
		lg.Info("backfill dm participant pins",
			zap.Int64("extra_pins_cleared", unpinnedExtra),
			zap.Int64("orphan_pinned_at_cleared", res.RowsAffected),
			zap.Int("users_with_multi_pin", len(multi)),
		)
	}
	return nil
}
