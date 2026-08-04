package engagement

import (
	"cakecake/internal/model/user"
	"cakecake/internal/model/video"
	"cakecake/internal/pkg/dailyreward"
	"cakecake/internal/pkg/usercoin"
	"context"

	"gorm.io/gorm"
)

// EngagementStore is the engagement-domain storage boundary.
// Phase 1: *gorm.DB impl. Phase 2+: replaced by gRPC client / per-domain store.
type EngagementStore interface {
	HasCoined(ctx context.Context, userID, videoID uint64) bool
	DecrementUserCoinsFallback(ctx context.Context, userID uint64, amount int) error
	IncrVideoCoinCount(ctx context.Context, videoID uint64, delta int) error
	ToggleWatchLater(ctx context.Context, userID, videoID uint64) (bool, error)
	ListWatchLater(ctx context.Context, userID uint64, page, pageSize int) ([]video.WatchLater, int64, error)
	ClearWatchLater(ctx context.Context, userID uint64) error
	ClearWatchedWatchLater(ctx context.Context, userID uint64) error
	MarkWatchLaterWatched(ctx context.Context, userID, videoID uint64) error
	BatchHasCoined(ctx context.Context, userID uint64, videoIDs []uint64) map[uint64]bool
	BatchWatchLater(ctx context.Context, userID uint64, videoIDs []uint64) map[uint64]bool
	BatchCoinedByUser(ctx context.Context, userID uint64, videoIDs []uint64) map[uint64]int
	GetVideoCoinRow(ctx context.Context, userID, videoID uint64) (*video.VideoCoin, bool, error)
	UpdateVideoCoinTx(ctx context.Context, uid, uploaderID, vid uint64, exist *video.VideoCoin) error
	CreateVideoCoinTx(ctx context.Context, uid, uploaderID, vid uint64, amount int) error
	CoinProgress(uid uint64) int
	GrantCoinExp(uid uint64, before, after int) error
	GetVideoByID(ctx context.Context, id uint64) (*video.Video, error)
	GetUserByID(ctx context.Context, id uint64) (*user.User, error)
	BatchVideoLikes(ctx context.Context, userID uint64, videoIDs []uint64) map[uint64]bool
	BatchPublishedVideosRaw(ctx context.Context, ids []uint64) (map[uint64]video.Video, error)
	ListUserCoinedVideosRows(ctx context.Context, ownerID uint64, limit int) ([]video.VideoCoin, int64, error)
}

// EngagementStoreImpl implements EngagementStore using *gorm.DB (Phase 1 monolith).
type EngagementStoreImpl struct {
	db *gorm.DB
}

func NewEngagementStore(db *gorm.DB) *EngagementStoreImpl {
	return &EngagementStoreImpl{db: db}
}

func (p *EngagementStoreImpl) HasCoined(ctx context.Context, userID, videoID uint64) bool {
	var cnt int64
	p.db.WithContext(ctx).Model(&video.VideoCoin{}).Where("user_id = ? AND video_id = ?", userID, videoID).Count(&cnt)
	return cnt > 0
}

func (p *EngagementStoreImpl) DecrementUserCoinsFallback(ctx context.Context, userID uint64, amount int) error {
	cost := usercoin.CostTenths(amount)
	return p.db.WithContext(ctx).Model(&user.User{}).Where("id = ? AND coin_balance_tenths >= ?", userID, cost).
		UpdateColumn("coin_balance_tenths", gorm.Expr("coin_balance_tenths - ?", cost)).Error
}

func (p *EngagementStoreImpl) IncrVideoCoinCount(ctx context.Context, videoID uint64, delta int) error {
	return p.db.WithContext(ctx).Model(&video.Video{}).Where("id = ?", videoID).
		UpdateColumn("coin_count", gorm.Expr("coin_count + ?", delta)).Error
}

func (p *EngagementStoreImpl) ToggleWatchLater(ctx context.Context, userID, videoID uint64) (bool, error) {
	var existing video.WatchLater
	if err := p.db.WithContext(ctx).Where("user_id = ? AND video_id = ?", userID, videoID).First(&existing).Error; err == nil {
		_ = p.db.WithContext(ctx).Delete(&existing).Error
		return false, nil
	}
	wl := video.WatchLater{UserID: userID, VideoID: videoID}
	if err := p.db.WithContext(ctx).Create(&wl).Error; err != nil {
		return false, err
	}
	return true, nil
}

func (p *EngagementStoreImpl) ListWatchLater(ctx context.Context, userID uint64, page, pageSize int) ([]video.WatchLater, int64, error) {
	var total int64
	_ = p.db.WithContext(ctx).Model(&video.WatchLater{}).Where("user_id = ?", userID).Count(&total).Error
	q := p.db.WithContext(ctx).Where("user_id = ?", userID)
	offset := (page - 1) * pageSize
	var list []video.WatchLater
	if err := q.Order("id DESC").Offset(offset).Limit(pageSize).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

func (p *EngagementStoreImpl) ClearWatchLater(ctx context.Context, userID uint64) error {
	return p.db.WithContext(ctx).Where("user_id = ?", userID).Delete(&video.WatchLater{}).Error
}

func (p *EngagementStoreImpl) ClearWatchedWatchLater(ctx context.Context, userID uint64) error {
	return p.db.WithContext(ctx).Where("user_id = ? AND watched = ?", userID, true).Delete(&video.WatchLater{}).Error
}

func (p *EngagementStoreImpl) MarkWatchLaterWatched(ctx context.Context, userID, videoID uint64) error {
	return p.db.WithContext(ctx).Model(&video.WatchLater{}).
		Where("user_id = ? AND video_id = ?", userID, videoID).Update("watched", true).Error
}

func (p *EngagementStoreImpl) BatchHasCoined(ctx context.Context, userID uint64, videoIDs []uint64) map[uint64]bool {
	result := make(map[uint64]bool)
	if userID == 0 || len(videoIDs) == 0 {
		return result
	}
	var coins []video.VideoCoin
	p.db.WithContext(ctx).Where("user_id = ? AND video_id IN ?", userID, videoIDs).Find(&coins)
	for _, c := range coins {
		result[c.VideoID] = true
	}
	return result
}

func (p *EngagementStoreImpl) BatchWatchLater(ctx context.Context, userID uint64, videoIDs []uint64) map[uint64]bool {
	result := make(map[uint64]bool)
	if userID == 0 || len(videoIDs) == 0 {
		return result
	}
	var wls []video.WatchLater
	p.db.WithContext(ctx).Where("user_id = ? AND video_id IN ?", userID, videoIDs).Find(&wls)
	for _, w := range wls {
		result[w.VideoID] = true
	}
	return result
}

func (p *EngagementStoreImpl) BatchCoinedByUser(ctx context.Context, userID uint64, videoIDs []uint64) map[uint64]int {
	result := make(map[uint64]int)
	if userID == 0 || len(videoIDs) == 0 {
		return result
	}
	var coins []video.VideoCoin
	p.db.WithContext(ctx).Where("user_id = ? AND video_id IN ?", userID, videoIDs).Find(&coins)
	for _, c := range coins {
		amt := c.Amount
		if amt < 0 {
			amt = 0
		}
		if amt > 2 {
			amt = 2
		}
		result[c.VideoID] = amt
	}
	return result
}

func (p *EngagementStoreImpl) GetVideoCoinRow(ctx context.Context, userID, videoID uint64) (*video.VideoCoin, bool, error) {
	var exist video.VideoCoin
	res := p.db.WithContext(ctx).Where("user_id = ? AND video_id = ?", userID, videoID).Limit(1).Find(&exist)
	if res.Error != nil {
		return nil, false, res.Error
	}
	if res.RowsAffected == 0 {
		return nil, false, nil
	}
	return &exist, true, nil
}

func (p *EngagementStoreImpl) UpdateVideoCoinTx(ctx context.Context, uid, uploaderID, vid uint64, exist *video.VideoCoin) error {
	return p.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := usercoin.SpendOnVideoCoin(tx, uid, uploaderID, vid, 1); err != nil {
			return err
		}
		if err := tx.Model(exist).Update("amount", 2).Error; err != nil {
			return err
		}
		return tx.Model(&video.Video{}).Where("id = ?", vid).UpdateColumn("coin_count", gorm.Expr("coin_count + ?", 1)).Error
	})
}

func (p *EngagementStoreImpl) CreateVideoCoinTx(ctx context.Context, uid, uploaderID, vid uint64, amount int) error {
	return p.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := usercoin.SpendOnVideoCoin(tx, uid, uploaderID, vid, amount); err != nil {
			return err
		}
		row := video.VideoCoin{UserID: uid, VideoID: vid, Amount: amount}
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
		return tx.Model(&video.Video{}).Where("id = ?", vid).UpdateColumn("coin_count", gorm.Expr("coin_count + ?", amount)).Error
	})
}

func (p *EngagementStoreImpl) CoinProgress(uid uint64) int {
	return dailyreward.CoinProgress(p.db, uid)
}

func (p *EngagementStoreImpl) GrantCoinExp(uid uint64, before, after int) error {
	return dailyreward.GrantCoinExp(p.db, uid, before, after)
}

func (p *EngagementStoreImpl) GetVideoByID(ctx context.Context, id uint64) (*video.Video, error) {
	var v video.Video
	if err := p.db.WithContext(ctx).First(&v, id).Error; err != nil {
		return nil, err
	}
	return &v, nil
}

func (p *EngagementStoreImpl) GetUserByID(ctx context.Context, id uint64) (*user.User, error) {
	var u user.User
	if err := p.db.WithContext(ctx).First(&u, id).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

func (p *EngagementStoreImpl) BatchVideoLikes(ctx context.Context, userID uint64, videoIDs []uint64) map[uint64]bool {
	result := make(map[uint64]bool)
	if userID == 0 || len(videoIDs) == 0 {
		return result
	}
	var likes []video.VideoLike
	p.db.WithContext(ctx).Where("user_id = ? AND video_id IN ?", userID, videoIDs).Find(&likes)
	for _, l := range likes {
		result[l.VideoID] = true
	}
	return result
}

func (p *EngagementStoreImpl) BatchPublishedVideosRaw(ctx context.Context, ids []uint64) (map[uint64]video.Video, error) {
	result := make(map[uint64]video.Video, len(ids))
	if len(ids) == 0 {
		return result, nil
	}
	var videos []video.Video
	if err := p.db.WithContext(ctx).Where("id IN ? AND status = ?", ids, video.StatusPublished).Find(&videos).Error; err != nil {
		return nil, err
	}
	for i := range videos {
		result[videos[i].ID] = videos[i]
	}
	return result, nil
}

func (p *EngagementStoreImpl) ListUserCoinedVideosRows(ctx context.Context, ownerID uint64, limit int) ([]video.VideoCoin, int64, error) {
	var coins []video.VideoCoin
	if err := p.db.WithContext(ctx).Where("user_id = ?", ownerID).Order("created_at DESC").Limit(limit).Find(&coins).Error; err != nil {
		return nil, 0, err
	}
	var total int64
	_ = p.db.WithContext(ctx).Model(&video.VideoCoin{}).Where("user_id = ?", ownerID).Count(&total).Error
	return coins, total, nil
}
