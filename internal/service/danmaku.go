package service

import (
	"minibili/internal/model/danmaku"
	"minibili/internal/model/user"
	"minibili/internal/model/video"
	"context"
	"time"
	"unicode/utf8"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"minibili/internal/data"
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
	Danmaku  *danmaku.Danmaku
	User     *user.User
	Cooldown bool
}

// PostDanmaku handles all business logic for posting a danmaku.
// Returns nil, *SvcError on validation/business failures.
func (s *DanmakuService) PostDanmaku(ctx context.Context, videoID, userID uint64, content, color, danmakuType, fontSize string, videoTime float64) (*DanmakuPostResult, error) {
	// Validate video exists and is published
	var v video.Video
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
	d := danmaku.Danmaku{
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
	_ = s.db.WithContext(ctx).Model(&video.Video{}).Where("id = ?", videoID).UpdateColumn("danmaku_count", gorm.Expr("danmaku_count + ?", 1)).Error

	// Load user for display name
	var u user.User
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
	var d danmaku.Danmaku
	if err := s.db.WithContext(ctx).First(&d, danmakuID).Error; err != nil {
		return nil, &SvcError{Code: 40400, Msg: "danmaku not found"}
	}
	var like danmaku.DanmakuLike
	res := s.db.WithContext(ctx).Where("user_id = ? AND danmaku_id = ?", userID, danmakuID).Limit(1).Find(&like)
	if res.Error != nil {
		return nil, ErrInternalError
	}
	if res.RowsAffected == 0 {
		// Add like
		like = danmaku.DanmakuLike{UserID: userID, DanmakuID: danmakuID}
		if err := s.db.WithContext(ctx).Create(&like).Error; err != nil {
			return nil, ErrInternalError
		}
		_ = s.db.WithContext(ctx).Model(&d).UpdateColumn("like_count", gorm.Expr("like_count + ?", 1)).Error
		var dm danmaku.Danmaku
		_ = s.db.WithContext(ctx).First(&dm, danmakuID).Error
		return &ToggleDanmakuLikeResult{Liked: true, LikeCount: dm.LikeCount}, nil
	}
	// Remove like
	if err := s.db.WithContext(ctx).Delete(&like).Error; err != nil {
		return nil, ErrInternalError
	}
	_ = s.db.WithContext(ctx).Model(&d).UpdateColumn("like_count", gorm.Expr("CASE WHEN like_count - ? < 0 THEN 0 ELSE like_count - ? END", 1, 1)).Error
	var dm danmaku.Danmaku
	_ = s.db.WithContext(ctx).First(&dm, danmakuID).Error
	return &ToggleDanmakuLikeResult{Liked: false, LikeCount: dm.LikeCount}, nil
}

// GetVideoAndUser loads basic video and user info (for handler convenience).
func (s *DanmakuService) GetVideoAndUser(ctx context.Context, videoID uint64) (*video.Video, *user.User, error) {
	var v video.Video
	if err := s.db.WithContext(ctx).First(&v, videoID).Error; err != nil {
		return nil, nil, err
	}
	var u user.User
	if err := s.db.WithContext(ctx).First(&u, v.UserID).Error; err != nil {
		return nil, nil, err
	}
	return &v, &u, nil
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
	base := s.db.WithContext(ctx).Model(&danmaku.Danmaku{}).
		Joins("INNER JOIN videos ON videos.id = danmakus.video_id AND videos.user_id = ?", uid)
	if filterVideoID > 0 {
		var owned video.Video
		if err := s.db.WithContext(ctx).Where("id = ? AND user_id = ?", filterVideoID, uid).First(&owned).Error; err != nil {
			return nil, err
		}
		base = base.Where("danmakus.video_id = ?", filterVideoID)
	}
	if keyword != "" {
		like := "%" + keyword + "%"
		base = base.Where("danmakus.content LIKE ?", like)
	}
	switch typeFilter {
	case "scroll", "top", "bottom":
		base = base.Where("danmakus.type = ?", typeFilter)
	}

	var total int64
	if err := base.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, err
	}

	var list []danmaku.Danmaku
	if err := base.Order("danmakus.id DESC").Limit(limit).Find(&list).Error; err != nil {
		return nil, err
	}

	// Batch load videos
	videoIDs := make([]uint64, 0, len(list))
	userIDs := make([]uint64, 0, len(list))
	for _, d := range list {
		videoIDs = append(videoIDs, d.VideoID)
		userIDs = append(userIDs, d.UserID)
	}

	videos := map[uint64]video.Video{}
	if len(videoIDs) > 0 {
		var vlist []video.Video
		_ = s.db.WithContext(ctx).Where("id IN ?", videoIDs).Find(&vlist).Error
		for i := range vlist {
			videos[vlist[i].ID] = vlist[i]
		}
	}

	usernames := map[uint64]string{}
	if len(userIDs) > 0 {
		var users []user.User
		_ = s.db.WithContext(ctx).Where("id IN ?", userIDs).Find(&users).Error
		for i := range users {
			usernames[users[i].ID] = user.DisplayUsername(&users[i])
		}
	}

	// Batch load like status
	danmakuIDs := make([]uint64, 0, len(list))
	for _, d := range list {
		danmakuIDs = append(danmakuIDs, d.ID)
	}
	likedByViewer := map[uint64]bool{}
	if viewerID > 0 && len(danmakuIDs) > 0 {
		var likes []danmaku.DanmakuLike
		_ = s.db.WithContext(ctx).Where("user_id = ? AND danmaku_id IN ?", viewerID, danmakuIDs).Find(&likes).Error
		for _, lk := range likes {
			likedByViewer[lk.DanmakuID] = true
		}
	}

	items := make([]CreatorDanmakuItem, 0, len(list))
	for _, d := range list {
		v := videos[d.VideoID]
		items = append(items, CreatorDanmakuItem{
			ID: d.ID, VideoID: d.VideoID, UserID: d.UserID,
			Content: d.Content, Color: d.Color, Type: d.Type,
			VideoTime: d.VideoTime, LikeCount: int64(d.LikeCount),
			CreatedAt: d.CreatedAt.Format("2006-01-02 15:04:05"),
			LikedByMe: likedByViewer[d.ID],
			VideoTitle: v.Title, CoverURL: v.CoverURL,
			Username: usernames[d.UserID],
		})
	}

	if total > int64(limit) {
		total = int64(limit)
	}
	return &ListCreatorDanmakusResult{Items: items, Total: total, Limit: limit}, nil
}

// DeleteCreatorDanmaku removes one danmaku on the uploader's video (????).
func (s *DanmakuService) DeleteCreatorDanmaku(ctx context.Context, uid, danmakuID uint64) (*danmaku.Danmaku, error) {
	var d danmaku.Danmaku
	if err := s.db.WithContext(ctx).First(&d, danmakuID).Error; err != nil {
		return nil, err
	}
	var v video.Video
	if err := s.db.WithContext(ctx).First(&v, d.VideoID).Error; err != nil {
		return nil, err
	}
	if v.UserID != uid {
		return nil, ErrForbidden
	}
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.WithContext(ctx).Where("danmaku_id = ?", danmakuID).Delete(&danmaku.DanmakuLike{}).Error; err != nil {
			return err
		}
		if err := tx.WithContext(ctx).Delete(&d).Error; err != nil {
			return err
		}
		return tx.WithContext(ctx).Model(&video.Video{}).Where("id = ?", v.ID).
			UpdateColumn("danmaku_count", gorm.Expr("CASE WHEN danmaku_count - ? < 0 THEN 0 ELSE danmaku_count - ? END", 1, 1)).Error
	})
	if err != nil {
		return nil, err
	}
	return &d, nil
}

// ListHistory returns danmaku history for a given video, optionally filtered by currentTime.
func (s *DanmakuService) ListHistory(ctx context.Context, videoID uint64, currentTime float64, limit int) ([]danmaku.Danmaku, error) {
	query := s.db.WithContext(ctx).Where("video_id = ?", videoID)
	if currentTime > 0 {
		query = query.Where("video_time BETWEEN ? AND ?", currentTime-10, currentTime+2)
	}
	query = query.Order("video_time ASC").Limit(limit)
	var hist []danmaku.Danmaku
	if err := query.Find(&hist).Error; err != nil {
		return nil, err
	}
	return hist, nil
}

