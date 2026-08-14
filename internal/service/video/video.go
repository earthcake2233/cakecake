package video

import (
	"cakecake/internal/model/video"
	"cakecake/internal/pkg/dbtx"
	"cakecake/internal/pkg/traceid"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
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
	// backpressure rejects new enqueues when the transcode queue is full.
	backpressure BackpressureChecker
}

// ErrRequeueSourceMissing reports that the raw media (or user cover) behind a
// dead letter no longer exists on disk, so requeue cannot succeed.
var ErrRequeueSourceMissing = errors.New("requeue source media missing")

// SourceObjectStore keeps the original media durable so dead-letter requeue
// works across instances and container rebuilds instead of depending on a
// single-host local path.
type SourceObjectStore interface {
	Exists(objectKey string) (bool, error)
	Size(objectKey string) (int64, error)
	PresignPut(objectKey string, expiry time.Duration, contentType string) (string, error)
}

// ErrDirectUploadUnavailable reports that OSS is not configured for
// client-side direct upload.
var ErrDirectUploadUnavailable = errors.New("direct upload unavailable: OSS not configured")

// ErrDirectUploadSourceMissing reports that the OSS object claimed by a
// direct-upload submission does not exist.
var ErrDirectUploadSourceMissing = errors.New("direct upload source object missing")

// ErrDirectUploadTooLarge reports that the OSS object exceeds the upload size
// limit; it is rejected before any download so a large object cannot be used
// as a disk/bandwidth DoS vector.
var ErrDirectUploadTooLarge = errors.New("direct upload source object too large")

// ErrDirectUploadInvalidKey reports that a submitted object key is outside
// the uploads/{uid}/ namespace.
var ErrDirectUploadInvalidKey = errors.New("direct upload object key invalid")

// ErrTranscodeQueueFull reports that the transcode queue is over capacity
// (backpressure); uploads should be retried later.
var ErrTranscodeQueueFull = errors.New("transcode queue full")

// BackpressureChecker rejects transcode enqueues when capacity is exhausted.
type BackpressureChecker interface {
	CheckTranscodeCapacity(ctx context.Context) error
}

// ErrDirectUploadAlreadyClaimed reports that the raw object belongs to a
// different user's claim.
var ErrDirectUploadAlreadyClaimed = errors.New("direct upload object already claimed")

// ErrDirectUploadInProgress reports that a concurrent submit for the same raw
// object is still creating its video row.
var ErrDirectUploadInProgress = errors.New("direct upload object claim in progress")

// directUploadMaxBytes matches the multipart upload limit (500 MB) so both
// upload paths enforce the same ceiling.
const directUploadMaxBytes int64 = 500 << 20

// maxMediaDurationSec caps the media duration hint stored at submit time.
// The authoritative duration check happens in the transcode worker, which
// already downloads the source; the hint is only used for the player UI.
const maxMediaDurationSec = 30 * 60

// NormalizeDurationHint clamps a client-provided duration to a safe display
// value. It is advisory: the worker re-probes and overwrites it, and rejects
// media that actually exceeds the limit.
func NormalizeDurationHint(d float64) float64 {
	if d < 0 || d != d { // negative or NaN
		return 0
	}
	if d > maxMediaDurationSec {
		return maxMediaDurationSec
	}
	return d
}

// directUploadExpiry bounds the presigned PUT URLs issued by
// CreateDirectUploadTicket.
const directUploadExpiry = 15 * time.Minute

// DirectUploadTicket is the presigned-PUT ticket a client uses to upload the
// raw video (and optional cover) straight to OSS before submitting metadata.
type DirectUploadTicket struct {
	RawKey   string `json:"raw_key"`
	RawURL   string `json:"raw_upload_url"`
	CoverKey string `json:"cover_key,omitempty"`
	CoverURL string `json:"cover_upload_url,omitempty"`
	// ExpiresIn is the ticket lifetime in seconds.
	ExpiresIn int64 `json:"expires_in"`
}

// VideoProbe is the media-duration probe used by the video services. It is a
// variable so tests can substitute a deterministic probe without invoking
// the external ffprobe binary.
var VideoProbe = ffmpeg.ProbeDurationSeconds

// probeTimeout bounds a single ffprobe run during upload validation; a hung
// probe must not hold the HTTP request forever.
var probeTimeout = 15 * time.Second

// NewVideoService creates a VideoService with storage, cache, logger,
// optional search client, and optional transcode queue publisher.
func NewVideoService(db *gorm.DB, rdb *redis.Client, log *zap.Logger, es *search.Client, mq queue.TranscodePublisher, oss SourceObjectStore, backpressure ...BackpressureChecker) *VideoService {
	s := &VideoService{videos: NewVideoProvider(db), rdb: rdb, log: log, es: es, mq: mq, oss: oss}
	if len(backpressure) > 0 {
		s.backpressure = backpressure[0]
	}
	return s
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

// CreateDirectUploadTicket issues presigned PUT URLs for a direct upload:
// the browser uploads the source files straight to OSS, then submits the
// metadata via CreateVideoFromDirectUpload. This keeps large files off the
// API server's bandwidth. Object keys are namespaced by the owning user
// (uploads/{uid}/{uuid}/...) so a later submit cannot reference someone
// else's objects.
func (s *VideoService) CreateDirectUploadTicket(_ context.Context, uid uint64, filename, coverFilename, rawContentType, coverContentType string) (*DirectUploadTicket, error) {
	if s.oss == nil {
		return nil, ErrDirectUploadUnavailable
	}
	prefix := directUploadKeyPrefix(uid)
	rawKey := fmt.Sprintf("%s%s/source%s", prefix, uuid.NewString(), safeObjectExt(filename))
	rawURL, err := s.oss.PresignPut(rawKey, directUploadExpiry, uploadContentType(rawContentType))
	if err != nil {
		return nil, fmt.Errorf("presign raw upload: %w", err)
	}
	ticket := &DirectUploadTicket{
		RawKey:    rawKey,
		RawURL:    rawURL,
		ExpiresIn: int64(directUploadExpiry.Seconds()),
	}
	if strings.TrimSpace(coverFilename) != "" {
		coverKey := fmt.Sprintf("%s%s/cover%s", prefix, uuid.NewString(), safeObjectExt(coverFilename))
		coverURL, err := s.oss.PresignPut(coverKey, directUploadExpiry, uploadContentType(coverContentType))
		if err != nil {
			return nil, fmt.Errorf("presign cover upload: %w", err)
		}
		ticket.CoverKey = coverKey
		ticket.CoverURL = coverURL
	}
	return ticket, nil
}

// uploadContentType normalizes the browser-reported MIME type. The value is
// signed into the presigned PUT URL, so it must exactly match the
// Content-Type header the browser sends.
func uploadContentType(ct string) string {
	ct = strings.TrimSpace(ct)
	if ct == "" {
		return "application/octet-stream"
	}
	return ct
}

// CreateVideoFromDirectUpload validates an OSS-uploaded source with cheap
// HEAD checks only (existence + size limit — no full download), creates the
// video row with the client-provided duration hint and enqueues transcoding.
// The authoritative duration check is done by the worker after it downloads
// the source.
func (s *VideoService) CreateVideoFromDirectUpload(ctx context.Context, uid uint64, title, description, tagsJSON, zone, rawKey, coverKey string, durationHint float64) (*video.Video, error) {
	if s.oss == nil {
		return nil, ErrDirectUploadUnavailable
	}
	if err := s.checkCapacity(ctx); err != nil {
		return nil, err
	}
	prefix := directUploadKeyPrefix(uid)
	if !strings.HasPrefix(rawKey, prefix) {
		return nil, fmt.Errorf("%w: raw key %s", ErrDirectUploadInvalidKey, rawKey)
	}
	if coverKey != "" && !strings.HasPrefix(coverKey, prefix) {
		return nil, fmt.Errorf("%w: cover key %s", ErrDirectUploadInvalidKey, coverKey)
	}
	ok, err := s.oss.Exists(rawKey)
	if err != nil {
		return nil, fmt.Errorf("check raw object: %w", err)
	}
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrDirectUploadSourceMissing, rawKey)
	}
	size, err := s.oss.Size(rawKey)
	if err != nil {
		return nil, fmt.Errorf("check raw object size: %w", err)
	}
	if size > directUploadMaxBytes {
		return nil, fmt.Errorf("%w: %d bytes", ErrDirectUploadTooLarge, size)
	}
	var v video.Video
	err = s.videos.WithTx(ctx, func(tx dbtx.Tx) error {
		claim := video.DirectUploadClaim{RawKey: rawKey, UserID: uid}
		if err := tx.Create(&claim).Error; err != nil {
			if !isUniqueViolation(err) {
				return err
			}
			var existing video.DirectUploadClaim
			if err := tx.Where("raw_key = ?", rawKey).First(&existing).Error; err != nil {
				return err
			}
			if existing.UserID != uid {
				return fmt.Errorf("%w: key %s", ErrDirectUploadAlreadyClaimed, rawKey)
			}
			if existing.VideoID == 0 {
				return fmt.Errorf("%w: key %s", ErrDirectUploadInProgress, rawKey)
			}
			if err := tx.First(&v, existing.VideoID).Error; err != nil {
				return err
			}
			return nil
		}
		v = video.Video{
			UserID:      uid,
			Title:       title,
			Description: description,
			DurationSec: NormalizeDurationHint(durationHint),
			Status:      video.StatusProcessing,
			TagsJSON:    tagsJSON,
			Zone:        zone,
		}
		if err := tx.Create(&v).Error; err != nil {
			return err
		}
		if err := tx.Model(&video.DirectUploadClaim{}).Where("id = ?", claim.ID).
			Updates(map[string]interface{}{"video_id": v.ID}).Error; err != nil {
			return err
		}
		// Outbox: the video row, the claim and the pending job are committed
		// atomically; the relay publishes it later.
		job := buildTranscodeJob(v.ID, rawKey, coverKey, "", "", traceid.FromContext(ctx))
		payload, err := json.Marshal(job)
		if err != nil {
			return err
		}
		return tx.Create(&video.TranscodeOutbox{
			JobID:   job.JobID,
			VideoID: v.ID,
			Payload: string(payload),
			Status:  video.OutboxStatusPending,
		}).Error
	})
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func (s *VideoService) checkCapacity(ctx context.Context) error {
	if s.backpressure == nil {
		return nil
	}
	if err := s.backpressure.CheckTranscodeCapacity(ctx); err != nil {
		return fmt.Errorf("%w: %v", ErrTranscodeQueueFull, err)
	}
	return nil
}

func directUploadKeyPrefix(uid uint64) string {
	return fmt.Sprintf("uploads/%d/", uid)
}

func safeObjectExt(filename string) string {
	ext := strings.ToLower(filepath.Ext(strings.TrimSpace(filename)))
	if len(ext) > 16 {
		return ""
	}
	for _, r := range ext {
		if !(r >= 'a' && r <= 'z') && !(r >= '0' && r <= '9') && r != '.' {
			return ""
		}
	}
	return ext
}

// enqueueOutboxJob writes a pending outbox row for a legacy local-path job
// (pre-migration local drafts / server-side ingest). The background relay
// publishes it to RabbitMQ with publisher confirm. Local files are the source
// of truth for these jobs and are kept until the worker consumes them.
func enqueueOutboxJob(ctx context.Context, videos VideoProvider, oss SourceObjectStore, videoID uint64, rawPath, coverPath, traceID string) error {
	job := buildTranscodeJob(videoID, "", "", rawPath, coverPath, traceID)
	payload, err := json.Marshal(job)
	if err != nil {
		return err
	}
	if err := videos.CreateTranscodeOutbox(ctx, &video.TranscodeOutbox{
		JobID:   job.JobID,
		VideoID: videoID,
		Payload: string(payload),
		Status:  video.OutboxStatusPending,
	}); err != nil {
		return err
	}
	return nil
}

// buildTranscodeJob assembles the durable job payload with a stable JobID for
// dedup and an optional trace ID for end-to-end correlation.
func buildTranscodeJob(videoID uint64, rawKey, coverKey, rawPath, coverPath, traceID string) queue.TranscodeJob {
	job := queue.TranscodeJob{
		VideoID:    videoID,
		RawPath:    rawPath,
		CoverPath:  coverPath,
		RawKey:     rawKey,
		CoverKey:   coverKey,
		JobID:      uuid.NewString(),
		TraceID:    traceID,
		RetryCount: 0,
	}
	if rawKey != "" {
		job.RawPath = ""
	}
	if coverKey != "" {
		job.CoverPath = ""
	}
	return job
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
	// A replay is a brand-new execution: mint a fresh JobID so the worker's
	// (job_id, retry_count) dedup row from the original attempt does not
	// swallow this job. TraceID is kept for end-to-end correlation.
	job.JobID = uuid.NewString()
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
	ctx, cancel := context.WithTimeout(context.Background(), probeTimeout)
	defer cancel()
	return VideoProbe(ctx, path)
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
	Videos []video.Video
	// UploaderNames maps video UserID to its display name ("" when the user
	// is missing; anonymized accounts map to the 已注销用户 marker).
	UploaderNames  map[uint64]string
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

// AdminRejectVideoWithAudit rejects a pending-review video and records the
// status transition in the transcode audit trail.
func (s *VideoService) AdminRejectVideoWithAudit(ctx context.Context, id uint64, reason string, adminID uint64) error {
	v, err := s.videos.GetVideoByID(ctx, id)
	if err != nil {
		return err
	}
	now := time.Now()
	if err := s.videos.UpdateVideoByID(ctx, id, map[string]interface{}{
		"status":               video.StatusRejected,
		"fail_reason":          reason,
		"reviewed_at":          now,
		"reviewed_by_admin_id": adminID,
	}); err != nil {
		return err
	}
	ev := video.TranscodeEvent{
		VideoID:    id,
		FromStatus: v.Status,
		ToStatus:   video.StatusRejected,
		Reason:     reason,
	}
	return s.videos.RecordTranscodeEvent(ctx, &ev)
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
