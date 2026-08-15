package worker

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"cakecake/internal/config"
	"cakecake/internal/model/user"
	"cakecake/internal/model/video"
	"cakecake/internal/queue"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func deadLetterPayloadFor(t *testing.T, job queue.TranscodeJob) string {
	t.Helper()
	b, err := json.Marshal(map[string]interface{}{"job": job, "reason": "x"})
	require.NoError(t, err)
	return string(b)
}

func countOutbox(t *testing.T, db *gorm.DB) int64 {
	t.Helper()
	var n int64
	require.NoError(t, db.Model(&video.TranscodeOutbox{}).Count(&n).Error)
	return n
}

func TestAutoRetryDeadLettersOnce_RequeuesTransient(t *testing.T) {
	db := newBehavioralWorkerDB(t)
	require.NoError(t, db.Create(&user.User{ID: 1, Username: "u", PasswordHash: "x"}).Error)
	require.NoError(t, db.Create(&video.Video{ID: 80, UserID: 1, Title: "v", Status: video.StatusFailed, FailReason: "oss down"}).Error)
	require.NoError(t, db.Create(&video.TranscodeDeadLetter{
		VideoID:     80,
		Reason:      "oss upload cover: open data/tmp/80_cover.jpg: no such file",
		RetryCount:  3,
		PayloadJSON: deadLetterPayloadFor(t, queue.TranscodeJob{VideoID: 80, RawKey: "raws/x/source.mp4", JobID: "job-a"}),
	}).Error)

	n, err := autoRetryDeadLettersOnce(context.Background(), db, zap.NewNop())
	require.NoError(t, err)
	require.Equal(t, 1, n)

	var ob video.TranscodeOutbox
	require.NoError(t, db.Where("video_id = ?", 80).First(&ob).Error)
	require.Equal(t, video.OutboxStatusPending, ob.Status)
	var job queue.TranscodeJob
	require.NoError(t, json.Unmarshal([]byte(ob.Payload), &job))
	require.Equal(t, uint64(80), job.VideoID)
	require.Zero(t, job.RetryCount, "auto retry resets retry_count")
	require.NotEmpty(t, job.JobID)
	require.NotEqual(t, "job-a", job.JobID, "auto retry must mint a fresh job_id")

	var row video.TranscodeDeadLetter
	require.NoError(t, db.First(&row).Error)
	require.Equal(t, 1, row.AutoRetryCount)
	require.NotNil(t, row.LastAutoRetryAt)
	require.Nil(t, row.ProcessedAt, "a scheduled replay leaves the row pending until its outcome is known")
	var v video.Video
	require.NoError(t, db.First(&v, 80).Error)
	require.Equal(t, video.StatusProcessing, v.Status)
	var ev video.TranscodeEvent
	require.NoError(t, db.Where("video_id = ?", 80).First(&ev).Error)
	require.Equal(t, video.StatusFailed, ev.FromStatus)
	require.Equal(t, video.StatusProcessing, ev.ToStatus)
}

func TestAutoRetryDeadLettersOnce_SkipsPermanent(t *testing.T) {
	db := newBehavioralWorkerDB(t)
	require.NoError(t, db.Create(&video.TranscodeDeadLetter{
		VideoID:     81,
		Reason:      "Invalid data found when processing input",
		RetryCount:  3,
		PayloadJSON: deadLetterPayloadFor(t, queue.TranscodeJob{VideoID: 81, JobID: "job-b"}),
	}).Error)

	n, err := autoRetryDeadLettersOnce(context.Background(), db, zap.NewNop())
	require.NoError(t, err)
	require.Zero(t, n)
	require.Zero(t, countOutbox(t, db))
	var row video.TranscodeDeadLetter
	require.NoError(t, db.First(&row).Error)
	require.NotNil(t, row.ProcessedAt, "permanent reasons must be resolved so retention can archive the row")
	require.Zero(t, row.AutoRetryCount)
}

func TestAutoRetryDeadLettersOnce_RespectsMaxAndBackoff(t *testing.T) {
	db := newBehavioralWorkerDB(t)
	now := time.Now()
	recent := now.Add(-10 * time.Second)
	require.NoError(t, db.Create(&video.TranscodeDeadLetter{
		VideoID:        82,
		Reason:         "oss upload failed",
		RetryCount:     3,
		AutoRetryCount: deadLetterAutoRetryMax,
		PayloadJSON:    deadLetterPayloadFor(t, queue.TranscodeJob{VideoID: 82, JobID: "job-c"}),
	}).Error)
	require.NoError(t, db.Create(&video.TranscodeDeadLetter{
		VideoID:         83,
		Reason:          "oss upload failed",
		RetryCount:      3,
		AutoRetryCount:  1,
		LastAutoRetryAt: &recent,
		PayloadJSON:     deadLetterPayloadFor(t, queue.TranscodeJob{VideoID: 83, JobID: "job-d"}),
	}).Error)

	n, err := autoRetryDeadLettersOnce(context.Background(), db, zap.NewNop())
	require.NoError(t, err)
	require.Zero(t, n, "maxed-out and backoff-pending rows must not be requeued")
	require.Zero(t, countOutbox(t, db))
	var maxed video.TranscodeDeadLetter
	require.NoError(t, db.Where("video_id = ?", 82).First(&maxed).Error)
	require.NotNil(t, maxed.ProcessedAt, "a maxed-out row is terminal and must be resolved")
	var backoff video.TranscodeDeadLetter
	require.NoError(t, db.Where("video_id = ?", 83).First(&backoff).Error)
	require.Nil(t, backoff.ProcessedAt, "a row inside its backoff window stays pending for the next scan")
}

func TestAutoRetryDeadLettersOnce_SkipsTerminalOrMissingVideo(t *testing.T) {
	db := newBehavioralWorkerDB(t)
	require.NoError(t, db.Create(&video.Video{ID: 84, UserID: 1, Title: "v", Status: video.StatusPublished}).Error)
	require.NoError(t, db.Create(&video.TranscodeDeadLetter{
		VideoID:     84,
		Reason:      "oss upload failed",
		RetryCount:  3,
		PayloadJSON: deadLetterPayloadFor(t, queue.TranscodeJob{VideoID: 84, JobID: "job-terminal"}),
	}).Error)
	require.NoError(t, db.Create(&video.TranscodeDeadLetter{
		VideoID:     85,
		Reason:      "oss upload failed",
		RetryCount:  3,
		PayloadJSON: deadLetterPayloadFor(t, queue.TranscodeJob{VideoID: 85, JobID: "job-missing"}),
	}).Error)

	n, err := autoRetryDeadLettersOnce(context.Background(), db, zap.NewNop())
	require.NoError(t, err)
	require.Zero(t, n, "terminal/missing videos must not be replayed")
	require.Zero(t, countOutbox(t, db))
	var row video.TranscodeDeadLetter
	require.NoError(t, db.Where("video_id = ?", 84).First(&row).Error)
	require.Zero(t, row.AutoRetryCount, "skipped rows must not consume retry budget")
	require.NotNil(t, row.ProcessedAt, "a terminal video resolves the row so it can archive")
	var missing video.TranscodeDeadLetter
	require.NoError(t, db.Where("video_id = ?", 85).First(&missing).Error)
	require.NotNil(t, missing.ProcessedAt, "a missing video resolves the row so it can archive")
}

func TestAutoRetryDeadLettersOnce_SkipsProcessingVideo(t *testing.T) {
	db := newBehavioralWorkerDB(t)
	// processing = a replay or normal job is already in flight; the minute-level
	// window before the dead letter is marked processed must not double-schedule.
	require.NoError(t, db.Create(&video.Video{ID: 86, UserID: 1, Title: "v", Status: video.StatusProcessing}).Error)
	require.NoError(t, db.Create(&video.TranscodeDeadLetter{
		VideoID:     86,
		Reason:      "oss upload failed",
		RetryCount:  3,
		PayloadJSON: deadLetterPayloadFor(t, queue.TranscodeJob{VideoID: 86, JobID: "job-inflight"}),
	}).Error)

	n, err := autoRetryDeadLettersOnce(context.Background(), db, zap.NewNop())
	require.NoError(t, err)
	require.Zero(t, n)
	require.Zero(t, countOutbox(t, db))
	var row video.TranscodeDeadLetter
	require.NoError(t, db.Where("video_id = ?", 86).First(&row).Error)
	require.Nil(t, row.ProcessedAt, "an in-flight replay leaves the row pending; its outcome resolves it")
	require.Zero(t, row.AutoRetryCount)
}

func TestAutoRetryDeadLettersOnce_LifetimeBudgetStopsNewRows(t *testing.T) {
	db := newBehavioralWorkerDB(t)
	require.NoError(t, db.Create(&video.Video{ID: 87, UserID: 1, Title: "v", Status: video.StatusFailed}).Error)
	// A failed replay produces a NEW dead-letter row with count 0. The old row
	// has already consumed the whole lifetime budget, so the new row must not
	// start another unbounded retry round.
	require.NoError(t, db.Create(&video.TranscodeDeadLetter{
		VideoID:        87,
		Reason:         "oss upload failed",
		RetryCount:     3,
		AutoRetryCount: deadLetterAutoRetryTotalMax,
		PayloadJSON:    deadLetterPayloadFor(t, queue.TranscodeJob{VideoID: 87, JobID: "job-old"}),
	}).Error)
	require.NoError(t, db.Create(&video.TranscodeDeadLetter{
		VideoID:     87,
		Reason:      "oss upload failed",
		RetryCount:  3,
		PayloadJSON: deadLetterPayloadFor(t, queue.TranscodeJob{VideoID: 87, JobID: "job-new"}),
	}).Error)

	n, err := autoRetryDeadLettersOnce(context.Background(), db, zap.NewNop())
	require.NoError(t, err)
	require.Zero(t, n, "lifetime budget exhausted: fresh rows must not restart retries")
	require.Zero(t, countOutbox(t, db))
	var rows []video.TranscodeDeadLetter
	require.NoError(t, db.Where("video_id = ?", 87).Order("id ASC").Find(&rows).Error)
	require.Len(t, rows, 2)
	for _, r := range rows {
		require.NotNil(t, r.ProcessedAt, "rows whose lifetime budget is exhausted must be resolved")
	}
}

func TestAutoRetryDeadLettersOnce_LifetimeBudgetStillAllowsWithinLimit(t *testing.T) {
	db := newBehavioralWorkerDB(t)
	require.NoError(t, db.Create(&video.Video{ID: 88, UserID: 1, Title: "v", Status: video.StatusFailed}).Error)
	require.NoError(t, db.Create(&video.TranscodeDeadLetter{
		VideoID:        88,
		Reason:         "oss upload failed",
		RetryCount:     3,
		AutoRetryCount: 2,
		PayloadJSON:    deadLetterPayloadFor(t, queue.TranscodeJob{VideoID: 88, JobID: "job-old"}),
	}).Error)
	require.NoError(t, db.Create(&video.TranscodeDeadLetter{
		VideoID:     88,
		Reason:      "oss upload failed",
		RetryCount:  3,
		PayloadJSON: deadLetterPayloadFor(t, queue.TranscodeJob{VideoID: 88, JobID: "job-new"}),
	}).Error)

	n, err := autoRetryDeadLettersOnce(context.Background(), db, zap.NewNop())
	require.NoError(t, err)
	require.Equal(t, 1, n, "budget not exhausted yet: the fresh row may be retried")
}

func TestAutoRetryableReason_ExcludesConfigErrors(t *testing.T) {
	require.False(t, autoRetryableReason("OSS 未配置"))
	require.False(t, autoRetryableReason("transcode queue not configured"))
	require.True(t, autoRetryableReason("oss upload cover: open ...: no such file"))
	require.True(t, autoRetryableReason("下载原始视频失败: connection refused"))
}

// TestAutoRetryReplay_NotSkippedByWorkerDedup is the end-to-end regression for
// the dedup collision: the original attempt's (job_id, 0) row must not swallow
// an auto-replayed job. The replay mints a fresh job_id, the outbox relay
// publishes it, and the worker processes it to completion.
func TestAutoRetryReplay_NotSkippedByWorkerDedup(t *testing.T) {
	db := newBehavioralWorkerDB(t)
	require.NoError(t, db.Create(&user.User{ID: 1, Username: "u", PasswordHash: "x"}).Error)
	require.NoError(t, db.Create(&video.Video{ID: 90, UserID: 1, Title: "v", Status: video.StatusFailed}).Error)
	require.NoError(t, db.Create(&video.TranscodeJobDedup{JobID: "job-old", RetryCount: 0, VideoID: 90}).Error)

	tmp := t.TempDir()
	raw := filepath.Join(tmp, "raw.mp4")
	require.NoError(t, os.WriteFile(raw, []byte("raw"), 0o644))
	require.NoError(t, db.Create(&video.TranscodeDeadLetter{
		VideoID:     90,
		Reason:      "oss upload failed",
		RetryCount:  3,
		PayloadJSON: deadLetterPayloadFor(t, queue.TranscodeJob{VideoID: 90, RawPath: raw, JobID: "job-old"}),
	}).Error)

	n, err := autoRetryDeadLettersOnce(context.Background(), db, zap.NewNop())
	require.NoError(t, err)
	require.Equal(t, 1, n)
	var ob video.TranscodeOutbox
	require.NoError(t, db.Where("video_id = ?", 90).First(&ob).Error)
	var replayed queue.TranscodeJob
	require.NoError(t, json.Unmarshal([]byte(ob.Payload), &replayed))
	require.NotEqual(t, "job-old", replayed.JobID, "replay must carry a fresh job_id")

	pub := &outboxPublisher{}
	sent, err := relayTranscodeOutboxOnce(context.Background(), db, pub, zap.NewNop())
	require.NoError(t, err)
	require.Equal(t, 1, sent)
	require.Len(t, pub.bodies, 1)

	// Feed the replayed message through the worker: it must transcode, not skip.
	ack := &fakeAck{}
	ff := &writingFFmpeg{}
	store := &recordingStore{}
	cfg := &config.C{
		TempUploadDir:       tmp,
		OSSPublicURLPrefix:  "https://cdn.example.com",
		VideoReviewRequired: false,
	}
	handleDeliveryWith(context.Background(), cfg, db, nil, store, nil,
		amqp.Delivery{Body: pub.bodies[0], Acknowledger: ack}, ff, zap.NewNop())

	require.Equal(t, 1, ack.acked)
	var v video.Video
	require.NoError(t, db.First(&v, 90).Error)
	require.Equal(t, video.StatusPublished, v.Status)
	var d video.TranscodeJobDedup
	require.NoError(t, db.Where("job_id = ?", replayed.JobID).First(&d).Error, "fresh job_id must be deduped as processed")
}
