package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// C holds application configuration loaded from environment variables.
type C struct {
	AppEnv        string
	HTTPAddr      string
	JWTSecret     string
	MySQLDSN      string
	RedisAddr     string
	RedisPassword string
	RedisDB       int
	RedisDial     time.Duration
	RedisRead     time.Duration
	RedisWrite    time.Duration
	RedisPoolSize int

	RabbitMQURL string

	// ElasticsearchURL empty disables search (optional, like OSS).
	ElasticsearchURL      string
	ElasticsearchUsername string
	ElasticsearchPassword string

	OSSEndpoint        string
	OSSAccessKeyID     string
	OSSAccessKeySecret string
	OSSBucket          string
	// OSSPublicURLPrefix optional full prefix without trailing slash, e.g. https://bucket.oss-cn-beijing.aliyuncs.com
	OSSPublicURLPrefix string

	SensitiveWordsFile string
	TempUploadDir      string
	// FFprobePath / FFmpegPath：可执行文件绝对路径或 PATH 中的名称；空则默认 ffprobe / ffmpeg（进程环境 PATH 需能找到）。
	FFprobePath string
	FFmpegPath  string

	// IP2RegionV4XDB optional ip2region IPv4 database for comment IP location.
	IP2RegionV4XDB string
	// IP2RegionDevClientIP optional public IP used for comment location when APP_ENV=development
	// and the real client is loopback/private (typical Vite → :8080 local proxy).
	IP2RegionDevClientIP string

	// AdminSeedUsername / AdminSeedPassword: create first admin when admins table is empty (optional).
	AdminSeedUsername string
	AdminSeedPassword string

	// VideoReviewRequired: transcode success → pending_review instead of published (default true).
	VideoReviewRequired bool
	// ArticleReviewRequired: column publish → pending_review instead of published (default true).
	ArticleReviewRequired bool
	// VideoUploadDisabled: reject video file upload/transcode; metadata-only drafts still allowed.
	VideoUploadDisabled bool

	// DeepSeek / AI assistant (optional; empty API key disables replies).
	DeepSeekAPIKey     string
	DeepSeekBaseURL    string
	DeepSeekModel      string
	AgentBotUsername   string
	AgentEnabled       bool
	AgentMaxHistory    int
	AgentHistoryTTL    time.Duration
	AgentDailyQuota    int
	AgentRequestTimeout time.Duration

	// RateLimitEnabled enables global IP-based token bucket rate limiter.
	// Use RATE_LIMIT_ENABLED=1 to turn on (default off).
	RateLimitEnabled bool
	// RateLimitRate tokens refilled per second.
	RateLimitRate float64
	// RateLimitBurst max token capacity (burst allowance).
	RateLimitBurst int
}

func getenv(key, def string) string {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	return v
}

func mustParseDuration(s string, def time.Duration) time.Duration {
	if s == "" {
		return def
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return def
	}
	return d
}

func atoi(s string, def int) int {
	if s == "" {
		return def
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return n
}

func parseFloatEnv(key string, def float64) float64 {
	s := strings.TrimSpace(os.Getenv(key))
	if s == "" {
		return def
	}
	n, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return def
	}
	return n
}

func parseBoolEnv(key string, def bool) bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	if v == "" {
		return def
	}
	switch v {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return def
	}
}

// Load reads configuration from environment variables.
func Load() *C {
	return &C{
		AppEnv:        getenv("APP_ENV", "development"),
		HTTPAddr:      getenv("HTTP_ADDR", ":8080"),
		JWTSecret:     os.Getenv("JWT_SECRET"),
		MySQLDSN:      os.Getenv("MYSQL_DSN"),
		RedisAddr:     getenv("REDIS_ADDR", "127.0.0.1:6379"),
		RedisPassword: os.Getenv("REDIS_PASSWORD"),
		RedisDB:       atoi(os.Getenv("REDIS_DB"), 0),
		RedisDial:     mustParseDuration(os.Getenv("REDIS_DIAL_TIMEOUT"), 5*time.Second),
		RedisRead:     mustParseDuration(os.Getenv("REDIS_READ_TIMEOUT"), 3*time.Second),
		RedisWrite:    mustParseDuration(os.Getenv("REDIS_WRITE_TIMEOUT"), 3*time.Second),
		RedisPoolSize: atoi(os.Getenv("REDIS_POOL_SIZE"), 20),

		RabbitMQURL: getenv("RABBITMQ_URL", "amqp://guest:guest@127.0.0.1:5672/"),

		ElasticsearchURL:      strings.TrimSpace(os.Getenv("ELASTICSEARCH_URL")),
		ElasticsearchUsername: strings.TrimSpace(os.Getenv("ELASTICSEARCH_USERNAME")),
		ElasticsearchPassword: os.Getenv("ELASTICSEARCH_PASSWORD"),

		OSSAccessKeyID:     os.Getenv("OSS_ACCESS_KEY_ID"),
		OSSAccessKeySecret: os.Getenv("OSS_ACCESS_KEY_SECRET"),
		OSSBucket:          os.Getenv("OSS_BUCKET"),
		OSSEndpoint: normalizeAliyunOSSEndpoint(
			os.Getenv("OSS_ENDPOINT"),
			os.Getenv("OSS_BUCKET"),
		),
		OSSPublicURLPrefix: os.Getenv("OSS_PUBLIC_URL_PREFIX"),

		SensitiveWordsFile: getenv("SENSITIVE_WORDS_FILE", "./configs/sensitive_words.txt"),
		TempUploadDir:      getenv("TEMP_UPLOAD_DIR", "./data/tmp"),
		FFprobePath:        strings.TrimSpace(os.Getenv("FFPROBE_PATH")),
		FFmpegPath:         strings.TrimSpace(os.Getenv("FFMPEG_PATH")),
		IP2RegionV4XDB:        getenv("IP2REGION_V4_XDB", "./configs/ip2region_v4.xdb"),
		IP2RegionDevClientIP:    strings.TrimSpace(os.Getenv("IP2REGION_DEV_CLIENT_IP")),
		AdminSeedUsername:       strings.TrimSpace(os.Getenv("ADMIN_SEED_USERNAME")),
		AdminSeedPassword:       os.Getenv("ADMIN_SEED_PASSWORD"),
		VideoReviewRequired:     parseBoolEnv("VIDEO_REVIEW_REQUIRED", true),
		ArticleReviewRequired:   parseBoolEnv("ARTICLE_REVIEW_REQUIRED", true),
		VideoUploadDisabled:     parseBoolEnv("VIDEO_UPLOAD_DISABLED", false),

		DeepSeekAPIKey:  strings.TrimSpace(os.Getenv("DEEPSEEK_API_KEY")),
		DeepSeekBaseURL: strings.TrimRight(strings.TrimSpace(getenv("DEEPSEEK_BASE_URL", "https://api.deepseek.com")), "/"),
		DeepSeekModel:   getenv("DEEPSEEK_MODEL", "deepseek-chat"),
		AgentBotUsername: getenv("AGENT_BOT_USERNAME", "minibili_ai"),
		AgentEnabled: parseBoolEnv("AGENT_ENABLED", strings.TrimSpace(os.Getenv("DEEPSEEK_API_KEY")) != ""),
		AgentMaxHistory:    atoi(os.Getenv("AGENT_MAX_HISTORY"), 20),
		AgentHistoryTTL:    mustParseDuration(os.Getenv("AGENT_HISTORY_TTL"), 30*24*time.Hour),
		AgentDailyQuota:    atoi(os.Getenv("AGENT_DAILY_QUOTA"), 80),
		RateLimitEnabled: parseBoolEnv("RATE_LIMIT_ENABLED", false),
		RateLimitRate:    parseFloatEnv("RATE_LIMIT_RATE", 20),
		RateLimitBurst:    atoi(os.Getenv("RATE_LIMIT_BURST"), 50),
		AgentRequestTimeout: mustParseDuration(os.Getenv("AGENT_REQUEST_TIMEOUT"), 90*time.Second),
	}
}

// OSSObjectURL builds the browser-accessible URL for an object key.
func (c *C) OSSObjectURL(objectKey string) string {
	prefix := strings.TrimSuffix(c.OSSPublicURLPrefix, "/")
	if prefix == "" && c.OSSEndpoint != "" && c.OSSBucket != "" {
		host := strings.TrimPrefix(strings.TrimPrefix(c.OSSEndpoint, "https://"), "http://")
		prefix = fmt.Sprintf("https://%s.%s", c.OSSBucket, host)
	}
	if prefix == "" {
		return objectKey
	}
	return prefix + "/" + strings.TrimPrefix(objectKey, "/")
}

func (c *C) ossPublicURLPrefix() string {
	prefix := strings.TrimSuffix(c.OSSPublicURLPrefix, "/")
	if prefix == "" && c.OSSEndpoint != "" && c.OSSBucket != "" {
		host := strings.TrimPrefix(strings.TrimPrefix(c.OSSEndpoint, "https://"), "http://")
		prefix = fmt.Sprintf("https://%s.%s", c.OSSBucket, host)
	}
	return prefix
}

// OSSObjectKeyFromURL extracts the object key from a stored public OSS URL.
func (c *C) OSSObjectKeyFromURL(publicURL string) string {
	publicURL = strings.TrimSpace(publicURL)
	if publicURL == "" {
		return ""
	}
	prefix := c.ossPublicURLPrefix()
	if prefix != "" {
		for _, p := range []string{prefix, strings.Replace(prefix, "https://", "http://", 1)} {
			if strings.HasPrefix(publicURL, p+"/") {
				return strings.TrimPrefix(publicURL[len(p):], "/")
			}
		}
	}
	u, err := url.Parse(publicURL)
	if err != nil {
		return ""
	}
	return strings.TrimPrefix(u.EscapedPath(), "/")
}
