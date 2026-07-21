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

type RateLimiter struct {
	rdb   *redis.Client
	rate  float64
	burst int
	sha   string
	mu    sync.Mutex
}

func NewRateLimiter(rdb *redis.Client, rate float64, burst int) *RateLimiter {
	return &RateLimiter{
		rdb:   rdb,
		rate:  rate,
		burst: burst,
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
		if strings.EqualFold(c.GetHeader("Upgrade"), "websocket") {
			c.Next()
			return
		}
		ip := c.ClientIP()
		key := "ratelimit:" + ip
		now := time.Now().Unix()
		sha, err := rl.loadScript()
		if err != nil {
			c.Next()
			return
		}
		result, err := rl.rdb.EvalSha(context.Background(), sha, []string{key}, rl.rate, rl.burst, now, 1).Result()
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
