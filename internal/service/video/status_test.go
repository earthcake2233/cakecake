package video

import (
	"context"
	"testing"

	"cakecake/internal/model/video"
	"cakecake/internal/service/servicetest"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestValidateTranscodeStatusTransition(t *testing.T) {
	valid := [][2]string{
		{video.StatusProcessing, video.StatusPendingReview},
		{video.StatusProcessing, video.StatusPublished},
		{video.StatusProcessing, video.StatusFailed},
		{video.StatusPendingReview, video.StatusPublished},
		{video.StatusPendingReview, video.StatusRejected},
	}
	for _, tc := range valid {
		require.True(t, ValidateTranscodeStatusTransition(tc[0], tc[1]), "%s -> %s should be legal", tc[0], tc[1])
	}

	invalid := [][2]string{
		{video.StatusPendingReview, video.StatusProcessing},
		{video.StatusPublished, video.StatusFailed},
		{video.StatusFailed, video.StatusPublished},
		{video.StatusProcessing, video.StatusRejected},
		{video.StatusRejected, video.StatusPendingReview},
	}
	for _, tc := range invalid {
		require.False(t, ValidateTranscodeStatusTransition(tc[0], tc[1]), "%s -> %s should be illegal", tc[0], tc[1])
	}
}

// TestPublishVideo_RejectsIllegalTransition proves a stale publish cannot
// overwrite a video that was already rejected.
func TestPublishVideo_RejectsIllegalTransition(t *testing.T) {
	db := servicetest.NewDB(t)
	servicetest.SeedUser(t, db, 1, "u")
	require.NoError(t, db.Create(&video.Video{ID: 77, UserID: 1, Title: "v", Status: video.StatusRejected}).Error)
	svc := NewVideoService(db, nil, zap.NewNop(), nil, nil, nil)

	err := svc.Publish(context.Background(), 77, nil)
	require.Error(t, err)
	var v video.Video
	require.NoError(t, db.First(&v, 77).Error)
	require.Equal(t, video.StatusRejected, v.Status)
}
