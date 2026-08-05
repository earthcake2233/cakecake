package userlevel

import (
	"cakecake/internal/model/user"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"cakecake/internal/logger"
)

// BatchCurrentLevels maps user id to account level (1–6) from stored experience.
func BatchCurrentLevels(db *gorm.DB, uids []uint64) map[uint64]int {
	out := make(map[uint64]int, len(uids))
	if db == nil || len(uids) == 0 {
		return out
	}
	seen := make(map[uint64]struct{}, len(uids))
	uniq := make([]uint64, 0, len(uids))
	for _, id := range uids {
		if id == 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		uniq = append(uniq, id)
	}
	if len(uniq) == 0 {
		return out
	}
	var users []user.User
	if err := db.Select("id", "experience").Where("id IN ?", uniq).Find(&users).Error; err != nil && logger.L != nil {
		logger.L.Warn("userlevel: batch load users failed", zap.Error(err))
	}
	for i := range users {
		u := &users[i]
		out[u.ID] = FromExperience(u.Experience).CurrentLevel
	}
	return out
}
