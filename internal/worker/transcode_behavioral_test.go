package worker

import (
	"cakecake/internal/config"
	"cakecake/internal/data"
	"cakecake/internal/model/user"
	"cakecake/internal/model/video"
	"cakecake/internal/queue"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/glebarez/sqlite"
)

// writingFFmpeg writes the requested output files so the pipeline can upload
// them, and reports success for every operation.
type writingFFmpeg struct{}

func (writingFFmpeg) TranscodeToH264MP4(_ string, outMP4 string) (string, error) {
	if err := os.WriteFile(outMP4, []byte("fake mp4"), 0o644); err != nil {
		return "", err
	}
	return "", nil
}

func (writingFFmpeg) ScreenshotJPEG(_ string, outJPEG string, _ float64) (string, error) {
	if err := os.WriteFile(outJPEG, []byte("fake jpg"), 0o644); err != nil {
		return "", err
	}
	return "", nil
}

func (writingFFmpeg) IsPermanentTranscodeFailure(string) bool { return false }

type recordingStore struct {
	uploads []string
}

func (s *recordingStore) UploadFile(objectKey, localPath string) error {
	s.uploads = append(s.uploads, objectKey)
	return nil
}

func newBehavioralWorkerDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, data.AutoMigrateAll(db, zap.NewNop()))
	return db
}

func deliveryForVideoID(videoID uint64, rawPath, coverPath string, ack *fakeAck) amqp.Delivery {
	body, _ := json.Marshal(TranscodeJob{VideoID: videoID, RawPath: rawPath, CoverPath: coverPath})
	return amqp.Delivery{Body: body, Acknowledger: ack}
}

// TestBehavioral_TranscodeSuccess verifies the full happy path with a real DB:
// ffmpeg output -> OSS uploads -> video becomes published with URLs -> temp
// files cleaned -> message acked.
func TestBehavioral_TranscodeSuccess(t *testing.T) {
	db := newBehavioralWorkerDB(t)
	require.NoError(t, db.Create(&user.User{ID: 1, Username: "uploader", PasswordHash: "x"}).Error)
	require.NoError(t, db.Create(&video.Video{ID: 10, UserID: 1, Title: "v", Status: video.StatusProcessing}).Error)

	tmp := t.TempDir()
	rawPath := filepath.Join(tmp, "raw.mp4")
	coverPath := filepath.Join(tmp, "cover.png")
	require.NoError(t, os.WriteFile(rawPath, []byte("raw"), 0o644))
	require.NoError(t, os.WriteFile(coverPath, []byte("cover"), 0o644))

	ack := &fakeAck{}
	store := &recordingStore{}
	cfg := &config.C{
		TempUploadDir:       tmp,
		OSSPublicURLPrefix:  "https://cdn.example.com",
		VideoReviewRequired: false,
	}
	handleDeliveryWith(context.Background(), cfg, db, nil, store, nil,
		deliveryForVideoID(10, rawPath, coverPath, ack), writingFFmpeg{}, zap.NewNop())

	require.Equal(t, 1, ack.acked, "successful job must be acked")
	require.Equal(t, []string{"videos/10.mp4", "covers/10.png"}, store.uploads)

	var v video.Video
	require.NoError(t, db.First(&v, 10).Error)
	require.Equal(t, video.StatusPublished, v.Status)
	require.Equal(t, "https://cdn.example.com/videos/10.mp4", v.VideoURL)
	require.Equal(t, "https://cdn.example.com/covers/10.png", v.CoverURL)

	// All staging files must be cleaned up.
	for _, p := range []string{
		rawPath, coverPath,
		filepath.Join(tmp, "10_out.mp4"),
		filepath.Join(tmp, "10_cover.jpg"),
	} {
		_, err := os.Stat(p)
		require.ErrorIs(t, err, os.ErrNotExist, "staging file %s should be removed", p)
	}
}

// TestBehavioral_TranscodePermanentFailure verifies the failure path: a
// permanent ffmpeg error marks the video failed, uploads nothing, cleans the
// raw file, and acks (the message is consumed, not requeued).
func TestBehavioral_TranscodePermanentFailure(t *testing.T) {
	db := newBehavioralWorkerDB(t)
	require.NoError(t, db.Create(&video.Video{ID: 11, UserID: 1, Title: "v", Status: video.StatusProcessing}).Error)

	tmp := t.TempDir()
	rawPath := filepath.Join(tmp, "bad.mp4")
	require.NoError(t, os.WriteFile(rawPath, []byte("raw"), 0o644))

	ff := &fakeFFmpeg{transcodeErr: os.ErrInvalid, transcodeStderr: "corrupt", permStderr: "corrupt"}
	ack := &fakeAck{}
	store := &recordingStore{}
	cfg := &config.C{TempUploadDir: tmp}
	handleDeliveryWith(context.Background(), cfg, db, nil, store, nil,
		deliveryForVideoID(11, rawPath, "", ack), ff, zap.NewNop())

	require.Equal(t, 1, ack.acked)
	require.Empty(t, store.uploads)
	var v video.Video
	require.NoError(t, db.First(&v, 11).Error)
	require.Equal(t, video.StatusFailed, v.Status)
	require.NotEmpty(t, v.FailReason)
	_, err := os.Stat(rawPath)
	require.ErrorIs(t, err, os.ErrNotExist, "raw file should be removed after terminal failure")
}

// TestBehavioral_TranscodeExhaustedRetriesKeepsRawSource verifies that going
// to the dead letter preserves the original media: the admin requeue re-runs
// ffmpeg from RawPath/CoverPath, so they must survive the terminal path.
func TestBehavioral_TranscodeExhaustedRetriesKeepsRawSource(t *testing.T) {
	db := newBehavioralWorkerDB(t)
	require.NoError(t, db.Create(&video.Video{ID: 21, UserID: 1, Title: "v", Status: video.StatusProcessing}).Error)

	tmp := t.TempDir()
	rawPath := filepath.Join(tmp, "raw.mp4")
	coverPath := filepath.Join(tmp, "cover.png")
	require.NoError(t, os.WriteFile(rawPath, []byte("raw"), 0o644))
	require.NoError(t, os.WriteFile(coverPath, []byte("cover"), 0o644))

	body, _ := json.Marshal(TranscodeJob{VideoID: 21, RawPath: rawPath, CoverPath: coverPath, RetryCount: 3})
	ack := &fakeAck{}
	store := &recordingStore{}
	pub := &fakePublisher{}
	ff := &fakeFFmpeg{transcodeErr: errors.New("retryable"), transcodeStderr: "boom", permStderr: "other"}
	handleDeliveryWith(context.Background(), &config.C{TempUploadDir: tmp}, db, pub, store, nil,
		amqp.Delivery{Body: body, Acknowledger: ack}, ff, zap.NewNop())

	require.Equal(t, 1, ack.acked)
	require.Empty(t, store.uploads)
	require.Equal(t, []string{queue.TranscodeDeadQueue}, pub.keys, "exhausted job must be dead-lettered")
	var v video.Video
	require.NoError(t, db.First(&v, 21).Error)
	require.Equal(t, video.StatusFailed, v.Status)
	var rec video.TranscodeDeadLetter
	require.NoError(t, db.Where("video_id = ?", 21).First(&rec).Error)
	require.Equal(t, 3, rec.RetryCount)
	_, err := os.Stat(rawPath)
	require.NoError(t, err, "raw source must be kept for admin requeue")
	_, err = os.Stat(coverPath)
	require.NoError(t, err, "cover source must be kept for admin requeue")
}

// TestBehavioral_TranscodeRepublishFailureKeepsRawSource covers the second
// dead-letter path: republishing a retryable job fails, so the job is
// dead-lettered; the original media must still be kept for compensation.
func TestBehavioral_TranscodeRepublishFailureKeepsRawSource(t *testing.T) {
	db := newBehavioralWorkerDB(t)
	require.NoError(t, db.Create(&video.Video{ID: 22, UserID: 1, Title: "v", Status: video.StatusProcessing}).Error)

	tmp := t.TempDir()
	rawPath := filepath.Join(tmp, "raw.mp4")
	require.NoError(t, os.WriteFile(rawPath, []byte("raw"), 0o644))

	oldDelay := transcodeRetryBaseDelay
	transcodeRetryBaseDelay = time.Millisecond
	defer func() { transcodeRetryBaseDelay = oldDelay }()

	body, _ := json.Marshal(TranscodeJob{VideoID: 22, RawPath: rawPath, RetryCount: 2})
	ack := &fakeAck{}
	store := &recordingStore{}
	pub := &fakePublisher{err: errors.New("broker down")}
	ff := &fakeFFmpeg{transcodeErr: errors.New("retryable"), transcodeStderr: "boom", permStderr: "other"}
	handleDeliveryWith(context.Background(), &config.C{TempUploadDir: tmp}, db, pub, store, nil,
		amqp.Delivery{Body: body, Acknowledger: ack}, ff, zap.NewNop())

	require.Equal(t, 1, ack.acked)
	var v video.Video
	require.NoError(t, db.First(&v, 22).Error)
	require.Equal(t, video.StatusFailed, v.Status)
	var rec video.TranscodeDeadLetter
	require.NoError(t, db.Where("video_id = ?", 22).First(&rec).Error)
	require.Equal(t, 3, rec.RetryCount, "republish failure stores the incremented retry count")
	_, err := os.Stat(rawPath)
	require.NoError(t, err, "raw source must be kept after republish failure")
}

type spyFFmpeg struct {
	called bool
}

func (s *spyFFmpeg) TranscodeToH264MP4(_, _ string) (string, error) {
	s.called = true
	return "", errors.New("should not run on redelivery")
}

func (s *spyFFmpeg) ScreenshotJPEG(_, _ string, _ float64) (string, error) {
	s.called = true
	return "", errors.New("should not run on redelivery")
}

func (s *spyFFmpeg) IsPermanentTranscodeFailure(string) bool { return false }

// TestBehavioral_TranscodeRedelivery_SkipsTerminalState verifies the
// at-least-once idempotency guard: a redelivered job for a video that already
// reached a terminal transcode state is acked and skipped without touching
// ffmpeg or OSS.
func TestBehavioral_TranscodeRedelivery_SkipsTerminalState(t *testing.T) {
	cases := []struct {
		name   string
		status string
	}{
		{"published", video.StatusPublished},
		{"pending_review", video.StatusPendingReview},
		{"failed", video.StatusFailed},
		{"rejected", video.StatusRejected},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db := newBehavioralWorkerDB(t)
			require.NoError(t, db.Create(&user.User{ID: 1, Username: "u", PasswordHash: "x"}).Error)
			require.NoError(t, db.Create(&video.Video{
				ID: 50, UserID: 1, Title: "v", Status: tc.status,
				VideoURL: "https://cdn.example.com/50.mp4",
			}).Error)

			tmp := t.TempDir()
			raw := filepath.Join(tmp, "raw.mp4")
			cover := filepath.Join(tmp, "cover.png")
			require.NoError(t, os.WriteFile(raw, []byte("raw"), 0o644))
			require.NoError(t, os.WriteFile(cover, []byte("cover"), 0o644))

			ack := &fakeAck{}
			store := &recordingStore{}
			ff := &spyFFmpeg{}
			handleDeliveryWith(context.Background(), &config.C{TempUploadDir: tmp}, db, nil, store, nil,
				deliveryForVideoID(50, raw, cover, ack), ff, zap.NewNop())

			require.Equal(t, 1, ack.acked, "terminal redelivery must be acked")
			require.False(t, ff.called, "terminal redelivery must not run ffmpeg")
			require.Empty(t, store.uploads, "terminal redelivery must not upload OSS")
			var v video.Video
			require.NoError(t, db.First(&v, 50).Error)
			require.Equal(t, tc.status, v.Status)
			_, err := os.Stat(raw)
			require.ErrorIs(t, err, os.ErrNotExist, "staging raw should be cleaned on skip")
			_, err = os.Stat(cover)
			require.ErrorIs(t, err, os.ErrNotExist, "staging cover should be cleaned on skip")
		})
	}
}

// TestBehavioral_TranscodeRedelivery_MissingVideoAcks verifies that a
// redelivered job whose video row no longer exists is acked and dropped
// instead of failing the queue or running ffmpeg.
func TestBehavioral_TranscodeRedelivery_MissingVideoAcks(t *testing.T) {
	db := newBehavioralWorkerDB(t)
	tmp := t.TempDir()
	raw := filepath.Join(tmp, "raw.mp4")
	require.NoError(t, os.WriteFile(raw, []byte("raw"), 0o644))

	ack := &fakeAck{}
	ff := &spyFFmpeg{}
	store := &recordingStore{}
	handleDeliveryWith(context.Background(), &config.C{TempUploadDir: tmp}, db, nil, store, nil,
		deliveryForVideoID(999, raw, "", ack), ff, zap.NewNop())

	require.Equal(t, 1, ack.acked)
	require.False(t, ff.called)
	require.Empty(t, store.uploads)
	_, err := os.Stat(raw)
	require.ErrorIs(t, err, os.ErrNotExist, "staging raw should be cleaned on missing video")
}
