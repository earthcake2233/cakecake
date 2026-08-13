package video

import (
	"cakecake/internal/model/video"
	"context"
	"errors"
	"fmt"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"cakecake/internal/ffmpeg"
	"cakecake/internal/queue"
)

// VideoDraftService handles video draft business logic.
type VideoDraftService struct {
	videos VideoProvider
	rdb    *redis.Client
	log    *zap.Logger
	mq     queue.TranscodePublisher
	oss    SourceObjectStore
}

// ErrReplaceMediaUpdate marks a failure at the first (field-update) stage of
// ReplaceMedia, where the caller should clean up newly saved temp files.
var ErrReplaceMediaUpdate = errors.New("replace media: draft field update failed")

// NewVideoDraftService creates a VideoDraftService for the draft editing flow.
func NewVideoDraftService(db *gorm.DB, rdb *redis.Client, log *zap.Logger, mq queue.TranscodePublisher, oss SourceObjectStore) *VideoDraftService {
	return &VideoDraftService{videos: NewVideoProvider(db), rdb: rdb, log: log, mq: mq, oss: oss}
}

// PublishTranscode enqueues a transcode job (best-effort via RabbitMQ).
func (s *VideoDraftService) PublishTranscode(ctx context.Context, body []byte) error {
	if s.mq == nil {
		return fmt.Errorf("transcode queue not configured")
	}
	return s.mq.PublishTranscode(ctx, body)
}

// EnqueueTranscode builds the transcode job and publishes it to the queue.
func (s *VideoDraftService) EnqueueTranscode(ctx context.Context, videoID uint64, rawPath, coverPath string) error {
	return enqueueTranscodeJob(ctx, s.mq, s.oss, videoID, rawPath, coverPath)
}

// ProbeDurationSeconds probes a raw media file's duration via ffprobe.
func (s *VideoDraftService) ProbeDurationSeconds(path string) (float64, error) {
	return VideoProbe(path)
}

// FFprobeExe returns the ffprobe executable path used by the probe helpers.
func (s *VideoDraftService) FFprobeExe() string {
	return ffmpeg.FFprobeExe()
}

// ReplaceMediaOpts carries the validated media-replacement fields.
type ReplaceMediaOpts struct {
	Title       string
	Description string
	TagsJSON    string
	Zone        string
	RawPath     string
	CoverPath   string
	DurationSec float64
}

// ReplaceMedia swaps a draft's media, marks it processing, and queues transcoding.
func (s *VideoDraftService) ReplaceMedia(ctx context.Context, v *video.Video, opts ReplaceMediaOpts) error {
	updates := map[string]interface{}{
		"title":            opts.Title,
		"description":      opts.Description,
		"tags_json":        opts.TagsJSON,
		"status":           video.StatusProcessing,
		"fail_reason":      "",
		"video_url":        "",
		"cover_url":        "",
		"duration_sec":     opts.DurationSec,
		"draft_raw_path":   opts.RawPath,
		"draft_cover_path": opts.CoverPath,
	}
	if opts.Zone != "" {
		updates["zone"] = opts.Zone
	}
	if err := s.UpdateDraft(ctx, v, updates); err != nil {
		return fmt.Errorf("%w: %v", ErrReplaceMediaUpdate, err)
	}
	if err := s.EnqueueTranscode(ctx, v.ID, opts.RawPath, opts.CoverPath); err != nil {
		return err
	}
	return s.UpdateDraft(ctx, v, map[string]interface{}{
		"draft_raw_path":   "",
		"draft_cover_path": "",
	})
}

// CreateDraft inserts a new draft video record.
func (s *VideoDraftService) CreateDraft(ctx context.Context, v *video.Video) error {
	return s.videos.CreateVideo(ctx, v)
}

// GetDraftByID returns a draft by its ID (without ownership check).
func (s *VideoDraftService) GetDraftByID(ctx context.Context, id uint64) (*video.Video, error) {
	return s.videos.GetVideoByID(ctx, id)
}

// GetOwnedDraft returns a draft by ID only if it belongs to uid and has draft status.
func (s *VideoDraftService) GetOwnedDraft(ctx context.Context, id, uid uint64) (*video.Video, error) {
	return s.videos.GetVideoByUser(ctx, id, uid)
}

// GetOwnedDraftByStatus returns a draft by ID, user ID, and status.
func (s *VideoDraftService) GetOwnedDraftByStatus(ctx context.Context, id, uid uint64, status string) (*video.Video, error) {
	return s.videos.GetVideoByUserStatus(ctx, id, uid, status)
}

// UpdateDraft updates the given draft record with the specified fields.
func (s *VideoDraftService) UpdateDraft(ctx context.Context, v *video.Video, updates map[string]interface{}) error {
	return s.videos.UpdateVideo(ctx, v, updates)
}

// UpdateDraftField updates a single field of a draft record.
func (s *VideoDraftService) UpdateDraftField(ctx context.Context, v *video.Video, field string, value interface{}) error {
	return s.videos.UpdateVideoField(ctx, v, field, value)
}

// DeleteDraft deletes a draft video by ID.
func (s *VideoDraftService) DeleteDraft(ctx context.Context, id uint64) error {
	return s.videos.DeleteVideo(ctx, id)
}

// CountUserDrafts counts draft videos for a user.
func (s *VideoDraftService) CountUserDrafts(ctx context.Context, uid uint64) (int64, error) {
	return s.videos.CountDrafts(ctx, uid)
}

// ListUserDrafts lists draft videos for a user with pagination.
func (s *VideoDraftService) ListUserDrafts(ctx context.Context, uid uint64, page, pageSize int) ([]video.Video, int64, error) {
	return s.videos.ListDrafts(ctx, uid, page, pageSize)
}

// RefetchDraft re-fetches a draft from the database by ID.
func (s *VideoDraftService) RefetchDraft(ctx context.Context, id uint64) (*video.Video, error) {
	return s.GetDraftByID(ctx, id)
}
