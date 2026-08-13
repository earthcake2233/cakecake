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

	ticket, err := svc.CreateDirectUploadTicket(context.Background(), 7, "my video.mp4", "cover.png")
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(ticket.RawKey, "uploads/7/"))
	require.True(t, strings.HasSuffix(ticket.RawKey, "/source.mp4"))
	require.Contains(t, ticket.RawURL, ticket.RawKey)
	require.True(t, strings.HasPrefix(ticket.CoverKey, "uploads/7/"))
	require.True(t, strings.HasSuffix(ticket.CoverKey, "/cover.png"))
	require.Contains(t, ticket.CoverURL, ticket.CoverKey)
	require.Equal(t, int64(900), ticket.ExpiresIn)
}

func TestCreateDirectUploadTicket_NoCover(t *testing.T) {
	db := servicetest.NewDB(t)
	_, rdb := servicetest.NewRedis(t)
	svc := NewVideoService(db, rdb, zap.NewNop(), nil, &fakeTranscodePublisher{}, &fakeSourceStore{})

	ticket, err := svc.CreateDirectUploadTicket(context.Background(), 7, "a.mp4", "")
	require.NoError(t, err)
	require.Empty(t, ticket.CoverKey)
	require.Empty(t, ticket.CoverURL)
}

func TestCreateDirectUploadTicket_OSSDisabled(t *testing.T) {
	db := servicetest.NewDB(t)
	_, rdb := servicetest.NewRedis(t)
	svc := NewVideoService(db, rdb, zap.NewNop(), nil, &fakeTranscodePublisher{}, nil)

	_, err := svc.CreateDirectUploadTicket(context.Background(), 7, "a.mp4", "")
	require.ErrorIs(t, err, ErrDirectUploadUnavailable)
}

func TestCreateVideoFromDirectUpload_CreatesAndEnqueues(t *testing.T) {
	db := servicetest.NewDB(t)
	_, rdb := servicetest.NewRedis(t)
	mq := &fakeTranscodePublisher{}
	rawKey := "uploads/7/abc/source.mp4"
	coverKey := "uploads/7/abc/cover.png"
	oss := &fakeSourceStore{exist: map[string]bool{rawKey: true}}
	svc := NewVideoService(db, rdb, zap.NewNop(), nil, mq, oss)

	oldProbe := VideoProbe
	VideoProbe = func(_ context.Context, _ string) (float64, error) { return 12.5, nil }
	defer func() { VideoProbe = oldProbe }()

	servicetest.SeedUser(t, db, 7, "uploader")
	v, err := svc.CreateVideoFromDirectUpload(context.Background(), 7, "t", "d", "[]", "anime", rawKey, coverKey)
	require.NoError(t, err)
	require.NotZero(t, v.ID)
	require.Equal(t, video.StatusProcessing, v.Status)
	require.Equal(t, 12.5, v.DurationSec)

	var row video.Video
	require.NoError(t, db.First(&row, v.ID).Error)
	require.Equal(t, "t", row.Title)
	require.Len(t, mq.bodies, 1)
	var job queue.TranscodeJob
	require.NoError(t, json.Unmarshal(mq.bodies[0], &job))
	require.Equal(t, rawKey, job.RawKey)
	require.Equal(t, coverKey, job.CoverKey)
	require.Empty(t, job.RawPath)
}

func TestCreateVideoFromDirectUpload_MissingSource(t *testing.T) {
	db := servicetest.NewDB(t)
	_, rdb := servicetest.NewRedis(t)
	mq := &fakeTranscodePublisher{}
	svc := NewVideoService(db, rdb, zap.NewNop(), nil, mq, &fakeSourceStore{exist: map[string]bool{}})

	_, err := svc.CreateVideoFromDirectUpload(context.Background(), 7, "t", "", "[]", "", "uploads/7/missing/source.mp4", "")
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

	_, err := svc.CreateVideoFromDirectUpload(context.Background(), 7, "t", "", "[]", "", rawKey, "")
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

	_, err := svc.CreateVideoFromDirectUpload(context.Background(), 7, "t", "", "[]", "", "raws/7/source.mp4", "")
	require.ErrorIs(t, err, ErrDirectUploadInvalidKey)

	_, err = svc.CreateVideoFromDirectUpload(context.Background(), 7, "t", "", "[]", "", "uploads/8/ok/source.mp4", "")
	require.ErrorIs(t, err, ErrDirectUploadInvalidKey)

	_, err = svc.CreateVideoFromDirectUpload(context.Background(), 7, "t", "", "[]", "", "uploads/7/ok/source.mp4", "uploads/8/ok/cover.png")
	require.ErrorIs(t, err, ErrDirectUploadInvalidKey)
	require.Empty(t, mq.bodies)
}

func TestCreateVideoFromDirectUpload_ProbeFailure(t *testing.T) {
	db := servicetest.NewDB(t)
	_, rdb := servicetest.NewRedis(t)
	mq := &fakeTranscodePublisher{}
	svc := NewVideoService(db, rdb, zap.NewNop(), nil, mq, &fakeSourceStore{exist: map[string]bool{"uploads/7/abc/source.mp4": true}})

	oldProbe := VideoProbe
	VideoProbe = func(_ context.Context, _ string) (float64, error) { return 0, errors.New("not a video") }
	defer func() { VideoProbe = oldProbe }()

	_, err := svc.CreateVideoFromDirectUpload(context.Background(), 7, "t", "", "[]", "", "uploads/7/abc/source.mp4", "")
	require.Error(t, err)
	require.Empty(t, mq.bodies)
	var count int64
	require.NoError(t, db.Model(&video.Video{}).Count(&count).Error)
	require.Zero(t, count)
}

func TestCreateVideoFromDirectUpload_Idempotent(t *testing.T) {
	db := servicetest.NewDB(t)
	_, rdb := servicetest.NewRedis(t)
	mq := &fakeTranscodePublisher{}
	rawKey := "uploads/7/abc/source.mp4"
	svc := NewVideoService(db, rdb, zap.NewNop(), nil, mq, &fakeSourceStore{exist: map[string]bool{rawKey: true}})

	oldProbe := VideoProbe
	VideoProbe = func(_ context.Context, _ string) (float64, error) { return 12.5, nil }
	defer func() { VideoProbe = oldProbe }()

	servicetest.SeedUser(t, db, 7, "uploader")
	v1, err := svc.CreateVideoFromDirectUpload(context.Background(), 7, "t", "", "[]", "", rawKey, "")
	require.NoError(t, err)
	v2, err := svc.CreateVideoFromDirectUpload(context.Background(), 7, "t", "", "[]", "", rawKey, "")
	require.NoError(t, err)
	require.Equal(t, v1.ID, v2.ID, "the same raw_key must not create a second video row")

	var count int64
	require.NoError(t, db.Model(&video.Video{}).Count(&count).Error)
	require.Equal(t, int64(1), count)
	require.Len(t, mq.bodies, 2)
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

	_, err := svc.CreateVideoFromDirectUpload(context.Background(), 7, "t", "", "[]", "", rawKey, "")
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

	_, err := svc.CreateVideoFromDirectUpload(context.Background(), 7, "t", "", "[]", "", rawKey, "")
	require.ErrorIs(t, err, ErrDirectUploadInProgress)
}

func TestCreateVideoFromDirectUpload_EnqueueFailureReleasesClaim(t *testing.T) {
	db := servicetest.NewDB(t)
	_, rdb := servicetest.NewRedis(t)
	mq := &fakeTranscodePublisher{err: errors.New("broker down")}
	rawKey := "uploads/7/abc/source.mp4"
	svc := NewVideoService(db, rdb, zap.NewNop(), nil, mq, &fakeSourceStore{exist: map[string]bool{rawKey: true}})

	oldProbe := VideoProbe
	VideoProbe = func(_ context.Context, _ string) (float64, error) { return 12.5, nil }
	defer func() { VideoProbe = oldProbe }()

	servicetest.SeedUser(t, db, 7, "uploader")
	_, err := svc.CreateVideoFromDirectUpload(context.Background(), 7, "t", "", "[]", "", rawKey, "")
	require.Error(t, err)
	var claims int64
	require.NoError(t, db.Model(&video.DirectUploadClaim{}).Count(&claims).Error)
	require.Zero(t, claims, "failed enqueue must release the idempotency claim")
	var videos int64
	require.NoError(t, db.Model(&video.Video{}).Count(&videos).Error)
	require.Zero(t, videos)

	mq.err = nil
	_, err = svc.CreateVideoFromDirectUpload(context.Background(), 7, "t", "", "[]", "", rawKey, "")
	require.NoError(t, err, "retry after a failed enqueue must succeed")
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
