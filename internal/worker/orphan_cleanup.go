package worker

import (
	"cakecake/internal/model/video"
	"cakecake/internal/queue"
	"cakecake/internal/storage"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

const (
	orphanCleanupDefaultInterval  = time.Hour
	orphanCleanupDefaultRetention = 24 * time.Hour
)

// orphanObjectStore is the minimal object-storage surface for the cleanup
// task: list staged uploads and delete unlinked ones.
type orphanObjectStore interface {
	ListObjects(prefix string, maxKeys int) ([]storage.ObjectMeta, error)
	DeleteObject(objectKey string) error
}

// StartOrphanObjectCleanup periodically deletes direct-upload/draft objects
// that are older than the retention window and referenced by no DB row
// (drafts, claims, outbox jobs or dead-letter payloads). The grace period
// keeps "uploaded but not yet submitted" files safe; in-flight uploads are
// protected by their recent LastModified.
func StartOrphanObjectCleanup(ctx context.Context, db *gorm.DB, oss *storage.OSS, retention, interval time.Duration, lg *zap.Logger) {
	if db == nil || oss == nil {
		return
	}
	if retention <= 0 {
		retention = orphanCleanupDefaultRetention
	}
	if interval <= 0 {
		interval = orphanCleanupDefaultInterval
	}
	if err := runOrphanObjectCleanupOnce(ctx, db, oss, retention, lg); err != nil {
		lg.Warn("orphan object cleanup", zap.Error(err))
	}
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if err := runOrphanObjectCleanupOnce(ctx, db, oss, retention, lg); err != nil {
				lg.Warn("orphan object cleanup", zap.Error(err))
			}
		}
	}
}

// runOrphanObjectCleanupOnce scans uploads/ and drafts/ and deletes objects
// older than the retention cutoff that no DB record references.
func runOrphanObjectCleanupOnce(ctx context.Context, db *gorm.DB, oss orphanObjectStore, retention time.Duration, lg *zap.Logger) error {
	cutoff := time.Now().Add(-retention)
	referenced := collectReferencedObjectKeys(ctx, db)

	scanned, deleted := 0, 0
	for _, prefix := range []string{"uploads/", "drafts/"} {
		objs, err := oss.ListObjects(prefix, 1000)
		if err != nil {
			return fmt.Errorf("list %s objects: %w", prefix, err)
		}
		for _, obj := range objs {
			scanned++
			if obj.LastModified.After(cutoff) {
				continue // still inside the grace period (incl. in-flight uploads)
			}
			if _, ok := referenced[obj.Key]; ok {
				continue // still needed by a draft/claim/outbox/dead letter
			}
			if err := oss.DeleteObject(obj.Key); err != nil {
				lg.Warn("delete orphan object", zap.String("key", obj.Key), zap.Error(err))
				continue
			}
			deleted++
			incrOrphanObjectsDeleted()
			lg.Info("deleted orphan object",
				zap.String("key", obj.Key),
				zap.Time("last_modified", obj.LastModified))
		}
	}
	lg.Info("orphan object cleanup",
		zap.Int("scanned", scanned),
		zap.Int("deleted", deleted),
		zap.Duration("retention", retention))
	return nil
}

// collectReferencedObjectKeys returns every staged object key still owned by
// the database: draft rows, direct-upload claims, outbox jobs (pending or
// sent) and dead-letter payloads (including archived ones, which are retained
// for audit).
func collectReferencedObjectKeys(ctx context.Context, db *gorm.DB) map[string]struct{} {
	refs := make(map[string]struct{})
	add := func(key string) {
		key = strings.TrimSpace(key)
		if key != "" {
			refs[key] = struct{}{}
		}
	}

	var drafts []video.Video
	if err := db.WithContext(ctx).
		Where("draft_raw_key <> '' OR draft_cover_key <> ''").
		Find(&drafts).Error; err == nil {
		for i := range drafts {
			add(drafts[i].DraftRawKey)
			add(drafts[i].DraftCoverKey)
		}
	}

	var claims []video.DirectUploadClaim
	if err := db.WithContext(ctx).Find(&claims).Error; err == nil {
		for i := range claims {
			add(claims[i].RawKey)
		}
	}

	var outboxes []video.TranscodeOutbox
	if err := db.WithContext(ctx).Find(&outboxes).Error; err == nil {
		for i := range outboxes {
			addJobPayloadKeys(outboxes[i].Payload, add)
		}
	}

	var deadLetters []video.TranscodeDeadLetter
	if err := db.WithContext(ctx).Find(&deadLetters).Error; err == nil {
		for i := range deadLetters {
			var payload struct {
				Job queue.TranscodeJob `json:"job"`
			}
			if json.Unmarshal([]byte(deadLetters[i].PayloadJSON), &payload) == nil {
				add(payload.Job.RawKey)
				add(payload.Job.CoverKey)
			}
		}
	}
	return refs
}

// addJobPayloadKeys extracts raw/cover keys from an outbox payload (the job
// JSON is stored flat) and feeds them to add.
func addJobPayloadKeys(payload string, add func(string)) {
	var job queue.TranscodeJob
	if json.Unmarshal([]byte(payload), &job) == nil {
		add(job.RawKey)
		add(job.CoverKey)
	}
}
