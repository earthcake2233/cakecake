package video

import (
	"cakecake/internal/model/user"
	"cakecake/internal/model/video"
	"cakecake/internal/pkg/cursor"
	"cakecake/internal/pkg/dbtx"
	"cakecake/internal/search"
	"cakecake/internal/service/queryutil"
	"context"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// VideoProviderImpl implements VideoProvider using *gorm.DB.
type VideoProviderImpl struct {
	db *gorm.DB
}

// TranscodeDeadLetterFilter filters dead-letter audit rows.
type TranscodeDeadLetterFilter struct {
	Page     int
	PageSize int
	Status   string // pending | requeued | processed | "" (all)
}

// ListTranscodeDeadLetters lists dead-letter audit rows with pagination and
// optional status filtering.
func (p *VideoProviderImpl) ListTranscodeDeadLetters(ctx context.Context, f TranscodeDeadLetterFilter) ([]video.TranscodeDeadLetter, int64, error) {
	q := p.db.WithContext(ctx).Model(&video.TranscodeDeadLetter{})
	// Archived rows are retained for audit but hidden from the active list.
	q = q.Where("archived_at IS NULL")
	switch f.Status {
	case "pending":
		q = q.Where("processed_at IS NULL AND requeued_at IS NULL")
	case "requeued":
		q = q.Where("requeued_at IS NOT NULL")
	case "processed":
		q = q.Where("processed_at IS NOT NULL")
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	page, size := f.Page, f.PageSize
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 20
	}
	var rows []video.TranscodeDeadLetter
	if err := q.Order("id DESC").Limit(size).Offset((page - 1) * size).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

// GetTranscodeDeadLetter loads a single dead-letter audit row by id.
func (p *VideoProviderImpl) GetTranscodeDeadLetter(ctx context.Context, id uint64) (*video.TranscodeDeadLetter, error) {
	var row video.TranscodeDeadLetter
	if err := p.db.WithContext(ctx).First(&row, id).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

// MarkTranscodeDeadLetterRequeued records a manual requeue on the audit row.
func (p *VideoProviderImpl) MarkTranscodeDeadLetterRequeued(ctx context.Context, id uint64, at time.Time) error {
	return p.db.WithContext(ctx).Model(&video.TranscodeDeadLetter{}).Where("id = ?", id).Updates(map[string]interface{}{
		"requeued_at":    at,
		"requeued_count": gorm.Expr("requeued_count + 1"),
		"processed_at":   nil,
	}).Error
}

// RevertTranscodeDeadLetterRequeue restores lifecycle fields to their
// pre-requeue values after a failed publish.
func (p *VideoProviderImpl) RevertTranscodeDeadLetterRequeue(ctx context.Context, id uint64, prevRequeuedAt *time.Time, prevRequeuedCount int, prevProcessedAt *time.Time) error {
	return p.db.WithContext(ctx).Model(&video.TranscodeDeadLetter{}).Where("id = ?", id).Updates(map[string]interface{}{
		"requeued_at":    prevRequeuedAt,
		"requeued_count": prevRequeuedCount,
		"processed_at":   prevProcessedAt,
	}).Error
}

// ResetVideoForTranscodeRequeue moves a video back to processing and clears
// stale output fields so a requeued job is not skipped by the idempotency guard.
func (p *VideoProviderImpl) ResetVideoForTranscodeRequeue(ctx context.Context, videoID uint64) error {
	return p.db.WithContext(ctx).Model(&video.Video{}).Where("id = ?", videoID).Updates(map[string]interface{}{
		"status":      video.StatusProcessing,
		"fail_reason": "",
		"video_url":   "",
		"cover_url":   "",
	}).Error
}

// MarkVideoFailedByID marks a video failed with a rune-safe reason.
func (p *VideoProviderImpl) MarkVideoFailedByID(ctx context.Context, videoID uint64, reason string) error {
	r := []rune(reason)
	if len(r) > 1900 {
		reason = string(r[:1900])
	}
	return p.db.WithContext(ctx).Model(&video.Video{}).Where("id = ?", videoID).Updates(map[string]interface{}{
		"status": video.StatusFailed, "fail_reason": reason,
	}).Error
}

var _ VideoProvider = (*VideoProviderImpl)(nil)

// NewVideoProvider creates a gorm-backed VideoProvider implementation.
func NewVideoProvider(db *gorm.DB) *VideoProviderImpl {
	return &VideoProviderImpl{db: db}
}

// GetPublishedVideo loads a published video as a VideoInfo projection.
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

// GetVideoAuthor returns the owner id of a video.
func (p *VideoProviderImpl) GetVideoAuthor(ctx context.Context, id uint64) (uint64, error) {
	var v video.Video
	if err := p.db.WithContext(ctx).Select("user_id").First(&v, id).Error; err != nil {
		return 0, err
	}
	return v.UserID, nil
}

// GetVideoByID loads a full video row by id.
func (p *VideoProviderImpl) GetVideoByID(ctx context.Context, id uint64) (*video.Video, error) {
	return queryutil.FirstByID[video.Video](ctx, p.db, id)
}

// GetVideoByUser loads a video by id and owner.
func (p *VideoProviderImpl) GetVideoByUser(ctx context.Context, id, uid uint64) (*video.Video, error) {
	var v video.Video
	if err := p.db.WithContext(ctx).Where("id = ? AND user_id = ?", id, uid).First(&v).Error; err != nil {
		return nil, err
	}
	return &v, nil
}

// GetVideoByUserStatus loads a video by id, owner, and status.
func (p *VideoProviderImpl) GetVideoByUserStatus(ctx context.Context, id, uid uint64, status string) (*video.Video, error) {
	var v video.Video
	if err := p.db.WithContext(ctx).Where("id = ? AND user_id = ? AND status = ?", id, uid, status).First(&v).Error; err != nil {
		return nil, err
	}
	return &v, nil
}

// CreateVideo inserts a video row.
func (p *VideoProviderImpl) CreateVideo(ctx context.Context, v *video.Video) error {
	return p.db.WithContext(ctx).Create(v).Error
}

// DeleteVideo removes a video row by id.
func (p *VideoProviderImpl) DeleteVideo(ctx context.Context, id uint64) error {
	return p.db.WithContext(ctx).Where("id = ?", id).Delete(&video.Video{}).Error
}

// UpdateVideo applies partial updates to a loaded video.
func (p *VideoProviderImpl) UpdateVideo(ctx context.Context, v *video.Video, updates map[string]interface{}) error {
	return p.db.WithContext(ctx).Model(v).Updates(updates).Error
}

// UpdateVideoByID applies partial updates to a video by id.
func (p *VideoProviderImpl) UpdateVideoByID(ctx context.Context, id uint64, updates map[string]interface{}) error {
	return p.db.WithContext(ctx).Model(&video.Video{}).Where("id = ?", id).Updates(updates).Error
}

// UpdateVideoField updates a single field on a loaded video.
func (p *VideoProviderImpl) UpdateVideoField(ctx context.Context, v *video.Video, field string, value interface{}) error {
	return p.db.WithContext(ctx).Model(v).Update(field, value).Error
}

// ListPublishedVideos pages published videos by zone/recency/order options.
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

// ListUserVideos pages a user's videos, optionally filtered by status.
func (p *VideoProviderImpl) ListUserVideos(ctx context.Context, uid uint64, status string, page, pageSize int) ([]video.Video, int64, error) {
	base := func() *gorm.DB {
		q := p.db.WithContext(ctx).Model(&video.Video{}).Where("user_id = ?", uid)
		if status != "" {
			q = q.Where("status = ?", status)
		}
		return q
	}
	var list []video.Video
	total, err := queryutil.ListPage(base, page, pageSize, "id DESC", &list)
	if err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

// ListUserVideosAdvanced pages a user's videos with multi-status/title filters.
func (p *VideoProviderImpl) ListUserVideosAdvanced(ctx context.Context, f MyVideoFilter) (*MyVideoPageResult, error) {
	base := func() *gorm.DB {
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
		return q
	}
	total, err := queryutil.Count(base)
	if err != nil {
		return nil, err
	}
	totalPages := int((total + int64(f.PageSize) - 1) / int64(f.PageSize))
	if totalPages < 1 {
		totalPages = 1
	}
	if f.Page > totalPages {
		f.Page = totalPages
	}
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
	if err := queryutil.FindPage(base, f.Page, f.PageSize, orderClause, &list); err != nil {
		return nil, err
	}
	return &MyVideoPageResult{Videos: list, Total: total, TotalPages: totalPages}, nil
}

// ListUserPublishedVideosCursor pages a user's published videos with a keyset cursor.
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

// ListDrafts pages a user's draft videos.
func (p *VideoProviderImpl) ListDrafts(ctx context.Context, uid uint64, page, pageSize int) ([]video.Video, int64, error) {
	base := func() *gorm.DB {
		return p.db.WithContext(ctx).Model(&video.Video{}).Where("user_id = ? AND status = 'draft'", uid)
	}
	var list []video.Video
	total, err := queryutil.ListPage(base, page, pageSize, "updated_at DESC, id DESC", &list)
	if err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

// CountDrafts counts a user's draft videos.
func (p *VideoProviderImpl) CountDrafts(ctx context.Context, uid uint64) (int64, error) {
	var count int64
	err := p.db.WithContext(ctx).Model(&video.Video{}).Where("user_id = ? AND status = 'draft'", uid).Count(&count).Error
	return count, err
}

// CountVideosByStatusForUser counts a user's videos per status.
func (p *VideoProviderImpl) CountVideosByStatusForUser(uid uint64) map[string]int64 {
	result := map[string]int64{}
	for _, st := range []string{video.StatusProcessing, video.StatusPending, video.StatusRejected, video.StatusPublished, video.StatusPrivate} {
		var n int64
		_ = p.db.Model(&video.Video{}).Where("user_id = ? AND status = ?", uid, st).Count(&n).Error
		result[st] = n
	}
	return result
}

// CountZoneVideos counts published videos under a zone (including children).
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

// CountPublishedVideos counts all published videos.
func (p *VideoProviderImpl) CountPublishedVideos(ctx context.Context) int64 {
	var n int64
	p.db.WithContext(ctx).Model(&video.Video{}).Where("status = ?", video.StatusPublished).Count(&n)
	return n
}

// CountByStatus counts videos with the given status.
func (p *VideoProviderImpl) CountByStatus(ctx context.Context, status string) (int64, error) {
	var cnt int64
	err := p.db.WithContext(ctx).Model(&video.Video{}).Where("status = ?", status).Count(&cnt).Error
	return cnt, err
}

// ToggleVideoLike toggles a video like atomically: the read-check-write
// sequence and the like_count adjustment run in one transaction, with a row
// lock on MySQL serializing concurrent toggles for the same video.
func (p *VideoProviderImpl) ToggleVideoLike(ctx context.Context, userID, videoID uint64) (bool, error) {
	var liked bool
	err := p.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// MySQL: lock the video row so concurrent toggles serialize on one
		// lock owner. SQLite has no SELECT ... FOR UPDATE; writers are
		// serialized and the unique-constraint race is handled below.
		if p.db.Dialector.Name() == "mysql" {
			var locked video.Video
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
				First(&locked, videoID).Error; err != nil {
				return err
			}
		}

		var n int64
		if err := tx.Model(&video.VideoLike{}).
			Where("user_id = ? AND video_id = ?", userID, videoID).
			Count(&n).Error; err != nil {
			return err
		}

		if n == 0 {
			lk := video.VideoLike{UserID: userID, VideoID: videoID}
			if err := tx.Create(&lk).Error; err != nil {
				// A concurrent toggle created the row first (SQLite has no
				// row lock): this toggle removes that like instead.
				if isUniqueViolation(err) {
					if derr := tx.Where("user_id = ? AND video_id = ?", userID, videoID).
						Delete(&video.VideoLike{}).Error; derr != nil {
						return derr
					}
					liked = false
					return decrVideoLikeCount(tx, videoID)
				}
				return err
			}
			liked = true
			return incrVideoLikeCount(tx, videoID)
		}

		if err := tx.Where("user_id = ? AND video_id = ?", userID, videoID).
			Delete(&video.VideoLike{}).Error; err != nil {
			return err
		}
		liked = false
		return decrVideoLikeCount(tx, videoID)
	})
	return liked, err
}

func incrVideoLikeCount(db *gorm.DB, videoID uint64) error {
	return db.Model(&video.Video{}).Where("id = ?", videoID).
		UpdateColumn("like_count", gorm.Expr("like_count + ?", 1)).Error
}

func decrVideoLikeCount(db *gorm.DB, videoID uint64) error {
	return db.Model(&video.Video{}).Where("id = ?", videoID).
		UpdateColumn("like_count", gorm.Expr("CASE WHEN like_count - ? < 0 THEN 0 ELSE like_count - ? END", 1, 1)).Error
}

func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "unique constraint") ||
		strings.Contains(msg, "duplicate entry") ||
		strings.Contains(msg, "constraint failed")
}

// DeleteDirectUploadClaim removes the idempotency claim for a raw object
// (used to compensate a failed enqueue so the submit can be retried).
func (p *VideoProviderImpl) DeleteDirectUploadClaim(ctx context.Context, rawKey string) error {
	return p.db.WithContext(ctx).Where("raw_key = ?", rawKey).Delete(&video.DirectUploadClaim{}).Error
}

// CreateTranscodeOutbox writes a pending outbox row (the local message table
// for the transcode queue).
func (p *VideoProviderImpl) CreateTranscodeOutbox(ctx context.Context, outbox *video.TranscodeOutbox) error {
	return p.db.WithContext(ctx).Create(outbox).Error
}

// RecordTranscodeEvent appends an audit row for a transcode status change.
func (p *VideoProviderImpl) RecordTranscodeEvent(ctx context.Context, ev *video.TranscodeEvent) error {
	return p.db.WithContext(ctx).Create(ev).Error
}

// PublishVideo marks a video published, stamps review metadata, and indexes it in Elasticsearch.
func (p *VideoProviderImpl) PublishVideo(ctx context.Context, esc *search.Client, log *zap.Logger, videoID uint64, adminID *uint64) error {
	var v video.Video
	if err := p.db.WithContext(ctx).First(&v, videoID).Error; err != nil {
		return err
	}
	if v.Status == video.StatusPublished {
		return nil
	}
	if !ValidateTranscodeStatusTransition(v.Status, video.StatusPublished) {
		return fmt.Errorf("illegal transcode status transition %s -> %s", v.Status, video.StatusPublished)
	}
	now := time.Now()
	updates := map[string]any{
		"status":      video.StatusPublished,
		"reviewed_at": now,
	}
	if adminID != nil && *adminID > 0 {
		updates["reviewed_by_admin_id"] = *adminID
	}
	// Conditional update on the status read moments ago: a concurrent
	// reject/update must not be overwritten by a stale publish.
	res := p.db.WithContext(ctx).Model(&video.Video{}).
		Where("id = ? AND status = ?", videoID, v.Status).
		Updates(updates)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return fmt.Errorf("video status changed concurrently while publishing")
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
	RecordTranscodeEvent(ctx, p.db, videoID, "", v.Status, video.StatusPublished, "publish")
	return nil
}

// AdminDeleteVideoCascade runs the cascade delete callback inside a transaction.
func (p *VideoProviderImpl) AdminDeleteVideoCascade(ctx context.Context, id uint64, fn func(tx dbtx.Tx) error) error {
	return p.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(tx)
	})
}

// WithTx runs fn inside a transaction.
func (p *VideoProviderImpl) WithTx(ctx context.Context, fn func(tx dbtx.Tx) error) error {
	return p.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(tx)
	})
}

// AdminListVideos pages all videos for the admin panel with status/title filters.
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

// IncrCommentCount adjusts a video's comment count by delta.
func (p *VideoProviderImpl) IncrCommentCount(ctx context.Context, id uint64, delta int) error {
	return p.db.WithContext(ctx).Model(&video.Video{}).Where("id = ?", id).
		UpdateColumn("comment_count", gorm.Expr("comment_count + ?", delta)).Error
}

// IncrFavCount adjusts a video's fav count by delta (negative clamps at zero).
func (p *VideoProviderImpl) IncrFavCount(ctx context.Context, id uint64, delta int) error {
	q := p.db.WithContext(ctx).Model(&video.Video{}).Where("id = ?", id)
	if delta < 0 {
		abs := -delta
		return q.UpdateColumn("fav_count",
			gorm.Expr("CASE WHEN fav_count < ? THEN 0 ELSE fav_count - ? END", abs, abs)).Error
	}
	return q.UpdateColumn("fav_count", gorm.Expr("fav_count + ?", delta)).Error
}

// BatchGetPublishedVideos loads published videos by ids as VideoInfo projections.
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
