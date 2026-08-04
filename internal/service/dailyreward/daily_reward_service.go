package dailyreward

import (
	"cakecake/internal/pkg/dailyreward"

	"gorm.io/gorm"
)

// DailyRewardService owns the DB access for daily reward operations so
// handlers never pass *gorm.DB around.
type DailyRewardService struct {
	store DailyRewardStore
}

func NewDailyRewardService(db *gorm.DB) *DailyRewardService {
	return &DailyRewardService{store: NewDailyRewardStore(db)}
}

// DailyRewardStore is the daily-reward storage boundary (Phase 1: *gorm.DB impl).
type DailyRewardStore interface {
	MarkLogin(uid uint64) error
	MarkWatch(uid uint64) error
	BuildSnapshot(uid uint64) (dailyreward.RewardsSnapshot, error)
	CoinProgress(uid uint64) int
}

// DailyRewardStoreImpl implements DailyRewardStore using *gorm.DB (Phase 1 monolith).
type DailyRewardStoreImpl struct {
	db *gorm.DB
}

func NewDailyRewardStore(db *gorm.DB) *DailyRewardStoreImpl {
	return &DailyRewardStoreImpl{db: db}
}

func (p *DailyRewardStoreImpl) MarkLogin(uid uint64) error {
	return dailyreward.MarkLogin(p.db, uid)
}

func (p *DailyRewardStoreImpl) MarkWatch(uid uint64) error {
	return dailyreward.MarkWatch(p.db, uid)
}

func (p *DailyRewardStoreImpl) BuildSnapshot(uid uint64) (dailyreward.RewardsSnapshot, error) {
	return dailyreward.BuildSnapshot(p.db, uid)
}

func (p *DailyRewardStoreImpl) CoinProgress(uid uint64) int {
	return dailyreward.CoinProgress(p.db, uid)
}

func (s *DailyRewardService) MarkLogin(uid uint64) error {
	return s.store.MarkLogin(uid)
}

func (s *DailyRewardService) MarkWatch(uid uint64) error {
	return s.store.MarkWatch(uid)
}

func (s *DailyRewardService) BuildSnapshot(uid uint64) (dailyreward.RewardsSnapshot, error) {
	return s.store.BuildSnapshot(uid)
}

func (s *DailyRewardService) CoinProgress(uid uint64) int {
	return s.store.CoinProgress(uid)
}
