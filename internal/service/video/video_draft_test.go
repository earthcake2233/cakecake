package video

import (
	"cakecake/internal/model/video"
	"cakecake/internal/service/servicetest"
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func newVideoDraftService(t *testing.T) *VideoDraftService {
	t.Helper()
	db := servicetest.NewDB(t)
	_, rdb := servicetest.NewRedis(t)
	return NewVideoDraftService(db, rdb, servicetest.ZapNop(), nil, nil)
}

func TestVideoDraftService_CRUD(t *testing.T) {
	s := newVideoDraftService(t)
	ctx := context.Background()

	d := &video.Video{UserID: 1, Title: "draft", Status: video.StatusDraft}
	require.NoError(t, s.CreateDraft(ctx, d))
	require.NotZero(t, d.ID)

	got, err := s.GetDraftByID(ctx, d.ID)
	require.NoError(t, err)
	require.Equal(t, video.StatusDraft, got.Status)

	owned, err := s.GetOwnedDraft(ctx, d.ID, 1)
	require.NoError(t, err)
	require.Equal(t, d.ID, owned.ID)
	_, err = s.GetOwnedDraft(ctx, d.ID, 2)
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)

	owned, err = s.GetOwnedDraftByStatus(ctx, d.ID, 1, video.StatusDraft)
	require.NoError(t, err)
	require.Equal(t, d.ID, owned.ID)
	_, err = s.GetOwnedDraftByStatus(ctx, d.ID, 1, video.StatusPublished)
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)

	require.NoError(t, s.UpdateDraft(ctx, d, map[string]interface{}{"title": "updated"}))
	require.NoError(t, s.UpdateDraftField(ctx, d, "description", "desc"))
	got, err = s.RefetchDraft(ctx, d.ID)
	require.NoError(t, err)
	require.Equal(t, "updated", got.Title)
	require.Equal(t, "desc", got.Description)

	// Count + list.
	require.NoError(t, s.CreateDraft(ctx, &video.Video{UserID: 1, Title: "d2", Status: video.StatusDraft}))
	n, err := s.CountUserDrafts(ctx, 1)
	require.NoError(t, err)
	require.Equal(t, int64(2), n)
	list, total, err := s.ListUserDrafts(ctx, 1, 1, 10)
	require.NoError(t, err)
	require.Equal(t, int64(2), total)
	require.Len(t, list, 2)

	// Delete + transcode publish (nil mq).
	require.NoError(t, s.DeleteDraft(ctx, d.ID))
	_, err = s.GetDraftByID(ctx, d.ID)
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)
	err = s.PublishTranscode(ctx, []byte("x"))
	require.Error(t, err)
}
