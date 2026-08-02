package service

import (
	"cakecake/internal/model/admin"
	"cakecake/internal/model/video"
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func newVideoService(t *testing.T) *VideoService {
	t.Helper()
	db := newAgentTestDB(t)
	_, rdb := newAgentTestRedis(t)
	return NewVideoService(db, rdb, zapNop(), nil, nil)
}

func seedVideoRow(t *testing.T, db *gorm.DB, id, userID uint64, status, zone string) {
	t.Helper()
	require.NoError(t, db.Create(&video.Video{
		ID: id, UserID: userID, Title: "v", Status: status, Zone: zone,
	}).Error)
}

func TestVideoService_CRUDAndPublish(t *testing.T) {
	s := newVideoService(t)
	ctx := context.Background()
	seedUser(t, s.db, 1, "alice")
	seedVideoRow(t, s.db, 10, 1, video.StatusDraft, "")

	v, err := s.GetVideoByID(ctx, 10)
	require.NoError(t, err)
	require.Equal(t, video.StatusDraft, v.Status)
	_, err = s.GetPublishedVideo(ctx, 10)
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)

	// Publish.
	require.NoError(t, s.Publish(ctx, 10, nil))
	pub, err := s.GetPublishedVideo(ctx, 10)
	require.NoError(t, err)
	require.Equal(t, video.StatusPublished, pub.Status)
	require.NoError(t, s.Publish(ctx, 10, nil)) // idempotent

	// Publish missing video.
	require.Error(t, s.Publish(ctx, 999, nil))

	// Update + create.
	require.NoError(t, s.UpdateVideo(ctx, v, map[string]interface{}{"title": "renamed"}))
	nv := &video.Video{UserID: 1, Title: "new", Status: video.StatusDraft}
	require.NoError(t, s.CreateVideoRecord(ctx, nv))
	require.NotZero(t, nv.ID)

	// Delete.
	require.NoError(t, s.DeleteVideoByID(ctx, nv.ID))
	_, err = s.GetVideoByID(ctx, nv.ID)
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)
}

func TestVideoService_ListPublished(t *testing.T) {
	s := newVideoService(t)
	ctx := context.Background()
	seedVideoRow(t, s.db, 10, 1, video.StatusPublished, "anime")
	seedVideoRow(t, s.db, 11, 1, video.StatusPublished, "anime-comedy")
	seedVideoRow(t, s.db, 12, 1, video.StatusDraft, "anime")

	res, err := s.ListPublishedVideos(ctx, VideoListOpts{Limit: 10, SortKey: "time"})
	require.NoError(t, err)
	require.Len(t, res.Videos, 2)

	// Zone filter.
	res, err = s.ListPublishedVideos(ctx, VideoListOpts{Limit: 10, SortKey: "hot", ZoneParent: "anime"})
	require.NoError(t, err)
	require.Len(t, res.Videos, 2)
	require.Equal(t, int64(2), res.ZoneVideoCount)

	// Recent only.
	res, err = s.ListPublishedVideos(ctx, VideoListOpts{Limit: 10, SortKey: "time", RecentOnly: true, Days: 1})
	require.NoError(t, err)
	require.Len(t, res.Videos, 2)

	// Counts.
	require.Equal(t, int64(2), s.CountPublishedVideos(ctx))
	require.Equal(t, int64(2), s.CountZoneVideos("anime"))
	require.Zero(t, s.CountZoneVideos(""))
	require.Equal(t, int64(2), s.GetZoneVideoCount("anime"))
	n, err := s.CountByStatus(ctx, video.StatusDraft)
	require.NoError(t, err)
	require.Equal(t, int64(1), n)

	counts := s.CountMyVideosByStatus(1)
	require.Equal(t, int64(2), counts[video.StatusPublished])
	require.Zero(t, counts[video.StatusDraft]) // drafts are not part of the status dashboard
}

func TestVideoService_MyVideosAndLike(t *testing.T) {
	s := newVideoService(t)
	ctx := context.Background()
	seedUser(t, s.db, 1, "alice")
	seedVideoRow(t, s.db, 10, 1, video.StatusPublished, "")
	seedVideoRow(t, s.db, 11, 1, video.StatusDraft, "")
	seedVideoRow(t, s.db, 12, 2, video.StatusPublished, "")

	list, total, err := s.ListMyVideos(ctx, 1, "", 1, 10)
	require.NoError(t, err)
	require.Equal(t, int64(2), total)
	require.Len(t, list, 2)

	// Advanced filter.
	page, err := s.ListMyVideosAdvanced(ctx, MyVideoFilter{UserID: 1, Statuses: []string{video.StatusPublished}, TitleQ: "v", Page: 1, PageSize: 10, SortKey: "like"})
	require.NoError(t, err)
	require.Equal(t, int64(1), page.Total)

	// Cursor list.
	rows, err := s.ListUserPublishedVideosCursor(ctx, 1, 0, 10)
	require.NoError(t, err)
	require.Len(t, rows, 1)

	// Like toggle.
	liked, err := s.ToggleVideoLike(ctx, 1, 10)
	require.NoError(t, err)
	require.True(t, liked)
	liked, err = s.ToggleVideoLike(ctx, 1, 10)
	require.NoError(t, err)
	require.False(t, liked)
	_, err = s.ToggleVideoLike(ctx, 1, 999)
	require.Error(t, err)

	// Delete cascade with custom fn.
	called := false
	require.NoError(t, s.DeleteVideoWithCascade(ctx, 11, func(tx *gorm.DB) error {
		called = true
		return nil
	}))
	require.True(t, called)
	// Default delete.
	require.NoError(t, s.DeleteVideoWithCascade(ctx, 11, nil))
}

func TestVideoService_BannersAndAdmin(t *testing.T) {
	s := newVideoService(t)
	ctx := context.Background()
	b := &admin.HomeBanner{Title: "b", ImageURL: "u", Enabled: true, SortOrder: 1}
	require.NoError(t, s.CreateBanner(ctx, b))
	require.NotZero(t, b.ID)

	got, err := s.GetBanner(ctx, b.ID)
	require.NoError(t, err)
	require.Equal(t, "b", got.Title)

	active, err := s.ListActiveBanners(ctx)
	require.NoError(t, err)
	require.Len(t, active, 1)
	all, err := s.ListBanners(ctx)
	require.NoError(t, err)
	require.Len(t, all, 1)

	require.NoError(t, s.UpdateBanner(ctx, b.ID, map[string]interface{}{"title": "b2"}))
	require.NoError(t, s.DeleteBanner(ctx, b.ID))
	_, err = s.GetBanner(ctx, b.ID)
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)

	// Admin.
	seedVideoRow(t, s.db, 10, 1, video.StatusPendingReview, "")
	require.NoError(t, s.AdminUpdateVideo(ctx, 10, map[string]interface{}{"status": video.StatusPublished}))
	adminRes, err := s.AdminListVideos(ctx, []string{video.StatusPublished}, "", 1, 10)
	require.NoError(t, err)
	require.Equal(t, int64(1), adminRes.Total)
	require.Zero(t, adminRes.PendingCount)
	called := false
	require.NoError(t, s.AdminDeleteVideoCascade(ctx, 10, func(tx *gorm.DB) error {
		called = true
		return nil
	}))
	require.True(t, called)
}

func TestVideoService_Transcode(t *testing.T) {
	s := newVideoService(t)
	ctx := context.Background()
	err := s.PublishTranscode(ctx, []byte("x"))
	require.Error(t, err) // mq nil -> not configured
}
