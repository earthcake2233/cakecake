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

// NewDailyRewardService creates a DailyRewardService.
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

var _ DailyRewardStore = (*DailyRewardStoreImpl)(nil)

// NewDailyRewardStore creates a gorm-backed daily-reward store.
func NewDailyRewardStore(db *gorm.DB) *DailyRewardStoreImpl {
	return &DailyRewardStoreImpl{db: db}
}

// MarkLogin records the daily login reward at the store layer.
func (p *DailyRewardStoreImpl) MarkLogin(uid uint64) error {
	return dailyreward.MarkLogin(p.db, uid)
}

// MarkWatch records the daily watch reward at the store layer.
func (p *DailyRewardStoreImpl) MarkWatch(uid uint64) error {
	return dailyreward.MarkWatch(p.db, uid)
}

// BuildSnapshot assembles the daily rewards snapshot from the store.
func (p *DailyRewardStoreImpl) BuildSnapshot(uid uint64) (dailyreward.RewardsSnapshot, error) {
	return dailyreward.BuildSnapshot(p.db, uid)
}

// CoinProgress reads the user's daily coin-task EXP progress.
func (p *DailyRewardStoreImpl) CoinProgress(uid uint64) int {
	return dailyreward.CoinProgress(p.db, uid)
}

// MarkLogin records the daily login reward and returns its snapshot.
func (s *DailyRewardService) MarkLogin(uid uint64) error {
	return s.store.MarkLogin(uid)
}

// MarkWatch records the daily watch reward and returns its snapshot.
func (s *DailyRewardService) MarkWatch(uid uint64) error {
	return s.store.MarkWatch(uid)
}

// BuildSnapshot returns the user's current daily rewards snapshot.
func (s *DailyRewardService) BuildSnapshot(uid uint64) (dailyreward.RewardsSnapshot, error) {
	return s.store.BuildSnapshot(uid)
}

// CoinProgress returns the user's daily coin-task EXP progress.
func (s *DailyRewardService) CoinProgress(uid uint64) int {
	return s.store.CoinProgress(uid)
}
