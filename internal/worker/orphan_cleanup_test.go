package worker

import (
	"cakecake/internal/model/video"
	"cakecake/internal/service/servicetest"
	"cakecake/internal/storage"
	"context"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type fakeOrphanStore struct {
	objects map[string]time.Time
	deleted []string
}

func (f *fakeOrphanStore) ListObjects(prefix string, _ int) ([]storage.ObjectMeta, error) {
	var out []storage.ObjectMeta
	for key, lm := range f.objects {
		if strings.HasPrefix(key, prefix) {
			out = append(out, storage.ObjectMeta{Key: key, LastModified: lm})
		}
	}
	return out, nil
}

func (f *fakeOrphanStore) DeleteObject(key string) error {
	f.deleted = append(f.deleted, key)
	return nil
}

func TestOrphanObjectCleanup_DeletesOnlyOldUnreferenced(t *testing.T) {
	db := servicetest.NewDB(t)
	now := time.Now()
	old := now.Add(-48 * time.Hour)
	recent := now.Add(-time.Hour)

	store := &fakeOrphanStore{objects: map[string]time.Time{
		// Referenced by DB rows -> must survive.
		"uploads/1/claim/source.mp4":      old,
		"uploads/1/outbox/source.mp4":     old,
		"uploads/1/deadletter/source.mp4": old,
		"drafts/1/draft/source.mp4":       old,
		// Fresh -> grace period, must survive even if unreferenced.
		"uploads/1/fresh/source.mp4": recent,
		// Old + unreferenced -> deleted.
		"uploads/1/orphan/source.mp4": old,
		"drafts/1/orphan/cover.jpg":   old,
	}}

	require.NoError(t, db.Create(&video.Video{
		UserID: 1, Title: "d", Status: video.StatusDraft,
		DraftRawKey: "drafts/1/draft/source.mp4",
	}).Error)
	require.NoError(t, db.Create(&video.DirectUploadClaim{RawKey: "uploads/1/claim/source.mp4", UserID: 1}).Error)
	require.NoError(t, db.Create(&video.TranscodeOutbox{
		JobID:   "j1",
		VideoID: 10,
		Payload: `{"video_id":10,"raw_key":"uploads/1/outbox/source.mp4"}`,
		Status:  video.OutboxStatusPending,
	}).Error)
	require.NoError(t, db.Create(&video.TranscodeDeadLetter{
		VideoID:     11,
		PayloadJSON: `{"job":{"video_id":11,"raw_key":"uploads/1/deadletter/source.mp4"},"reason":"x"}`,
	}).Error)

	err := runOrphanObjectCleanupOnce(context.Background(), db, store, 24*time.Hour, zap.NewNop())
	require.NoError(t, err)
	sort.Strings(store.deleted)
	require.Equal(t, []string{"drafts/1/orphan/cover.jpg", "uploads/1/orphan/source.mp4"}, store.deleted)

	// Protected and fresh keys must never be deleted.
	for _, key := range []string{
		"uploads/1/claim/source.mp4",
		"uploads/1/outbox/source.mp4",
		"uploads/1/deadletter/source.mp4",
		"drafts/1/draft/source.mp4",
		"uploads/1/fresh/source.mp4",
	} {
		require.NotContains(t, store.deleted, key)
	}
}

func TestCollectReferencedObjectKeys_ParsesBothPayloadShapes(t *testing.T) {
	db := servicetest.NewDB(t)
	require.NoError(t, db.Create(&video.Video{
		UserID: 1, Title: "d", Status: video.StatusDraft,
		DraftRawKey: "drafts/1/d/source.mp4", DraftCoverKey: "drafts/1/d/cover.jpg",
	}).Error)
	require.NoError(t, db.Create(&video.TranscodeOutbox{
		JobID: "j", VideoID: 1,
		Payload: `{"video_id":1,"raw_key":"uploads/1/o/source.mp4","cover_key":"uploads/1/o/cover.jpg"}`,
		Status:  video.OutboxStatusPending,
	}).Error)
	require.NoError(t, db.Create(&video.TranscodeDeadLetter{
		VideoID:     1,
		PayloadJSON: `{"job":{"video_id":1,"raw_key":"uploads/1/dl/source.mp4","cover_key":"uploads/1/dl/cover.jpg"},"reason":"x"}`,
	}).Error)

	refs := collectReferencedObjectKeys(context.Background(), db)
	for _, key := range []string{
		"drafts/1/d/source.mp4",
		"drafts/1/d/cover.jpg",
		"uploads/1/o/source.mp4",
		"uploads/1/o/cover.jpg",
		"uploads/1/dl/source.mp4",
		"uploads/1/dl/cover.jpg",
	} {
		_, ok := refs[key]
		require.True(t, ok, key)
	}
}
