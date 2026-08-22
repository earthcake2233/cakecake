package search

import (
	"context"
	"testing"
	"time"

	searchmodel "cakecake/internal/model/search"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func newSyncDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&searchmodel.SearchSyncJob{}))
	return db
}

type fakeExec struct {
	fail  bool
	calls []string
}

func (f *fakeExec) Exec(_ context.Context, entityType string, entityID uint64, action string) error {
	f.calls = append(f.calls, entityType+":"+action)
	if f.fail {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func TestEnqueueSyncJob_DedupeReset(t *testing.T) {
	db := newSyncDB(t)
	ctx := context.Background()
	require.NoError(t, EnqueueSyncJob(ctx, db, SyncJobEntityVideo, 1, SyncJobActionUpsert))
	require.NoError(t, EnqueueSyncJob(ctx, db, SyncJobEntityVideo, 1, SyncJobActionUpsert))

	var rows []searchmodel.SearchSyncJob
	require.NoError(t, db.Find(&rows).Error)
	require.Len(t, rows, 1)

	// Different action is a separate job.
	require.NoError(t, EnqueueSyncJob(ctx, db, SyncJobEntityVideo, 1, SyncJobActionDelete))
	require.NoError(t, db.Find(&rows).Error)
	require.Len(t, rows, 2)
}

func TestProcessDueSyncJobs_Success(t *testing.T) {
	db := newSyncDB(t)
	ctx := context.Background()
	require.NoError(t, EnqueueSyncJob(ctx, db, SyncJobEntityArticle, 7, SyncJobActionUpsert))

	exec := &fakeExec{}
	n, err := ProcessDueSyncJobs(ctx, db, exec, 10, time.Now(), DefaultSyncBackoff)
	require.NoError(t, err)
	require.Equal(t, 1, n)
	require.Equal(t, []string{"article:upsert"}, exec.calls)

	var row searchmodel.SearchSyncJob
	require.NoError(t, db.First(&row).Error)
	require.Equal(t, SyncJobStatusDone, row.Status)
}

func TestProcessDueSyncJobs_FailureBackoff(t *testing.T) {
	db := newSyncDB(t)
	ctx := context.Background()
	require.NoError(t, EnqueueSyncJob(ctx, db, SyncJobEntityVideo, 3, SyncJobActionUpsert))

	now := time.Now()
	exec := &fakeExec{fail: true}
	n, err := ProcessDueSyncJobs(ctx, db, exec, 10, now, DefaultSyncBackoff)
	require.NoError(t, err)
	require.Equal(t, 1, n)

	var row searchmodel.SearchSyncJob
	require.NoError(t, db.First(&row).Error)
	require.Equal(t, SyncJobStatusPending, row.Status)
	require.Equal(t, 1, row.Attempts)
	require.NotNil(t, row.NextRetryAt)
	require.True(t, row.NextRetryAt.After(now))

	// Not due yet: second pass skips the row.
	exec2 := &fakeExec{}
	_, err = ProcessDueSyncJobs(ctx, db, exec2, 10, now.Add(2*time.Second), DefaultSyncBackoff)
	require.NoError(t, err)
	require.Empty(t, exec2.calls)
}

func TestProcessDueSyncJobs_OnlyDueRows(t *testing.T) {
	db := newSyncDB(t)
	ctx := context.Background()
	require.NoError(t, EnqueueSyncJob(ctx, db, SyncJobEntityUser, 9, SyncJobActionUpsert))
	future := time.Now().Add(time.Hour)
	require.NoError(t, db.Model(&searchmodel.SearchSyncJob{}).
		Where("entity_type = ?", SyncJobEntityUser).Update("next_retry_at", future).Error)

	exec := &fakeExec{}
	n, err := ProcessDueSyncJobs(ctx, db, exec, 10, time.Now(), DefaultSyncBackoff)
	require.NoError(t, err)
	require.Equal(t, 0, n)
	require.Empty(t, exec.calls)
}
