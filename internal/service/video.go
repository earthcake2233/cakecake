package service

import (
	"minibili/internal/model/admin"
	"minibili/internal/model/video"
	"context"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"minibili/internal/pkg/cursor"
)

// VideoService handles video business logic.
type VideoService struct {
	db  *gorm.DB
	rdb *redis.Client
	log *zap.Logger
}

func NewVideoService(db *gorm.DB, rdb *redis.Client, log *zap.Logger) *VideoService {
	return &VideoService{db: db, rdb: rdb, log: log}
}

// CreateVideoRecord inserts a new video record into the database.
func (s *VideoService) CreateVideoRecord(ctx context.Context, v *video.Video) error {
	return s.db.WithContext(ctx).Create(v).Error
}

// DeleteVideoByID removes a video record by ID.
func (s *VideoService) DeleteVideoByID(ctx context.Context, id uint64) error {
	return s.db.WithContext(ctx).Where("id = ?", id).Delete(&video.Video{}).Error
}

// ListPublishedVideos queries published videos with optional filtering and cursor-based pagination.
func (s *VideoService) ListPublishedVideos(ctx context.Context, opts VideoListOpts) (*VideoListResult, error) {
	q := s.db.Model(&video.Video{}).Where("status = ?", "published")
	if opts.ZoneParent != "" {
		q = q.Where("zone = ? OR zone LIKE ?", opts.ZoneParent, opts.ZoneParent+"-%")
	}
	if opts.RecentOnly && opts.Days > 0 {
		cutoff := time.Now().AddDate(0, 0, -opts.Days)
		q = q.Where("created_at >= ?", cutoff)
	}
	useHotCursor := opts.ZoneParent == "" && opts.SortKey != "time" && !opts.RecentOnly
	var orderClause string
	switch opts.SortKey {
	case "time":
		orderClause = "created_at DESC, id DESC"
	default:
		orderClause = "play_count DESC, created_at DESC, danmaku_count DESC, id DESC"
	}
	cur, _ := cursor.Decode(opts.Cursor)
	if useHotCursor && cur != nil {
		q = q.Where(
			"(play_count < ?) OR (play_count = ? AND created_at < ?) OR (play_count = ? AND created_at = ? AND danmaku_count < ?) OR (play_count = ? AND created_at = ? AND danmaku_count = ? AND id < ?)",
			cur.PlayCount, cur.PlayCount, cur.CreatedAt,
			cur.PlayCount, cur.CreatedAt, cur.DanmakuCount,
			cur.PlayCount, cur.CreatedAt, cur.DanmakuCount, cur.ID,
		)
	}
	fetchLimit := opts.Limit + 1
	if !useHotCursor {
		fetchLimit = opts.Limit
	}
	var list []video.Video
	if err := q.Order(orderClause).Limit(fetchLimit).Find(&list).Error; err != nil {
		return nil, err
	}
	hasMore := useHotCursor && len(list) > opts.Limit
	if hasMore {
		list = list[:opts.Limit]
	}
	var next string
	if hasMore && len(list) > 0 {
		last := list[len(list)-1]
		next = cursor.Encode(cursor.VideoListC{
			PlayCount: last.PlayCount, CreatedAt: last.CreatedAt, DanmakuCount: last.DanmakuCount, ID: last.ID,
		})
	}
	var zoneCount int64
	if opts.ZoneParent != "" {
		_ = s.db.Model(&video.Video{}).
			Where("status = ?", "published").
			Where("zone = ? OR zone LIKE ?", opts.ZoneParent, opts.ZoneParent+"-%").
			Count(&zoneCount).Error
	}
	return &VideoListResult{
		Videos:         list,
		NextCursor:     next,
		HasMore:        hasMore,
		ZoneVideoCount: zoneCount,
	}, nil
}

// CountZoneVideos returns the count of published videos in a zone.
func (s *VideoService) CountZoneVideos(zoneParent string) int64 {
	if zoneParent == "" { return 0 }
	var n int64
	_ = s.db.Model(&video.Video{}).
		Where("status = ?", "published").
		Where("zone = ? OR zone LIKE ?", zoneParent, zoneParent+"-%").
		Count(&n).Error
	return n
}

// CountMyVideosByStatus returns counts for each status for a user.
func (s *VideoService) CountMyVideosByStatus(uid uint64) map[string]int64 {
	result := map[string]int64{}
	for _, st := range []string{"processing", "pending", "rejected", "published", "private"} {
		var n int64
		_ = s.db.Model(&video.Video{}).Where("user_id = ? AND status = ?", uid, st).Count(&n).Error
		result[st] = n
	}
	return result
}

// GetPublishedVideo returns a published video by ID.
func (s *VideoService) GetPublishedVideo(ctx context.Context, id uint64) (*video.Video, error) {
	var v video.Video
	if err := s.db.WithContext(ctx).First(&v, id).Error; err != nil {
		return nil, err
	}
	if v.Status != "published" {
		return nil, gorm.ErrRecordNotFound
	}
	return &v, nil
}

// GetVideoByID fetches a single video by ID.
func (s *VideoService) GetVideoByID(ctx context.Context, id uint64) (*video.Video, error) {
	var v video.Video
	if err := s.db.WithContext(ctx).First(&v, id).Error; err != nil {
		return nil, err
	}
	return &v, nil
}

// UpdateVideo updates a video record with the given fields.
func (s *VideoService) UpdateVideo(ctx context.Context, v *video.Video, updates map[string]interface{}) error {
	return s.db.WithContext(ctx).Model(v).Updates(updates).Error
}

// DeleteVideoWithCascade removes a video and its related data in a transaction.
func (s *VideoService) DeleteVideoWithCascade(ctx context.Context, id uint64, fn func(tx *gorm.DB) error) error {
	return s.db.WithContext(ctx).Transaction(fn)
}

// ListMyVideos queries a user's own videos with optional status filter.
func (s *VideoService) ListMyVideos(ctx context.Context, uid uint64, status string, page, pageSize int) ([]video.Video, int64, error) {
	q := s.db.Model(&video.Video{}).Where("user_id = ?", uid)
	if status != "" {
		q = q.Where("status = ?", status)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	offset := (page - 1) * pageSize
	var list []video.Video
	if err := q.Order("id DESC").Offset(offset).Limit(pageSize).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

// GetZoneVideoCount returns counted published videos for a zone.
func (s *VideoService) GetZoneVideoCount(zoneParent string) int64 {
	return s.CountZoneVideos(zoneParent)
}

// VideoListOpts carries filtering options for ListPublishedVideos.
type VideoListOpts struct {
	Limit      int
	SortKey    string
	ZoneParent string
	Days       int
	RecentOnly bool
	Cursor     string
}

// VideoListResult is the result of ListPublishedVideos.
type VideoListResult struct {
	Videos         []video.Video
	NextCursor     string
	HasMore        bool
	ZoneVideoCount int64
}

// ToggleVideoLike toggles the current user's like on a published video.
// Returns true if liked, false if unliked.
func (s *VideoService) ToggleVideoLike(ctx context.Context, userID, videoID uint64) (bool, error) {
	_, err := s.GetPublishedVideo(ctx, videoID)
	if err != nil {
		return false, err
	}
	var like video.VideoLike
	res := s.db.WithContext(ctx).Where("user_id = ? AND video_id = ?", userID, videoID).Limit(1).Find(&like)
	if res.Error != nil {
		return false, res.Error
	}
	if res.RowsAffected == 0 {
		lk := video.VideoLike{UserID: userID, VideoID: videoID}
		if err := s.db.WithContext(ctx).Create(&lk).Error; err != nil {
			return false, err
		}
		_ = s.db.WithContext(ctx).Model(&video.Video{}).Where("id = ?", videoID).
			UpdateColumn("like_count", gorm.Expr("like_count + ?", 1)).Error
		return true, nil
	}
	if err := s.db.WithContext(ctx).Delete(&like).Error; err != nil {
		return false, err
	}
	_ = s.db.WithContext(ctx).Model(&video.Video{}).Where("id = ?", videoID).
		UpdateColumn("like_count", gorm.Expr("CASE WHEN like_count - ? < 0 THEN 0 ELSE like_count - ? END", 1, 1)).Error
	return false, nil
}

// CountPublishedVideos returns the total number of published videos.
func (s *VideoService) CountPublishedVideos(ctx context.Context) int64 {
	var n int64
	s.db.WithContext(ctx).Model(&video.Video{}).Where("status = ?", "published").Count(&n)
	return n
}

// ListActiveBanners returns active home page banners.
func (s *VideoService) ListActiveBanners(ctx context.Context) ([]admin.HomeBanner, error) {
	now := time.Now()
	var rows []admin.HomeBanner
	q := s.db.WithContext(ctx).Where("enabled = ?", true).
		Where("(start_at IS NULL OR start_at <= ?)", now).
		Where("(end_at IS NULL OR end_at >= ?)", now).
		Order("sort_order ASC, id ASC")
	if err := q.Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// ListUserPublishedVideosCursor lists published videos for a user with cursor-based pagination.
func (s *VideoService) ListUserPublishedVideosCursor(ctx context.Context, uid uint64, cursorID uint64, limit int) ([]video.Video, error) {
	q := s.db.WithContext(ctx).Model(&video.Video{}).Where("user_id = ? AND status = ?", uid, "published")
	if cursorID > 0 {
		q = q.Where("id < ?", cursorID)
	}
	var list []video.Video
	if err := q.Order("id DESC").Limit(limit).Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

// ListMyVideosAdvanced lists user videos with advanced filtering (title search, custom sort, multi-status).
type MyVideoFilter struct {
	UserID   uint64
	Status   string   // single status
	Statuses []string // multi status (used when Status is empty)
	TitleQ   string
	SortKey  string   // "time", "reply", "like"
	Page     int
	PageSize int
}

type MyVideoPageResult struct {
	Videos     []video.Video
	Total      int64
	TotalPages int
}

func (s *VideoService) ListMyVideosAdvanced(ctx context.Context, f MyVideoFilter) (*MyVideoPageResult, error) {
	q := s.db.WithContext(ctx).Model(&video.Video{}).Where("user_id = ?", f.UserID)
	if f.Status != "" {
		q = q.Where("status = ?", f.Status)
	}
	if len(f.Statuses) > 0 {
		q = q.Where("status IN ?", f.Statuses)
	}
	if f.TitleQ != "" {
		q = q.Where("title LIKE ?", "%"+f.TitleQ+"%")
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, err
	}
	totalPages := int((total + int64(f.PageSize) - 1) / int64(f.PageSize))
	if totalPages < 1 {
		totalPages = 1
	}
	if f.Page > totalPages {
		f.Page = totalPages
	}
	offset := (f.Page - 1) * f.PageSize
	var orderClause string
	switch f.SortKey {
	case "reply":
		orderClause = "comment_count DESC, id DESC"
	case "like":
		orderClause = "like_count DESC, id DESC"
	default:
		orderClause = "id DESC"
	}
	var list []video.Video
	if err := q.Order(orderClause).Offset(offset).Limit(f.PageSize).Find(&list).Error; err != nil {
		return nil, err
	}
	return &MyVideoPageResult{Videos: list, Total: total, TotalPages: totalPages}, nil
}

// ListBanners returns all home banners ordered by sort_order and id.
func (s *VideoService) ListBanners(ctx context.Context) ([]admin.HomeBanner, error) {
	var rows []admin.HomeBanner
	if err := s.db.WithContext(ctx).Order("sort_order ASC, id ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// CreateBanner creates a new home banner.
func (s *VideoService) CreateBanner(ctx context.Context, b *admin.HomeBanner) error {
	return s.db.WithContext(ctx).Create(b).Error
}

// GetBanner returns a home banner by ID.
func (s *VideoService) GetBanner(ctx context.Context, id uint64) (*admin.HomeBanner, error) {
	var b admin.HomeBanner
	if err := s.db.WithContext(ctx).First(&b, id).Error; err != nil {
		return nil, err
	}
	return &b, nil
}

// UpdateBanner updates fields of a home banner.
func (s *VideoService) UpdateBanner(ctx context.Context, id uint64, updates map[string]interface{}) error {
	return s.db.WithContext(ctx).Model(&admin.HomeBanner{}).Where("id = ?", id).Updates(updates).Error
}

// DeleteBanner deletes a home banner by ID.
func (s *VideoService) DeleteBanner(ctx context.Context, id uint64) error {
	return s.db.WithContext(ctx).Delete(&admin.HomeBanner{}, id).Error
}


// CountByStatus returns video count by status.
func (s *VideoService) CountByStatus(ctx context.Context, status string) (int64, error) {
	var cnt int64
	err := s.db.WithContext(ctx).Model(&video.Video{}).Where("status = ?", status).Count(&cnt).Error
	return cnt, err
}

// AdminUpdateVideo updates video fields by ID (admin operation).
func (s *VideoService) AdminUpdateVideo(ctx context.Context, id uint64, updates map[string]interface{}) error {
	return s.db.WithContext(ctx).Model(&video.Video{}).Where("id = ?", id).Updates(updates).Error
}

// AdminDeleteVideoCascade deletes a video and cascades to related data within a transaction.
func (s *VideoService) AdminDeleteVideoCascade(ctx context.Context, id uint64, fn func(tx *gorm.DB) error) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(tx)
	})
}

// AdminListVideosResult holds paginated admin video list results.
type AdminListVideosResult struct {
	Total        int64
	Rows         []video.Video
	PendingCount int64
}

// AdminListVideos returns paginated videos with filters for admin panel.
func (s *VideoService) AdminListVideos(ctx context.Context, statuses []string, titleQ string, page, pageSize int) (*AdminListVideosResult, error) {
	q := s.db.WithContext(ctx).Model(&video.Video{})
	if len(statuses) > 0 {
		q = q.Where("status IN ?", statuses)
	}
	if titleQ != "" {
		q = q.Where("title LIKE ?", "%"+titleQ+"%")
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, err
	}
	offset := (page - 1) * pageSize
	var rows []video.Video
	if err := q.Order("created_at DESC, id DESC").Offset(offset).Limit(pageSize).Find(&rows).Error; err != nil {
		return nil, err
	}
	pending, _ := s.CountByStatus(ctx, "pending_review")
	return &AdminListVideosResult{
		Total:        total,
		Rows:         rows,
		PendingCount: pending,
	}, nil
}
