package video

import (
	"cakecake/internal/model/video"
	"cakecake/internal/pkg/dbtx"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"cakecake/internal/ffmpeg"
	"cakecake/internal/queue"
	"cakecake/internal/search"
)

// VideoService handles video business logic.
type VideoService struct {
	videos VideoProvider
	rdb    *redis.Client
	log    *zap.Logger
	es     *search.Client
	mq     queue.TranscodePublisher
	oss    SourceObjectStore
}

// ErrRequeueSourceMissing reports that the raw media (or user cover) behind a
// dead letter no longer exists on disk, so requeue cannot succeed.
var ErrRequeueSourceMissing = errors.New("requeue source media missing")

// SourceObjectStore keeps the original media durable so dead-letter requeue
// works across instances and container rebuilds instead of depending on a
// single-host local path.
type SourceObjectStore interface {
	UploadFile(objectKey, localPath string) error
	Exists(objectKey string) (bool, error)
	DeleteObject(objectKey string) error
}

// VideoProbe is the media-duration probe used by the video services. It is a
// variable so tests can substitute a deterministic probe without invoking
// the external ffprobe binary.
var VideoProbe = ffmpeg.ProbeDurationSeconds

// NewVideoService creates a VideoService with storage, cache, logger,
// optional search client, and optional transcode queue publisher.
func NewVideoService(db *gorm.DB, rdb *redis.Client, log *zap.Logger, es *search.Client, mq queue.TranscodePublisher, oss SourceObjectStore) *VideoService {
	return &VideoService{videos: NewVideoProvider(db), rdb: rdb, log: log, es: es, mq: mq, oss: oss}
}

// Publish marks a video published and re-indexes search (post-review or direct publish).
func (s *VideoService) Publish(ctx context.Context, videoID uint64, adminID *uint64) error {
	return s.videos.PublishVideo(ctx, s.es, s.log, videoID, adminID)
}

// PublishTranscode enqueues a transcode job (best-effort via RabbitMQ).
func (s *VideoService) PublishTranscode(ctx context.Context, body []byte) error {
	if s.mq == nil {
		return fmt.Errorf("transcode queue not configured")
	}
	return s.mq.PublishTranscode(ctx, body)
}

// EnqueueTranscode builds the transcode job and publishes it to the queue.
func (s *VideoService) EnqueueTranscode(ctx context.Context, videoID uint64, rawPath, coverPath string) error {
	return enqueueTranscodeJob(ctx, s.mq, s.oss, videoID, rawPath, coverPath)
}

// enqueueTranscodeJob uploads the original media to object storage when OSS is
// configured, then publishes the job. The job carries object keys (RawKey /
// CoverKey) so a worker on any host can download and retry them; without OSS
// it falls back to the legacy local-path payload.
//
// Failure safety: the local temp files are deleted ONLY after the publish is
// confirmed, and already-uploaded source objects are KEPT on any failure. A
// transient broker error therefore never destroys the user's video: the OSS
// object (or the local file, in fallback mode) always survives for retry or
// manual compensation.
func enqueueTranscodeJob(ctx context.Context, mq queue.TranscodePublisher, oss SourceObjectStore, videoID uint64, rawPath, coverPath string) error {
	if mq == nil {
		return fmt.Errorf("transcode queue not configured")
	}
	job := queue.TranscodeJob{VideoID: videoID, RawPath: rawPath, CoverPath: coverPath, RetryCount: 0}
	if oss != nil && rawPath != "" {
		rawKey := fmt.Sprintf("raws/%d/source%s", videoID, filepath.Ext(rawPath))
		if err := oss.UploadFile(rawKey, rawPath); err != nil {
			return fmt.Errorf("upload raw source: %w", err)
		}
		job.RawKey, job.RawPath = rawKey, ""
	}
	if oss != nil && coverPath != "" {
		coverKey := fmt.Sprintf("raws/%d/cover%s", videoID, filepath.Ext(coverPath))
		if err := oss.UploadFile(coverKey, coverPath); err != nil {
			return fmt.Errorf("upload cover source: %w", err)
		}
		job.CoverKey, job.CoverPath = coverKey, ""
	}
	body, err := json.Marshal(job)
	if err != nil {
		return err
	}
	if err := mq.PublishTranscode(ctx, body); err != nil {
		return err
	}
	// Publish confirmed: the OSS objects are now the authoritative source.
	// Local temp copies are no longer needed on this host.
	if oss != nil {
		_ = os.Remove(rawPath)
		_ = os.Remove(coverPath)
	}
	return nil
}

// ListTranscodeDeadLetters returns dead-letter audit rows with pagination.
func (s *VideoService) ListTranscodeDeadLetters(ctx context.Context, f TranscodeDeadLetterFilter) ([]video.TranscodeDeadLetter, int64, error) {
	return s.videos.ListTranscodeDeadLetters(ctx, f)
}

// RequeueTranscodeDeadLetter re-publishes a dead letter to the main queue with
// retries reset, and marks the audit row requeued. The video is moved back to
// processing first so the idempotency guard does not skip the redelivery.
func (s *VideoService) RequeueTranscodeDeadLetter(ctx context.Context, id uint64) error {
	row, err := s.videos.GetTranscodeDeadLetter(ctx, id)
	if err != nil {
		return err
	}
	var payload struct {
		Job    queue.TranscodeJob `json:"job"`
		Reason string             `json:"reason"`
	}
	if err := json.Unmarshal([]byte(row.PayloadJSON), &payload); err != nil {
		return fmt.Errorf("dead letter payload invalid: %w", err)
	}
	if payload.Job.VideoID == 0 {
		return fmt.Errorf("dead letter payload missing job")
	}
	// The dead letter keeps RawKey/RawPath (and the cover equivalents) on
	// purpose, but the object/file can still be gone (retention, manual
	// cleanup). Fail fast with a clear error instead of publishing a job
	// that is guaranteed to fail again.
	if err := s.ensureRequeueSource("raw", payload.Job.RawKey, payload.Job.RawPath); err != nil {
		return err
	}
	if payload.Job.CoverKey != "" || payload.Job.CoverPath != "" {
		if err := s.ensureRequeueSource("cover", payload.Job.CoverKey, payload.Job.CoverPath); err != nil {
			return err
		}
	}
	job := payload.Job
	job.RetryCount = 0
	body, err := json.Marshal(job)
	if err != nil {
		return err
	}

	// Keep the pre-requeue audit state so a failed publish can be compensated:
	// "requeued" must mean the message actually reached the main queue, not
	// merely that an attempt was made.
	prevRequeuedAt, prevRequeuedCount, prevProcessedAt := row.RequeuedAt, row.RequeuedCount, row.ProcessedAt

	// DB first, then publish: if publish fails, revert both the video and the
	// audit row so the user sees a failed video and a still-pending dead letter.
	if err := s.videos.ResetVideoForTranscodeRequeue(ctx, job.VideoID); err != nil {
		return err
	}
	if err := s.videos.MarkTranscodeDeadLetterRequeued(ctx, id, time.Now()); err != nil {
		_ = s.videos.MarkVideoFailedByID(ctx, job.VideoID, "requeue db mark failed: "+err.Error())
		return err
	}
	revertAudit := func(reason string) {
		if rerr := s.videos.RevertTranscodeDeadLetterRequeue(ctx, id, prevRequeuedAt, prevRequeuedCount, prevProcessedAt); rerr != nil {
			s.log.Error("revert dead letter requeue audit failed",
				zap.Uint64("dead_letter_id", id),
				zap.Error(rerr))
		}
		_ = s.videos.MarkVideoFailedByID(ctx, job.VideoID, reason)
	}
	if s.mq == nil {
		revertAudit("requeue failed: transcode queue not configured")
		return fmt.Errorf("transcode queue not configured")
	}
	if err := s.mq.PublishTranscode(ctx, body); err != nil {
		revertAudit("requeue publish failed: " + err.Error())
		return err
	}
	incrTranscodeRequeued()
	return nil
}

// ensureRequeueSource verifies the compensation input behind a dead letter
// still exists: an OSS object when the job was enqueued with object keys,
// otherwise the legacy local path.
func (s *VideoService) ensureRequeueSource(kind, objectKey, localPath string) error {
	if objectKey != "" {
		if s.oss == nil {
			return fmt.Errorf("OSS 未配置，无法校验%s源对象 %s", kind, objectKey)
		}
		ok, err := s.oss.Exists(objectKey)
		if err != nil {
			return fmt.Errorf("check %s source object %s: %w", kind, objectKey, err)
		}
		if !ok {
			return fmt.Errorf("%w: %s object %s", ErrRequeueSourceMissing, kind, objectKey)
		}
		return nil
	}
	if localPath == "" {
		return nil
	}
	if _, err := os.Stat(localPath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("%w: %s media %s", ErrRequeueSourceMissing, kind, localPath)
		}
		return fmt.Errorf("check %s media before requeue: %w", kind, err)
	}
	return nil
}

// ProbeDurationSeconds probes a raw media file's duration via ffprobe.
func (s *VideoService) ProbeDurationSeconds(path string) (float64, error) {
	return VideoProbe(path)
}

// FFprobeExe returns the ffprobe executable path used by the probe helpers.
func (s *VideoService) FFprobeExe() string {
	return ffmpeg.FFprobeExe()
}

// HumanizeFailReason maps a stored failure code to a user-facing reason.
func (s *VideoService) HumanizeFailReason(reason string) string {
	return ffmpeg.HumanizeFailReason(reason)
}

// HumanizeFailReason maps a stored failure code to a user-facing reason.
func HumanizeFailReason(reason string) string {
	return ffmpeg.HumanizeFailReason(reason)
}

// CreateVideoRecord inserts a new video record into the database.
func (s *VideoService) CreateVideoRecord(ctx context.Context, v *video.Video) error {
	return s.videos.CreateVideo(ctx, v)
}

// DeleteVideoByID removes a video record by ID.
func (s *VideoService) DeleteVideoByID(ctx context.Context, id uint64) error {
	return s.videos.DeleteVideo(ctx, id)
}

// ListPublishedVideos queries published videos with optional filtering and cursor-based pagination.
func (s *VideoService) ListPublishedVideos(ctx context.Context, opts VideoListOpts) (*VideoListResult, error) {
	return s.videos.ListPublishedVideos(ctx, opts)
}

// CountZoneVideos returns the count of published videos in a zone.
func (s *VideoService) CountZoneVideos(zoneParent string) int64 {
	return s.videos.CountZoneVideos(zoneParent)
}

// CountMyVideosByStatus returns counts for each status for a user.
func (s *VideoService) CountMyVideosByStatus(uid uint64) map[string]int64 {
	return s.videos.CountVideosByStatusForUser(uid)
}

// GetPublishedVideo returns a published video by ID.
func (s *VideoService) GetPublishedVideo(ctx context.Context, id uint64) (*video.Video, error) {
	v, err := s.videos.GetVideoByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if v.Status != video.StatusPublished {
		return nil, gorm.ErrRecordNotFound
	}
	return v, nil
}

// GetVideoByID fetches a single video by ID.
func (s *VideoService) GetVideoByID(ctx context.Context, id uint64) (*video.Video, error) {
	return s.videos.GetVideoByID(ctx, id)
}

// UpdateVideo updates a video record with the given fields.
func (s *VideoService) UpdateVideo(ctx context.Context, v *video.Video, updates map[string]interface{}) error {
	return s.videos.UpdateVideo(ctx, v, updates)
}

// DeleteVideoWithCascade removes a video and its related data in a transaction.
func (s *VideoService) DeleteVideoWithCascade(ctx context.Context, id uint64, fn func(tx dbtx.Tx) error) error {
	if fn != nil {
		return s.videos.WithTx(ctx, fn)
	}
	// default: just delete the video record inside a transaction
	return s.videos.WithTx(ctx, func(tx dbtx.Tx) error {
		return tx.Delete(&video.Video{}, id).Error
	})
}

// ListMyVideos queries a user's own videos with optional status filter.
func (s *VideoService) ListMyVideos(ctx context.Context, uid uint64, status string, page, pageSize int) ([]video.Video, int64, error) {
	return s.videos.ListUserVideos(ctx, uid, status, page, pageSize)
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
	return s.videos.ToggleVideoLike(ctx, userID, videoID)
}

// CountPublishedVideos returns the total number of published videos.
func (s *VideoService) CountPublishedVideos(ctx context.Context) int64 {
	return s.videos.CountPublishedVideos(ctx)
}

// ListUserPublishedVideosCursor lists published videos for a user with cursor-based pagination.
func (s *VideoService) ListUserPublishedVideosCursor(ctx context.Context, uid uint64, cursorID uint64, limit int) ([]video.Video, error) {
	return s.videos.ListUserPublishedVideosCursor(ctx, uid, cursorID, limit)
}

// MyVideoFilter filters the caller's videos by status.
type MyVideoFilter struct {
	UserID   uint64
	Status   string   // single status
	Statuses []string // multi status (used when Status is empty)
	TitleQ   string
	SortKey  string // "time", "reply", "like"
	Page     int
	PageSize int
}

// MyVideoPageResult is a paginated video list for the creator panel.
type MyVideoPageResult struct {
	Videos     []video.Video
	Total      int64
	TotalPages int
}

// ListMyVideosAdvanced pages the caller's videos with the given filter.
func (s *VideoService) ListMyVideosAdvanced(ctx context.Context, f MyVideoFilter) (*MyVideoPageResult, error) {
	return s.videos.ListUserVideosAdvanced(ctx, f)
}

// CountByStatus returns video count by status.
func (s *VideoService) CountByStatus(ctx context.Context, status string) (int64, error) {
	return s.videos.CountByStatus(ctx, status)
}

// AdminUpdateVideo updates video fields by ID (admin operation).
func (s *VideoService) AdminUpdateVideo(ctx context.Context, id uint64, updates map[string]interface{}) error {
	return s.videos.UpdateVideoByID(ctx, id, updates)
}

// AdminDeleteVideoCascade deletes a video and cascades to related data within a transaction.
func (s *VideoService) AdminDeleteVideoCascade(ctx context.Context, id uint64, fn func(tx dbtx.Tx) error) error {
	return s.videos.AdminDeleteVideoCascade(ctx, id, fn)
}

// AdminListVideosResult holds paginated admin video list results.
type AdminListVideosResult struct {
	Total        int64
	Rows         []video.Video
	PendingCount int64
}

// AdminListVideos returns paginated videos with filters for admin panel.
func (s *VideoService) AdminListVideos(ctx context.Context, statuses []string, titleQ string, page, pageSize int) (*AdminListVideosResult, error) {
	return s.videos.AdminListVideos(ctx, statuses, titleQ, page, pageSize)
}
