package engagement

import (
	"cakecake/internal/model/video"
	"cakecake/internal/service"
	vsvc "cakecake/internal/service/video"
	"context"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"cakecake/internal/pkg/dailyreward"
	"cakecake/internal/pkg/usercoin"
)

// EngagementService handles video engagement operations (coin, watch later).
type EngagementService struct {
	store EngagementStore
	rdb   *redis.Client
	log   *zap.Logger

	users  service.UserProvider
	videos vsvc.VideoProvider
}

func NewEngagementService(db *gorm.DB, rdb *redis.Client, log *zap.Logger, users service.UserProvider, videos vsvc.VideoProvider) *EngagementService {
	return &EngagementService{store: NewEngagementStore(db), rdb: rdb, log: log, users: users, videos: videos}
}

// HasCoined checks if user already coined a video.
func (s *EngagementService) HasCoined(ctx context.Context, userID, videoID uint64) bool {
	return s.store.HasCoined(ctx, userID, videoID)
}

// GetUserCoinBalance returns the user current coin balance.
func (s *EngagementService) GetUserCoinBalance(ctx context.Context, userID uint64) int64 {
	if s.users == nil {
		return 0
	}
	u, err := s.users.GetUser(ctx, userID)
	if err != nil {
		return 0
	}
	return u.CoinBalanceTenths
}

// DecrementUserCoins subtracts coins from user balance.
func (s *EngagementService) DecrementUserCoins(ctx context.Context, userID uint64, amount int) error {
	if s.users == nil {
		return s.store.DecrementUserCoinsFallback(ctx, userID, amount)
	}
	return s.users.DecrementCoins(ctx, userID, amount)
}

// IncrementVideoCoinCount increments the coin count on a video.
func (s *EngagementService) IncrementVideoCoinCount(ctx context.Context, videoID uint64, delta int) error {
	return s.store.IncrVideoCoinCount(ctx, videoID, delta)
}

// ToggleWatchLater adds or removes a watch-later entry.
func (s *EngagementService) ToggleWatchLater(ctx context.Context, userID, videoID uint64) (bool, error) {
	return s.store.ToggleWatchLater(ctx, userID, videoID)
}

// ListWatchLater returns watch-later entries with pagination.
func (s *EngagementService) ListWatchLater(ctx context.Context, userID uint64, page, pageSize int) ([]video.WatchLater, int64, error) {
	return s.store.ListWatchLater(ctx, userID, page, pageSize)
}

// ClearWatchLater removes all watch-later entries for a user.
func (s *EngagementService) ClearWatchLater(ctx context.Context, userID uint64) error {
	return s.store.ClearWatchLater(ctx, userID)
}

// ClearWatchedWatchLater removes watched watch-later entries.
func (s *EngagementService) ClearWatchedWatchLater(ctx context.Context, userID uint64) error {
	return s.store.ClearWatchedWatchLater(ctx, userID)
}

// MarkWatchLaterWatched marks a watch-later entry as watched.
func (s *EngagementService) MarkWatchLaterWatched(ctx context.Context, userID, videoID uint64) error {
	return s.store.MarkWatchLaterWatched(ctx, userID, videoID)
}

// BatchHasCoined returns a map of video_id -> coined for a user.
func (s *EngagementService) BatchHasCoined(ctx context.Context, userID uint64, videoIDs []uint64) map[uint64]bool {
	return s.store.BatchHasCoined(ctx, userID, videoIDs)
}

// BatchWatchLater returns a map of video_id -> in-watch-later for a user.
func (s *EngagementService) BatchWatchLater(ctx context.Context, userID uint64, videoIDs []uint64) map[uint64]bool {
	return s.store.BatchWatchLater(ctx, userID, videoIDs)
}

// BatchCoinedByUser returns a map of video_id -> coin amount for a user.
func (s *EngagementService) BatchCoinedByUser(ctx context.Context, userID uint64, videoIDs []uint64) map[uint64]int {
	return s.store.BatchCoinedByUser(ctx, userID, videoIDs)
}

// PostVideoCoinResult holds the result of a coin operation.
type PostVideoCoinResult struct {
	Coined        bool
	CoinCount     uint64
	Amount        int
	MyCoinAmount  int
	CoinBalance   float64
	DailyProgress int
	DailyMax      int
}

// PostVideoCoin performs the full coin transaction for a user.
func (s *EngagementService) PostVideoCoin(ctx context.Context, uid, vid, uploaderID uint64, amount int) (*PostVideoCoinResult, error) {
	exist, hasCoin, err := s.store.GetVideoCoinRow(ctx, uid, vid)
	if err != nil {
		return nil, err
	}
	coinBefore := s.store.CoinProgress(uid)
	var spentAmount int
	var myCoinAmount int

	if hasCoin {
		if exist.Amount >= 2 {
			return nil, nil
		}
		spentAmount = 1
		myCoinAmount = 2
		if err := s.store.UpdateVideoCoinTx(ctx, uid, uploaderID, vid, exist); err != nil {
			return nil, err
		}
	} else {
		if amount != 1 && amount != 2 {
			amount = 1
		}
		spentAmount = amount
		myCoinAmount = amount
		if err := s.store.CreateVideoCoinTx(ctx, uid, uploaderID, vid, amount); err != nil {
			return nil, err
		}
	}

	coinAfter := s.store.CoinProgress(uid)
	_ = s.store.GrantCoinExp(uid, coinBefore, coinAfter)
	var coinCount uint64
	if v, err := s.store.GetVideoByID(ctx, vid); err == nil {
		coinCount = v.CoinCount
	}
	var balance int64
	if viewer, err := s.store.GetUserByID(ctx, uid); err == nil {
		balance = viewer.CoinBalanceTenths
	}
	return &PostVideoCoinResult{
		Coined: true, CoinCount: coinCount, Amount: spentAmount,
		MyCoinAmount: myCoinAmount, CoinBalance: usercoin.BalanceFloat(balance),
		DailyProgress: coinAfter, DailyMax: dailyreward.ExpCoinMax,
	}, nil
}

// BatchVideoLikes returns a map of video_id -> liked for a user.
func (s *EngagementService) BatchVideoLikes(ctx context.Context, userID uint64, videoIDs []uint64) map[uint64]bool {
	return s.store.BatchVideoLikes(ctx, userID, videoIDs)
}

// WatchLaterVideoItem holds a watch-later entry with video details.
type WatchLaterVideoItem struct {
	ID             uint64  `json:"id"`
	Title          string  `json:"title"`
	CoverURL       string  `json:"cover_url"`
	PlayCount      uint64  `json:"play_count"`
	DanmakuCount   uint64  `json:"danmaku_count"`
	Duration       float64 `json:"duration"`
	UploaderName   string  `json:"uploader"`
	UploaderAvatar string  `json:"uploader_avatar_url"`
	UploaderID     uint64  `json:"uploader_id"`
	CreatedAt      string  `json:"created_at"`
	AddedAt        string  `json:"added_at"`
	Watched        bool    `json:"watched"`
}

// ListWatchLaterWithVideos returns watch-later items with video details.
func (s *EngagementService) ListWatchLaterWithVideos(ctx context.Context, userID uint64, page, pageSize int) ([]WatchLaterVideoItem, int64, error) {
	list, total, err := s.ListWatchLater(ctx, userID, page, pageSize)
	if err != nil {
		return nil, 0, err
	}
	if len(list) == 0 {
		return []WatchLaterVideoItem{}, total, nil
	}

	vids := make([]uint64, 0, len(list))
	for _, wl := range list {
		vids = append(vids, wl.VideoID)
	}

	vmap := make(map[uint64]*vsvc.VideoInfo)
	uids := make([]uint64, 0)
	uidSeen := make(map[uint64]struct{})
	if s.videos != nil {
		var verr error
		vmap, verr = s.videos.BatchGetPublishedVideos(ctx, vids)
		if verr != nil {
			return nil, 0, verr
		}
		for _, vi := range vmap {
			if _, ok := uidSeen[vi.UserID]; !ok {
				uidSeen[vi.UserID] = struct{}{}
				uids = append(uids, vi.UserID)
			}
		}
	} else {
		videos, err := s.store.BatchPublishedVideosRaw(ctx, vids)
		if err != nil {
			return nil, 0, err
		}
		for id, v := range videos {
			vv := v
			vmap[id] = &vsvc.VideoInfo{ID: vv.ID, UserID: vv.UserID, Title: vv.Title, CoverURL: vv.CoverURL,
				PlayCount: vv.PlayCount, DurationSec: vv.DurationSec, DanmakuClosed: vv.DanmakuClosed, CreatedAt: vv.CreatedAt}
			if _, ok := uidSeen[vv.UserID]; !ok {
				uidSeen[vv.UserID] = struct{}{}
				uids = append(uids, vv.UserID)
			}
		}
	}

	users_u := make(map[uint64]service.UserInfo)
	if s.users != nil && len(uids) > 0 {
		umap, err := s.users.GetUsersByIDs(ctx, uids)
		if err == nil {
			users_u = umap
		}
	}

	items := make([]WatchLaterVideoItem, 0, len(list))
	for _, wl := range list {
		vi, ok := vmap[wl.VideoID]
		if !ok {
			continue
		}
		u, _ := users_u[vi.UserID]
		items = append(items, WatchLaterVideoItem{
			ID:             vi.ID,
			Title:          vi.Title,
			CoverURL:       vi.CoverURL,
			PlayCount:      vi.PlayCount,
			DanmakuCount:   vi.DanmakuCount,
			Duration:       vi.DurationSec,
			UploaderName:   u.Nickname,
			UploaderAvatar: u.AvatarURL,
			UploaderID:     vi.UserID,
			CreatedAt:      vi.CreatedAt.Format("2006-01-02 15:04:05"),
			AddedAt:        wl.CreatedAt.Format("2006-01-02 15:04:05"),
			Watched:        wl.Watched,
		})
	}
	return items, total, nil
}

// CoinedVideoItem holds a coin record with video details.
type CoinedVideoItem struct {
	ID             uint64
	Title          string
	CoverURL       string
	PlayCount      uint64
	DanmakuCount   uint64
	CommentCount   uint64
	Duration       float64
	UploaderName   string
	UploaderAvatar string
	CreatedAt      string
	CoinedAt       string
}

// ListUserCoinedVideos returns coin records with video details for a user.
func (s *EngagementService) ListUserCoinedVideos(ctx context.Context, ownerID uint64, limit int) ([]CoinedVideoItem, int64, error) {
	coins, total, err := s.store.ListUserCoinedVideosRows(ctx, ownerID, limit)
	if err != nil {
		return nil, 0, err
	}
	if len(coins) == 0 {
		return []CoinedVideoItem{}, total, nil
	}

	vids := make([]uint64, 0, len(coins))
	seen := make(map[uint64]struct{}, len(coins))
	for i := range coins {
		vid := coins[i].VideoID
		if _, ok := seen[vid]; ok {
			continue
		}
		seen[vid] = struct{}{}
		vids = append(vids, vid)
	}

	vmap := make(map[uint64]*vsvc.VideoInfo)
	uids := make([]uint64, 0)
	uidSeen := make(map[uint64]struct{})
	if s.videos != nil {
		var verr error
		vmap, verr = s.videos.BatchGetPublishedVideos(ctx, vids)
		if verr != nil {
			return nil, 0, verr
		}
		for _, vi := range vmap {
			if _, ok := uidSeen[vi.UserID]; !ok {
				uidSeen[vi.UserID] = struct{}{}
				uids = append(uids, vi.UserID)
			}
		}
	} else {
		videos, err := s.store.BatchPublishedVideosRaw(ctx, vids)
		if err != nil {
			return nil, 0, err
		}
		for id, v := range videos {
			vv := v
			vmap[id] = &vsvc.VideoInfo{ID: vv.ID, UserID: vv.UserID, Title: vv.Title, CoverURL: vv.CoverURL,
				PlayCount: vv.PlayCount, DurationSec: vv.DurationSec, DanmakuClosed: vv.DanmakuClosed, CreatedAt: vv.CreatedAt}
			if _, ok := uidSeen[vv.UserID]; !ok {
				uidSeen[vv.UserID] = struct{}{}
				uids = append(uids, vv.UserID)
			}
		}
	}

	users_u := make(map[uint64]service.UserInfo)
	if s.users != nil && len(uids) > 0 {
		umap, err := s.users.GetUsersByIDs(ctx, uids)
		if err == nil {
			users_u = umap
		}
	}

	items := make([]CoinedVideoItem, 0, len(coins))
	for i := range coins {
		vi, ok := vmap[coins[i].VideoID]
		if !ok {
			continue
		}
		u, _ := users_u[vi.UserID]
		items = append(items, CoinedVideoItem{
			ID: vi.ID, Title: vi.Title, CoverURL: vi.CoverURL,
			PlayCount: vi.PlayCount, DanmakuCount: vi.DanmakuCount, CommentCount: vi.CommentCount,
			Duration: vi.DurationSec, UploaderName: u.Nickname, UploaderAvatar: u.AvatarURL,
			CreatedAt: vi.CreatedAt.Format("2006-01-02 15:04:05"),
			CoinedAt:  coins[i].CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}
	return items, total, nil
}
