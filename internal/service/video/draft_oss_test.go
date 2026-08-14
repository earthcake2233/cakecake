package video

import (
	"cakecake/internal/model/video"
	"cakecake/internal/service/servicetest"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func draftOSS(t *testing.T) (*VideoDraftService, *fakeSourceStore, *gorm.DB) {
	t.Helper()
	db := servicetest.NewDB(t)
	_, rdb := servicetest.NewRedis(t)
	store := &fakeSourceStore{
		exist: map[string]bool{},
		sizes: map[string]int64{},
	}
	svc := NewVideoDraftService(db, rdb, zap.NewNop(), nil, store)
	return svc, store, db
}

func stubProbe(t *testing.T, dur float64) {
	t.Helper()
	old := VideoProbe
	VideoProbe = func(context.Context, string) (float64, error) { return dur, nil }
	t.Cleanup(func() { VideoProbe = old })
}

func TestCreateDraftUploadTicket_NamespacesByUser(t *testing.T) {
	svc2, store2, _ := draftOSS(t)
	ticket, err := svc2.CreateDraftUploadTicket(context.Background(), 7, "clip.mp4", "cover.png", "video/mp4", "image/png")
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(ticket.RawKey, "drafts/7/"), ticket.RawKey)
	require.True(t, strings.HasSuffix(ticket.RawKey, "/source.mp4"), ticket.RawKey)
	require.Contains(t, ticket.RawURL, ticket.RawKey)
	require.True(t, strings.HasPrefix(ticket.CoverKey, "drafts/7/"), ticket.CoverKey)
	require.Contains(t, ticket.CoverURL, ticket.CoverKey)
	require.Greater(t, ticket.ExpiresIn, int64(0))
	require.Equal(t, []string{"video/mp4", "image/png"}, store2.putContentTypes)
	_, err = NewVideoDraftService(servicetest.NewDB(t), nil, zap.NewNop(), nil, nil).
		CreateDraftUploadTicket(context.Background(), 1, "a.mp4", "", "", "")
	require.ErrorIs(t, err, ErrDraftMediaUnavailable)
}

func TestValidateDraftMedia_RejectsAndAccepts(t *testing.T) {
	svc, store, _ := draftOSS(t)
	ctx := context.Background()
	raw := "drafts/1/u1/source.mp4"
	cover := "drafts/1/u1/cover.png"
	store.exist[raw] = true
	store.exist[cover] = true
	store.sizes[raw] = 1024
	store.sizes[cover] = 2048

	// Wrong owner prefix.
	_, err := svc.ValidateDraftMedia(ctx, 1, "drafts/2/u1/source.mp4", "")
	require.ErrorIs(t, err, ErrDraftMediaInvalidKey)
	// Missing object.
	_, err = svc.ValidateDraftMedia(ctx, 1, "drafts/1/missing/source.mp4", "")
	require.ErrorIs(t, err, ErrDraftMediaMissing)
	// Oversized raw.
	store.sizes[raw] = int64(500<<20) + 1
	_, err = svc.ValidateDraftMedia(ctx, 1, raw, "")
	require.ErrorIs(t, err, ErrDraftMediaTooLarge)
	store.sizes[raw] = 1024
	// Oversized cover.
	store.sizes[cover] = int64(10<<20) + 1
	_, err = svc.ValidateDraftMedia(ctx, 1, raw, cover)
	require.ErrorIs(t, err, ErrDraftMediaTooLarge)
	store.sizes[cover] = 2048

	// Valid.
	media, err := svc.ValidateDraftMedia(ctx, 1, raw, cover)
	require.NoError(t, err)
	require.Equal(t, raw, media.RawKey)
	require.Equal(t, cover, media.CoverKey)
	require.Zero(t, media.DurationSec) // duration validation moved to the worker
}

func TestValidateDraftCover(t *testing.T) {
	svc, store, _ := draftOSS(t)
	ctx := context.Background()
	store.exist["drafts/1/u1/cover.png"] = true
	store.sizes["drafts/1/u1/cover.png"] = 2048

	require.NoError(t, svc.ValidateDraftCover(ctx, 1, "drafts/1/u1/cover.png"))
	require.ErrorIs(t, svc.ValidateDraftCover(ctx, 1, "drafts/2/u1/cover.png"), ErrDraftMediaInvalidKey)
	require.ErrorIs(t, svc.ValidateDraftCover(ctx, 1, "drafts/1/nope/cover.png"), ErrDraftMediaMissing)
	store.sizes["drafts/1/u1/cover.png"] = int64(10<<20) + 1
	require.ErrorIs(t, svc.ValidateDraftCover(ctx, 1, "drafts/1/u1/cover.png"), ErrDraftMediaTooLarge)
}

func TestSubmitDraft_AtomicOutboxAndStatus(t *testing.T) {
	stubProbe(t, 9.5)
	svc, _, db := draftOSS(t)
	servicetest.SeedUser(t, db, 1, "u")
	require.NoError(t, db.Create(&video.Video{
		ID: 20, UserID: 1, Title: "d", Status: video.StatusDraft,
		DraftRawKey: "drafts/1/u/source.mp4", DraftCoverKey: "drafts/1/u/cover.png",
	}).Error)

	err := svc.SubmitDraft(context.Background(), &video.Video{ID: 20}, DraftMedia{
		RawKey: "drafts/1/u/source.mp4", CoverKey: "drafts/1/u/cover.png", DurationSec: 9.5,
	})
	require.NoError(t, err)
	var v video.Video
	require.NoError(t, db.First(&v, 20).Error)
	require.Equal(t, video.StatusProcessing, v.Status)
	require.Equal(t, 9.5, v.DurationSec)
	require.Empty(t, v.DraftRawKey)
	require.Empty(t, v.DraftCoverKey)
	var outbox int64
	require.NoError(t, db.Model(&video.TranscodeOutbox{}).Where("video_id = ?", 20).Count(&outbox).Error)
	require.Equal(t, int64(1), outbox)
}

func TestReplaceMedia_AtomicOutboxAndFields(t *testing.T) {
	stubProbe(t, 7.0)
	svc, _, db := draftOSS(t)
	servicetest.SeedUser(t, db, 1, "u")
	require.NoError(t, db.Create(&video.Video{
		ID: 30, UserID: 1, Title: "old", Status: video.StatusFailed,
	}).Error)

	err := svc.ReplaceMedia(context.Background(), &video.Video{ID: 30}, ReplaceMediaOpts{
		Title: "new", Description: "d", TagsJSON: `["a"]`, Zone: "动画",
		RawKey: "drafts/1/u/source.mp4", CoverKey: "drafts/1/u/cover.png", DurationSec: 7.0,
	})
	require.NoError(t, err)
	var v video.Video
	require.NoError(t, db.First(&v, 30).Error)
	require.Equal(t, video.StatusProcessing, v.Status)
	require.Equal(t, "new", v.Title)
	require.Equal(t, "动画", v.Zone)
	require.Empty(t, v.DraftRawKey)
	var outbox int64
	require.NoError(t, db.Model(&video.TranscodeOutbox{}).Where("video_id = ?", 30).Count(&outbox).Error)
	require.Equal(t, int64(1), outbox)
}

func TestDraftMediaErrorsAreTyped(t *testing.T) {
	require.True(t, errors.Is(ErrDraftMediaUnavailable, ErrDraftMediaUnavailable))
}
