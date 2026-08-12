package worker

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/DATA-DOG/go-sqlmock"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"cakecake/internal/config"
	"cakecake/internal/model/video"
	"cakecake/internal/queue"
)

type fakeAck struct {
	acked  int
	nacked int
}

func (f *fakeAck) Ack(tag uint64, multiple bool) error {
	f.acked++
	return nil
}

func (f *fakeAck) Nack(tag uint64, multiple, requeue bool) error {
	f.nacked++
	return nil
}

func (f *fakeAck) Reject(tag uint64, requeue bool) error {
	f.nacked++
	return nil
}

type fakeFFmpeg struct {
	transcodeStderr string
	transcodeErr    error
	shotStderr      string
	shotErr         error
	permStderr      string
}

func (f *fakeFFmpeg) TranscodeToH264MP4(_, _ string) (string, error) {
	return f.transcodeStderr, f.transcodeErr
}

func (f *fakeFFmpeg) ScreenshotJPEG(_, _ string, _ float64) (string, error) {
	return f.shotStderr, f.shotErr
}

func (f *fakeFFmpeg) IsPermanentTranscodeFailure(stderr string) bool {
	return stderr != "" && stderr == f.permStderr
}

type fakePublisher struct {
	published []amqp.Publishing
	keys      []string
	err       error
}

func (f *fakePublisher) PublishWithContext(ctx context.Context, exchange, key string, mandatory, immediate bool, msg amqp.Publishing) error {
	f.published = append(f.published, msg)
	f.keys = append(f.keys, key)
	return f.err
}

type fakeObjectStore struct {
	uploads []string
	err     error
}

func (f *fakeObjectStore) UploadFile(objectKey, localPath string) error {
	f.uploads = append(f.uploads, objectKey)
	return f.err
}

func newMockDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	gormDB, err := gorm.Open(mysql.New(mysql.Config{Conn: db, SkipInitializeWithVersion: true}), &gorm.Config{
		SkipDefaultTransaction: true,
	})
	require.NoError(t, err)
	return gormDB, mock
}

func expectProcessingVideo(mock sqlmock.Sqlmock, id uint64) {
	mock.ExpectQuery("SELECT \\* FROM `videos` WHERE `videos`.`id` = .*").
		WithArgs(id, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "status"}).
			AddRow(id, 1, "processing"))
}

func deliveryFor(job queue.TranscodeJob, ack *fakeAck) amqp.Delivery {
	body, _ := json.Marshal(job)
	return amqp.Delivery{Body: body, Acknowledger: ack}
}

func TestFailVideo(t *testing.T) {
	db, mock := newMockDB(t)
	mock.ExpectExec("UPDATE `videos` SET .*fail_reason.* WHERE id = .*").
		WillReturnResult(sqlmock.NewResult(1, 1))
	failVideo(db, 7, "   ")
	require.NoError(t, mock.ExpectationsWereMet())

	db2, mock2 := newMockDB(t)
	mock2.ExpectExec("UPDATE `videos` SET .*fail_reason.* WHERE id = .*").
		WillReturnResult(sqlmock.NewResult(1, 1))
	failVideo(db2, 8, "bad reason")
	require.NoError(t, mock2.ExpectationsWereMet())
}

func TestHandleDelivery_BadJSON(t *testing.T) {
	db, mock := newMockDB(t)
	ack := &fakeAck{}
	d := amqp.Delivery{Body: []byte("{not-json"), Acknowledger: ack}
	handleDeliveryWith(context.Background(), &config.C{}, db, nil, nil, nil, d, &fakeFFmpeg{}, zap.NewNop())
	if ack.acked != 1 {
		t.Errorf("expected 1 ack, got %d", ack.acked)
	}
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestHandleDelivery_OSSNotConfigured(t *testing.T) {
	db, mock := newMockDB(t)
	expectProcessingVideo(mock, 1)
	mock.ExpectExec("UPDATE `videos` SET .*fail_reason.* WHERE id = .*").
		WillReturnResult(sqlmock.NewResult(1, 1))
	ack := &fakeAck{}
	d := amqp.Delivery{Body: []byte(`{"video_id":1,"raw_path":"/tmp/x.mp4"}`), Acknowledger: ack}
	handleDeliveryWith(context.Background(), &config.C{}, db, nil, nil, nil, d, &fakeFFmpeg{}, zap.NewNop())
	if ack.acked != 1 {
		t.Errorf("expected 1 ack, got %d", ack.acked)
	}
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestHandleDelivery_TranscodePermanent(t *testing.T) {
	db, mock := newMockDB(t)
	expectProcessingVideo(mock, 5)
	mock.ExpectExec("UPDATE `videos` SET .*fail_reason.* WHERE id = .*").
		WillReturnResult(sqlmock.NewResult(1, 1))
	ack := &fakeAck{}
	job := queue.TranscodeJob{VideoID: 5, RawPath: "/tmp/r.mp4"}
	d := deliveryFor(job, ack)
	ff := &fakeFFmpeg{transcodeErr: errors.New("boom"), transcodeStderr: "corrupt", permStderr: "corrupt"}
	handleDeliveryWith(context.Background(), &config.C{TempUploadDir: t.TempDir()}, db, nil, &fakeObjectStore{}, nil, d, ff, zap.NewNop())
	if ack.acked != 1 {
		t.Errorf("expected 1 ack, got %d", ack.acked)
	}
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestHandleDelivery_TranscodeRetryableExhausted(t *testing.T) {
	db, mock := newMockDB(t)
	expectProcessingVideo(mock, 6)
	mock.ExpectExec("INSERT INTO `transcode_dead_letters`").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("UPDATE `videos` SET .*fail_reason.* WHERE id = .*").
		WillReturnResult(sqlmock.NewResult(1, 1))
	ack := &fakeAck{}
	job := queue.TranscodeJob{VideoID: 6, RawPath: "/tmp/r.mp4", RetryCount: 3}
	d := deliveryFor(job, ack)
	ff := &fakeFFmpeg{transcodeErr: errors.New("flaky"), transcodeStderr: "flaky error"}
	pub := &fakePublisher{}
	handleDeliveryWith(context.Background(), &config.C{TempUploadDir: t.TempDir()}, db, pub, &fakeObjectStore{}, nil, d, ff, zap.NewNop())
	if ack.acked != 1 {
		t.Errorf("expected 1 ack, got %d", ack.acked)
	}
	require.Len(t, pub.published, 1)
	require.Equal(t, queue.TranscodeDeadQueue, pub.keys[0])
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestHandleDelivery_TranscodeRetryableRepublish(t *testing.T) {
	db, mock := newMockDB(t)
	expectProcessingVideo(mock, 9)
	oldDelay := transcodeRetryBaseDelay
	transcodeRetryBaseDelay = time.Millisecond
	defer func() { transcodeRetryBaseDelay = oldDelay }()

	ack := &fakeAck{}
	job := queue.TranscodeJob{VideoID: 9, RawPath: "/tmp/r.mp4"}
	d := deliveryFor(job, ack)
	ff := &fakeFFmpeg{transcodeErr: errors.New("flaky"), transcodeStderr: "flaky error"}
	pub := &fakePublisher{}
	handleDeliveryWith(context.Background(), &config.C{TempUploadDir: t.TempDir()}, db, pub, &fakeObjectStore{}, nil, d, ff, zap.NewNop())
	if ack.acked != 1 {
		t.Errorf("expected 1 ack, got %d", ack.acked)
	}
	require.Len(t, pub.published, 1)
	require.Equal(t, queue.TranscodeQueue, pub.keys[0])
	var republished queue.TranscodeJob
	require.NoError(t, json.Unmarshal(pub.published[0].Body, &republished))
	if republished.RetryCount != 1 {
		t.Errorf("expected RetryCount 1, got %d", republished.RetryCount)
	}
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestHandleDelivery_RepublishFailure(t *testing.T) {
	db, mock := newMockDB(t)
	expectProcessingVideo(mock, 10)
	mock.ExpectExec("INSERT INTO `transcode_dead_letters`").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("UPDATE `videos` SET .*fail_reason.* WHERE id = .*").
		WillReturnResult(sqlmock.NewResult(1, 1))
	oldDelay := transcodeRetryBaseDelay
	transcodeRetryBaseDelay = time.Millisecond
	defer func() { transcodeRetryBaseDelay = oldDelay }()

	ack := &fakeAck{}
	job := queue.TranscodeJob{VideoID: 10, RawPath: "/tmp/r.mp4"}
	d := deliveryFor(job, ack)
	ff := &fakeFFmpeg{transcodeErr: errors.New("flaky"), transcodeStderr: "flaky error"}
	pub := &fakePublisher{err: errors.New("channel closed")}
	handleDeliveryWith(context.Background(), &config.C{TempUploadDir: t.TempDir()}, db, pub, &fakeObjectStore{}, nil, d, ff, zap.NewNop())
	if ack.acked != 1 {
		t.Errorf("expected 1 ack, got %d", ack.acked)
	}
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestHandleDelivery_UploadVideoFailsExhausted(t *testing.T) {
	db, mock := newMockDB(t)
	expectProcessingVideo(mock, 11)
	mock.ExpectExec("INSERT INTO `transcode_dead_letters`").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("UPDATE `videos` SET .*fail_reason.* WHERE id = .*").
		WillReturnResult(sqlmock.NewResult(1, 1))
	ack := &fakeAck{}
	job := queue.TranscodeJob{VideoID: 11, RawPath: "/tmp/r.mp4", CoverPath: "/tmp/c.jpg", RetryCount: 3}
	d := deliveryFor(job, ack)
	ff := &fakeFFmpeg{} // transcode + screenshot succeed
	store := &fakeObjectStore{err: errors.New("oss down")}
	handleDeliveryWith(context.Background(), &config.C{TempUploadDir: t.TempDir()}, db, nil, store, nil, d, ff, zap.NewNop())
	if ack.acked != 1 {
		t.Errorf("expected 1 ack, got %d", ack.acked)
	}
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDeadLetterTranscode_PersistsAndPublishes(t *testing.T) {
	db := newBehavioralWorkerDB(t)
	pub := &fakePublisher{}
	job := queue.TranscodeJob{VideoID: 7, RawPath: "/tmp/r.mp4", RetryCount: 3}
	deadLetterTranscode(context.Background(), db, pub, zap.NewNop(), job, "oss down")

	require.Len(t, pub.published, 1)
	require.Equal(t, queue.TranscodeDeadQueue, pub.keys[0])
	var dead queue.TranscodeJob
	require.NoError(t, json.Unmarshal(pub.published[0].Body, &dead))
	require.Equal(t, uint64(7), dead.VideoID)

	var rec video.TranscodeDeadLetter
	require.NoError(t, db.Where("video_id = ?", 7).First(&rec).Error)
	require.Equal(t, "oss down", rec.Reason)
	require.Equal(t, 3, rec.RetryCount)
	require.Contains(t, rec.PayloadJSON, "oss down")
}

func TestTruncate_RuneSafeUTF8(t *testing.T) {
	// Byte slicing could split multi-byte UTF-8 runes; truncation must stay
	// valid UTF-8 and never exceed the rune budget.
	reason := "ffmpeg 转码失败: 文件损坏"
	for n := 1; n <= len(reason); n++ {
		got := truncate(reason, n)
		require.True(t, utf8.ValidString(got), "truncate(%q, %d) produced invalid UTF-8", reason, n)
		require.LessOrEqual(t, utf8.RuneCountInString(got), n)
	}
	require.Equal(t, "ffmpeg ", truncate(reason, 7))
	require.Equal(t, "abc", truncate("abcdef", 3))
	require.Equal(t, "abcdef", truncate("abcdef", 6))
	require.Equal(t, "", truncate("abc", 0))
}

func TestHandleDelivery_SuccessDBError(t *testing.T) {
	db, mock := newMockDB(t)
	expectProcessingVideo(mock, 12)
	mock.ExpectExec("UPDATE `videos` SET .* WHERE id = .*").
		WillReturnError(errors.New("db down"))
	ack := &fakeAck{}
	job := queue.TranscodeJob{VideoID: 12, RawPath: "/tmp/r.mp4", CoverPath: "/tmp/c.jpg"}
	d := deliveryFor(job, ack)
	ff := &fakeFFmpeg{}
	store := &fakeObjectStore{}
	handleDeliveryWith(context.Background(), &config.C{TempUploadDir: t.TempDir()}, db, nil, store, nil, d, ff, zap.NewNop())
	if ack.acked != 1 {
		t.Errorf("expected 1 ack, got %d", ack.acked)
	}
	require.Len(t, store.uploads, 2)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestHandleDelivery_SuccessScreenshotPublish(t *testing.T) {
	db, mock := newMockDB(t)
	expectProcessingVideo(mock, 13)
	// handleDelivery update
	mock.ExpectExec("UPDATE .*videos.* SET .*").
		WillReturnResult(sqlmock.NewResult(1, 1))
	// video.PublishVideo: SELECT video
	mock.ExpectQuery("SELECT \\* FROM `videos`").
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "status", "created_at"}).
			AddRow(13, 1, "processing", time.Now()))
	// video.PublishVideo: UPDATE status
	mock.ExpectExec("UPDATE `videos` SET .* WHERE `id` = .*").
		WillReturnResult(sqlmock.NewResult(1, 1))
	// video.PublishVideo: first_published_at backfill
	mock.ExpectExec("UPDATE `users` SET .*first_published_at.* WHERE .*").
		WillReturnResult(sqlmock.NewResult(1, 1))

	ack := &fakeAck{}
	job := queue.TranscodeJob{VideoID: 13, RawPath: "/tmp/r.mp4"}
	d := deliveryFor(job, ack)
	ff := &fakeFFmpeg{}
	store := &fakeObjectStore{}
	cfg := &config.C{TempUploadDir: t.TempDir()}
	handleDeliveryWith(context.Background(), cfg, db, nil, store, nil, d, ff, zap.NewNop())
	if ack.acked != 1 {
		t.Errorf("expected 1 ack, got %d", ack.acked)
	}
	require.Len(t, store.uploads, 2)
	require.NoError(t, mock.ExpectationsWereMet())
}
