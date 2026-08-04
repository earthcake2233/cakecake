package video

import (
	"cakecake/internal/model/user"
	"cakecake/internal/model/video"
	"cakecake/internal/pkg/cursor"
	"cakecake/internal/search"
	"context"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// VideoProviderImpl implements VideoProvider using *gorm.DB.
type VideoProviderImpl struct {
	db *gorm.DB
}

func NewVideoProvider(db *gorm.DB) *VideoProviderImpl {
	return &VideoProviderImpl{db: db}
}

func (p *VideoProviderImpl) GetPublishedVideo(ctx context.Context, id uint64) (*VideoInfo, error) {
	var v video.Video
	if err := p.db.WithContext(ctx).First(&v, id).Error; err != nil {
		return nil, err
	}
	if v.Status != video.StatusPublished {
		return nil, gorm.ErrRecordNotFound
	}
	return &VideoInfo{
		ID: v.ID, UserID: v.UserID, Title: v.Title, CoverURL: v.CoverURL,
		PlayCount: v.PlayCount, DanmakuCount: v.DanmakuCount, CommentCount: v.CommentCount, DurationSec: v.DurationSec,
		FavCount: v.FavCount, Status: v.Status,
		CommentsClosed: v.CommentsClosed, CommentsCurated: v.CommentsCurated,
		DanmakuClosed: v.DanmakuClosed, CreatedAt: v.CreatedAt,
	}, nil
}

func (p *VideoProviderImpl) GetVideoAuthor(ctx context.Context, id uint64) (uint64, error) {
	var v video.Video
	if err := p.db.WithContext(ctx).Select("user_id").First(&v, id).Error; err != nil {
		return 0, err
	}
	return v.UserID, nil
}

func (p *VideoProviderImpl) GetVideoByID(ctx context.Context, id uint64) (*video.Video, error) {
	var v video.Video
	if err := p.db.WithContext(ctx).First(&v, id).Error; err != nil {
		return nil, err
	}
	return &v, nil
}

func (p *VideoProviderImpl) GetVideoByUser(ctx context.Context, id, uid uint64) (*video.Video, error) {
	var v video.Video
	if err := p.db.WithContext(ctx).Where("id = ? AND user_id = ?", id, uid).First(&v).Error; err != nil {
		return nil, err
	}
	return &v, nil
}

func (p *VideoProviderImpl) GetVideoByUserStatus(ctx context.Context, id, uid uint64, status string) (*video.Video, error) {
	var v video.Video
	if err := p.db.WithContext(ctx).Where("id = ? AND user_id = ? AND status = ?", id, uid, status).First(&v).Error; err != nil {
		return nil, err
	}
	return &v, nil
}

func (p *VideoProviderImpl) CreateVideo(ctx context.Context, v *video.Video) error {
	return p.db.WithContext(ctx).Create(v).Error
}

func (p *VideoProviderImpl) DeleteVideo(ctx context.Context, id uint64) error {
	return p.db.WithContext(ctx).Where("id = ?", id).Delete(&video.Video{}).Error
}

func (p *VideoProviderImpl) UpdateVideo(ctx context.Context, v *video.Video, updates map[string]interface{}) error {
	return p.db.WithContext(ctx).Model(v).Updates(updates).Error
}

func (p *VideoProviderImpl) UpdateVideoByID(ctx context.Context, id uint64, updates map[string]interface{}) error {
	return p.db.WithContext(ctx).Model(&video.Video{}).Where("id = ?", id).Updates(updates).Error
}

func (p *VideoProviderImpl) UpdateVideoField(ctx context.Context, v *video.Video, field string, value interface{}) error {
	return p.db.WithContext(ctx).Model(v).Update(field, value).Error
}

func (p *VideoProviderImpl) ListPublishedVideos(ctx context.Context, opts VideoListOpts) (*VideoListResult, error) {
	q := p.db.WithContext(ctx).Model(&video.Video{}).Where("status = ?", video.StatusPublished)
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
		_ = p.db.Model(&video.Video{}).
			Where("status = ?", video.StatusPublished).
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

func (p *VideoProviderImpl) ListUserVideos(ctx context.Context, uid uint64, status string, page, pageSize int) ([]video.Video, int64, error) {
	q := p.db.WithContext(ctx).Model(&video.Video{}).Where("user_id = ?", uid)
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

func (p *VideoProviderImpl) ListUserVideosAdvanced(ctx context.Context, f MyVideoFilter) (*MyVideoPageResult, error) {
	q := p.db.WithContext(ctx).Model(&video.Video{}).Where("user_id = ?", f.UserID)
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

func (p *VideoProviderImpl) ListUserPublishedVideosCursor(ctx context.Context, uid uint64, cursorID uint64, limit int) ([]video.Video, error) {
	q := p.db.WithContext(ctx).Model(&video.Video{}).Where("user_id = ? AND status = ?", uid, video.StatusPublished)
	if cursorID > 0 {
		q = q.Where("id < ?", cursorID)
	}
	var list []video.Video
	if err := q.Order("id DESC").Limit(limit).Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

func (p *VideoProviderImpl) ListDrafts(ctx context.Context, uid uint64, page, pageSize int) ([]video.Video, int64, error) {
	q := p.db.WithContext(ctx).Model(&video.Video{}).Where("user_id = ? AND status = 'draft'", uid)
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	offset := (page - 1) * pageSize
	var list []video.Video
	if err := q.Order("updated_at DESC, id DESC").Offset(offset).Limit(pageSize).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

func (p *VideoProviderImpl) CountDrafts(ctx context.Context, uid uint64) (int64, error) {
	var count int64
	err := p.db.WithContext(ctx).Model(&video.Video{}).Where("user_id = ? AND status = 'draft'", uid).Count(&count).Error
	return count, err
}

func (p *VideoProviderImpl) CountVideosByStatusForUser(uid uint64) map[string]int64 {
	result := map[string]int64{}
	for _, st := range []string{video.StatusProcessing, video.StatusPending, video.StatusRejected, video.StatusPublished, video.StatusPrivate} {
		var n int64
		_ = p.db.Model(&video.Video{}).Where("user_id = ? AND status = ?", uid, st).Count(&n).Error
		result[st] = n
	}
	return result
}

func (p *VideoProviderImpl) CountZoneVideos(zoneParent string) int64 {
	if zoneParent == "" {
		return 0
	}
	var n int64
	_ = p.db.Model(&video.Video{}).
		Where("status = ?", video.StatusPublished).
		Where("zone = ? OR zone LIKE ?", zoneParent, zoneParent+"-%").
		Count(&n).Error
	return n
}

func (p *VideoProviderImpl) CountPublishedVideos(ctx context.Context) int64 {
	var n int64
	p.db.WithContext(ctx).Model(&video.Video{}).Where("status = ?", video.StatusPublished).Count(&n)
	return n
}

func (p *VideoProviderImpl) CountByStatus(ctx context.Context, status string) (int64, error) {
	var cnt int64
	err := p.db.WithContext(ctx).Model(&video.Video{}).Where("status = ?", status).Count(&cnt).Error
	return cnt, err
}

func (p *VideoProviderImpl) ToggleVideoLike(ctx context.Context, userID, videoID uint64) (bool, error) {
	var like video.VideoLike
	res := p.db.WithContext(ctx).Where("user_id = ? AND video_id = ?", userID, videoID).Limit(1).Find(&like)
	if res.Error != nil {
		return false, res.Error
	}
	if res.RowsAffected == 0 {
		lk := video.VideoLike{UserID: userID, VideoID: videoID}
		if err := p.db.WithContext(ctx).Create(&lk).Error; err != nil {
			return false, err
		}
		_ = p.db.WithContext(ctx).Model(&video.Video{}).Where("id = ?", videoID).
			UpdateColumn("like_count", gorm.Expr("like_count + ?", 1)).Error
		return true, nil
	}
	if err := p.db.WithContext(ctx).Delete(&like).Error; err != nil {
		return false, err
	}
	_ = p.db.WithContext(ctx).Model(&video.Video{}).Where("id = ?", videoID).
		UpdateColumn("like_count", gorm.Expr("CASE WHEN like_count - ? < 0 THEN 0 ELSE like_count - ? END", 1, 1)).Error
	return false, nil
}

func (p *VideoProviderImpl) PublishVideo(ctx context.Context, esc *search.Client, log *zap.Logger, videoID uint64, adminID *uint64) error {
	var v video.Video
	if err := p.db.WithContext(ctx).First(&v, videoID).Error; err != nil {
		return err
	}
	if v.Status == video.StatusPublished {
		return nil
	}
	now := time.Now()
	updates := map[string]any{
		"status":      video.StatusPublished,
		"reviewed_at": now,
	}
	if adminID != nil && *adminID > 0 {
		updates["reviewed_by_admin_id"] = *adminID
	}
	if err := p.db.WithContext(ctx).Model(&v).Updates(updates).Error; err != nil {
		return err
	}
	_ = p.db.WithContext(ctx).Model(&user.User{}).
		Where("id = ? AND first_published_at IS NULL", v.UserID).
		Update("first_published_at", v.CreatedAt).Error
	if esc != nil && esc.Enabled() {
		ictx, cancel := context.WithTimeout(ctx, 15*time.Second)
		defer cancel()
		if err := esc.IndexVideoFromDB(ictx, p.db, videoID); err != nil && log != nil {
			log.Warn("elasticsearch index video on publish", zap.Uint64("video_id", videoID), zap.Error(err))
		}
	}
	return nil
}

func (p *VideoProviderImpl) AdminDeleteVideoCascade(ctx context.Context, id uint64, fn func(tx *gorm.DB) error) error {
	return p.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(tx)
	})
}

func (p *VideoProviderImpl) WithTx(ctx context.Context, fn func(tx *gorm.DB) error) error {
	return p.db.WithContext(ctx).Transaction(fn)
}

func (p *VideoProviderImpl) AdminListVideos(ctx context.Context, statuses []string, titleQ string, page, pageSize int) (*AdminListVideosResult, error) {
	q := p.db.WithContext(ctx).Model(&video.Video{})
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
	var pending int64
	_ = p.db.WithContext(ctx).Model(&video.Video{}).Where("status = ?", video.StatusPendingReview).Count(&pending).Error
	return &AdminListVideosResult{
		Total:        total,
		Rows:         rows,
		PendingCount: pending,
	}, nil
}

func (p *VideoProviderImpl) IncrCommentCount(ctx context.Context, id uint64, delta int) error {
	return p.db.WithContext(ctx).Model(&video.Video{}).Where("id = ?", id).
		UpdateColumn("comment_count", gorm.Expr("comment_count + ?", delta)).Error
}

func (p *VideoProviderImpl) IncrFavCount(ctx context.Context, id uint64, delta int) error {
	q := p.db.WithContext(ctx).Model(&video.Video{}).Where("id = ?", id)
	if delta < 0 {
		abs := -delta
		return q.UpdateColumn("fav_count",
			gorm.Expr("CASE WHEN fav_count < ? THEN 0 ELSE fav_count - ? END", abs, abs)).Error
	}
	return q.UpdateColumn("fav_count", gorm.Expr("fav_count + ?", delta)).Error
}

func (p *VideoProviderImpl) BatchGetPublishedVideos(ctx context.Context, ids []uint64) (map[uint64]*VideoInfo, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var videos []video.Video
	if err := p.db.WithContext(ctx).Where("id IN ? AND status = ?", ids, video.StatusPublished).Find(&videos).Error; err != nil {
		return nil, err
	}
	result := make(map[uint64]*VideoInfo, len(videos))
	for i := range videos {
		v := &videos[i]
		result[v.ID] = &VideoInfo{
			ID: v.ID, UserID: v.UserID, Title: v.Title, CoverURL: v.CoverURL,
			PlayCount: v.PlayCount, DanmakuCount: v.DanmakuCount, CommentCount: v.CommentCount, DurationSec: v.DurationSec,
			FavCount: v.FavCount, Status: v.Status,
			CommentsClosed: v.CommentsClosed, CommentsCurated: v.CommentsCurated,
			DanmakuClosed: v.DanmakuClosed, CreatedAt: v.CreatedAt,
		}
	}
	return result, nil
}
