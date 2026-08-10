package danmaku

import (
	"cakecake/internal/model/danmaku"
	"cakecake/internal/model/user"
	"cakecake/internal/model/video"
	"cakecake/internal/service"
	"context"
	"time"
	"unicode/utf8"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"cakecake/internal/data"
	"cakecake/internal/errcode"
	"cakecake/internal/pkg/sensitive"
)

// DanmakuService handles danmaku business logic.
type DanmakuService struct {
	store DanmakuStore
	rdb   *redis.Client
	log   *zap.Logger
	sens  *sensitive.Filter
}

// NewDanmakuService creates a DanmakuService.
func NewDanmakuService(db *gorm.DB, rdb *redis.Client, log *zap.Logger, sens *sensitive.Filter) *DanmakuService {
	return &DanmakuService{store: NewDanmakuStore(db), rdb: rdb, log: log, sens: sens}
}

// DanmakuPostResult is returned after a danmaku is created.
type DanmakuPostResult struct {
	Danmaku  *danmaku.Danmaku
	User     *user.User
	Cooldown bool
}

// PostDanmaku handles all business logic for posting a danmaku.
// Returns nil, *service.SvcError on validation/business failures.
func (s *DanmakuService) PostDanmaku(ctx context.Context, videoID, userID uint64, content, color, danmakuType, fontSize string, videoTime float64) (*DanmakuPostResult, error) {
	// Validate video exists and is published
	v, err := s.store.GetVideoByID(ctx, videoID)
	if err != nil || v.Status != video.StatusPublished {
		return nil, &service.SvcError{Code: 40400, Msg: "video not found"}
	}
	if v.DanmakuClosed {
		return nil, &service.SvcError{Code: 40304, Msg: "danmaku closed"}
	}

	// Validate content length
	if n := utf8.RuneCountInString(content); n < 1 || n > 100 {
		return nil, service.ErrParamError
	}

	// Cooldown check
	key := data.DanmakuCooldownKey(userID, videoID)
	okSet, err := s.rdb.SetNX(ctx, key, "1", 5*time.Second).Result()
	if err != nil {
		s.log.Error("redis cooldown", zap.Error(err))
		return nil, service.ErrInternalError
	}
	if !okSet {
		return nil, &service.SvcError{Code: errcode.CodeDanmakuCooldown, Msg: "danmaku cooldown"}
	}

	// Sensitive content check
	if err := s.sens.Check(content); err != nil {
		s.rdb.Del(ctx, key)
		if _, ok := err.(sensitive.ErrBlocked); ok {
			return nil, &service.SvcError{Code: errcode.CodeDanmakuSensitive, Msg: "danmaku sensitive"}
		}
		s.log.Error("sensitive check", zap.Error(err))
		return nil, service.ErrInternalError
	}

	// Create danmaku
	d := danmaku.Danmaku{
		VideoID:   videoID,
		UserID:    userID,
		Content:   content,
		Color:     color,
		Type:      danmakuType,
		FontSize:  fontSize,
		VideoTime: videoTime,
	}
	if err := s.store.CreateDanmaku(ctx, &d); err != nil {
		s.rdb.Del(ctx, key)
		return nil, service.ErrInternalError
	}

	// Increment danmaku count
	_ = s.store.IncrDanmakuCount(ctx, videoID)

	// Load user for display name
	u, _ := s.store.GetUserByID(ctx, userID)
	if u == nil {
		u = &user.User{}
	}

	return &DanmakuPostResult{Danmaku: &d, User: u}, nil
}

// ToggleDanmakuLikeResult is returned after toggling a danmaku like.
type ToggleDanmakuLikeResult struct {
	Liked     bool
	LikeCount uint64
}

// ToggleDanmakuLike toggles the like status for a danmaku.
func (s *DanmakuService) ToggleDanmakuLike(ctx context.Context, danmakuID, userID uint64) (*ToggleDanmakuLikeResult, error) {
	if _, err := s.store.GetDanmakuByID(ctx, danmakuID); err != nil {
		return nil, &service.SvcError{Code: 40400, Msg: "danmaku not found"}
	}
	liked, err := s.store.HasDanmakuLike(ctx, userID, danmakuID)
	if err != nil {
		return nil, service.ErrInternalError
	}
	if !liked {
		if err := s.store.CreateDanmakuLike(ctx, userID, danmakuID); err != nil {
			return nil, service.ErrInternalError
		}
		_ = s.store.IncrDanmakuLikeCount(ctx, danmakuID, 1)
	} else {
		if err := s.store.DeleteDanmakuLike(ctx, userID, danmakuID); err != nil {
			return nil, service.ErrInternalError
		}
		_ = s.store.IncrDanmakuLikeCount(ctx, danmakuID, -1)
	}
	var count uint64
	if dm, err := s.store.GetDanmakuByID(ctx, danmakuID); err == nil {
		count = dm.LikeCount
	}
	return &ToggleDanmakuLikeResult{Liked: !liked, LikeCount: count}, nil
}

// GetVideoAndUser loads basic video and user info (for handler convenience).
func (s *DanmakuService) GetVideoAndUser(ctx context.Context, videoID uint64) (*video.Video, *user.User, error) {
	v, err := s.store.GetVideoByID(ctx, videoID)
	if err != nil {
		return nil, nil, err
	}
	u, err := s.store.GetUserByID(ctx, v.UserID)
	if err != nil {
		return nil, nil, err
	}
	return v, u, nil
}

// ListCreatorDanmakusResult holds the result of listing creator danmakus.
type ListCreatorDanmakusResult struct {
	Items []CreatorDanmakuItem
	Total int64
	Limit int
}

// CreatorDanmakuItem represents a danmaku with related data for creator view.
type CreatorDanmakuItem struct {
	ID         uint64
	VideoID    uint64
	UserID     uint64
	Content    string
	Color      string
	Type       string
	VideoTime  float64
	LikeCount  int64
	CreatedAt  string
	LikedByMe  bool
	VideoTitle string
	CoverURL   string
	Username   string
}

// ListCreatorDanmakus lists danmaku on the authenticated uploader's videos (????).
func (s *DanmakuService) ListCreatorDanmakus(ctx context.Context, uid uint64, limit int, keyword, typeFilter string, filterVideoID uint64, viewerID uint64) (*ListCreatorDanmakusResult, error) {
	return s.store.ListCreatorDanmakus(ctx, uid, limit, keyword, typeFilter, filterVideoID, viewerID)
}

// DeleteCreatorDanmaku removes one danmaku on the uploader's video (????).
func (s *DanmakuService) DeleteCreatorDanmaku(ctx context.Context, uid, danmakuID uint64) (*danmaku.Danmaku, error) {
	d, err := s.store.GetDanmakuByID(ctx, danmakuID)
	if err != nil {
		return nil, err
	}
	v, err := s.store.GetVideoByID(ctx, d.VideoID)
	if err != nil {
		return nil, err
	}
	if v.UserID != uid {
		return nil, service.ErrForbidden
	}
	if err := s.store.DeleteDanmakuCascade(ctx, d, v.ID); err != nil {
		return nil, err
	}
	return d, nil
}

// ListHistory returns danmaku history for a given video, optionally filtered by currentTime.
func (s *DanmakuService) ListHistory(ctx context.Context, videoID uint64, currentTime float64, limit int) ([]danmaku.Danmaku, error) {
	return s.store.ListHistory(ctx, videoID, currentTime, limit)
}
