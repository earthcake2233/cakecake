package service

import (
	"gorm.io/gorm"

	"cakecake/internal/pkg/dailyreward"
)

// DailyRewardService owns the DB access for daily reward operations so
// handlers never pass *gorm.DB around.
type DailyRewardService struct {
	db *gorm.DB
}

func NewDailyRewardService(db *gorm.DB) *DailyRewardService {
	return &DailyRewardService{db: db}
}

func (s *DailyRewardService) MarkLogin(uid uint64) error {
	return dailyreward.MarkLogin(s.db, uid)
}

func (s *DailyRewardService) MarkWatch(uid uint64) error {
	return dailyreward.MarkWatch(s.db, uid)
}

func (s *DailyRewardService) BuildSnapshot(uid uint64) (dailyreward.RewardsSnapshot, error) {
	return dailyreward.BuildSnapshot(s.db, uid)
}

func (s *DailyRewardService) CoinProgress(uid uint64) int {
	return dailyreward.CoinProgress(s.db, uid)
}
