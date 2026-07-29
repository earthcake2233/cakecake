package service

import (
	"context"
	"time"
	"unicode/utf8"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"minibili/internal/data"
	"minibili/internal/model"
	"minibili/internal/pkg/sensitive"
)

// DanmakuService handles danmaku business logic.
type DanmakuService struct {
	db   *gorm.DB
	rdb  *redis.Client
	log  *zap.Logger
	sens *sensitive.Filter
}

// NewDanmakuService creates a DanmakuService.
func NewDanmakuService(db *gorm.DB, rdb *redis.Client, log *zap.Logger, sens *sensitive.Filter) *DanmakuService {
	return &DanmakuService{db: db, rdb: rdb, log: log, sens: sens}
}

// DanmakuPostResult is returned after a danmaku is created.
type DanmakuPostResult struct {
	Danmaku  *model.Danmaku
	User     *model.User
	Cooldown bool
}

// PostDanmaku handles all business logic for posting a danmaku.
// Returns nil, *SvcError on validation/business failures.
func (s *DanmakuService) PostDanmaku(ctx context.Context, videoID, userID uint64, content, color, danmakuType, fontSize string, videoTime float64) (*DanmakuPostResult, error) {
	// Validate video exists and is published
	var v model.Video
	if err := s.db.WithContext(ctx).First(&v, videoID).Error; err != nil || v.Status != "published" {
		return nil, &SvcError{Code: 40400, Msg: "video not found"}
	}
	if v.DanmakuClosed {
		return nil, &SvcError{Code: 40304, Msg: "danmaku closed"}
	}

	// Validate content length
	if n := utf8.RuneCountInString(content); n < 1 || n > 100 {
		return nil, ErrParamError
	}

	// Cooldown check
	key := data.DanmakuCooldownKey(userID, videoID)
	okSet, err := s.rdb.SetNX(ctx, key, "1", 5*time.Second).Result()
	if err != nil {
		s.log.Error("redis cooldown", zap.Error(err))
		return nil, ErrInternalError
	}
	if !okSet {
		return nil, &SvcError{Code: 40025, Msg: "danmaku cooldown"}
	}

	// Sensitive content check
	if err := s.sens.Check(content); err != nil {
		s.rdb.Del(ctx, key)
		if _, ok := err.(sensitive.ErrBlocked); ok {
			return nil, &SvcError{Code: 40022, Msg: "danmaku sensitive"}
		}
		s.log.Error("sensitive check", zap.Error(err))
		return nil, ErrInternalError
	}

	// Create danmaku
	d := model.Danmaku{
		VideoID:   videoID,
		UserID:    userID,
		Content:   content,
		Color:     color,
		Type:      danmakuType,
		FontSize:  fontSize,
		VideoTime: videoTime,
	}
	if err := s.db.WithContext(ctx).Create(&d).Error; err != nil {
		s.rdb.Del(ctx, key)
		return nil, ErrInternalError
	}

	// Increment danmaku count
	_ = s.db.WithContext(ctx).Model(&model.Video{}).Where("id = ?", videoID).UpdateColumn("danmaku_count", gorm.Expr("danmaku_count + ?", 1)).Error

	// Load user for display name
	var u model.User
	_ = s.db.WithContext(ctx).First(&u, userID).Error

	return &DanmakuPostResult{Danmaku: &d, User: &u}, nil
}

// ToggleDanmakuLikeResult is returned after toggling a danmaku like.
type ToggleDanmakuLikeResult struct {
	Liked     bool
	LikeCount uint64
}

// ToggleDanmakuLike toggles the like status for a danmaku.
func (s *DanmakuService) ToggleDanmakuLike(ctx context.Context, danmakuID, userID uint64) (*ToggleDanmakuLikeResult, error) {
	var d model.Danmaku
	if err := s.db.WithContext(ctx).First(&d, danmakuID).Error; err != nil {
		return nil, &SvcError{Code: 40400, Msg: "danmaku not found"}
	}
	var like model.DanmakuLike
	res := s.db.WithContext(ctx).Where("user_id = ? AND danmaku_id = ?", userID, danmakuID).Limit(1).Find(&like)
	if res.Error != nil {
		return nil, ErrInternalError
	}
	if res.RowsAffected == 0 {
		// Add like
		like = model.DanmakuLike{UserID: userID, DanmakuID: danmakuID}
		if err := s.db.WithContext(ctx).Create(&like).Error; err != nil {
			return nil, ErrInternalError
		}
		_ = s.db.WithContext(ctx).Model(&d).UpdateColumn("like_count", gorm.Expr("like_count + ?", 1)).Error
		var dm model.Danmaku
		_ = s.db.WithContext(ctx).First(&dm, danmakuID).Error
		return &ToggleDanmakuLikeResult{Liked: true, LikeCount: dm.LikeCount}, nil
	}
	// Remove like
	if err := s.db.WithContext(ctx).Delete(&like).Error; err != nil {
		return nil, ErrInternalError
	}
	_ = s.db.WithContext(ctx).Model(&d).UpdateColumn("like_count", gorm.Expr("CASE WHEN like_count - ? < 0 THEN 0 ELSE like_count - ? END", 1, 1)).Error
	var dm model.Danmaku
	_ = s.db.WithContext(ctx).First(&dm, danmakuID).Error
	return &ToggleDanmakuLikeResult{Liked: false, LikeCount: dm.LikeCount}, nil
}

// GetVideoAndUser loads basic video and user info (for handler convenience).
func (s *DanmakuService) GetVideoAndUser(ctx context.Context, videoID uint64) (*model.Video, *model.User, error) {
	var v model.Video
	if err := s.db.WithContext(ctx).First(&v, videoID).Error; err != nil {
		return nil, nil, err
	}
	var u model.User
	if err := s.db.WithContext(ctx).First(&u, v.UserID).Error; err != nil {
		return nil, nil, err
	}
	return &v, &u, nil
}
