package service

import (
	"minibili/internal/model/user"
	"minibili/internal/model/video"
	"context"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"minibili/internal/pkg/dailyreward"
	"minibili/internal/pkg/usercoin"
)

// EngagementService handles video engagement operations (coin, watch later).
type EngagementService struct {
	db  *gorm.DB
	rdb *redis.Client
	log *zap.Logger

	users  UserProvider
	videos VideoProvider
}

func NewEngagementService(db *gorm.DB, rdb *redis.Client, log *zap.Logger) *EngagementService {
	return &EngagementService{db: db, rdb: rdb, log: log}
}

func (s *EngagementService) SetProviders(users UserProvider, videos VideoProvider) {
	s.users = users
	s.videos = videos
}



// HasCoined checks if user already coined a video.
func (s *EngagementService) HasCoined(ctx context.Context, userID, videoID uint64) bool {
	var cnt int64
	s.db.WithContext(ctx).Model(&video.VideoCoin{}).Where("user_id = ? AND video_id = ?", userID, videoID).Count(&cnt)
	return cnt > 0
}

// GetUserCoinBalance returns the user current coin balance.
func (s *EngagementService) GetUserCoinBalance(ctx context.Context, userID uint64) int64 {
	if s.users == nil { return 0 }
	u, err := s.users.GetUser(ctx, userID)
	if err != nil { return 0 }
	return u.CoinBalanceTenths
}

// DecrementUserCoins subtracts coins from user balance.
func (s *EngagementService) DecrementUserCoins(ctx context.Context, userID uint64, amount int) error {
	if s.users == nil {
		return s.db.WithContext(ctx).Model(&user.User{}).Where("id = ? AND coins >= ?", userID, amount).
			UpdateColumn("coins", gorm.Expr("coins - ?", amount)).Error
	}
	return s.users.DecrementCoins(ctx, userID, amount)
}

// IncrementVideoCoinCount increments the coin count on a video.
func (s *EngagementService) IncrementVideoCoinCount(ctx context.Context, videoID uint64, delta int) error {
	return s.db.WithContext(ctx).Model(&video.Video{}).Where("id = ?", videoID).
		UpdateColumn("coin_count", gorm.Expr("coin_count + ?", delta)).Error
}

// ToggleWatchLater adds or removes a watch-later entry.
func (s *EngagementService) ToggleWatchLater(ctx context.Context, userID, videoID uint64) (bool, error) {
	var existing video.WatchLater
	if err := s.db.WithContext(ctx).Where("user_id = ? AND video_id = ?", userID, videoID).First(&existing).Error; err == nil {
		_ = s.db.WithContext(ctx).Delete(&existing).Error
		return false, nil
	}
	wl := video.WatchLater{UserID: userID, VideoID: videoID}
	if err := s.db.WithContext(ctx).Create(&wl).Error; err != nil { return false, err }
	return true, nil
}

// ListWatchLater returns watch-later entries with pagination.
func (s *EngagementService) ListWatchLater(ctx context.Context, userID uint64, page, pageSize int) ([]video.WatchLater, int64, error) {
	var total int64
	_ = s.db.WithContext(ctx).Model(&video.WatchLater{}).Where("user_id = ?", userID).Count(&total).Error
	q := s.db.WithContext(ctx).Where("user_id = ?", userID)
	offset := (page - 1) * pageSize
	var list []video.WatchLater
	if err := q.Order("id DESC").Offset(offset).Limit(pageSize).Find(&list).Error; err != nil { return nil, 0, err }
	return list, total, nil
}

// ClearWatchLater removes all watch-later entries for a user.
func (s *EngagementService) ClearWatchLater(ctx context.Context, userID uint64) error {
	return s.db.WithContext(ctx).Where("user_id = ?", userID).Delete(&video.WatchLater{}).Error
}

// ClearWatchedWatchLater removes watched watch-later entries.
func (s *EngagementService) ClearWatchedWatchLater(ctx context.Context, userID uint64) error {
	return s.db.WithContext(ctx).Where("user_id = ? AND watched = ?", userID, true).Delete(&video.WatchLater{}).Error
}

// MarkWatchLaterWatched marks a watch-later entry as watched.
func (s *EngagementService) MarkWatchLaterWatched(ctx context.Context, userID, videoID uint64) error {
	return s.db.WithContext(ctx).Model(&video.WatchLater{}).
		Where("user_id = ? AND video_id = ?", userID, videoID).Update("watched", true).Error
}

// BatchHasCoined returns a map of video_id -> coined for a user.
func (s *EngagementService) BatchHasCoined(ctx context.Context, userID uint64, videoIDs []uint64) map[uint64]bool {
	result := make(map[uint64]bool)
	if userID == 0 || len(videoIDs) == 0 { return result }
	var coins []video.VideoCoin
	s.db.WithContext(ctx).Where("user_id = ? AND video_id IN ?", userID, videoIDs).Find(&coins)
	for _, c := range coins { result[c.VideoID] = true }
	return result
}

// BatchWatchLater returns a map of video_id -> in-watch-later for a user.
func (s *EngagementService) BatchWatchLater(ctx context.Context, userID uint64, videoIDs []uint64) map[uint64]bool {
	result := make(map[uint64]bool)
	if userID == 0 || len(videoIDs) == 0 { return result }
	var wls []video.WatchLater
	s.db.WithContext(ctx).Where("user_id = ? AND video_id IN ?", userID, videoIDs).Find(&wls)
	for _, w := range wls { result[w.VideoID] = true }
	return result
}
// BatchFavoritedByUser returns a map of video_id -> favorited for a user.
func (s *EngagementService) BatchFavoritedByUser(ctx context.Context, userID uint64, videoIDs []uint64) map[uint64]bool {
	result := make(map[uint64]bool)
	if userID == 0 || len(videoIDs) == 0 { return result }
	var favs []video.VideoFavorite
	s.db.WithContext(ctx).Where("user_id = ? AND video_id IN ?", userID, videoIDs).Find(&favs)
	for _, f := range favs { result[f.VideoID] = true }
	return result
}

// BatchCoinedByUser returns a map of video_id -> coin amount for a user.
func (s *EngagementService) BatchCoinedByUser(ctx context.Context, userID uint64, videoIDs []uint64) map[uint64]int {
	result := make(map[uint64]int)
	if userID == 0 || len(videoIDs) == 0 { return result }
	var coins []video.VideoCoin
	s.db.WithContext(ctx).Where("user_id = ? AND video_id IN ?", userID, videoIDs).Find(&coins)
	for _, c := range coins {
		amt := c.Amount
		if amt < 0 { amt = 0 }
		if amt > 2 { amt = 2 }
		result[c.VideoID] = amt
	}
	return result
}

// ToggleFavorite adds or removes a favorite, returning the new state and video count.
func (s *EngagementService) ToggleFavorite(ctx context.Context, userID, videoID uint64) (bool, uint64, error) {
	var rows []video.VideoFavorite
	res := s.db.WithContext(ctx).Where("user_id = ? AND video_id = ?", userID, videoID).Find(&rows)
	if res.Error != nil { return false, 0, res.Error }
	if len(rows) == 0 {
		row := video.VideoFavorite{UserID: userID, VideoID: videoID}
		if err := s.db.WithContext(ctx).Create(&row).Error; err != nil { return false, 0, err }
		_ = s.db.WithContext(ctx).Model(&video.Video{}).Where("id = ?", videoID).UpdateColumn("fav_count", gorm.Expr("fav_count + ?", 1)).Error
		var v video.Video
		if err := s.db.WithContext(ctx).First(&v, videoID).Error; err != nil { return true, 0, nil }
		return true, v.FavCount, nil
	}
	if err := s.db.WithContext(ctx).Where("user_id = ? AND video_id = ?", userID, videoID).Delete(&video.VideoFavorite{}).Error; err != nil {
		return false, 0, err
	}
	_ = s.db.WithContext(ctx).Model(&video.Video{}).Where("id = ?", videoID).UpdateColumn("fav_count", gorm.Expr("CASE WHEN fav_count - ? < 0 THEN 0 ELSE fav_count - ? END", 1, 1)).Error
	var v video.Video
	if err := s.db.WithContext(ctx).First(&v, videoID).Error; err != nil { return false, 0, nil }
	return false, v.FavCount, nil
}

// MoveFavoritesBetweenFolders moves favorites from one folder to another.
func (s *EngagementService) MoveFavoritesBetweenFolders(ctx context.Context, uid, fromFolderID, toFolderID uint64) (int64, error) {
	var favs []video.VideoFavorite
	if err := s.db.WithContext(ctx).Where("user_id = ? AND folder_id = ?", uid, fromFolderID).
		Order("created_at ASC").Find(&favs).Error; err != nil { return 0, err }
	if len(favs) == 0 { return 0, nil }
	for _, fav := range favs {
		var already video.VideoFavorite
		_ = s.db.WithContext(ctx).Where("user_id = ? AND video_id = ?", uid, fav.VideoID).Limit(1).Find(&already).Error
		if already.ID > 0 && already.FolderID == toFolderID {
			_ = s.db.WithContext(ctx).Delete(&fav).Error
		} else {
			_ = s.db.WithContext(ctx).Model(&fav).Update("folder_id", toFolderID).Error
		}
	}
	var remaining int64
	_ = s.db.WithContext(ctx).Model(&video.VideoFavorite{}).Where("user_id = ? AND folder_id = ?", uid, fromFolderID).Count(&remaining).Error
	return remaining, nil
}

// VideoFavCount returns the fav_count for a video.
func (s *EngagementService) VideoFavCount(ctx context.Context, videoID uint64) (uint64, error) {
	var v video.Video
	if err := s.db.WithContext(ctx).Select("fav_count").First(&v, videoID).Error; err != nil {
		return 0, err
	}
	return v.FavCount, nil
}

// ToggleVideoFavoriteWithFolder adds or removes a favorite in a specific folder, returning new state and fav_count.
func (s *EngagementService) ToggleVideoFavoriteWithFolder(ctx context.Context, userID, videoID, folderID uint64) (bool, uint64, error) {
	var rows []video.VideoFavorite
	res := s.db.WithContext(ctx).Where("user_id = ? AND video_id = ?", userID, videoID).Find(&rows)
	if res.Error != nil { return false, 0, res.Error }
	if len(rows) == 0 {
		row := video.VideoFavorite{UserID: userID, VideoID: videoID, FolderID: folderID}
		if err := s.db.WithContext(ctx).Create(&row).Error; err != nil { return false, 0, err }
		_ = s.db.WithContext(ctx).Model(&video.Video{}).Where("id = ?", videoID).UpdateColumn("fav_count", gorm.Expr("fav_count + ?", 1)).Error
		var v video.Video
		if err := s.db.WithContext(ctx).First(&v, videoID).Error; err != nil { return true, 0, nil }
		return true, v.FavCount, nil
	}
	if err := s.db.WithContext(ctx).Where("user_id = ? AND video_id = ?", userID, videoID).Delete(&video.VideoFavorite{}).Error; err != nil {
		return false, 0, err
	}
	_ = s.db.WithContext(ctx).Model(&video.Video{}).Where("id = ?", videoID).UpdateColumn("fav_count", gorm.Expr("CASE WHEN fav_count - ? < 0 THEN 0 ELSE fav_count - ? END", 1, 1)).Error
	var v video.Video
	if err := s.db.WithContext(ctx).First(&v, videoID).Error; err != nil { return false, 0, nil }
	return false, v.FavCount, nil
}

// PostVideoCoinResult holds the result of a coin operation.
type PostVideoCoinResult struct {
	Coined         bool
	CoinCount      uint64
	Amount         int
	MyCoinAmount   int
	CoinBalance    float64
	DailyProgress  int
	DailyMax       int
}

// PostVideoCoin performs the full coin transaction for a user.
func (s *EngagementService) PostVideoCoin(ctx context.Context, uid, vid, uploaderID uint64, amount int) (*PostVideoCoinResult, error) {
	var exist video.VideoCoin
	res := s.db.WithContext(ctx).Where("user_id = ? AND video_id = ?", uid, vid).Limit(1).Find(&exist)
	if res.Error != nil { return nil, res.Error }

	coinBefore := dailyreward.CoinProgress(s.db, uid)
	var spentAmount int
	var myCoinAmount int

	if res.RowsAffected > 0 {
		if exist.Amount >= 2 { return nil, nil }
		spentAmount = 1
		myCoinAmount = 2
		if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			if err := usercoin.SpendOnVideoCoin(tx, uid, uploaderID, vid, spentAmount); err != nil { return err }
			if err := tx.Model(&exist).Update("amount", 2).Error; err != nil { return err }
			return tx.Model(&video.Video{}).Where("id = ?", vid).UpdateColumn("coin_count", gorm.Expr("coin_count + ?", 1)).Error
		}); err != nil { return nil, err }
	} else {
		if amount != 1 && amount != 2 { amount = 1 }
		spentAmount = amount
		myCoinAmount = amount
		if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			if err := usercoin.SpendOnVideoCoin(tx, uid, uploaderID, vid, spentAmount); err != nil { return err }
			row := video.VideoCoin{UserID: uid, VideoID: vid, Amount: amount}
			if err := tx.Create(&row).Error; err != nil { return err }
			return tx.Model(&video.Video{}).Where("id = ?", vid).UpdateColumn("coin_count", gorm.Expr("coin_count + ?", amount)).Error
		}); err != nil { return nil, err }
	}

	coinAfter := dailyreward.CoinProgress(s.db, uid)
	_ = dailyreward.GrantCoinExp(s.db, uid, coinBefore, coinAfter)
	var v video.Video
	_ = s.db.WithContext(ctx).First(&v, vid).Error
	var viewer user.User
	_ = s.db.WithContext(ctx).First(&viewer, uid).Error
	return &PostVideoCoinResult{
		Coined: true, CoinCount: v.CoinCount, Amount: spentAmount,
		MyCoinAmount: myCoinAmount, CoinBalance: usercoin.BalanceFloat(viewer.CoinBalanceTenths),
		DailyProgress: coinAfter, DailyMax: dailyreward.ExpCoinMax,
	}, nil
}

// BatchVideoLikes returns a map of video_id -> liked for a user.
func (s *EngagementService) BatchVideoLikes(ctx context.Context, userID uint64, videoIDs []uint64) map[uint64]bool {
	result := make(map[uint64]bool)
	if userID == 0 || len(videoIDs) == 0 { return result }
	var likes []video.VideoLike
	s.db.WithContext(ctx).Where("user_id = ? AND video_id IN ?", userID, videoIDs).Find(&likes)
	for _, l := range likes { result[l.VideoID] = true }
	return result
}

// AdjustVideoFavCount increments or decrements the fav_count on a video.
func (s *EngagementService) AdjustVideoFavCount(ctx context.Context, videoID uint64, delta int) error {
	if delta >= 0 {
		return s.db.WithContext(ctx).Model(&video.Video{}).Where("id = ?", videoID).
			UpdateColumn("fav_count", gorm.Expr("fav_count + ?", delta)).Error
	}
	return s.db.WithContext(ctx).Model(&video.Video{}).Where("id = ?", videoID).
		UpdateColumn("fav_count", gorm.Expr("CASE WHEN fav_count - ? < 0 THEN 0 ELSE fav_count - ? END", -delta, -delta)).Error
}

// UserFavoriteCount returns the number of times a user has favorited a specific video.
func (s *EngagementService) UserFavoriteCount(ctx context.Context, userID, videoID uint64) (int64, error) {
	var cnt int64
	err := s.db.WithContext(ctx).Model(&video.VideoFavorite{}).Where("user_id = ? AND video_id = ?", userID, videoID).Count(&cnt).Error
	return cnt, err
}
// WatchLaterVideoItem holds a watch-later entry with video details.
type WatchLaterVideoItem struct {
	ID                uint64
	Title             string
	CoverURL          string
	PlayCount         uint64
	Duration          float64
	UploaderName      string
	UploaderAvatar    string
	CreatedAt         string
	Watched           bool
}

// ListWatchLaterWithVideos returns watch-later items with video details.
func (s *EngagementService) ListWatchLaterWithVideos(ctx context.Context, userID uint64, page, pageSize int) ([]WatchLaterVideoItem, int64, error) {
	list, total, err := s.ListWatchLater(ctx, userID, page, pageSize)
	if err != nil { return nil, 0, err }
	if len(list) == 0 { return []WatchLaterVideoItem{}, total, nil }
	
	vids := make([]uint64, 0, len(list))
	for _, wl := range list { vids = append(vids, wl.VideoID) }
	
	vmap := make(map[uint64]*VideoInfo)
	uids := make([]uint64, 0)
	uidSeen := make(map[uint64]struct{})
	if s.videos != nil {
		var verr error
		vmap, verr = s.videos.BatchGetPublishedVideos(ctx, vids)
		if verr != nil { return nil, 0, verr }
		for _, vi := range vmap {
			if _, ok := uidSeen[vi.UserID]; !ok {
				uidSeen[vi.UserID] = struct{}{}
				uids = append(uids, vi.UserID)
			}
		}
	} else {
		var videos []video.Video
		if err := s.db.WithContext(ctx).Where("id IN ? AND status = ?", vids, "published").Find(&videos).Error; err != nil {
			return nil, 0, err
		}
		for i := range videos {
			v := &videos[i]
			vmap[v.ID] = &VideoInfo{ID: v.ID, UserID: v.UserID, Title: v.Title, CoverURL: v.CoverURL,
				PlayCount: v.PlayCount, DurationSec: v.DurationSec, DanmakuClosed: v.DanmakuClosed, CreatedAt: v.CreatedAt}
			if _, ok := uidSeen[v.UserID]; !ok {
				uidSeen[v.UserID] = struct{}{}
				uids = append(uids, v.UserID)
			}
		}
	}
	
	users_u := make(map[uint64]UserInfo)
	if s.users != nil && len(uids) > 0 {
		umap, err := s.users.GetUsersByIDs(ctx, uids)
		if err == nil { users_u = umap }
	}
	
	items := make([]WatchLaterVideoItem, 0, len(list))
	for _, wl := range list {
		vi, ok := vmap[wl.VideoID]
		if !ok { continue }
		u, _ := users_u[vi.UserID]
		items = append(items, WatchLaterVideoItem{
			ID: vi.ID, Title: vi.Title, CoverURL: vi.CoverURL,
			PlayCount: vi.PlayCount, Duration: vi.DurationSec,
			UploaderName: u.Nickname, UploaderAvatar: u.AvatarURL,
			CreatedAt: vi.CreatedAt.Format("2006-01-02 15:04:05"),
			Watched: wl.Watched,
		})
	}
	return items, total, nil
}

// CoinedVideoItem holds a coin record with video details.
type CoinedVideoItem struct {
	ID                uint64
	Title             string
	CoverURL          string
	PlayCount         uint64
	DanmakuCount      uint64
	CommentCount      uint64
	Duration          float64
	UploaderName      string
	UploaderAvatar    string
	CreatedAt         string
	CoinedAt          string
}

// ListUserCoinedVideos returns coin records with video details for a user.
func (s *EngagementService) ListUserCoinedVideos(ctx context.Context, ownerID uint64, limit int) ([]CoinedVideoItem, int64, error) {
	var coins []video.VideoCoin
	if err := s.db.WithContext(ctx).Where("user_id = ?", ownerID).Order("created_at DESC").Limit(limit).Find(&coins).Error; err != nil {
		return nil, 0, err
	}
	var total int64
	_ = s.db.WithContext(ctx).Model(&video.VideoCoin{}).Where("user_id = ?", ownerID).Count(&total).Error
	if len(coins) == 0 { return []CoinedVideoItem{}, total, nil }
	
	vids := make([]uint64, 0, len(coins))
	seen := make(map[uint64]struct{}, len(coins))
	for i := range coins {
		vid := coins[i].VideoID
		if _, ok := seen[vid]; ok { continue }
		seen[vid] = struct{}{}
		vids = append(vids, vid)
	}
	
	vmap := make(map[uint64]*VideoInfo)
	uids := make([]uint64, 0)
	uidSeen := make(map[uint64]struct{})
	if s.videos != nil {
		var verr error
		vmap, verr = s.videos.BatchGetPublishedVideos(ctx, vids)
		if verr != nil { return nil, 0, verr }
		for _, vi := range vmap {
			if _, ok := uidSeen[vi.UserID]; !ok {
				uidSeen[vi.UserID] = struct{}{}
				uids = append(uids, vi.UserID)
			}
		}
	} else {
		var videos []video.Video
		if err := s.db.WithContext(ctx).Where("id IN ? AND status = ?", vids, "published").Find(&videos).Error; err != nil {
			return nil, 0, err
		}
		for i := range videos {
			v := &videos[i]
			vmap[v.ID] = &VideoInfo{ID: v.ID, UserID: v.UserID, Title: v.Title, CoverURL: v.CoverURL,
				PlayCount: v.PlayCount, DurationSec: v.DurationSec, DanmakuClosed: v.DanmakuClosed, CreatedAt: v.CreatedAt}
			if _, ok := uidSeen[v.UserID]; !ok {
				uidSeen[v.UserID] = struct{}{}
				uids = append(uids, v.UserID)
			}
		}
	}
	
	users_u := make(map[uint64]UserInfo)
	if s.users != nil && len(uids) > 0 {
		umap, err := s.users.GetUsersByIDs(ctx, uids)
		if err == nil { users_u = umap }
	}
	
	items := make([]CoinedVideoItem, 0, len(coins))
	for i := range coins {
		vi, ok := vmap[coins[i].VideoID]
		if !ok { continue }
		u, _ := users_u[vi.UserID]
		items = append(items, CoinedVideoItem{
			ID: vi.ID, Title: vi.Title, CoverURL: vi.CoverURL,
			PlayCount: vi.PlayCount, DanmakuCount: vi.DanmakuCount, CommentCount: vi.CommentCount,
			Duration: vi.DurationSec, UploaderName: u.Nickname, UploaderAvatar: u.AvatarURL,
			CreatedAt: vi.CreatedAt.Format("2006-01-02 15:04:05"),
			CoinedAt: coins[i].CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}
	return items, total, nil
}

