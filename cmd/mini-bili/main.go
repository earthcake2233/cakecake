package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"go.uber.org/zap"

	"minibili/internal/aigateway"
	"minibili/internal/config"
	"minibili/internal/data"
	"minibili/internal/ffmpeg"
	"minibili/internal/handler"
	"minibili/internal/middleware"
	"minibili/internal/logger"
	"minibili/internal/pkg/iplocate"
	"minibili/internal/pkg/jwttoken"
	"minibili/internal/pkg/sensitive"
	"minibili/internal/queue"
	"minibili/internal/search"
	"minibili/internal/service"
	"minibili/internal/storage"
	"minibili/internal/worker"
	"minibili/internal/ws"
)

func main() {
	_ = godotenv.Load()
	logger.Init()
	log := logger.L

	cfg := config.Load()
	if cfg.JWTSecret == "" || cfg.MySQLDSN == "" {
		log.Fatal("missing required env: JWT_SECRET, MYSQL_DSN")
	}

	ffmpeg.Init(cfg.FFprobePath, cfg.FFmpegPath)
	if err := ffmpeg.CheckFFprobe(); err != nil {
		log.Warn("ffprobe 不可用，视频上传将返回 40009，直至 PATH 或 FFPROBE_PATH 配置正确",
			zap.String("ffprobe", ffmpeg.FFprobeExe()),
			zap.Error(err),
		)
	} else {
		log.Info("ffprobe ok", zap.String("path", ffmpeg.FFprobeExe()))
	}

	db, err := data.NewDB(cfg.MySQLDSN, log)
	if err != nil {
		log.Fatal("database", zap.Error(err))
	}
	rdb, err := data.NewRedis(cfg)
	if err != nil {
		log.Fatal("redis", zap.Error(err))
	}
	mq, err := queue.Dial(cfg.RabbitMQURL)
	if err != nil {
		log.Fatal("rabbitmq", zap.Error(err))
	}
	defer func() { _ = mq.Close() }()

	jm, err := jwttoken.NewManager(cfg.JWTSecret)
	if err != nil {
		log.Fatal("jwt", zap.Error(err))
	}

	sens := sensitive.NewFilter(cfg.SensitiveWordsFile, log)
	if err := sens.Reload(); err != nil {
		log.Warn("sensitive words initial load", zap.Error(err))
	}

	var ossc *storage.OSS
	if o, err := storage.NewOSS(cfg.OSSEndpoint, cfg.OSSAccessKeyID, cfg.OSSAccessKeySecret, cfg.OSSBucket); err == nil {
		ossc = o
		log.Info("oss client initialized")
	} else {
		log.Warn("oss client disabled", zap.Error(err))
	}

	if err := os.MkdirAll(cfg.TempUploadDir, 0o755); err != nil {
		log.Fatal("temp upload dir", zap.Error(err))
	}

	if err := data.SeedDefaultAdmin(db, cfg, log); err != nil {
		log.Warn("seed default admin", zap.Error(err))
	}
	if err := data.EnsureAgentProfiles(db, cfg, log); err != nil {
		log.Warn("ensure agent profiles", zap.Error(err))
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var esc *search.Client
	if cfg.ElasticsearchURL != "" {
		if c, err := search.Dial(cfg); err != nil {
			log.Warn("elasticsearch disabled", zap.String("url", cfg.ElasticsearchURL), zap.Error(err))
		} else {
			esc = c
			if err := esc.EnsureIndices(context.Background()); err != nil {
				log.Warn("elasticsearch ensure indices", zap.Error(err))
			} else {
				log.Info("elasticsearch client initialized", zap.String("url", cfg.ElasticsearchURL))
				go func() {
					rctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
					defer cancel()
					if err := esc.ReindexAll(rctx, db); err != nil {
						log.Warn("elasticsearch reindex all", zap.Error(err))
					} else {
						log.Info("elasticsearch reindex all completed")
					}
				}()
			}
		}
	} else {
		log.Info("elasticsearch disabled (ELASTICSEARCH_URL empty)")
	}
	defer func() { _ = esc.Close() }()

	go worker.StartTranscodeConsumer(ctx, cfg, db, mq, ossc, esc)

	pc := &service.PlayCounter{Rdb: rdb, DB: db}
	go func() {
		t := time.NewTicker(10 * time.Second)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				if err := pc.Flush(context.Background()); err != nil {
					log.Error("flush playcount", zap.Error(err))
				}
			}
		}
	}()

	hub := ws.NewHub()
	chatHub := ws.NewChatHub()
	relay := service.NewDanmakuRelay(rdb, hub, log)
	go relay.RunSubscriber(ctx)

	var ipLoc *iplocate.Searcher
	if ipLoc, err = iplocate.Open(cfg.IP2RegionV4XDB); err != nil {
		log.Warn("ip2region disabled", zap.String("path", cfg.IP2RegionV4XDB), zap.Error(err))
		ipLoc = nil
	} else if ipLoc != nil {
		log.Info("ip2region enabled", zap.String("path", cfg.IP2RegionV4XDB))
	}

	searchHot := &service.SearchHotRecorder{Rdb: rdb, Sens: sens}

	var agentGW *aigateway.Gateway
	if cfg.DeepSeekAPIKey != "" {
		agentGW = &aigateway.Gateway{
			LLM: &aigateway.Client{
				APIKey:  cfg.DeepSeekAPIKey,
				BaseURL: cfg.DeepSeekBaseURL,
				Model:   cfg.DeepSeekModel,
				HTTPClient: &http.Client{Timeout: cfg.AgentRequestTimeout},
			},
			Redis:       rdb,
			MaxHistory:  cfg.AgentMaxHistory,
			HistoryTTL:  cfg.AgentHistoryTTL,
		}
		log.Info("ai gateway enabled",
			zap.String("model", cfg.DeepSeekModel),
			zap.String("base_url", cfg.DeepSeekBaseURL),
		)
	} else {
		log.Info("ai gateway disabled (DEEPSEEK_API_KEY empty)")
	}
	var rl *middleware.RateLimiter
	if cfg.RateLimitEnabled {
		rl = middleware.NewRateLimiter(rdb, cfg.RateLimitRate, cfg.RateLimitBurst)
		log.Info("rate limiter enabled",
			zap.Float64("rate", cfg.RateLimitRate),
			zap.Int("burst", cfg.RateLimitBurst),
		)
	}
	agentSvc := &service.AgentService{
		Cfg: cfg, DB: db, Redis: rdb, Gateway: agentGW, Sens: sens,
		ChatHub: chatHub, Log: log,
	}

	deps := &handler.Dependencies{
		Cfg: cfg, DB: db, Redis: rdb, Log: log, Hub: hub, ChatHub: chatHub,
		JWT: jm, Sens: sens, OSS: ossc, MQ: mq, ES: esc, Play: pc,
		SearchHot: searchHot, DanmakuRelay: relay, IPLocate: ipLoc, Agent: agentSvc,
		RateLimiter:  rl,
	}
	api := &handler.API{Dependencies: deps}

	if cfg.AppEnv == "development" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.New()
	r.Use(gin.Recovery())
	handler.RegisterRoutes(r, api, jm)

	go func() {
		if err := r.Run(cfg.HTTPAddr); err != nil {
			log.Fatal("http server", zap.Error(err))
		}
	}()
	log.Info("mini-bili listening", zap.String("addr", cfg.HTTPAddr))

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch
	log.Info("shutting down")
	cancel()
	time.Sleep(300 * time.Millisecond)
}
