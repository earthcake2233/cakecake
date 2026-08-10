package config

import (
	"fmt"
	"go.uber.org/zap/zapcore"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// C holds application configuration loaded from environment variables.
type C struct {
	AppEnv string `json:"app_env"`

	HTTPAddr string `json:"http_addr"`

	JWTSecret string `json:"-"`

	MySQLDSN string `json:"-"`

	RedisAddr string `json:"redis_addr"`

	RedisPassword string `json:"-"`

	RedisDB int `json:"redis_db"`

	RedisDial time.Duration `json:"redis_dial_timeout"`

	RedisRead time.Duration `json:"redis_read_timeout"`

	RedisWrite time.Duration `json:"redis_write_timeout"`

	RedisPoolSize int `json:"redis_pool_size"`

	RabbitMQURL string `json:"-"`

	// ElasticsearchURL empty disables search (optional, like OSS).
	ElasticsearchURL string `json:"elasticsearch_url"`

	ElasticsearchUsername string `json:"-"`

	ElasticsearchPassword string `json:"-"`

	OSSEndpoint string `json:"oss_endpoint"`

	OSSAccessKeyID string `json:"-"`

	OSSAccessKeySecret string `json:"-"`

	OSSBucket string `json:"oss_bucket"`

	// OSSPublicURLPrefix optional full prefix without trailing slash, e.g. https://bucket.oss-cn-beijing.aliyuncs.com
	OSSPublicURLPrefix string `json:"oss_public_url_prefix"`

	SensitiveWordsFile string `json:"sensitive_words_file"`

	TempUploadDir string `json:"temp_upload_dir"`

	// FFprobePath / FFmpegPath: absolute path or name in PATH; empty defaults to ffprobe / ffmpeg (must be findable via process environment PATH).
	FFprobePath string `json:"ffprobe_path"`

	FFmpegPath string `json:"ffmpeg_path"`

	// IP2RegionV4XDB optional ip2region IPv4 database for comment IP location.
	IP2RegionV4XDB string `json:"ip2region_v4_xdb"`

	// IP2RegionDevClientIP optional public IP used for comment location when APP_ENV=development
	// and the real client is loopback/private (typical Vite → :8080 local proxy).
	IP2RegionDevClientIP string `json:"ip2region_dev_client_ip"`

	// AdminSeedUsername / AdminSeedPassword: create first admin when admins table is empty (optional).
	AdminSeedUsername string `json:"admin_seed_username"`

	AdminSeedPassword string `json:"-"`

	// DemoUserPassword: login password for seeded demo users (SEED_DEMO_DATA=true).
	DemoUserPassword string `json:"-"`

	// VideoReviewRequired: transcode success → pending_review instead of published (default true).
	VideoReviewRequired bool `json:"video_review_required"`

	// ArticleReviewRequired: column publish → pending_review instead of published (default true).
	ArticleReviewRequired bool `json:"article_review_required"`

	// VideoUploadDisabled: reject video file upload/transcode; metadata-only drafts still allowed.
	VideoUploadDisabled bool `json:"video_upload_disabled"`

	// SeedDemoData: when true and the videos table is empty, insert demo users/videos/danmaku
	// pointing at public demo media URLs (Docker Compose one-command experience).
	SeedDemoData bool `json:"seed_demo_data"`

	// DBAutoMigrate enables automatic schema migration on startup (default true).
	// Set DB_AUTO_MIGRATE=0 in production when using goose or another migration tool.
	DBAutoMigrate bool `json:"db_auto_migrate"`

	// DeepSeek / AI assistant (optional; empty API key disables replies).
	DeepSeekAPIKey string `json:"-"`

	DeepSeekBaseURL string `json:"deepseek_base_url"`

	DeepSeekModel string `json:"deepseek_model"`

	AgentBotUsername string `json:"agent_bot_username"`

	AgentEnabled bool `json:"agent_enabled"`

	AgentMaxHistory int `json:"agent_max_history"`

	AgentHistoryTTL time.Duration `json:"agent_history_ttl"`

	AgentDailyQuota int `json:"agent_daily_quota"`

	AgentRequestTimeout time.Duration `json:"agent_request_timeout"`

	// ShutdownTimeout max wait for background tasks during graceful shutdown (default 30s).
	ShutdownTimeout time.Duration `json:"shutdown_timeout"`

	// RateLimitEnabled enables global IP-based token bucket rate limiter.
	// Use RATE_LIMIT_ENABLED=1 to turn on (default off).
	RateLimitEnabled bool `json:"rate_limit_enabled"`

	// RateLimitRate tokens refilled per second.
	RateLimitRate float64 `json:"rate_limit_rate"`

	// RateLimitBurst max token capacity (burst allowance).
	RateLimitBurst int `json:"rate_limit_burst"`

	// MetricsToken optional bearer token guarding GET /metrics. Empty disables
	// auth (local development); set a strong random value in production.
	MetricsToken string `json:"-"`
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
	appEnv := getenv("APP_ENV", "development")
	return &C{
		AppEnv:        appEnv,
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

		SensitiveWordsFile:    getenv("SENSITIVE_WORDS_FILE", "./configs/sensitive_words.txt"),
		TempUploadDir:         getenv("TEMP_UPLOAD_DIR", "./data/tmp"),
		FFprobePath:           strings.TrimSpace(os.Getenv("FFPROBE_PATH")),
		FFmpegPath:            strings.TrimSpace(os.Getenv("FFMPEG_PATH")),
		IP2RegionV4XDB:        getenv("IP2REGION_V4_XDB", "./configs/ip2region_v4.xdb"),
		IP2RegionDevClientIP:  strings.TrimSpace(os.Getenv("IP2REGION_DEV_CLIENT_IP")),
		AdminSeedUsername:     strings.TrimSpace(os.Getenv("ADMIN_SEED_USERNAME")),
		AdminSeedPassword:     os.Getenv("ADMIN_SEED_PASSWORD"),
		DemoUserPassword:      strings.TrimSpace(getenv("DEMO_USER_PASSWORD", "demo123456")),
		VideoReviewRequired:   parseBoolEnv("VIDEO_REVIEW_REQUIRED", true),
		ArticleReviewRequired: parseBoolEnv("ARTICLE_REVIEW_REQUIRED", true),
		VideoUploadDisabled:   parseBoolEnv("VIDEO_UPLOAD_DISABLED", false),
		SeedDemoData:          parseBoolEnv("SEED_DEMO_DATA", false),
		DBAutoMigrate:         parseBoolEnv("DB_AUTO_MIGRATE", appEnv != "production"),

		DeepSeekAPIKey:      strings.TrimSpace(os.Getenv("DEEPSEEK_API_KEY")),
		DeepSeekBaseURL:     strings.TrimRight(strings.TrimSpace(getenv("DEEPSEEK_BASE_URL", "https://api.deepseek.com")), "/"),
		DeepSeekModel:       getenv("DEEPSEEK_MODEL", "deepseek-v4-flash"),
		AgentBotUsername:    getenv("AGENT_BOT_USERNAME", "minibili_ai"),
		AgentEnabled:        parseBoolEnv("AGENT_ENABLED", strings.TrimSpace(os.Getenv("DEEPSEEK_API_KEY")) != ""),
		AgentMaxHistory:     atoi(os.Getenv("AGENT_MAX_HISTORY"), 20),
		AgentHistoryTTL:     mustParseDuration(os.Getenv("AGENT_HISTORY_TTL"), 30*24*time.Hour),
		AgentDailyQuota:     atoi(os.Getenv("AGENT_DAILY_QUOTA"), 80),
		RateLimitEnabled:    parseBoolEnv("RATE_LIMIT_ENABLED", false),
		RateLimitRate:       parseFloatEnv("RATE_LIMIT_RATE", 20),
		RateLimitBurst:      atoi(os.Getenv("RATE_LIMIT_BURST"), 50),
		MetricsToken:        strings.TrimSpace(os.Getenv("METRICS_TOKEN")),
		AgentRequestTimeout: mustParseDuration(os.Getenv("AGENT_REQUEST_TIMEOUT"), 90*time.Second),
		ShutdownTimeout:     mustParseDuration(os.Getenv("SHUTDOWN_TIMEOUT"), 30*time.Second),
	}
}

// MarshalLogObject implements zapcore.ObjectMarshaler for safe zap.Any logging.
func (c *C) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddString("app_env", c.AppEnv)
	enc.AddString("http_addr", c.HTTPAddr)
	enc.AddString("redis_addr", c.RedisAddr)
	enc.AddInt("redis_db", c.RedisDB)
	enc.AddDuration("redis_dial_timeout", c.RedisDial)
	enc.AddDuration("redis_read_timeout", c.RedisRead)
	enc.AddDuration("redis_write_timeout", c.RedisWrite)
	enc.AddInt("redis_pool_size", c.RedisPoolSize)
	enc.AddString("elasticsearch_url", c.ElasticsearchURL)
	enc.AddString("oss_endpoint", c.OSSEndpoint)
	enc.AddString("oss_bucket", c.OSSBucket)
	enc.AddString("oss_public_url_prefix", c.OSSPublicURLPrefix)
	enc.AddString("sensitive_words_file", c.SensitiveWordsFile)
	enc.AddString("temp_upload_dir", c.TempUploadDir)
	enc.AddString("ffprobe_path", c.FFprobePath)
	enc.AddString("ffmpeg_path", c.FFmpegPath)
	enc.AddString("ip2region_v4_xdb", c.IP2RegionV4XDB)
	enc.AddString("ip2region_dev_client_ip", c.IP2RegionDevClientIP)
	enc.AddString("admin_seed_username", c.AdminSeedUsername)
	enc.AddString("demo_user_password", c.DemoUserPassword)
	enc.AddBool("video_review_required", c.VideoReviewRequired)
	enc.AddBool("article_review_required", c.ArticleReviewRequired)
	enc.AddBool("video_upload_disabled", c.VideoUploadDisabled)
	enc.AddBool("seed_demo_data", c.SeedDemoData)
	enc.AddBool("db_auto_migrate", c.DBAutoMigrate)
	enc.AddString("deepseek_base_url", c.DeepSeekBaseURL)
	enc.AddString("deepseek_model", c.DeepSeekModel)
	enc.AddString("agent_bot_username", c.AgentBotUsername)
	enc.AddBool("agent_enabled", c.AgentEnabled)
	enc.AddInt("agent_max_history", c.AgentMaxHistory)
	enc.AddDuration("agent_history_ttl", c.AgentHistoryTTL)
	enc.AddInt("agent_daily_quota", c.AgentDailyQuota)
	enc.AddDuration("agent_request_timeout", c.AgentRequestTimeout)
	enc.AddDuration("shutdown_timeout", c.ShutdownTimeout)
	enc.AddBool("rate_limit_enabled", c.RateLimitEnabled)
	enc.AddFloat64("rate_limit_rate", c.RateLimitRate)
	enc.AddInt("rate_limit_burst", c.RateLimitBurst)
	return nil
}

// OSSObjectURL builds the public URL for an OSS object key.
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
