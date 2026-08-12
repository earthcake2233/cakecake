package video

import (
	"cakecake/internal/model/video"
	"cakecake/internal/queue"
	"cakecake/internal/service/servicetest"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type fakeTranscodePublisher struct {
	bodies [][]byte
	err    error
}

func (f *fakeTranscodePublisher) PublishTranscode(_ context.Context, body []byte) error {
	if f.err != nil {
		return f.err
	}
	f.bodies = append(f.bodies, body)
	return nil
}

func deadLetterPayload(t *testing.T, job queue.TranscodeJob) string {
	t.Helper()
	b, err := json.Marshal(map[string]interface{}{"job": job, "reason": "x"})
	require.NoError(t, err)
	return string(b)
}

func tempMedia(t *testing.T, name string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), name)
	require.NoError(t, os.WriteFile(p, []byte("media"), 0o644))
	return p
}

func TestRequeueTranscodeDeadLetter_ResetsAndPublishes(t *testing.T) {
	db := servicetest.NewDB(t)
	_, rdb := servicetest.NewRedis(t)
	mq := &fakeTranscodePublisher{}
	svc := NewVideoService(db, rdb, zap.NewNop(), nil, mq)
	ctx := context.Background()

	servicetest.SeedUser(t, db, 1, "u")
	require.NoError(t, db.Create(&video.Video{ID: 10, UserID: 1, Title: "v", Status: video.StatusFailed, FailReason: "oss down"}).Error)
	dl := video.TranscodeDeadLetter{
		VideoID: 10, RetryCount: 3, Reason: "oss down",
		PayloadJSON: deadLetterPayload(t, queue.TranscodeJob{VideoID: 10, RawPath: tempMedia(t, "raw.mp4"), RetryCount: 3}),
	}
	require.NoError(t, db.Create(&dl).Error)

	require.NoError(t, svc.RequeueTranscodeDeadLetter(ctx, dl.ID))

	require.Len(t, mq.bodies, 1)
	var job queue.TranscodeJob
	require.NoError(t, json.Unmarshal(mq.bodies[0], &job))
	require.Equal(t, uint64(10), job.VideoID)
	require.Zero(t, job.RetryCount)

	var v video.Video
	require.NoError(t, db.First(&v, 10).Error)
	require.Equal(t, video.StatusProcessing, v.Status)
	require.Empty(t, v.FailReason)

	var row video.TranscodeDeadLetter
	require.NoError(t, db.First(&row, dl.ID).Error)
	require.NotNil(t, row.RequeuedAt)
	require.Equal(t, 1, row.RequeuedCount)
}

func TestRequeueTranscodeDeadLetter_MissingSourceRejects(t *testing.T) {
	db := servicetest.NewDB(t)
	_, rdb := servicetest.NewRedis(t)
	mq := &fakeTranscodePublisher{}
	svc := NewVideoService(db, rdb, zap.NewNop(), nil, mq)
	ctx := context.Background()

	servicetest.SeedUser(t, db, 1, "u")
	require.NoError(t, db.Create(&video.Video{ID: 10, UserID: 1, Title: "v", Status: video.StatusFailed}).Error)
	dl := video.TranscodeDeadLetter{
		VideoID: 10, RetryCount: 3, Reason: "x",
		PayloadJSON: deadLetterPayload(t, queue.TranscodeJob{VideoID: 10, RawPath: filepath.Join(t.TempDir(), "gone.mp4"), RetryCount: 3}),
	}
	require.NoError(t, db.Create(&dl).Error)

	err := svc.RequeueTranscodeDeadLetter(ctx, dl.ID)
	require.ErrorIs(t, err, ErrRequeueSourceMissing)
	require.Empty(t, mq.bodies, "missing source must not publish")
	var v video.Video
	require.NoError(t, db.First(&v, 10).Error)
	require.Equal(t, video.StatusFailed, v.Status)
	var row video.TranscodeDeadLetter
	require.NoError(t, db.First(&row, dl.ID).Error)
	require.Nil(t, row.RequeuedAt)
	require.Zero(t, row.RequeuedCount)
}

func TestRequeueTranscodeDeadLetter_MissingCoverRejects(t *testing.T) {
	db := servicetest.NewDB(t)
	_, rdb := servicetest.NewRedis(t)
	mq := &fakeTranscodePublisher{}
	svc := NewVideoService(db, rdb, zap.NewNop(), nil, mq)
	ctx := context.Background()

	servicetest.SeedUser(t, db, 1, "u")
	require.NoError(t, db.Create(&video.Video{ID: 10, UserID: 1, Title: "v", Status: video.StatusFailed}).Error)
	dl := video.TranscodeDeadLetter{
		VideoID: 10, RetryCount: 3, Reason: "x",
		PayloadJSON: deadLetterPayload(t, queue.TranscodeJob{
			VideoID: 10, RawPath: tempMedia(t, "raw.mp4"),
			CoverPath: filepath.Join(t.TempDir(), "gone.jpg"), RetryCount: 3,
		}),
	}
	require.NoError(t, db.Create(&dl).Error)

	err := svc.RequeueTranscodeDeadLetter(ctx, dl.ID)
	require.ErrorIs(t, err, ErrRequeueSourceMissing)
	require.Empty(t, mq.bodies)
}

func TestRequeueTranscodeDeadLetter_PublishFailureRevertsVideoAndAuditRow(t *testing.T) {
	db := servicetest.NewDB(t)
	_, rdb := servicetest.NewRedis(t)
	mq := &fakeTranscodePublisher{err: errors.New("broker down")}
	svc := NewVideoService(db, rdb, zap.NewNop(), nil, mq)
	ctx := context.Background()

	servicetest.SeedUser(t, db, 1, "u")
	require.NoError(t, db.Create(&video.Video{ID: 11, UserID: 1, Title: "v", Status: video.StatusFailed, FailReason: "old"}).Error)
	dl := video.TranscodeDeadLetter{
		VideoID: 11, RetryCount: 3, Reason: "x",
		PayloadJSON: deadLetterPayload(t, queue.TranscodeJob{VideoID: 11, RawPath: tempMedia(t, "raw.mp4"), RetryCount: 3}),
	}
	require.NoError(t, db.Create(&dl).Error)

	err := svc.RequeueTranscodeDeadLetter(ctx, dl.ID)
	require.Error(t, err)
	var v video.Video
	require.NoError(t, db.First(&v, 11).Error)
	require.Equal(t, video.StatusFailed, v.Status)
	require.Contains(t, v.FailReason, "requeue publish failed")
	var row video.TranscodeDeadLetter
	require.NoError(t, db.First(&row, dl.ID).Error)
	require.Nil(t, row.RequeuedAt, "failed publish must not leave the audit row marked requeued")
	require.Zero(t, row.RequeuedCount)
}

func TestRequeueTranscodeDeadLetter_PublishFailureRestoresPreviousAuditState(t *testing.T) {
	db := servicetest.NewDB(t)
	_, rdb := servicetest.NewRedis(t)
	mq := &fakeTranscodePublisher{err: errors.New("broker down")}
	svc := NewVideoService(db, rdb, zap.NewNop(), nil, mq)
	ctx := context.Background()

	servicetest.SeedUser(t, db, 1, "u")
	require.NoError(t, db.Create(&video.Video{ID: 12, UserID: 1, Title: "v", Status: video.StatusFailed}).Error)
	prevRequeuedAt := time.Now().Add(-time.Hour)
	prevProcessedAt := time.Now().Add(-2 * time.Hour)
	dl := video.TranscodeDeadLetter{
		VideoID: 12, RetryCount: 3, Reason: "x", RequeuedAt: &prevRequeuedAt, RequeuedCount: 2, ProcessedAt: &prevProcessedAt,
		PayloadJSON: deadLetterPayload(t, queue.TranscodeJob{VideoID: 12, RawPath: tempMedia(t, "raw.mp4"), RetryCount: 3}),
	}
	require.NoError(t, db.Create(&dl).Error)

	err := svc.RequeueTranscodeDeadLetter(ctx, dl.ID)
	require.Error(t, err)
	var row video.TranscodeDeadLetter
	require.NoError(t, db.First(&row, dl.ID).Error)
	require.NotNil(t, row.RequeuedAt)
	require.Equal(t, prevRequeuedAt.Unix(), row.RequeuedAt.Unix(), "requeued_at must be restored to the pre-attempt value")
	require.Equal(t, 2, row.RequeuedCount, "requeued_count must be restored to the pre-attempt value")
	require.NotNil(t, row.ProcessedAt)
	require.Equal(t, prevProcessedAt.Unix(), row.ProcessedAt.Unix(), "processed_at must be restored to the pre-attempt value")
}

func TestRequeueTranscodeDeadLetter_NoMQRollsBackAuditRow(t *testing.T) {
	db := servicetest.NewDB(t)
	_, rdb := servicetest.NewRedis(t)
	svc := NewVideoService(db, rdb, zap.NewNop(), nil, nil)
	ctx := context.Background()

	servicetest.SeedUser(t, db, 1, "u")
	require.NoError(t, db.Create(&video.Video{ID: 13, UserID: 1, Title: "v", Status: video.StatusFailed}).Error)
	dl := video.TranscodeDeadLetter{
		VideoID: 13, RetryCount: 3, Reason: "x",
		PayloadJSON: deadLetterPayload(t, queue.TranscodeJob{VideoID: 13, RawPath: tempMedia(t, "raw.mp4"), RetryCount: 3}),
	}
	require.NoError(t, db.Create(&dl).Error)

	err := svc.RequeueTranscodeDeadLetter(ctx, dl.ID)
	require.Error(t, err)
	var v video.Video
	require.NoError(t, db.First(&v, 13).Error)
	require.Equal(t, video.StatusFailed, v.Status)
	require.Contains(t, v.FailReason, "transcode queue not configured")
	var row video.TranscodeDeadLetter
	require.NoError(t, db.First(&row, dl.ID).Error)
	require.Nil(t, row.RequeuedAt)
	require.Zero(t, row.RequeuedCount)
}

func TestListTranscodeDeadLetters_Filters(t *testing.T) {
	db := servicetest.NewDB(t)
	_, rdb := servicetest.NewRedis(t)
	svc := NewVideoService(db, rdb, zap.NewNop(), nil, nil)
	ctx := context.Background()

	require.NoError(t, db.Create(&video.TranscodeDeadLetter{VideoID: 1, RetryCount: 1, Reason: "pending"}).Error)
	now := time.Now()
	require.NoError(t, db.Create(&video.TranscodeDeadLetter{VideoID: 2, RetryCount: 1, Reason: "requeued", RequeuedAt: &now}).Error)

	rows, total, err := svc.ListTranscodeDeadLetters(ctx, TranscodeDeadLetterFilter{Page: 1, PageSize: 10, Status: "requeued"})
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, rows, 1)
	require.Equal(t, uint64(2), rows[0].VideoID)
}
