package video

import (
	"context"
	"testing"

	"cakecake/internal/service/servicetest"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestCreateVideoFromDirectUpload_RejectsBadCoverMagic(t *testing.T) {
	db := servicetest.NewDB(t)
	servicetest.SeedUser(t, db, 1, "u")

	rawKey := "uploads/1/x/source.mp4"
	coverKey := "uploads/1/x/cover.png"
	oss := &fakeSourceStore{
		exist:           map[string]bool{rawKey: true, coverKey: true},
		sizes:           map[string]int64{rawKey: 100, coverKey: 100},
		readPrefixBytes: []byte("plain text, definitely not an image"),
	}
	svc := NewVideoService(db, nil, zap.NewNop(), nil, &fakeTranscodePublisher{}, oss)
	ctx := context.Background()

	_, err := svc.CreateVideoFromDirectUpload(ctx, 1, "t", "d", "[]", "动画", rawKey, coverKey, 5)
	require.ErrorIs(t, err, ErrDirectUploadInvalidCover)
}

func TestCreateVideoFromDirectUpload_AcceptsValidCoverMagic(t *testing.T) {
	db := servicetest.NewDB(t)
	servicetest.SeedUser(t, db, 1, "u")

	rawKey := "uploads/1/x/source.mp4"
	coverKey := "uploads/1/x/cover.png"
	oss := &fakeSourceStore{
		exist: map[string]bool{rawKey: true, coverKey: true},
		sizes: map[string]int64{rawKey: 100, coverKey: 100},
		// default readPrefixBytes = JPEG magic
	}
	svc := NewVideoService(db, nil, zap.NewNop(), nil, &fakeTranscodePublisher{}, oss)
	ctx := context.Background()

	v, err := svc.CreateVideoFromDirectUpload(ctx, 1, "t", "d", "[]", "动画", rawKey, coverKey, 5)
	require.NoError(t, err)
	require.Equal(t, "t", v.Title)
}
