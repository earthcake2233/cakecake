package service

import (
	"context"
	"strconv"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"minibili/internal/data"
	"minibili/internal/model"
)

// PlayCounter syncs Redis hot counters with MySQL (SPEC F3).
type PlayCounter struct {
	Rdb *redis.Client
	DB  *gorm.DB
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
func (p *PlayCounter) Display(ctx context.Context, v *model.Video) (uint64, error) {
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
		if err := p.DB.WithContext(ctx).Model(&model.Video{}).Where("id = ?", vid).
			UpdateColumn("play_count", gorm.Expr("play_count + ?", d)).Error; err != nil {
			continue
		}
		_, _ = p.Rdb.Del(ctx, key).Result()
		_, _ = p.Rdb.SRem(ctx, data.SetPlayDirty, sid).Result()
	}
	return nil
}
