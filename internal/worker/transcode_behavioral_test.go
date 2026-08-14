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

	"github.com/prometheus/client_golang/prometheus/testutil"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/glebarez/sqlite"
)

// writingFFmpeg writes the requested output files so the pipeline can upload
// them, and reports success for every operation.
type writingFFmpeg struct{}

func (writingFFmpeg) TranscodeToH264MP4(_ context.Context, _ string, outMP4 string) (string, error) {
	if err := os.WriteFile(outMP4, []byte("fake mp4"), 0o644); err != nil {
		return "", err
	}
	return "", nil
}

func (writingFFmpeg) ScreenshotJPEG(_ context.Context, _ string, outJPEG string, _ float64) (string, error) {
	if err := os.WriteFile(outJPEG, []byte("fake jpg"), 0o644); err != nil {
		return "", err
	}
	return "", nil
}

func (writingFFmpeg) IsPermanentTranscodeFailure(string) bool { return false }

func (writingFFmpeg) ProbeDurationSeconds(_ context.Context, _ string) (float64, error) {
	return 60, nil
}

type recordingStore struct {
	uploads []string
	deleted []string
}

func (s *recordingStore) UploadFile(objectKey, localPath string) error {
	s.uploads = append(s.uploads, objectKey)
	return nil
}

func (s *recordingStore) DownloadFile(_ string, _ string) error {
	// Tests seed source files locally before the call; OSS-keyed behavior
	// is covered by fakeObjectStore, so no-op downloads are fine here.
	return nil
}

func (s *recordingStore) DeleteObject(objectKey string) error {
	s.deleted = append(s.deleted, objectKey)
	return nil
}

// fileCheckingStore writes downloaded sources as real files and verifies
// every uploaded local path exists, catching "downloaded then deleted" bugs.
type fileCheckingStore struct {
	uploads []string
	missing []string
}

func (s *fileCheckingStore) UploadFile(objectKey, localPath string) error {
	if _, err := os.Stat(localPath); err != nil {
		s.missing = append(s.missing, objectKey+" -> "+localPath)
		return nil
	}
	s.uploads = append(s.uploads, objectKey)
	return nil
}

func (s *fileCheckingStore) DownloadFile(_ string, localPath string) error {
	return os.WriteFile(localPath, []byte("cover-bytes"), 0o644)
}

func (s *fileCheckingStore) DeleteObject(string) error { return nil }

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
	permBefore := testutil.ToFloat64(transcodeJobsTotal.WithLabelValues("permanent_failure"))
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
	require.Greater(t, testutil.ToFloat64(transcodeJobsTotal.WithLabelValues("permanent_failure")), permBefore)
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

func (s *spyFFmpeg) TranscodeToH264MP4(_ context.Context, _, _ string) (string, error) {
	s.called = true
	return "", errors.New("should not run on redelivery")
}

func (s *spyFFmpeg) ScreenshotJPEG(_ context.Context, _, _ string, _ float64) (string, error) {
	s.called = true
	return "", errors.New("should not run on redelivery")
}

func (s *spyFFmpeg) IsPermanentTranscodeFailure(string) bool { return false }

func (s *spyFFmpeg) ProbeDurationSeconds(_ context.Context, _ string) (float64, error) {
	return 60, nil
}

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

// TestBehavioral_TranscodeSuccessWithOSSKeys verifies the durable-source path:
// the job carries OSS object keys, the worker downloads them to temp files,
// transcodes, uploads the results, and deletes both the temp copies and the
// source objects on success.
func TestBehavioral_TranscodeSuccessWithOSSKeys(t *testing.T) {
	successBefore := testutil.ToFloat64(transcodeJobsTotal.WithLabelValues("success"))
	db := newBehavioralWorkerDB(t)
	require.NoError(t, db.Create(&user.User{ID: 1, Username: "uploader", PasswordHash: "x"}).Error)
	require.NoError(t, db.Create(&video.Video{ID: 30, UserID: 1, Title: "v", Status: video.StatusProcessing}).Error)

	tmp := t.TempDir()
	cfg := &config.C{
		TempUploadDir:       tmp,
		OSSPublicURLPrefix:  "https://cdn.example.com",
		VideoReviewRequired: false,
	}
	store := &fakeObjectStore{}
	ack := &fakeAck{}
	body, _ := json.Marshal(TranscodeJob{VideoID: 30, RawKey: "raws/30/source.mp4", CoverKey: "raws/30/cover.png"})
	handleDeliveryWith(context.Background(), cfg, db, nil, store, nil,
		amqp.Delivery{Body: body, Acknowledger: ack}, writingFFmpeg{}, zap.NewNop())

	require.Equal(t, 1, ack.acked)
	require.Equal(t, []string{"videos/30.mp4", "covers/30.png"}, store.uploads)
	require.ElementsMatch(t, []string{"raws/30/source.mp4", "raws/30/cover.png"}, store.deleted,
		"source objects are released after a successful transcode")

	var v video.Video
	require.NoError(t, db.First(&v, 30).Error)
	require.Equal(t, video.StatusPublished, v.Status)
	for _, p := range []string{
		filepath.Join(tmp, "30_source.mp4"),
		filepath.Join(tmp, "30_cover.png"),
		filepath.Join(tmp, "30_out.mp4"),
		filepath.Join(tmp, "30_cover.jpg"),
	} {
		_, err := os.Stat(p)
		require.ErrorIs(t, err, os.ErrNotExist, "temp file %s should be removed", p)
	}
	require.Greater(t, testutil.ToFloat64(transcodeJobsTotal.WithLabelValues("success")), successBefore)
}

// TestBehavioral_TranscodeUserCoverSurvivesUpload guards the regression where
// coverOut (default cover path) and the downloaded user cover share the same
// file: removing coverOut deleted the downloaded cover before upload.
func TestBehavioral_TranscodeUserCoverSurvivesUpload(t *testing.T) {
	db := newBehavioralWorkerDB(t)
	require.NoError(t, db.Create(&user.User{ID: 1, Username: "uploader", PasswordHash: "x"}).Error)
	require.NoError(t, db.Create(&video.Video{ID: 70, UserID: 1, Title: "v", Status: video.StatusProcessing}).Error)

	tmp := t.TempDir()
	cfg := &config.C{
		TempUploadDir:       tmp,
		OSSPublicURLPrefix:  "https://cdn.example.com",
		VideoReviewRequired: false,
	}
	store := &fileCheckingStore{}
	ack := &fakeAck{}
	body, _ := json.Marshal(TranscodeJob{VideoID: 70, RawKey: "raws/x/source.mp4", CoverKey: "raws/x/cover.jpg"})
	handleDeliveryWith(context.Background(), cfg, db, nil, store, nil,
		amqp.Delivery{Body: body, Acknowledger: ack}, writingFFmpeg{}, zap.NewNop())

	require.Equal(t, 1, ack.acked)
	require.Empty(t, store.missing, "every uploaded file must exist on disk: %v", store.missing)
	require.Contains(t, store.uploads, "covers/70.jpg")
	var v video.Video
	require.NoError(t, db.First(&v, 70).Error)
	require.Equal(t, video.StatusPublished, v.Status)
}

// TestBehavioral_TranscodeDownloadFailureRequeues verifies that a failed
// source download schedules a retry with the OSS key intact: the object is
// the compensation input and must survive, only the local copy is cleaned.
func TestBehavioral_TranscodeDownloadFailureRequeues(t *testing.T) {
	retriesBefore := testutil.ToFloat64(transcodeRetriesScheduledTotal)
	db := newBehavioralWorkerDB(t)
	require.NoError(t, db.Create(&user.User{ID: 1, Username: "uploader", PasswordHash: "x"}).Error)
	require.NoError(t, db.Create(&video.Video{ID: 31, UserID: 1, Title: "v", Status: video.StatusProcessing}).Error)

	tmp := t.TempDir()
	store := &fakeObjectStore{downloadErr: errors.New("oss down")}
	pub := &fakePublisher{}
	ack := &fakeAck{}
	body, _ := json.Marshal(TranscodeJob{VideoID: 31, RawKey: "raws/31/source.mp4"})
	handleDeliveryWith(context.Background(), &config.C{TempUploadDir: tmp}, db, pub, store, nil,
		amqp.Delivery{Body: body, Acknowledger: ack}, writingFFmpeg{}, zap.NewNop())

	require.Equal(t, 1, ack.acked)
	require.Empty(t, store.uploads)
	require.Len(t, pub.published, 1)
	require.Equal(t, queue.RetryQueueForAttempt(1), pub.keys[0])
	var requeued TranscodeJob
	require.NoError(t, json.Unmarshal(pub.published[0].Body, &requeued))
	require.Equal(t, 1, requeued.RetryCount)
	require.Equal(t, "raws/31/source.mp4", requeued.RawKey, "OSS source key must survive retry scheduling")
	require.Empty(t, store.deleted)
	require.Greater(t, testutil.ToFloat64(transcodeRetriesScheduledTotal), retriesBefore)
}

// TestBehavioral_TranscodeDedupSkipsDuplicateJob verifies message-level
// dedup: a redelivered (job_id, retry_count) pair is skipped before ffmpeg
// runs, so an in-flight duplicate cannot double-transcode.
func TestBehavioral_TranscodeDedupSkipsDuplicateJob(t *testing.T) {
	db := newBehavioralWorkerDB(t)
	require.NoError(t, db.Create(&user.User{ID: 1, Username: "u", PasswordHash: "x"}).Error)
	require.NoError(t, db.Create(&video.Video{ID: 60, UserID: 1, Title: "v", Status: video.StatusProcessing}).Error)
	require.NoError(t, db.Create(&video.TranscodeJobDedup{JobID: "job-x", RetryCount: 0, VideoID: 60}).Error)

	tmp := t.TempDir()
	raw := filepath.Join(tmp, "raw.mp4")
	require.NoError(t, os.WriteFile(raw, []byte("raw"), 0o644))
	ack := &fakeAck{}
	body, _ := json.Marshal(TranscodeJob{VideoID: 60, RawPath: raw, JobID: "job-x", RetryCount: 0})
	ff := &spyFFmpeg{}
	store := &recordingStore{}
	handleDeliveryWith(context.Background(), &config.C{TempUploadDir: tmp}, db, nil, store, nil,
		amqp.Delivery{Body: body, Acknowledger: ack}, ff, zap.NewNop())

	require.Equal(t, 1, ack.acked)
	require.False(t, ff.called, "duplicate job must not run ffmpeg")
	require.Empty(t, store.uploads)
	var v video.Video
	require.NoError(t, db.First(&v, 60).Error)
	require.Equal(t, video.StatusProcessing, v.Status)
}

// TestBehavioral_TranscodeDedupAllowsNextRetry verifies the dedup key is
// (job_id, retry_count): the next retry attempt is a new key and still runs.
func TestBehavioral_TranscodeDedupAllowsNextRetry(t *testing.T) {
	db := newBehavioralWorkerDB(t)
	require.NoError(t, db.Create(&user.User{ID: 1, Username: "u", PasswordHash: "x"}).Error)
	require.NoError(t, db.Create(&video.Video{ID: 61, UserID: 1, Title: "v", Status: video.StatusProcessing}).Error)
	require.NoError(t, db.Create(&video.TranscodeJobDedup{JobID: "job-y", RetryCount: 0, VideoID: 61}).Error)

	tmp := t.TempDir()
	raw := filepath.Join(tmp, "raw.mp4")
	require.NoError(t, os.WriteFile(raw, []byte("raw"), 0o644))
	ack := &fakeAck{}
	body, _ := json.Marshal(TranscodeJob{VideoID: 61, RawPath: raw, JobID: "job-y", RetryCount: 1})
	ff := &writingFFmpeg{}
	store := &recordingStore{}
	cfg := &config.C{TempUploadDir: tmp, OSSPublicURLPrefix: "https://cdn.example.com", VideoReviewRequired: false}
	handleDeliveryWith(context.Background(), cfg, db, nil, store, nil,
		amqp.Delivery{Body: body, Acknowledger: ack}, ff, zap.NewNop())

	require.Equal(t, 1, ack.acked)
	require.Equal(t, []string{"videos/61.mp4", "covers/61.jpg"}, store.uploads)
	var v video.Video
	require.NoError(t, db.First(&v, 61).Error)
	require.Equal(t, video.StatusPublished, v.Status)
}

// TestBehavioral_TranscodeFailureWritesAuditEvent verifies the state machine
// audit trail records processing -> failed.
func TestBehavioral_TranscodeFailureWritesAuditEvent(t *testing.T) {
	db := newBehavioralWorkerDB(t)
	require.NoError(t, db.Create(&user.User{ID: 1, Username: "u", PasswordHash: "x"}).Error)
	require.NoError(t, db.Create(&video.Video{ID: 62, UserID: 1, Title: "v", Status: video.StatusProcessing}).Error)

	tmp := t.TempDir()
	raw := filepath.Join(tmp, "bad.mp4")
	require.NoError(t, os.WriteFile(raw, []byte("raw"), 0o644))
	ack := &fakeAck{}
	body, _ := json.Marshal(TranscodeJob{VideoID: 62, RawPath: raw, JobID: "job-z"})
	ff := &fakeFFmpeg{transcodeErr: errors.New("boom"), transcodeStderr: "corrupt", permStderr: "corrupt"}
	handleDeliveryWith(context.Background(), &config.C{TempUploadDir: tmp}, db, nil, &recordingStore{}, nil,
		amqp.Delivery{Body: body, Acknowledger: ack}, ff, zap.NewNop())

	var ev video.TranscodeEvent
	require.NoError(t, db.Where("video_id = ?", 62).First(&ev).Error)
	require.Equal(t, video.StatusProcessing, ev.FromStatus)
	require.Equal(t, video.StatusFailed, ev.ToStatus)
	require.Equal(t, "job-z", ev.JobID)
	require.NotEmpty(t, ev.Reason)
}
