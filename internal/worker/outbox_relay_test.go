package worker

import (
	"context"
	"testing"
	"time"

	"cakecake/internal/model/video"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type outboxPublisher struct {
	bodies [][]byte
	err    error
}

func (o *outboxPublisher) PublishTranscode(_ context.Context, body []byte) error {
	if o.err != nil {
		return o.err
	}
	o.bodies = append(o.bodies, body)
	return nil
}

func TestRelayTranscodeOutboxOnce_PublishesAndMarksSent(t *testing.T) {
	db := newBehavioralWorkerDB(t)
	require.NoError(t, db.Create(&video.TranscodeOutbox{
		JobID:   "job-1",
		VideoID: 1,
		Payload: `{"video_id":1,"job_id":"job-1"}`,
		Status:  video.OutboxStatusPending,
	}).Error)
	pub := &outboxPublisher{}

	sent, err := relayTranscodeOutboxOnce(context.Background(), db, pub, zap.NewNop())
	require.NoError(t, err)
	require.Equal(t, 1, sent)
	require.Len(t, pub.bodies, 1)

	var row video.TranscodeOutbox
	require.NoError(t, db.First(&row).Error)
	require.Equal(t, video.OutboxStatusSent, row.Status)
	require.NotNil(t, row.SentAt)
}

func TestRelayTranscodeOutboxOnce_RetriesWithBackoff(t *testing.T) {
	db := newBehavioralWorkerDB(t)
	require.NoError(t, db.Create(&video.TranscodeOutbox{
		JobID:   "job-2",
		VideoID: 2,
		Payload: `{"video_id":2,"job_id":"job-2"}`,
		Status:  video.OutboxStatusPending,
	}).Error)
	pub := &outboxPublisher{err: context.DeadlineExceeded}

	sent, err := relayTranscodeOutboxOnce(context.Background(), db, pub, zap.NewNop())
	require.NoError(t, err)
	require.Zero(t, sent)

	var row video.TranscodeOutbox
	require.NoError(t, db.First(&row).Error)
	require.Equal(t, video.OutboxStatusPending, row.Status)
	require.Equal(t, 1, row.Attempts)
	require.NotNil(t, row.NextRetryAt)
	require.True(t, row.NextRetryAt.After(time.Now()), "failed publish must be retried later")

	// Before next_retry_at, the row is not selected again.
	sent, err = relayTranscodeOutboxOnce(context.Background(), db, pub, zap.NewNop())
	require.NoError(t, err)
	require.Zero(t, sent)
	require.Len(t, pub.bodies, 0, "no publish attempt before backoff elapses")
}

func TestRelayTranscodeOutboxOnce_SkipsAlreadySent(t *testing.T) {
	db := newBehavioralWorkerDB(t)
	now := time.Now()
	require.NoError(t, db.Create(&video.TranscodeOutbox{
		JobID:   "job-3",
		VideoID: 3,
		Payload: `{"video_id":3,"job_id":"job-3"}`,
		Status:  video.OutboxStatusSent,
		SentAt:  &now,
	}).Error)
	pub := &outboxPublisher{}

	sent, err := relayTranscodeOutboxOnce(context.Background(), db, pub, zap.NewNop())
	require.NoError(t, err)
	require.Zero(t, sent)
	require.Empty(t, pub.bodies)
}

func TestRelayTranscodeOutboxOnce_SaturatesBackoffExponent(t *testing.T) {
	db := newBehavioralWorkerDB(t)
	require.NoError(t, db.Create(&video.TranscodeOutbox{
		JobID:    "job-overflow",
		VideoID:  9,
		Payload:  `{"video_id":9,"job_id":"job-overflow"}`,
		Status:   video.OutboxStatusPending,
		Attempts: 62,
	}).Error)
	pub := &outboxPublisher{err: context.DeadlineExceeded}

	sent, err := relayTranscodeOutboxOnce(context.Background(), db, pub, zap.NewNop())
	require.NoError(t, err)
	require.Zero(t, sent)

	var row video.TranscodeOutbox
	require.NoError(t, db.First(&row).Error)
	require.Equal(t, video.OutboxStatusPending, row.Status)
	require.NotNil(t, row.NextRetryAt)
	require.True(t, row.NextRetryAt.After(time.Now()), "backoff must stay positive after many failures")
}
