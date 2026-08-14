package video

import (
	"context"
	"errors"
	"testing"

	"cakecake/internal/queue"
	"cakecake/internal/service/servicetest"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type fakeBackpressure struct {
	err error
}

func (f *fakeBackpressure) CheckTranscodeCapacity(context.Context) error {
	return f.err
}

func TestBackpressure_RejectsEnqueue(t *testing.T) {
	db := servicetest.NewDB(t)
	_, rdb := servicetest.NewRedis(t)
	oss := &fakeSourceStore{}
	svc := NewVideoDraftService(db, rdb, zap.NewNop(), &fakeTranscodePublisher{}, oss,
		&fakeBackpressure{err: queue.ErrTranscodeQueueFull})

	err := svc.EnqueueTranscode(context.Background(), 1, tempMedia(t, "a.mp4"), "")
	require.ErrorIs(t, err, ErrTranscodeQueueFull)
}

func TestBackpressure_RejectsDirectUpload(t *testing.T) {
	db := servicetest.NewDB(t)
	_, rdb := servicetest.NewRedis(t)
	rawKey := "uploads/7/abc/source.mp4"
	svc := NewVideoService(db, rdb, zap.NewNop(), nil, &fakeTranscodePublisher{},
		&fakeSourceStore{exist: map[string]bool{rawKey: true}},
		&fakeBackpressure{err: errors.New("capacity exceeded")})

	_, err := svc.CreateVideoFromDirectUpload(context.Background(), 7, "t", "", "[]", "", rawKey, "", 0)
	require.ErrorIs(t, err, ErrTranscodeQueueFull)
}
