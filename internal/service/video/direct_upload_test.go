package video

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"cakecake/internal/model/video"
	"cakecake/internal/queue"
	"cakecake/internal/service/servicetest"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestCreateDirectUploadTicket_IssuesPresignedURLs(t *testing.T) {
	db := servicetest.NewDB(t)
	_, rdb := servicetest.NewRedis(t)
	mq := &fakeTranscodePublisher{}
	oss := &fakeSourceStore{}
	svc := NewVideoService(db, rdb, zap.NewNop(), nil, mq, oss)

	ticket, err := svc.CreateDirectUploadTicket(context.Background(), 7, "my video.mp4", "cover.png", "video/mp4", "image/png")
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(ticket.RawKey, "uploads/7/"))
	require.True(t, strings.HasSuffix(ticket.RawKey, "/source.mp4"))
	require.Contains(t, ticket.RawURL, ticket.RawKey)
	require.True(t, strings.HasPrefix(ticket.CoverKey, "uploads/7/"))
	require.True(t, strings.HasSuffix(ticket.CoverKey, "/cover.png"))
	require.Contains(t, ticket.CoverURL, ticket.CoverKey)
	require.Equal(t, int64(900), ticket.ExpiresIn)
	require.Equal(t, []string{"video/mp4", "image/png"}, oss.putContentTypes)
}

func TestCreateDirectUploadTicket_NoCover(t *testing.T) {
	db := servicetest.NewDB(t)
	_, rdb := servicetest.NewRedis(t)
	oss := &fakeSourceStore{}
	svc := NewVideoService(db, rdb, zap.NewNop(), nil, &fakeTranscodePublisher{}, oss)

	ticket, err := svc.CreateDirectUploadTicket(context.Background(), 7, "a.mp4", "", "video/mp4", "")
	require.NoError(t, err)
	require.Empty(t, ticket.CoverKey)
	require.Empty(t, ticket.CoverURL)
	require.Equal(t, []string{"video/mp4"}, oss.putContentTypes)
}

func TestCreateDirectUploadTicket_OSSDisabled(t *testing.T) {
	db := servicetest.NewDB(t)
	_, rdb := servicetest.NewRedis(t)
	svc := NewVideoService(db, rdb, zap.NewNop(), nil, &fakeTranscodePublisher{}, nil)

	_, err := svc.CreateDirectUploadTicket(context.Background(), 7, "a.mp4", "", "", "")
	require.ErrorIs(t, err, ErrDirectUploadUnavailable)
}

func TestCreateVideoFromDirectUpload_CreatesVideoAndOutbox(t *testing.T) {
	db := servicetest.NewDB(t)
	_, rdb := servicetest.NewRedis(t)
	rawKey := "uploads/7/abc/source.mp4"
	coverKey := "uploads/7/abc/cover.png"
	oss := &fakeSourceStore{exist: map[string]bool{rawKey: true}}
	svc := NewVideoService(db, rdb, zap.NewNop(), nil, &fakeTranscodePublisher{}, oss)

	servicetest.SeedUser(t, db, 7, "uploader")
	v, err := svc.CreateVideoFromDirectUpload(context.Background(), 7, "t", "d", "[]", "anime", rawKey, coverKey, 12.5)
	require.NoError(t, err)
	require.NotZero(t, v.ID)
	require.Equal(t, video.StatusProcessing, v.Status)
	require.Equal(t, 12.5, v.DurationSec) // client duration hint is stored; worker re-probes later

	var row video.Video
	require.NoError(t, db.First(&row, v.ID).Error)
	require.Equal(t, "t", row.Title)
	var ob video.TranscodeOutbox
	require.NoError(t, db.Where("video_id = ?", v.ID).First(&ob).Error)
	require.Equal(t, video.OutboxStatusPending, ob.Status)
	var job queue.TranscodeJob
	require.NoError(t, json.Unmarshal([]byte(ob.Payload), &job))
	require.Equal(t, rawKey, job.RawKey)
	require.Equal(t, coverKey, job.CoverKey)
	require.Empty(t, job.RawPath)
	require.NotEmpty(t, job.JobID)
}

func TestCreateVideoFromDirectUpload_MissingSource(t *testing.T) {
	db := servicetest.NewDB(t)
	_, rdb := servicetest.NewRedis(t)
	mq := &fakeTranscodePublisher{}
	svc := NewVideoService(db, rdb, zap.NewNop(), nil, mq, &fakeSourceStore{exist: map[string]bool{}})

	_, err := svc.CreateVideoFromDirectUpload(context.Background(), 7, "t", "", "[]", "", "uploads/7/missing/source.mp4", "", 0)
	require.ErrorIs(t, err, ErrDirectUploadSourceMissing)
	require.Empty(t, mq.bodies)
}

func TestCreateVideoFromDirectUpload_TooLarge(t *testing.T) {
	db := servicetest.NewDB(t)
	_, rdb := servicetest.NewRedis(t)
	mq := &fakeTranscodePublisher{}
	rawKey := "uploads/7/big/source.mp4"
	svc := NewVideoService(db, rdb, zap.NewNop(), nil, mq, &fakeSourceStore{
		exist: map[string]bool{rawKey: true},
		sizes: map[string]int64{rawKey: int64(500<<20) + 1},
	})

	_, err := svc.CreateVideoFromDirectUpload(context.Background(), 7, "t", "", "[]", "", rawKey, "", 0)
	require.ErrorIs(t, err, ErrDirectUploadTooLarge)
	require.Empty(t, mq.bodies)
	var count int64
	require.NoError(t, db.Model(&video.Video{}).Count(&count).Error)
	require.Zero(t, count)
}

func TestCreateVideoFromDirectUpload_InvalidKeys(t *testing.T) {
	db := servicetest.NewDB(t)
	_, rdb := servicetest.NewRedis(t)
	mq := &fakeTranscodePublisher{}
	svc := NewVideoService(db, rdb, zap.NewNop(), nil, mq, &fakeSourceStore{})

	_, err := svc.CreateVideoFromDirectUpload(context.Background(), 7, "t", "", "[]", "", "raws/7/source.mp4", "", 0)
	require.ErrorIs(t, err, ErrDirectUploadInvalidKey)

	_, err = svc.CreateVideoFromDirectUpload(context.Background(), 7, "t", "", "[]", "", "uploads/8/ok/source.mp4", "", 0)
	require.ErrorIs(t, err, ErrDirectUploadInvalidKey)

	_, err = svc.CreateVideoFromDirectUpload(context.Background(), 7, "t", "", "[]", "", "uploads/7/ok/source.mp4", "uploads/8/ok/cover.png", 0)
	require.ErrorIs(t, err, ErrDirectUploadInvalidKey)
	require.Empty(t, mq.bodies)
}

func TestCreateVideoFromDirectUpload_DoesNotProbeAtSubmit(t *testing.T) {
	db := servicetest.NewDB(t)
	_, rdb := servicetest.NewRedis(t)
	mq := &fakeTranscodePublisher{}
	svc := NewVideoService(db, rdb, zap.NewNop(), nil, mq, &fakeSourceStore{exist: map[string]bool{"uploads/7/abc/source.mp4": true}})

	probed := false
	oldProbe := VideoProbe
	VideoProbe = func(_ context.Context, _ string) (float64, error) {
		probed = true
		return 0, errors.New("probe should not run at submit")
	}
	defer func() { VideoProbe = oldProbe }()

	_, err := svc.CreateVideoFromDirectUpload(context.Background(), 7, "t", "", "[]", "", "uploads/7/abc/source.mp4", "", 0)
	require.NoError(t, err)
	require.False(t, probed, "submit must not download/probe the object")
	require.Empty(t, mq.bodies)
	var v video.Video
	require.NoError(t, db.First(&v, "title = ?", "t").Error)
	require.Equal(t, video.StatusProcessing, v.Status)
}

func TestCreateVideoFromDirectUpload_Idempotent(t *testing.T) {
	db := servicetest.NewDB(t)
	_, rdb := servicetest.NewRedis(t)
	rawKey := "uploads/7/abc/source.mp4"
	svc := NewVideoService(db, rdb, zap.NewNop(), nil, &fakeTranscodePublisher{}, &fakeSourceStore{exist: map[string]bool{rawKey: true}})

	oldProbe := VideoProbe
	VideoProbe = func(_ context.Context, _ string) (float64, error) { return 12.5, nil }
	defer func() { VideoProbe = oldProbe }()

	servicetest.SeedUser(t, db, 7, "uploader")
	v1, err := svc.CreateVideoFromDirectUpload(context.Background(), 7, "t", "", "[]", "", rawKey, "", 0)
	require.NoError(t, err)
	v2, err := svc.CreateVideoFromDirectUpload(context.Background(), 7, "t", "", "[]", "", rawKey, "", 0)
	require.NoError(t, err)
	require.Equal(t, v1.ID, v2.ID, "the same raw_key must not create a second video row")

	var count int64
	require.NoError(t, db.Model(&video.Video{}).Count(&count).Error)
	require.Equal(t, int64(1), count)
	var outboxCount int64
	require.NoError(t, db.Model(&video.TranscodeOutbox{}).Count(&outboxCount).Error)
	require.Equal(t, int64(1), outboxCount, "one raw_key -> one outbox row")
}

func TestCreateVideoFromDirectUpload_ClaimedByOtherUser(t *testing.T) {
	db := servicetest.NewDB(t)
	_, rdb := servicetest.NewRedis(t)
	rawKey := "uploads/7/abc/source.mp4"
	require.NoError(t, db.Create(&video.DirectUploadClaim{RawKey: rawKey, UserID: 8, VideoID: 99}).Error)
	svc := NewVideoService(db, rdb, zap.NewNop(), nil, &fakeTranscodePublisher{}, &fakeSourceStore{exist: map[string]bool{rawKey: true}})
	oldProbe := VideoProbe
	VideoProbe = func(_ context.Context, _ string) (float64, error) { return 5, nil }
	defer func() { VideoProbe = oldProbe }()

	_, err := svc.CreateVideoFromDirectUpload(context.Background(), 7, "t", "", "[]", "", rawKey, "", 0)
	require.ErrorIs(t, err, ErrDirectUploadAlreadyClaimed)
}

func TestCreateVideoFromDirectUpload_ClaimInProgress(t *testing.T) {
	db := servicetest.NewDB(t)
	_, rdb := servicetest.NewRedis(t)
	rawKey := "uploads/7/abc/source.mp4"
	require.NoError(t, db.Create(&video.DirectUploadClaim{RawKey: rawKey, UserID: 7}).Error)
	svc := NewVideoService(db, rdb, zap.NewNop(), nil, &fakeTranscodePublisher{}, &fakeSourceStore{exist: map[string]bool{rawKey: true}})
	oldProbe := VideoProbe
	VideoProbe = func(_ context.Context, _ string) (float64, error) { return 5, nil }
	defer func() { VideoProbe = oldProbe }()

	_, err := svc.CreateVideoFromDirectUpload(context.Background(), 7, "t", "", "[]", "", rawKey, "", 0)
	require.ErrorIs(t, err, ErrDirectUploadInProgress)
}

func TestSafeObjectExt(t *testing.T) {
	longName := "a." + strings.Repeat("a", 20)
	require.Equal(t, ".mp4", safeObjectExt("a.mp4"))
	require.Equal(t, ".mp4", safeObjectExt("a.MP4"))
	require.Equal(t, ".png", safeObjectExt("c.png"))
	require.Equal(t, "", safeObjectExt("noext"))
	require.Equal(t, ".mp4", safeObjectExt("bad;name.mp4"))
	require.Equal(t, "", safeObjectExt(longName))
}
