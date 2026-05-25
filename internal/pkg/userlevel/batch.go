package userlevel

import (
	"gorm.io/gorm"

	"minibili/internal/model"
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
	var users []model.User
	_ = db.Select("id", "experience").Where("id IN ?", uniq).Find(&users).Error
	for i := range users {
		u := &users[i]
		out[u.ID] = FromExperience(u.Experience).CurrentLevel
	}
	return out
}
