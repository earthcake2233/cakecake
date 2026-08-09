package danmaku

import (
	"cakecake/internal/model/danmaku"
	"cakecake/internal/model/user"
	"cakecake/internal/model/video"
	"context"

	"gorm.io/gorm"
)

// DanmakuStore is the danmaku-domain storage boundary.
// Phase 1: *gorm.DB impl. Phase 2+: replaced by gRPC client / per-domain store.
type DanmakuStore interface {
	GetVideoByID(ctx context.Context, id uint64) (*video.Video, error)
	GetUserByID(ctx context.Context, id uint64) (*user.User, error)
	GetDanmakuByID(ctx context.Context, id uint64) (*danmaku.Danmaku, error)
	CreateDanmaku(ctx context.Context, d *danmaku.Danmaku) error
	IncrDanmakuCount(ctx context.Context, videoID uint64) error
	HasDanmakuLike(ctx context.Context, userID, danmakuID uint64) (bool, error)
	CreateDanmakuLike(ctx context.Context, userID, danmakuID uint64) error
	DeleteDanmakuLike(ctx context.Context, userID, danmakuID uint64) error
	IncrDanmakuLikeCount(ctx context.Context, danmakuID uint64, delta int) error
	ListCreatorDanmakus(ctx context.Context, uid uint64, limit int, keyword, typeFilter string, filterVideoID uint64, viewerID uint64) (*ListCreatorDanmakusResult, error)
	DeleteDanmakuCascade(ctx context.Context, d *danmaku.Danmaku, videoID uint64) error
	ListHistory(ctx context.Context, videoID uint64, currentTime float64, limit int) ([]danmaku.Danmaku, error)
}

// DanmakuStoreImpl implements DanmakuStore using *gorm.DB (Phase 1 monolith).
type DanmakuStoreImpl struct {
	db *gorm.DB
}

var _ DanmakuStore = (*DanmakuStoreImpl)(nil)

// NewDanmakuStore creates a gorm-backed DanmakuStore implementation.
func NewDanmakuStore(db *gorm.DB) *DanmakuStoreImpl {
	return &DanmakuStoreImpl{db: db}
}

// GetVideoByID loads a video row by id.
func (p *DanmakuStoreImpl) GetVideoByID(ctx context.Context, id uint64) (*video.Video, error) {
	var v video.Video
	if err := p.db.WithContext(ctx).First(&v, id).Error; err != nil {
		return nil, err
	}
	return &v, nil
}

// GetUserByID loads a user row by id.
func (p *DanmakuStoreImpl) GetUserByID(ctx context.Context, id uint64) (*user.User, error) {
	var u user.User
	if err := p.db.WithContext(ctx).First(&u, id).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

// GetDanmakuByID loads a danmaku row by id.
func (p *DanmakuStoreImpl) GetDanmakuByID(ctx context.Context, id uint64) (*danmaku.Danmaku, error) {
	var d danmaku.Danmaku
	if err := p.db.WithContext(ctx).First(&d, id).Error; err != nil {
		return nil, err
	}
	return &d, nil
}

// CreateDanmaku inserts a danmaku row.
func (p *DanmakuStoreImpl) CreateDanmaku(ctx context.Context, d *danmaku.Danmaku) error {
	return p.db.WithContext(ctx).Create(d).Error
}

// IncrDanmakuCount increments a video's danmaku count.
func (p *DanmakuStoreImpl) IncrDanmakuCount(ctx context.Context, videoID uint64) error {
	return p.db.WithContext(ctx).Model(&video.Video{}).Where("id = ?", videoID).
		UpdateColumn("danmaku_count", gorm.Expr("danmaku_count + ?", 1)).Error
}

// HasDanmakuLike reports whether the user liked the danmaku.
func (p *DanmakuStoreImpl) HasDanmakuLike(ctx context.Context, userID, danmakuID uint64) (bool, error) {
	var like danmaku.DanmakuLike
	res := p.db.WithContext(ctx).Where("user_id = ? AND danmaku_id = ?", userID, danmakuID).Limit(1).Find(&like)
	return res.RowsAffected > 0, res.Error
}

// CreateDanmakuLike records a danmaku like.
func (p *DanmakuStoreImpl) CreateDanmakuLike(ctx context.Context, userID, danmakuID uint64) error {
	return p.db.WithContext(ctx).Create(&danmaku.DanmakuLike{UserID: userID, DanmakuID: danmakuID}).Error
}

// DeleteDanmakuLike removes a danmaku like.
func (p *DanmakuStoreImpl) DeleteDanmakuLike(ctx context.Context, userID, danmakuID uint64) error {
	return p.db.WithContext(ctx).Where("user_id = ? AND danmaku_id = ?", userID, danmakuID).Delete(&danmaku.DanmakuLike{}).Error
}

// IncrDanmakuLikeCount adjusts a danmaku's like count by delta.
func (p *DanmakuStoreImpl) IncrDanmakuLikeCount(ctx context.Context, danmakuID uint64, delta int) error {
	if delta < 0 {
		return p.db.WithContext(ctx).Model(&danmaku.Danmaku{}).Where("id = ?", danmakuID).
			UpdateColumn("like_count", gorm.Expr("CASE WHEN like_count - ? < 0 THEN 0 ELSE like_count - ? END", -delta, -delta)).Error
	}
	return p.db.WithContext(ctx).Model(&danmaku.Danmaku{}).Where("id = ?", danmakuID).
		UpdateColumn("like_count", gorm.Expr("like_count + ?", delta)).Error
}

// ListCreatorDanmakus lists a creator's danmakus with filters for the creator panel.
func (p *DanmakuStoreImpl) ListCreatorDanmakus(ctx context.Context, uid uint64, limit int, keyword, typeFilter string, filterVideoID uint64, viewerID uint64) (*ListCreatorDanmakusResult, error) {
	base := p.db.WithContext(ctx).Model(&danmaku.Danmaku{}).
		Joins("INNER JOIN videos ON videos.id = danmakus.video_id AND videos.user_id = ?", uid)
	if filterVideoID > 0 {
		var owned video.Video
		if err := p.db.WithContext(ctx).Where("id = ? AND user_id = ?", filterVideoID, uid).First(&owned).Error; err != nil {
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

	videoIDs := make([]uint64, 0, len(list))
	userIDs := make([]uint64, 0, len(list))
	for _, d := range list {
		videoIDs = append(videoIDs, d.VideoID)
		userIDs = append(userIDs, d.UserID)
	}

	videos := map[uint64]video.Video{}
	if len(videoIDs) > 0 {
		var vlist []video.Video
		_ = p.db.WithContext(ctx).Where("id IN ?", videoIDs).Find(&vlist).Error
		for i := range vlist {
			videos[vlist[i].ID] = vlist[i]
		}
	}

	usernames := map[uint64]string{}
	if len(userIDs) > 0 {
		var users []user.User
		_ = p.db.WithContext(ctx).Where("id IN ?", userIDs).Find(&users).Error
		for i := range users {
			usernames[users[i].ID] = user.DisplayUsername(&users[i])
		}
	}

	danmakuIDs := make([]uint64, 0, len(list))
	for _, d := range list {
		danmakuIDs = append(danmakuIDs, d.ID)
	}
	likedByViewer := map[uint64]bool{}
	if viewerID > 0 && len(danmakuIDs) > 0 {
		var likes []danmaku.DanmakuLike
		_ = p.db.WithContext(ctx).Where("user_id = ? AND danmaku_id IN ?", viewerID, danmakuIDs).Find(&likes).Error
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
			CreatedAt:  d.CreatedAt.Format("2006-01-02 15:04:05"),
			LikedByMe:  likedByViewer[d.ID],
			VideoTitle: v.Title, CoverURL: v.CoverURL,
			Username: usernames[d.UserID],
		})
	}

	if total > int64(limit) {
		total = int64(limit)
	}
	return &ListCreatorDanmakusResult{Items: items, Total: total, Limit: limit}, nil
}

// DeleteDanmakuCascade deletes a danmaku and its likes atomically.
func (p *DanmakuStoreImpl) DeleteDanmakuCascade(ctx context.Context, d *danmaku.Danmaku, videoID uint64) error {
	return p.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.WithContext(ctx).Where("danmaku_id = ?", d.ID).Delete(&danmaku.DanmakuLike{}).Error; err != nil {
			return err
		}
		if err := tx.WithContext(ctx).Delete(d).Error; err != nil {
			return err
		}
		return tx.WithContext(ctx).Model(&video.Video{}).Where("id = ?", videoID).
			UpdateColumn("danmaku_count", gorm.Expr("CASE WHEN danmaku_count - ? < 0 THEN 0 ELSE danmaku_count - ? END", 1, 1)).Error
	})
}

// ListHistory lists danmakus for a video before a timestamp (history replay).
func (p *DanmakuStoreImpl) ListHistory(ctx context.Context, videoID uint64, currentTime float64, limit int) ([]danmaku.Danmaku, error) {
	query := p.db.WithContext(ctx).Where("video_id = ?", videoID)
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
