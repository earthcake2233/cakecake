package config

import (
	"os"
	"testing"
	"time"
)

// --- Helper functions ---

func TestGetenv(t *testing.T) {
	const key = "TEST_GETENV_KEY"
	const val = "hello"

	os.Unsetenv(key)
	if got := getenv(key, "default"); got != "default" {
		t.Errorf("getenv(%q, default) = %q; want default", key, got)
	}

	os.Setenv(key, val)
	defer os.Unsetenv(key)
	if got := getenv(key, "default"); got != val {
		t.Errorf("getenv(%q, default) = %q; want %q", key, got, val)
	}
}

func TestMustParseDuration(t *testing.T) {
	if got := mustParseDuration("", 5*time.Second); got != 5*time.Second {
		t.Errorf("mustParseDuration(empty, 5s) = %v; want 5s", got)
	}
	if got := mustParseDuration("invalid", 3*time.Second); got != 3*time.Second {
		t.Errorf("mustParseDuration(invalid, 3s) = %v; want 3s", got)
	}
	if got := mustParseDuration("10s", 1*time.Second); got != 10*time.Second {
		t.Errorf("mustParseDuration(10s, 1s) = %v; want 10s", got)
	}
}

func TestAtoi(t *testing.T) {
	if got := atoi("", 42); got != 42 {
		t.Errorf("atoi(empty, 42) = %d; want 42", got)
	}
	if got := atoi("abc", 99); got != 99 {
		t.Errorf("atoi(abc, 99) = %d; want 99", got)
	}
	if got := atoi("123", 0); got != 123 {
		t.Errorf("atoi(123, 0) = %d; want 123", got)
	}
}

func TestParseFloatEnv(t *testing.T) {
	const key = "TEST_PARSEFLOAT_KEY"

	os.Unsetenv(key)
	if got := parseFloatEnv(key, 1.5); got != 1.5 {
		t.Errorf("parseFloatEnv(unset) = %f; want 1.5", got)
	}

	os.Setenv(key, "invalid")
	defer os.Unsetenv(key)
	if got := parseFloatEnv(key, 2.5); got != 2.5 {
		t.Errorf("parseFloatEnv(invalid) = %f; want 2.5", got)
	}

	os.Setenv(key, "  42.5  ")
	if got := parseFloatEnv(key, 0); got != 42.5 {
		t.Errorf("parseFloatEnv(42.5) = %f; want 42.5", got)
	}
}

func TestParseBoolEnv(t *testing.T) {
	const key = "TEST_PARSEBOOL_KEY"

	os.Unsetenv(key)
	if got := parseBoolEnv(key, true); got != true {
		t.Errorf("parseBoolEnv(unset, true) = %v; want true", got)
	}
	if got := parseBoolEnv(key, false); got != false {
		t.Errorf("parseBoolEnv(unset, false) = %v; want false", got)
	}

	truthy := []string{"1", "true", "TRUE", "yes", "YES", "on", "ON"}
	for _, v := range truthy {
		os.Setenv(key, v)
		if got := parseBoolEnv(key, false); got != true {
			t.Errorf("parseBoolEnv(%q) = %v; want true", v, got)
		}
	}

	falsy := []string{"0", "false", "FALSE", "no", "NO", "off", "OFF"}
	for _, v := range falsy {
		os.Setenv(key, v)
		if got := parseBoolEnv(key, true); got != false {
			t.Errorf("parseBoolEnv(%q) = %v; want false", v, got)
		}
	}

	os.Setenv(key, "maybe")
	if got := parseBoolEnv(key, true); got != true {
		t.Errorf("parseBoolEnv(maybe, true) = %v; want true", got)
	}
	os.Unsetenv(key)
}

// --- Load() ---

func TestLoad_Defaults(t *testing.T) {
	cfg := Load()

	if cfg.AppEnv != "development" {
		t.Errorf("AppEnv = %q; want development", cfg.AppEnv)
	}
	if cfg.HTTPAddr != ":8080" {
		t.Errorf("HTTPAddr = %q; want :8080", cfg.HTTPAddr)
	}
	if cfg.RedisAddr != "127.0.0.1:6379" {
		t.Errorf("RedisAddr = %q; want 127.0.0.1:6379", cfg.RedisAddr)
	}
	if cfg.RedisDB != 0 {
		t.Errorf("RedisDB = %d; want 0", cfg.RedisDB)
	}
	if cfg.RedisDial != 5*time.Second {
		t.Errorf("RedisDial = %v; want 5s", cfg.RedisDial)
	}
	if cfg.RedisRead != 3*time.Second {
		t.Errorf("RedisRead = %v; want 3s", cfg.RedisRead)
	}
	if cfg.RedisWrite != 3*time.Second {
		t.Errorf("RedisWrite = %v; want 3s", cfg.RedisWrite)
	}
	if cfg.RabbitMQURL != "amqp://guest:guest@127.0.0.1:5672/" {
		t.Errorf("RabbitMQURL = %q; want amqp://...", cfg.RabbitMQURL)
	}
	if cfg.SensitiveWordsFile != "./configs/sensitive_words.txt" {
		t.Errorf("SensitiveWordsFile = %q; want ./configs/sensitive_words.txt", cfg.SensitiveWordsFile)
	}
	if cfg.TempUploadDir != "./data/tmp" {
		t.Errorf("TempUploadDir = %q; want ./data/tmp", cfg.TempUploadDir)
	}
	if cfg.IP2RegionV4XDB != "./configs/ip2region_v4.xdb" {
		t.Errorf("IP2RegionV4XDB = %q; want ./configs/ip2region_v4.xdb", cfg.IP2RegionV4XDB)
	}
	if cfg.VideoReviewRequired != true {
		t.Error("VideoReviewRequired defaults to true")
	}
	if cfg.ArticleReviewRequired != true {
		t.Error("ArticleReviewRequired defaults to true")
	}
	if cfg.VideoUploadDisabled != false {
		t.Error("VideoUploadDisabled defaults to false")
	}
	if cfg.DeepSeekModel != "deepseek-chat" {
		t.Errorf("DeepSeekModel = %q; want deepseek-chat", cfg.DeepSeekModel)
	}
	if cfg.DeepSeekBaseURL != "https://api.deepseek.com" {
		t.Errorf("DeepSeekBaseURL = %q; want https://api.deepseek.com", cfg.DeepSeekBaseURL)
	}
	if cfg.AgentBotUsername != "minibili_ai" {
		t.Errorf("AgentBotUsername = %q; want minibili_ai", cfg.AgentBotUsername)
	}
	if cfg.AgentMaxHistory != 20 {
		t.Errorf("AgentMaxHistory = %d; want 20", cfg.AgentMaxHistory)
	}
	if cfg.AgentHistoryTTL != 30*24*time.Hour {
		t.Errorf("AgentHistoryTTL = %v; want 720h", cfg.AgentHistoryTTL)
	}
	if cfg.AgentDailyQuota != 80 {
		t.Errorf("AgentDailyQuota = %d; want 80", cfg.AgentDailyQuota)
	}
	if cfg.RateLimitEnabled != false {
		t.Error("RateLimitEnabled defaults to false")
	}
	if cfg.RateLimitRate != 20.0 {
		t.Errorf("RateLimitRate = %f; want 20.0", cfg.RateLimitRate)
	}
	if cfg.RateLimitBurst != 50 {
		t.Errorf("RateLimitBurst = %d; want 50", cfg.RateLimitBurst)
	}
	if cfg.AgentRequestTimeout != 90*time.Second {
		t.Errorf("AgentRequestTimeout = %v; want 90s", cfg.AgentRequestTimeout)
	}
}

func TestLoad_WithEnv(t *testing.T) {
	setenv := func(k, v string) {
		os.Setenv(k, v)
	}

	setenv("APP_ENV", "production")
	setenv("HTTP_ADDR", ":9090")
	setenv("JWT_SECRET", "mysecret")
	setenv("MYSQL_DSN", "user:pass@tcp(localhost:3306)/db")
	setenv("REDIS_ADDR", "10.0.0.1:6379")
	setenv("REDIS_PASSWORD", "redispass")
	setenv("REDIS_DB", "2")
	setenv("REDIS_DIAL_TIMEOUT", "10s")
	setenv("REDIS_READ_TIMEOUT", "8s")
	setenv("REDIS_WRITE_TIMEOUT", "7s")
	setenv("REDIS_POOL_SIZE", "50")
	setenv("RABBITMQ_URL", "amqp://admin:admin@10.0.0.2:5672/")
	setenv("ELASTICSEARCH_URL", "https://es.example.com")
	setenv("ELASTICSEARCH_USERNAME", "esuser")
	setenv("ELASTICSEARCH_PASSWORD", "espass")
	setenv("OSS_ENDPOINT", "https://oss-cn-beijing.aliyuncs.com")
	setenv("OSS_ACCESS_KEY_ID", "akid")
	setenv("OSS_ACCESS_KEY_SECRET", "aksecret")
	setenv("OSS_BUCKET", "mybucket")
	setenv("OSS_PUBLIC_URL_PREFIX", "https://mybucket.oss-cn-beijing.aliyuncs.com")
	setenv("SENSITIVE_WORDS_FILE", "/etc/sensitive.txt")
	setenv("TEMP_UPLOAD_DIR", "/tmp/uploads")
	setenv("FFPROBE_PATH", "/usr/bin/ffprobe")
	setenv("FFMPEG_PATH", "/usr/bin/ffmpeg")
	setenv("IP2REGION_V4_XDB", "/data/ip2region.xdb")
	setenv("IP2REGION_DEV_CLIENT_IP", "1.2.3.4")
	setenv("ADMIN_SEED_USERNAME", "admin")
	setenv("ADMIN_SEED_PASSWORD", "admin123")
	setenv("VIDEO_REVIEW_REQUIRED", "false")
	setenv("ARTICLE_REVIEW_REQUIRED", "0")
	setenv("VIDEO_UPLOAD_DISABLED", "true")
	setenv("DEEPSEEK_API_KEY", "dskey")
	setenv("DEEPSEEK_BASE_URL", "https://custom.deepseek.com/")
	setenv("DEEPSEEK_MODEL", "deepseek-reasoner")
	setenv("AGENT_BOT_USERNAME", "bot")
	setenv("AGENT_ENABLED", "true")
	setenv("AGENT_MAX_HISTORY", "100")
	setenv("AGENT_HISTORY_TTL", "72h")
	setenv("AGENT_DAILY_QUOTA", "200")
	setenv("AGENT_REQUEST_TIMEOUT", "60s")
	setenv("RATE_LIMIT_ENABLED", "1")
	setenv("RATE_LIMIT_RATE", "50.5")
	setenv("RATE_LIMIT_BURST", "200")

	defer func() {
		for _, k := range []string{
			"APP_ENV", "HTTP_ADDR", "JWT_SECRET", "MYSQL_DSN",
			"REDIS_ADDR", "REDIS_PASSWORD", "REDIS_DB",
			"REDIS_DIAL_TIMEOUT", "REDIS_READ_TIMEOUT", "REDIS_WRITE_TIMEOUT", "REDIS_POOL_SIZE",
			"RABBITMQ_URL",
			"ELASTICSEARCH_URL", "ELASTICSEARCH_USERNAME", "ELASTICSEARCH_PASSWORD",
			"OSS_ENDPOINT", "OSS_ACCESS_KEY_ID", "OSS_ACCESS_KEY_SECRET", "OSS_BUCKET", "OSS_PUBLIC_URL_PREFIX",
			"SENSITIVE_WORDS_FILE", "TEMP_UPLOAD_DIR",
			"FFPROBE_PATH", "FFMPEG_PATH",
			"IP2REGION_V4_XDB", "IP2REGION_DEV_CLIENT_IP",
			"ADMIN_SEED_USERNAME", "ADMIN_SEED_PASSWORD",
			"VIDEO_REVIEW_REQUIRED", "ARTICLE_REVIEW_REQUIRED", "VIDEO_UPLOAD_DISABLED",
			"DEEPSEEK_API_KEY", "DEEPSEEK_BASE_URL", "DEEPSEEK_MODEL",
			"AGENT_BOT_USERNAME", "AGENT_ENABLED", "AGENT_MAX_HISTORY", "AGENT_HISTORY_TTL", "AGENT_DAILY_QUOTA", "AGENT_REQUEST_TIMEOUT",
			"RATE_LIMIT_ENABLED", "RATE_LIMIT_RATE", "RATE_LIMIT_BURST",
		} {
			os.Unsetenv(k)
		}
	}()

	cfg := Load()

	if cfg.AppEnv != "production" {
		t.Errorf("AppEnv = %q; want production", cfg.AppEnv)
	}
	if cfg.HTTPAddr != ":9090" {
		t.Errorf("HTTPAddr = %q; want :9090", cfg.HTTPAddr)
	}
	if cfg.JWTSecret != "mysecret" {
		t.Errorf("JWTSecret = %q; want mysecret", cfg.JWTSecret)
	}
	if cfg.MySQLDSN != "user:pass@tcp(localhost:3306)/db" {
		t.Errorf("MySQLDSN = %q; want ...", cfg.MySQLDSN)
	}
	if cfg.RedisAddr != "10.0.0.1:6379" {
		t.Errorf("RedisAddr = %q", cfg.RedisAddr)
	}
	if cfg.RedisPassword != "redispass" {
		t.Errorf("RedisPassword = %q", cfg.RedisPassword)
	}
	if cfg.RedisDB != 2 {
		t.Errorf("RedisDB = %d; want 2", cfg.RedisDB)
	}
	if cfg.RedisDial != 10*time.Second {
		t.Errorf("RedisDial = %v", cfg.RedisDial)
	}
	if cfg.RedisRead != 8*time.Second {
		t.Errorf("RedisRead = %v", cfg.RedisRead)
	}
	if cfg.RedisWrite != 7*time.Second {
		t.Errorf("RedisWrite = %v", cfg.RedisWrite)
	}
	if cfg.RedisPoolSize != 50 {
		t.Errorf("RedisPoolSize = %d; want 50", cfg.RedisPoolSize)
	}
	if cfg.RabbitMQURL != "amqp://admin:admin@10.0.0.2:5672/" {
		t.Errorf("RabbitMQURL = %q", cfg.RabbitMQURL)
	}
	if cfg.ElasticsearchURL != "https://es.example.com" {
		t.Errorf("ElasticsearchURL = %q", cfg.ElasticsearchURL)
	}
	if cfg.ElasticsearchUsername != "esuser" {
		t.Errorf("ElasticsearchUsername = %q", cfg.ElasticsearchUsername)
	}
	if cfg.ElasticsearchPassword != "espass" {
		t.Errorf("ElasticsearchPassword = %q", cfg.ElasticsearchPassword)
	}
	if cfg.OSSEndpoint != "https://oss-cn-beijing.aliyuncs.com" {
		t.Errorf("OSSEndpoint = %q", cfg.OSSEndpoint)
	}
	if cfg.OSSAccessKeyID != "akid" {
		t.Errorf("OSSAccessKeyID = %q", cfg.OSSAccessKeyID)
	}
	if cfg.OSSAccessKeySecret != "aksecret" {
		t.Errorf("OSSAccessKeySecret = %q", cfg.OSSAccessKeySecret)
	}
	if cfg.OSSBucket != "mybucket" {
		t.Errorf("OSSBucket = %q", cfg.OSSBucket)
	}
	if cfg.OSSPublicURLPrefix != "https://mybucket.oss-cn-beijing.aliyuncs.com" {
		t.Errorf("OSSPublicURLPrefix = %q", cfg.OSSPublicURLPrefix)
	}
	if cfg.SensitiveWordsFile != "/etc/sensitive.txt" {
		t.Errorf("SensitiveWordsFile = %q", cfg.SensitiveWordsFile)
	}
	if cfg.TempUploadDir != "/tmp/uploads" {
		t.Errorf("TempUploadDir = %q", cfg.TempUploadDir)
	}
	if cfg.FFprobePath != "/usr/bin/ffprobe" {
		t.Errorf("FFprobePath = %q", cfg.FFprobePath)
	}
	if cfg.FFmpegPath != "/usr/bin/ffmpeg" {
		t.Errorf("FFmpegPath = %q", cfg.FFmpegPath)
	}
	if cfg.IP2RegionV4XDB != "/data/ip2region.xdb" {
		t.Errorf("IP2RegionV4XDB = %q", cfg.IP2RegionV4XDB)
	}
	if cfg.IP2RegionDevClientIP != "1.2.3.4" {
		t.Errorf("IP2RegionDevClientIP = %q", cfg.IP2RegionDevClientIP)
	}
	if cfg.AdminSeedUsername != "admin" {
		t.Errorf("AdminSeedUsername = %q", cfg.AdminSeedUsername)
	}
	if cfg.AdminSeedPassword != "admin123" {
		t.Errorf("AdminSeedPassword = %q", cfg.AdminSeedPassword)
	}
	if cfg.VideoReviewRequired != false {
		t.Error("VideoReviewRequired should be false")
	}
	if cfg.ArticleReviewRequired != false {
		t.Error("ArticleReviewRequired should be false")
	}
	if cfg.VideoUploadDisabled != true {
		t.Error("VideoUploadDisabled should be true")
	}
	if cfg.DeepSeekAPIKey != "dskey" {
		t.Errorf("DeepSeekAPIKey = %q", cfg.DeepSeekAPIKey)
	}
	if cfg.DeepSeekBaseURL != "https://custom.deepseek.com" {
		t.Errorf("DeepSeekBaseURL = %q; want https://custom.deepseek.com", cfg.DeepSeekBaseURL)
	}
	if cfg.DeepSeekModel != "deepseek-reasoner" {
		t.Errorf("DeepSeekModel = %q", cfg.DeepSeekModel)
	}
	if cfg.AgentBotUsername != "bot" {
		t.Errorf("AgentBotUsername = %q", cfg.AgentBotUsername)
	}
	if cfg.AgentEnabled != true {
		t.Error("AgentEnabled should be true")
	}
	if cfg.AgentMaxHistory != 100 {
		t.Errorf("AgentMaxHistory = %d; want 100", cfg.AgentMaxHistory)
	}
	if cfg.AgentHistoryTTL != 72*time.Hour {
		t.Errorf("AgentHistoryTTL = %v; want 72h", cfg.AgentHistoryTTL)
	}
	if cfg.AgentDailyQuota != 200 {
		t.Errorf("AgentDailyQuota = %d; want 200", cfg.AgentDailyQuota)
	}
	if cfg.AgentRequestTimeout != 60*time.Second {
		t.Errorf("AgentRequestTimeout = %v; want 60s", cfg.AgentRequestTimeout)
	}
	if cfg.RateLimitEnabled != true {
		t.Error("RateLimitEnabled should be true")
	}
	if cfg.RateLimitRate != 50.5 {
		t.Errorf("RateLimitRate = %f; want 50.5", cfg.RateLimitRate)
	}
	if cfg.RateLimitBurst != 200 {
		t.Errorf("RateLimitBurst = %d; want 200", cfg.RateLimitBurst)
	}
}

func TestLoad_AgentEnabled_AutoDetect(t *testing.T) {
	os.Unsetenv("AGENT_ENABLED")
	os.Unsetenv("DEEPSEEK_API_KEY")
	cfg := Load()
	if cfg.AgentEnabled {
		t.Error("AgentEnabled should be false when no DEEPSEEK_API_KEY")
	}

	os.Setenv("DEEPSEEK_API_KEY", "somekey")
	defer os.Unsetenv("DEEPSEEK_API_KEY")
	cfg2 := Load()
	if !cfg2.AgentEnabled {
		t.Error("AgentEnabled should be true when DEEPSEEK_API_KEY is set")
	}

	os.Setenv("AGENT_ENABLED", "false")
	cfg3 := Load()
	if cfg3.AgentEnabled {
		t.Error("AgentEnabled should be false when explicitly disabled")
	}
	os.Unsetenv("AGENT_ENABLED")
}

// --- OSS additional tests ---

func TestOSSObjectURL_NoPrefix(t *testing.T) {
	cfg := &C{
		OSSPublicURLPrefix: "",
		OSSEndpoint:        "",
		OSSBucket:          "",
	}
	if got := cfg.OSSObjectURL("videos/42.mp4"); got != "videos/42.mp4" {
		t.Errorf("OSSObjectURL with no prefix = %q; want key as-is", got)
	}
}

func TestOSSObjectURL_WithPrefix(t *testing.T) {
	cfg := &C{
		OSSPublicURLPrefix: "https://cdn.example.com",
		OSSEndpoint:        "",
		OSSBucket:          "",
	}
	if got := cfg.OSSObjectURL("videos/42.mp4"); got != "https://cdn.example.com/videos/42.mp4" {
		t.Errorf("OSSObjectURL = %q; want https://cdn.example.com/videos/42.mp4", got)
	}
}

func TestOSSObjectURL_FromEndpointAndBucket(t *testing.T) {
	cfg := &C{
		OSSPublicURLPrefix: "",
		OSSEndpoint:        "https://oss-cn-beijing.aliyuncs.com",
		OSSBucket:          "mybucket",
	}
	if got := cfg.OSSObjectURL("videos/42.mp4"); got != "https://mybucket.oss-cn-beijing.aliyuncs.com/videos/42.mp4" {
		t.Errorf("OSSObjectURL = %q; want constructed from endpoint+bucket", got)
	}
}

func TestOSSObjectURL_WithLeadingSlash(t *testing.T) {
	cfg := &C{
		OSSPublicURLPrefix: "https://cdn.example.com",
	}
	if got := cfg.OSSObjectURL("/videos/42.mp4"); got != "https://cdn.example.com/videos/42.mp4" {
		t.Errorf("OSSObjectURL with leading slash = %q; want cleaned", got)
	}
}

func TestOSSObjectKeyFromURL_Empty(t *testing.T) {
	cfg := &C{}
	if got := cfg.OSSObjectKeyFromURL(""); got != "" {
		t.Errorf("OSSObjectKeyFromURL('') = %q; want ''", got)
	}
}

func TestOSSObjectKeyFromURL_NoPrefix(t *testing.T) {
	cfg := &C{
		OSSBucket:   "mybucket",
		OSSEndpoint: "https://oss-cn-beijing.aliyuncs.com",
	}
	got := cfg.OSSObjectKeyFromURL("https://mybucket.oss-cn-beijing.aliyuncs.com/videos/42.mp4")
	if got != "videos/42.mp4" {
		t.Errorf("OSSObjectKeyFromURL = %q; want videos/42.mp4", got)
	}
}

func TestOSSObjectKeyFromURL_Invalid(t *testing.T) {
	cfg := &C{
		OSSPublicURLPrefix: "https://cdn.example.com",
	}
	got := cfg.OSSObjectKeyFromURL("://invalid-url")
	if got != "" {
		t.Errorf("OSSObjectKeyFromURL(invalid) = %q; want ''", got)
	}
}

func TestNormalizeAliyunOSSEndpoint_EdgeCases(t *testing.T) {
	if got := normalizeAliyunOSSEndpoint("", ""); got != "" {
		t.Errorf("normalize('', '') = %q; want ''", got)
	}
	if got := normalizeAliyunOSSEndpoint("https://oss-cn-beijing.aliyuncs.com", ""); got != "https://oss-cn-beijing.aliyuncs.com" {
		t.Errorf("normalize with empty bucket = %q", got)
	}
	if got := normalizeAliyunOSSEndpoint("", "mybucket"); got != "" {
		t.Errorf("normalize with empty endpoint = %q; want ''", got)
	}

	ep := "https://oss-cn-beijing.aliyuncs.com"
	if got := normalizeAliyunOSSEndpoint(ep, "mybucket"); got != ep {
		t.Errorf("normalize already normal endpoint = %q; want %q", got, ep)
	}

	ep = "http://bucket.oss-cn-hangzhou.aliyuncs.com"
	want := "http://oss-cn-hangzhou.aliyuncs.com"
	if got := normalizeAliyunOSSEndpoint(ep, "bucket"); got != want {
		t.Errorf("normalize http endpoint = %q; want %q", got, want)
	}

	ep = "https://wrong.oss-cn-beijing.aliyuncs.com"
	if got := normalizeAliyunOSSEndpoint(ep, "mybucket"); got != ep {
		t.Errorf("non-matching bucket should return endpoint as-is = %q", got)
	}

	ep = "https://a.b"
	if got := normalizeAliyunOSSEndpoint(ep, "a"); got != ep {
		t.Errorf("too-short host returns endpoint as-is = %q", got)
	}

	ep = "https://bucket.example.com"
	if got := normalizeAliyunOSSEndpoint(ep, "bucket"); got != ep {
		t.Errorf("non-oss host returns endpoint as-is = %q", got)
	}
}

// --- RuntimeConfig coverage gaps ---

func TestNewRuntimeConfig_NilDefaults(t *testing.T) {
	rc := NewRuntimeConfig(nil, nil)
	if rc.defaults == nil {
		t.Error("NewRuntimeConfig(nil, nil) should initialize defaults map")
	}
}

func TestRuntimeConfig_GetBool_MissingKey(t *testing.T) {
	rc := NewRuntimeConfig(nil, nil)
	if rc.GetBool("nonexistent", true) != true {
		t.Error("GetBool should return fallback for missing key")
	}
}

func TestRuntimeConfig_GetInt_InvalidValue(t *testing.T) {
	rc := NewRuntimeConfig(nil, nil)
	rc.mu.Lock()
	rc.cache["badint"] = "not-a-number"
	rc.cache["goodint"] = "42"
	rc.mu.Unlock()

	if got := rc.GetInt("badint", 10); got != 10 {
		t.Errorf("GetInt(badint, 10) = %d; want 10", got)
	}
	if got := rc.GetInt("goodint", 0); got != 42 {
		t.Errorf("GetInt(goodint, 0) = %d; want 42", got)
	}
}

func TestRuntimeConfig_GetFloat_InvalidValue(t *testing.T) {
	rc := NewRuntimeConfig(nil, nil)
	rc.mu.Lock()
	rc.cache["badfloat"] = "not-a-number"
	rc.cache["goodfloat"] = "3.14"
	rc.mu.Unlock()

	if got := rc.GetFloat("badfloat", 1.5); got != 1.5 {
		t.Errorf("GetFloat(badfloat, 1.5) = %f; want 1.5", got)
	}
	if got := rc.GetFloat("goodfloat", 0); got != 3.14 {
		t.Errorf("GetFloat(goodfloat, 0) = %f; want 3.14", got)
	}
}

func TestRuntimeConfig_Get_MissingInCache(t *testing.T) {
	rc := NewRuntimeConfig(nil, map[string]string{"mykey": "myval"})
	if got := rc.Get("mykey", "fallback"); got != "myval" {
		t.Errorf("Get(mykey, fallback) = %q; want 'myval' from defaults", got)
	}
}

func TestRuntimeConfig_Get_CachePrecedence(t *testing.T) {
	rc := NewRuntimeConfig(nil, map[string]string{"k": "defaultval"})
	rc.mu.Lock()
	rc.cache["k"] = "cachedval"
	rc.mu.Unlock()
	if got := rc.Get("k", "fallback"); got != "cachedval" {
		t.Errorf("Get(k) = %q; want 'cachedval' from cache", got)
	}
}

func TestRuntimeConfig_Set_UpdatesCache(t *testing.T) {
	rc := NewRuntimeConfig(nil, nil)
	rc.mu.Lock()
	rc.cache["existing"] = "old"
	rc.mu.Unlock()
	if got := rc.Get("existing", ""); got != "old" {
		t.Errorf("Get(existing) = %q; want 'old'", got)
	}
}

func TestRuntimeConfig_Stop_Twice(t *testing.T) {
	rc := NewRuntimeConfig(nil, nil)
	rc.Stop()
	rc.Stop()
}

func TestRuntimeConfig_NewAndGetFloat_Missing(t *testing.T) {
	rc := NewRuntimeConfig(nil, nil)
	if got := rc.GetFloat("missing", 99.9); got != 99.9 {
		t.Errorf("GetFloat(missing) = %f; want 99.9", got)
	}
}
func TestOSSObjectKeyFromURL_HTTPPrefix(t *testing.T) {
	cfg := &C{
		OSSPublicURLPrefix: "https://mybucket.oss-cn-beijing.aliyuncs.com",
	}
	// Test that http:// prefix variant also matches
	got := cfg.OSSObjectKeyFromURL("http://mybucket.oss-cn-beijing.aliyuncs.com/videos/42.mp4")
	if got != "videos/42.mp4" {
		t.Errorf("OSSObjectKeyFromURL(http) = %q; want videos/42.mp4", got)
	}
}

func TestNormalizeAliyunOSSEndpoint_EmptyHost(t *testing.T) {
	// https:// with no actual host -> should return as-is
	if got := normalizeAliyunOSSEndpoint("https://", "mybucket"); got != "https://" {
		t.Errorf("normalize('https://', 'mybucket') = %q; want 'https://'", got)
	}
}
