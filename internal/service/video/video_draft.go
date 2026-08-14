package video

import (
	"cakecake/internal/model/video"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"cakecake/internal/ffmpeg"
	"cakecake/internal/pkg/dbtx"
	"cakecake/internal/pkg/traceid"
	"cakecake/internal/queue"
)

// VideoDraftService handles video draft business logic. Since the draft media
// lives in object storage (drafts/{uid}/...), drafts are safe across instances
// and container rebuilds, unlike the legacy local-temp-file drafts.
type VideoDraftService struct {
	videos       VideoProvider
	rdb          *redis.Client
	log          *zap.Logger
	mq           queue.TranscodePublisher
	oss          SourceObjectStore
	backpressure BackpressureChecker
}

var (
	// ErrDraftMediaUnavailable reports that OSS is not configured, so draft
	// media cannot be staged or submitted.
	ErrDraftMediaUnavailable = errors.New("draft media unavailable: OSS not configured")
	// ErrDraftMediaEmpty reports a missing raw object key.
	ErrDraftMediaEmpty = errors.New("draft media raw object key empty")
	// ErrDraftMediaInvalidKey reports a raw/cover key outside the user's drafts namespace.
	ErrDraftMediaInvalidKey = errors.New("draft media object key invalid")
	// ErrDraftMediaMissing reports that a claimed draft object does not exist.
	ErrDraftMediaMissing = errors.New("draft media object missing")
	// ErrDraftMediaTooLarge reports that a draft object exceeds the size limit.
	ErrDraftMediaTooLarge = errors.New("draft media object too large")
)

// DraftUploadTicket is the presigned-PUT ticket a client uses to stage draft
// media (raw video + optional cover) straight to OSS before saving metadata.
type DraftUploadTicket struct {
	RawKey   string `json:"raw_key"`
	RawURL   string `json:"raw_upload_url"`
	CoverKey string `json:"cover_key,omitempty"`
	CoverURL string `json:"cover_upload_url,omitempty"`
	// ExpiresIn is the ticket lifetime in seconds.
	ExpiresIn int64 `json:"expires_in"`
}

// DraftMedia is validated draft media ready for submission.
type DraftMedia struct {
	RawKey      string
	CoverKey    string
	DurationSec float64
}

const (
	// draftUploadExpiry bounds presigned PUT URLs for draft media.
	draftUploadExpiry = 15 * time.Minute
	// draftUploadMaxBytes mirrors the direct-upload ceiling (500 MB).
	draftUploadMaxBytes int64 = 500 << 20
	// draftCoverMaxBytes bounds the optional cover object (10 MB).
	draftCoverMaxBytes int64 = 10 << 20
)

// NewVideoDraftService creates a VideoDraftService for the draft editing flow.
func NewVideoDraftService(db *gorm.DB, rdb *redis.Client, log *zap.Logger, mq queue.TranscodePublisher, oss SourceObjectStore, backpressure ...BackpressureChecker) *VideoDraftService {
	s := &VideoDraftService{videos: NewVideoProvider(db), rdb: rdb, log: log, mq: mq, oss: oss}
	if len(backpressure) > 0 {
		s.backpressure = backpressure[0]
	}
	return s
}

// draftKeyPrefix namespaces draft objects per owner so a submit can never
// reference another user's media.
func draftKeyPrefix(uid uint64) string {
	return fmt.Sprintf("drafts/%d/", uid)
}

func isDraftKey(uid uint64, key string) bool {
	return strings.HasPrefix(key, draftKeyPrefix(uid))
}

// PublishTranscode enqueues a transcode job (best-effort via RabbitMQ).
func (s *VideoDraftService) PublishTranscode(ctx context.Context, body []byte) error {
	if s.mq == nil {
		return fmt.Errorf("transcode queue not configured")
	}
	return s.mq.PublishTranscode(ctx, body)
}

// EnqueueTranscode builds the transcode job and publishes it to the queue
// (legacy local-path path; kept for old drafts and server-side ingest).
func (s *VideoDraftService) EnqueueTranscode(ctx context.Context, videoID uint64, rawPath, coverPath string) error {
	if err := s.checkCapacity(ctx); err != nil {
		return err
	}
	return enqueueOutboxJob(ctx, s.videos, s.oss, videoID, rawPath, coverPath, traceid.FromContext(ctx))
}

// CreateDraftUploadTicket issues presigned PUT URLs for staging draft media.
func (s *VideoDraftService) CreateDraftUploadTicket(_ context.Context, uid uint64, filename, coverFilename, rawContentType, coverContentType string) (*DraftUploadTicket, error) {
	if s.oss == nil {
		return nil, ErrDraftMediaUnavailable
	}
	prefix := draftKeyPrefix(uid)
	rawKey := fmt.Sprintf("%s%s/source%s", prefix, uuid.NewString(), safeObjectExt(filename))
	rawURL, err := s.oss.PresignPut(rawKey, draftUploadExpiry, uploadContentType(rawContentType))
	if err != nil {
		return nil, fmt.Errorf("presign draft raw upload: %w", err)
	}
	ticket := &DraftUploadTicket{
		RawKey:    rawKey,
		RawURL:    rawURL,
		ExpiresIn: int64(draftUploadExpiry.Seconds()),
	}
	if strings.TrimSpace(coverFilename) != "" {
		coverKey := fmt.Sprintf("%s%s/cover%s", prefix, uuid.NewString(), safeObjectExt(coverFilename))
		coverURL, err := s.oss.PresignPut(coverKey, draftUploadExpiry, uploadContentType(coverContentType))
		if err != nil {
			return nil, fmt.Errorf("presign draft cover upload: %w", err)
		}
		ticket.CoverKey = coverKey
		ticket.CoverURL = coverURL
	}
	return ticket, nil
}

// ValidateDraftMedia verifies that the submitted objects exist, stay inside
// the owner's namespace and size limits. It performs cheap HEAD checks only:
// duration validation happens in the transcode worker after it downloads the
// source, so saving/publishing a draft never blocks on a full object download.
func (s *VideoDraftService) ValidateDraftMedia(ctx context.Context, uid uint64, rawKey, coverKey string) (DraftMedia, error) {
	if s.oss == nil {
		return DraftMedia{}, ErrDraftMediaUnavailable
	}
	rawKey = strings.TrimSpace(rawKey)
	coverKey = strings.TrimSpace(coverKey)
	if rawKey == "" {
		return DraftMedia{}, ErrDraftMediaEmpty
	}
	if !isDraftKey(uid, rawKey) {
		return DraftMedia{}, fmt.Errorf("%w: raw key %s", ErrDraftMediaInvalidKey, rawKey)
	}
	if coverKey != "" && !isDraftKey(uid, coverKey) {
		return DraftMedia{}, fmt.Errorf("%w: cover key %s", ErrDraftMediaInvalidKey, coverKey)
	}
	ok, err := s.oss.Exists(rawKey)
	if err != nil {
		return DraftMedia{}, fmt.Errorf("check draft raw object: %w", err)
	}
	if !ok {
		return DraftMedia{}, fmt.Errorf("%w: raw %s", ErrDraftMediaMissing, rawKey)
	}
	size, err := s.oss.Size(rawKey)
	if err != nil {
		return DraftMedia{}, fmt.Errorf("check draft raw size: %w", err)
	}
	if size > draftUploadMaxBytes {
		return DraftMedia{}, fmt.Errorf("%w: raw %d bytes", ErrDraftMediaTooLarge, size)
	}
	if coverKey != "" {
		ok, err = s.oss.Exists(coverKey)
		if err != nil {
			return DraftMedia{}, fmt.Errorf("check draft cover object: %w", err)
		}
		if !ok {
			return DraftMedia{}, fmt.Errorf("%w: cover %s", ErrDraftMediaMissing, coverKey)
		}
		coverSize, err := s.oss.Size(coverKey)
		if err != nil {
			return DraftMedia{}, fmt.Errorf("check draft cover size: %w", err)
		}
		if coverSize > draftCoverMaxBytes {
			return DraftMedia{}, fmt.Errorf("%w: cover %d bytes", ErrDraftMediaTooLarge, coverSize)
		}
	}
	return DraftMedia{RawKey: rawKey, CoverKey: coverKey}, nil
}

// ValidateDraftCover verifies a cover-only replacement: the object exists,
// belongs to the owner's draft namespace and stays within the size limit.
func (s *VideoDraftService) ValidateDraftCover(ctx context.Context, uid uint64, coverKey string) error {
	if s.oss == nil {
		return ErrDraftMediaUnavailable
	}
	coverKey = strings.TrimSpace(coverKey)
	if coverKey == "" {
		return fmt.Errorf("%w: cover key empty", ErrDraftMediaEmpty)
	}
	if !isDraftKey(uid, coverKey) {
		return fmt.Errorf("%w: cover key %s", ErrDraftMediaInvalidKey, coverKey)
	}
	ok, err := s.oss.Exists(coverKey)
	if err != nil {
		return fmt.Errorf("check draft cover object: %w", err)
	}
	if !ok {
		return fmt.Errorf("%w: cover %s", ErrDraftMediaMissing, coverKey)
	}
	size, err := s.oss.Size(coverKey)
	if err != nil {
		return fmt.Errorf("check draft cover size: %w", err)
	}
	if size > draftCoverMaxBytes {
		return fmt.Errorf("%w: cover %d bytes", ErrDraftMediaTooLarge, size)
	}
	return nil
}

// SubmitDraft atomically writes the transcode outbox row and moves the draft
// to processing. Either both happen or neither does; the draft keys are
// cleared so a retry cannot double-submit the same staging objects.
func (s *VideoDraftService) SubmitDraft(ctx context.Context, v *video.Video, media DraftMedia) error {
	if err := s.checkCapacity(ctx); err != nil {
		return err
	}
	return s.videos.WithTx(ctx, func(tx dbtx.Tx) error {
		job := buildTranscodeJob(v.ID, media.RawKey, media.CoverKey, "", "", traceid.FromContext(ctx))
		payload, err := json.Marshal(job)
		if err != nil {
			return err
		}
		if err := tx.Create(&video.TranscodeOutbox{
			JobID:   job.JobID,
			VideoID: v.ID,
			Payload: string(payload),
			Status:  video.OutboxStatusPending,
		}).Error; err != nil {
			return err
		}
		updates := map[string]interface{}{
			"status":          video.StatusProcessing,
			"fail_reason":     "",
			"video_url":       "",
			"duration_sec":    media.DurationSec,
			"draft_raw_key":   "",
			"draft_cover_key": "",
		}
		return tx.Model(&video.Video{}).Where("id = ?", v.ID).Updates(updates).Error
	})
}

// ProbeDurationSeconds probes a raw media file's duration via ffprobe.
func (s *VideoDraftService) ProbeDurationSeconds(path string) (float64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), probeTimeout)
	defer cancel()
	return VideoProbe(ctx, path)
}

// FFprobeExe returns the ffprobe executable path used by the probe helpers.
func (s *VideoDraftService) FFprobeExe() string {
	return ffmpeg.FFprobeExe()
}

// ReplaceMediaOpts carries the validated media-replacement fields. Media is
// referenced by OSS object keys staged through CreateDraftUploadTicket.
type ReplaceMediaOpts struct {
	Title       string
	Description string
	TagsJSON    string
	Zone        string
	RawKey      string
	CoverKey    string
	DurationSec float64
}

// ReplaceMedia swaps a failed/rejected video's media and atomically enqueues
// transcoding: the outbox row and the status/meta update commit together, so
// there is no window where a job is queued but the row still says failed.
func (s *VideoDraftService) ReplaceMedia(ctx context.Context, v *video.Video, opts ReplaceMediaOpts) error {
	if err := s.checkCapacity(ctx); err != nil {
		return err
	}
	return s.videos.WithTx(ctx, func(tx dbtx.Tx) error {
		job := buildTranscodeJob(v.ID, opts.RawKey, opts.CoverKey, "", "", traceid.FromContext(ctx))
		payload, err := json.Marshal(job)
		if err != nil {
			return err
		}
		if err := tx.Create(&video.TranscodeOutbox{
			JobID:   job.JobID,
			VideoID: v.ID,
			Payload: string(payload),
			Status:  video.OutboxStatusPending,
		}).Error; err != nil {
			return err
		}
		updates := map[string]interface{}{
			"title":           opts.Title,
			"description":     opts.Description,
			"tags_json":       opts.TagsJSON,
			"status":          video.StatusProcessing,
			"fail_reason":     "",
			"video_url":       "",
			"duration_sec":    opts.DurationSec,
			"draft_raw_key":   "",
			"draft_cover_key": "",
		}
		if opts.Zone != "" {
			updates["zone"] = opts.Zone
		}
		return tx.Model(&video.Video{}).Where("id = ?", v.ID).Updates(updates).Error
	})
}

func (s *VideoDraftService) checkCapacity(ctx context.Context) error {
	if s.backpressure == nil {
		return nil
	}
	if err := s.backpressure.CheckTranscodeCapacity(ctx); err != nil {
		return fmt.Errorf("%w: %v", ErrTranscodeQueueFull, err)
	}
	return nil
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
