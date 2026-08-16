package hotsearch

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// mergedDetailCacheTTL bounds staleness of the public hot-search list.
// Ops/layout changes invalidate explicitly; this TTL is a safety net.
const mergedDetailCacheTTL = 30 * time.Second

const mergedDetailCachePrefix = "minibili:hotsearch:merged:"

func mergedDetailCacheKey(limit int) string {
	return fmt.Sprintf("%s%d", mergedDetailCachePrefix, limit)
}

// cachedMergedDetail returns cached merged detail when present. Any read
// error (including redis.Nil) degrades to the DB path.
func cachedMergedDetail(ctx context.Context, rdb *redis.Client, limit int) ([]HotSearchMergedDetail, bool) {
	if rdb == nil {
		return nil, false
	}
	raw, err := rdb.Get(ctx, mergedDetailCacheKey(limit)).Bytes()
	if err != nil {
		return nil, false
	}
	var items []HotSearchMergedDetail
	if err := json.Unmarshal(raw, &items); err != nil {
		return nil, false
	}
	return items, true
}

// storeMergedDetail caches merged detail; failures degrade silently.
func storeMergedDetail(ctx context.Context, rdb *redis.Client, limit int, items []HotSearchMergedDetail) {
	if rdb == nil {
		return
	}
	raw, err := json.Marshal(items)
	if err != nil {
		return
	}
	rdb.Set(ctx, mergedDetailCacheKey(limit), raw, mergedDetailCacheTTL)
}

// invalidateMergedCache drops all merged-detail cache keys for allowed limits.
// Called after any ops/layout/Redis-rank mutation.
func invalidateMergedCache(ctx context.Context, rdb *redis.Client) {
	if rdb == nil {
		return
	}
	for limit := 1; limit <= 20; limit++ {
		rdb.Del(ctx, mergedDetailCacheKey(limit))
	}
}
