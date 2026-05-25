package data

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"minibili/internal/config"
)

// NewRedis creates a Redis client with explicit timeouts (Rule R-DEV-5).
func NewRedis(cfg *config.C) (*redis.Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:         cfg.RedisAddr,
		Password:     cfg.RedisPassword,
		DB:           cfg.RedisDB,
		DialTimeout:  cfg.RedisDial,
		ReadTimeout:  cfg.RedisRead,
		WriteTimeout: cfg.RedisWrite,
		PoolSize:     cfg.RedisPoolSize,
	})
	ctx, cancel := context.WithTimeout(context.Background(), cfg.RedisDial)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis ping: %w", err)
	}
	return rdb, nil
}

// RedisKey helpers
const (
	PrefixDanmakuCooldown = "danmaku:cooldown:"
	PrefixRefreshInvalid      = "refresh_token:invalid:"
	PrefixAdminRefreshInvalid = "refresh_token:admin:invalid:"
	// PrefixVideoPlayDelta stores incremental views since last flush to MySQL.
	PrefixVideoPlayDelta = "videodelta:"
	SetPlayDirty         = "playcount:dirty"
	// ChannelDanmakuFanout is Redis Pub/Sub for cross-process danmaku room fan-out (SPEC NF-3).
	ChannelDanmakuFanout = "minibili:danmaku:fanout"
)

func DanmakuCooldownKey(userID, videoID uint64) string {
	return fmt.Sprintf("%s%d:%d", PrefixDanmakuCooldown, userID, videoID)
}

func RefreshInvalidKey(tokenID string) string {
	return PrefixRefreshInvalid + tokenID
}

func AdminRefreshInvalidKey(tokenID string) string {
	return PrefixAdminRefreshInvalid + tokenID
}

func VideoPlayDeltaKey(videoID uint64) string {
	return fmt.Sprintf("%s%d", PrefixVideoPlayDelta, videoID)
}

// RefreshInvalidTTL matches refresh token lifetime (3 days).
var RefreshInvalidTTL = 72 * time.Hour
