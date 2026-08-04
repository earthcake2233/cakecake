package video

import (
	"cakecake/internal/model/video"
	"cakecake/internal/search"
	"context"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// VideoProvider is the video domain boundary.
type VideoProvider interface {
	GetPublishedVideo(ctx context.Context, id uint64) (*VideoInfo, error)
	GetVideoAuthor(ctx context.Context, id uint64) (uint64, error)
	IncrCommentCount(ctx context.Context, id uint64, delta int) error
	// IncrFavCount updates the favorite count by delta (can be negative).
	IncrFavCount(ctx context.Context, id uint64, delta int) error
	// BatchGetPublishedVideos returns a map of video id to published VideoInfo.
	BatchGetPublishedVideos(ctx context.Context, ids []uint64) (map[uint64]*VideoInfo, error)

	// Video-domain storage boundary (Phase 1: *gorm.DB impl; Phase 2+: gRPC client).
	GetVideoByID(ctx context.Context, id uint64) (*video.Video, error)
	GetVideoByUser(ctx context.Context, id, uid uint64) (*video.Video, error)
	GetVideoByUserStatus(ctx context.Context, id, uid uint64, status string) (*video.Video, error)
	CreateVideo(ctx context.Context, v *video.Video) error
	DeleteVideo(ctx context.Context, id uint64) error
	UpdateVideo(ctx context.Context, v *video.Video, updates map[string]interface{}) error
	UpdateVideoByID(ctx context.Context, id uint64, updates map[string]interface{}) error
	UpdateVideoField(ctx context.Context, v *video.Video, field string, value interface{}) error
	// ListPublishedVideos queries published videos with optional zone/recent filters and hot cursor.
	ListPublishedVideos(ctx context.Context, opts VideoListOpts) (*VideoListResult, error)
	// ListUserVideos returns a user's videos with an optional status filter, page-ordered by id DESC.
	ListUserVideos(ctx context.Context, uid uint64, status string, page, pageSize int) ([]video.Video, int64, error)
	// ListUserVideosAdvanced returns a user's videos with title search and custom sort.
	ListUserVideosAdvanced(ctx context.Context, f MyVideoFilter) (*MyVideoPageResult, error)
	// ListUserPublishedVideosCursor returns a user's published videos, cursor-based.
	ListUserPublishedVideosCursor(ctx context.Context, uid uint64, cursorID uint64, limit int) ([]video.Video, error)
	// ListDrafts returns a user's drafts, page-ordered by updated_at DESC.
	ListDrafts(ctx context.Context, uid uint64, page, pageSize int) ([]video.Video, int64, error)
	// CountDrafts counts a user's draft videos.
	CountDrafts(ctx context.Context, uid uint64) (int64, error)
	// CountVideosByStatusForUser returns per-status counts for a user's videos.
	CountVideosByStatusForUser(uid uint64) map[string]int64
	// CountZoneVideos returns the published-video count for a zone.
	CountZoneVideos(zoneParent string) int64
	// CountPublishedVideos returns the total published-video count.
	CountPublishedVideos(ctx context.Context) int64
	// CountByStatus returns the video count for a single status.
	CountByStatus(ctx context.Context, status string) (int64, error)
	// ToggleVideoLike creates/removes a like row and adjusts like_count (business check done by caller).
	ToggleVideoLike(ctx context.Context, userID, videoID uint64) (bool, error)
	// PublishVideo marks a video published and (re)indexes search. Monolith-phase flow.
	PublishVideo(ctx context.Context, esc *search.Client, log *zap.Logger, videoID uint64, adminID *uint64) error
	// AdminDeleteVideoCascade runs the cascade-delete callback inside a transaction.
	AdminDeleteVideoCascade(ctx context.Context, id uint64, fn func(tx *gorm.DB) error) error
	// WithTx runs fn inside a database transaction (Phase 1 monolith seam).
	WithTx(ctx context.Context, fn func(tx *gorm.DB) error) error

	// AdminListVideos returns paginated videos with status/title filters for the admin panel.
	AdminListVideos(ctx context.Context, statuses []string, titleQ string, page, pageSize int) (*AdminListVideosResult, error)
}

// VideoInfo is the cross-domain video data.
type VideoInfo struct {
	ID              uint64
	UserID          uint64
	Title           string
	CoverURL        string
	PlayCount       uint64
	DanmakuCount    uint64
	CommentCount    uint64
	DurationSec     float64
	FavCount        uint64
	Status          string
	CommentsClosed  bool
	CommentsCurated bool
	DanmakuClosed   bool
	CreatedAt       time.Time
}
