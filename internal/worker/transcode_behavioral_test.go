package worker

import (
	"cakecake/internal/config"
	"cakecake/internal/data"
	"cakecake/internal/model/user"
	"cakecake/internal/model/video"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

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
