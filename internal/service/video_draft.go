package service

import (
	"cakecake/internal/model/video"
	"context"
	"encoding/json"
	"fmt"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"cakecake/internal/ffmpeg"
	"cakecake/internal/queue"
)

// VideoDraftService handles video draft business logic.
type VideoDraftService struct {
	db  *gorm.DB
	rdb *redis.Client
	log *zap.Logger
	mq  queue.TranscodePublisher
}

func NewVideoDraftService(db *gorm.DB, rdb *redis.Client, log *zap.Logger, mq queue.TranscodePublisher) *VideoDraftService {
	return &VideoDraftService{db: db, rdb: rdb, log: log, mq: mq}
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
	job := queue.TranscodeJob{VideoID: videoID, RawPath: rawPath, CoverPath: coverPath, RetryCount: 0}
	body, err := json.Marshal(job)
	if err != nil {
		return err
	}
	return s.PublishTranscode(ctx, body)
}

// ProbeDurationSeconds probes a raw media file's duration via ffprobe.
func (s *VideoDraftService) ProbeDurationSeconds(path string) (float64, error) {
	return ffmpeg.ProbeDurationSeconds(path)
}

// FFprobeExe returns the ffprobe executable path used by the probe helpers.
func (s *VideoDraftService) FFprobeExe() string {
	return ffmpeg.FFprobeExe()
}

// CreateDraft inserts a new draft video record.
func (s *VideoDraftService) CreateDraft(ctx context.Context, v *video.Video) error {
	return s.db.WithContext(ctx).Create(v).Error
}

// GetDraftByID returns a draft by its ID (without ownership check).
func (s *VideoDraftService) GetDraftByID(ctx context.Context, id uint64) (*video.Video, error) {
	var v video.Video
	if err := s.db.WithContext(ctx).First(&v, id).Error; err != nil {
		return nil, err
	}
	return &v, nil
}

// GetOwnedDraft returns a draft by ID only if it belongs to uid and has draft status.
func (s *VideoDraftService) GetOwnedDraft(ctx context.Context, id, uid uint64) (*video.Video, error) {
	var v video.Video
	if err := s.db.WithContext(ctx).Where("id = ? AND user_id = ?", id, uid).First(&v).Error; err != nil {
		return nil, err
	}
	return &v, nil
}

// GetOwnedDraftByStatus returns a draft by ID, user ID, and status.
func (s *VideoDraftService) GetOwnedDraftByStatus(ctx context.Context, id, uid uint64, status string) (*video.Video, error) {
	var v video.Video
	if err := s.db.WithContext(ctx).Where("id = ? AND user_id = ? AND status = ?", id, uid, status).First(&v).Error; err != nil {
		return nil, err
	}
	return &v, nil
}

// UpdateDraft updates the given draft record with the specified fields.
func (s *VideoDraftService) UpdateDraft(ctx context.Context, v *video.Video, updates map[string]interface{}) error {
	return s.db.WithContext(ctx).Model(v).Updates(updates).Error
}

// UpdateDraftField updates a single field of a draft record.
func (s *VideoDraftService) UpdateDraftField(ctx context.Context, v *video.Video, field string, value interface{}) error {
	return s.db.WithContext(ctx).Model(v).Update(field, value).Error
}

// DeleteDraft deletes a draft video by ID.
func (s *VideoDraftService) DeleteDraft(ctx context.Context, id uint64) error {
	return s.db.WithContext(ctx).Where("id = ?", id).Delete(&video.Video{}).Error
}

// CountUserDrafts counts draft videos for a user.
func (s *VideoDraftService) CountUserDrafts(ctx context.Context, uid uint64) (int64, error) {
	var count int64
	err := s.db.WithContext(ctx).Model(&video.Video{}).Where("user_id = ? AND status = 'draft'", uid).Count(&count).Error
	return count, err
}

// ListUserDrafts lists draft videos for a user with pagination.
func (s *VideoDraftService) ListUserDrafts(ctx context.Context, uid uint64, page, pageSize int) ([]video.Video, int64, error) {
	q := s.db.WithContext(ctx).Model(&video.Video{}).Where("user_id = ? AND status = 'draft'", uid)
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

// RefetchDraft re-fetches a draft from the database by ID.
func (s *VideoDraftService) RefetchDraft(ctx context.Context, id uint64) (*video.Video, error) {
	return s.GetDraftByID(ctx, id)
}
