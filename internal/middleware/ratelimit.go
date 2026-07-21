package middleware

import (
	"fmt"
	"net/http"
	"strings"
	"context"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"minibili/internal/config"
	"minibili/internal/errcode"
	"minibili/internal/pkg/resp"
)

const tokenBucketScript = `
local key = KEYS[1]
local rate = tonumber(ARGV[1])
local burst = tonumber(ARGV[2])
local now = tonumber(ARGV[3])
local cost = tonumber(ARGV[4])
local bucket = redis.call("HMGET", key, "tokens", "last_refill")
local tokens = tonumber(bucket[1]) or burst
local last_refill = tonumber(bucket[2]) or now
-- Refill tokens based on elapsed time
local elapsed = now - last_refill
if elapsed > 0 then
    tokens = math.min(burst, tokens + elapsed * rate)
end
if tokens >= cost then
    tokens = tokens - cost
    redis.call("HSET", key, "tokens", tokens, "last_refill", now)
    redis.call("EXPIRE", key, math.ceil(burst / rate) + 1)
    return {1, tokens}
else
    redis.call("HSET", key, "tokens", tokens, "last_refill", now)
    redis.call("EXPIRE", key, math.ceil(burst / rate) + 1)
    return {0, tokens}
end
`

var tokenBucketScriptSHA string
var scriptLoadOnce sync.Once

// RateLimiter implements an IP-based token bucket rate limiter backed by Redis.
// When a RuntimeConfig is provided, rate/burst values are read dynamically per-request.
type RateLimiter struct {
	rdb          *redis.Client
	rc           *config.RuntimeConfig
	defaultRate  float64
	defaultBurst int
	sha          string
	mu           sync.Mutex
}

// NewRateLimiter creates a RateLimiter. Pass an optional RuntimeConfig for dynamic tuning.
func NewRateLimiter(rdb *redis.Client, rc *config.RuntimeConfig, defaultRate float64, defaultBurst int) *RateLimiter {
	return &RateLimiter{
		rdb:          rdb,
		rc:           rc,
		defaultRate:  defaultRate,
		defaultBurst: defaultBurst,
	}
}

func (rl *RateLimiter) loadScript() (string, error) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	if rl.sha != "" {
		return rl.sha, nil
	}
	sha, err := rl.rdb.ScriptLoad(context.Background(), tokenBucketScript).Result()
	if err != nil {
		return "", err
	}
	rl.sha = sha
	return sha, nil
}

func (rl *RateLimiter) RateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check dynamic enabled flag
		if rl.rc != nil && !rl.rc.GetBool("rate_limit_enabled", rl.defaultRate > 0) {
			c.Next()
			return
		}
		if strings.EqualFold(c.GetHeader("Upgrade"), "websocket") {
			c.Next()
			return
		}

		// Read dynamic rate/burst from RuntimeConfig
		rate := rl.defaultRate
		burst := rl.defaultBurst
		if rl.rc != nil {
			rate = rl.rc.GetFloat("rate_limit_rate", rate)
			burst = rl.rc.GetInt("rate_limit_burst", burst)
		}

		ip := c.ClientIP()
		key := "ratelimit:" + ip
		now := time.Now().Unix()
		sha, err := rl.loadScript()
		if err != nil {
			c.Next()
			return
		}
		result, err := rl.rdb.EvalSha(context.Background(), sha, []string{key}, rate, burst, now, 1).Result()
		if err != nil {
			c.Next()
			return
		}
		vals, ok := result.([]interface{})
		if !ok || len(vals) < 2 {
			c.Next()
			return
		}
		allowed, _ := vals[0].(int64)
		remaining, _ := vals[1].(float64)
		c.Header("X-RateLimit-Remaining", fmt.Sprintf("%.0f", remaining))
		if allowed == 0 {
			c.Header("Retry-After", "1")
			resp.Err(c, http.StatusTooManyRequests, errcode.CodeTooManyRequests)
			c.Abort()
			return
		}
		c.Next()
	}
}
