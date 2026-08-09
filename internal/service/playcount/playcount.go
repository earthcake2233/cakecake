package playcount

import (
	"cakecake/internal/model/video"
	"context"
	"strconv"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"cakecake/internal/data"
)

// PlayCounter syncs Redis hot counters with MySQL (SPEC F3).
type PlayCounter struct {
	Rdb   *redis.Client
	Store PlayCountStore
}

// PlayCountStore is the play-count flush boundary (Phase 1: *gorm.DB impl).
type PlayCountStore interface {
	AddPlayCount(ctx context.Context, videoID uint64, delta uint64) error
}

// PlayCountStoreImpl implements PlayCountStore using *gorm.DB (Phase 1 monolith).
type PlayCountStoreImpl struct {
	db *gorm.DB
}

var _ PlayCountStore = (*PlayCountStoreImpl)(nil)

// NewPlayCountStore creates a gorm-backed play-count store.
func NewPlayCountStore(db *gorm.DB) *PlayCountStoreImpl {
	return &PlayCountStoreImpl{db: db}
}

// AddPlayCount adjusts a video's play count by delta.
func (p *PlayCountStoreImpl) AddPlayCount(ctx context.Context, videoID uint64, delta uint64) error {
	return p.db.WithContext(ctx).Model(&video.Video{}).Where("id = ?", videoID).
		UpdateColumn("play_count", gorm.Expr("play_count + ?", delta)).Error
}

// Incr increments Redis delta for views and marks the video for DB flush.
func (p *PlayCounter) Incr(ctx context.Context, videoID uint64) error {
	pipe := p.Rdb.Pipeline()
	pipe.Incr(ctx, data.VideoPlayDeltaKey(videoID))
	pipe.SAdd(ctx, data.SetPlayDirty, strconv.FormatUint(videoID, 10))
	_, err := pipe.Exec(ctx)
	return err
}

// Display returns MySQL play_count plus unflushed Redis delta.
func (p *PlayCounter) Display(ctx context.Context, v *video.Video) (uint64, error) {
	key := data.VideoPlayDeltaKey(v.ID)
	d, err := p.Rdb.Get(ctx, key).Uint64()
	if err == redis.Nil {
		return v.PlayCount, nil
	}
	if err != nil {
		return v.PlayCount, err
	}
	return v.PlayCount + d, nil
}

// Flush merges Redis deltas into MySQL (every 10s job).
func (p *PlayCounter) Flush(ctx context.Context) error {
	ids, err := p.Rdb.SMembers(ctx, data.SetPlayDirty).Result()
	if err != nil {
		return err
	}
	for _, sid := range ids {
		vid, err := strconv.ParseUint(sid, 10, 64)
		if err != nil {
			continue
		}
		key := data.VideoPlayDeltaKey(vid)
		d, err := p.Rdb.Get(ctx, key).Uint64()
		if err == redis.Nil {
			_, _ = p.Rdb.SRem(ctx, data.SetPlayDirty, sid).Result()
			continue
		}
		if err != nil {
			continue
		}
		if d == 0 {
			_, _ = p.Rdb.Del(ctx, key).Result()
			_, _ = p.Rdb.SRem(ctx, data.SetPlayDirty, sid).Result()
			continue
		}
		if err := p.Store.AddPlayCount(ctx, vid, d); err != nil {
			continue
		}
		_, _ = p.Rdb.Del(ctx, key).Result()
		_, _ = p.Rdb.SRem(ctx, data.SetPlayDirty, sid).Result()
	}
	return nil
}
