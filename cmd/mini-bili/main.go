// Package main Mini-Bili API.
//
// @title           Mini-Bili API
// @version         1.0
// @description     Mini-Bili - 轻量级弹幕视频分享平台后端API
// @termsOfService  https://github.com/earthcake2233/cakecake/blob/main/LICENSE
//
// @contact.name   earthcake2233
// @contact.url    https://github.com/earthcake2233
//
// @license.name  Custom Non-Commercial License
// @license.url   https://github.com/earthcake2233/cakecake/blob/main/LICENSE
//
// @host           localhost:8080
// @BasePath       /api/v1
// @schemes        http
//
// @securityDefinitions.apikey  BearerAuth
// @in                          header
// @name                        Authorization
// @description                 JWT Bearer token, e.g. "Bearer {token}"
package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"go.uber.org/zap"

	"minibili/internal/aigateway"
	"minibili/internal/aigateway/toolkit"
	"minibili/internal/config"
	"minibili/internal/data"
	"minibili/internal/ffmpeg"
	"minibili/internal/handler"
	"minibili/internal/logger"
	"minibili/internal/middleware"
	"minibili/internal/pkg/iplocate"
	"minibili/internal/pkg/jwttoken"
	"minibili/internal/pkg/sensitive"
	"minibili/internal/queue"
	"minibili/internal/search"
	"minibili/internal/service"
	"minibili/internal/storage"
	"minibili/internal/worker"
	"minibili/internal/ws"

	_ "minibili/docs/swagger"
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
		log.Warn("ffprobe 不可用，视频上传将返回40009，直到PATH 或FFPROBE_PATH 配置正确",
			zap.String("ffprobe", ffmpeg.FFprobeExe()),
			zap.Error(err),
		)
	} else {
		log.Info("ffprobe ok", zap.String("path", ffmpeg.FFprobeExe()))
	}

	db, err := data.NewDB(cfg.MySQLDSN, log, cfg.DBAutoMigrate)
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

	// Runtime config: seeded from env, periodically refreshed from DB.
	runtimeCfg := config.NewRuntimeConfig(db, map[string]string{
		"agent_enabled":         strconv.FormatBool(cfg.AgentEnabled),
		"agent_daily_quota":     strconv.Itoa(cfg.AgentDailyQuota),
		"agent_max_history":     strconv.Itoa(cfg.AgentMaxHistory),
		"agent_history_ttl":     cfg.AgentHistoryTTL.String(),
		"agent_request_timeout": cfg.AgentRequestTimeout.String(),
		"rate_limit_enabled":    strconv.FormatBool(cfg.RateLimitEnabled),
		"rate_limit_rate":       strconv.FormatFloat(cfg.RateLimitRate, 'f', -1, 64),
		"rate_limit_burst":      strconv.Itoa(cfg.RateLimitBurst),
	})
	runtimeCtx, runtimeCancel := context.WithCancel(context.Background())
	defer runtimeCancel()
	runtimeCfg.Start(runtimeCtx)
	log.Info("runtime config initialized")

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

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		worker.StartTranscodeConsumer(ctx, cfg, db, mq, ossc, esc)
	}()

	pc := &service.PlayCounter{Rdb: rdb, DB: db}
	wg.Add(1)
	go func() {
		defer wg.Done()
		// Final flush on exit.
		defer func() {
			if err := pc.Flush(context.Background()); err != nil {
				log.Error("final flush playcount", zap.Error(err))
			}
		}()
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
	wg.Add(1)
	go func() {
		defer wg.Done()
		relay.RunSubscriber(ctx)
	}()

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
				APIKey:     cfg.DeepSeekAPIKey,
				BaseURL:    cfg.DeepSeekBaseURL,
				Model:      cfg.DeepSeekModel,
				HTTPClient: &http.Client{Timeout: cfg.AgentRequestTimeout},
			},
			Redis:      rdb,
			MaxHistory: cfg.AgentMaxHistory,
			HistoryTTL: cfg.AgentHistoryTTL,
		}
		log.Info("ai gateway enabled",
			zap.String("model", cfg.DeepSeekModel),
			zap.String("base_url", cfg.DeepSeekBaseURL),
		)
	} else {
		log.Info("ai gateway disabled (DEEPSEEK_API_KEY empty)")
	}
	// Always create the rate limiter middleware; dynamic config controls whether it blocks.
	rl := middleware.NewRateLimiter(rdb, runtimeCfg, cfg.RateLimitRate, cfg.RateLimitBurst)
	log.Info("rate limiter middleware mounted",
		zap.Float64("default_rate", cfg.RateLimitRate),
		zap.Int("default_burst", cfg.RateLimitBurst),
	)
	agentSvc := &service.AgentService{
		Cfg: cfg, DB: db, Redis: rdb, Gateway: agentGW, Sens: sens,
		ChatHub: chatHub, Log: log, RC: runtimeCfg,
		ToolExec: &toolkit.PlatformExecutor{DB: db, ES: esc, Sens: sens},
	}

	userProv := service.NewUserProvider(db)
	videoProv := service.NewVideoProvider(db)
	articleProv := service.NewArticleProvider(db)
	dynamicProv := service.NewDynamicProvider(db)

	notifSvc := service.NewNotificationService(db, rdb, log, userProv)
	commentSvc := service.NewCommentService(db, rdb, log, sens, notifSvc, userProv, videoProv, articleProv, dynamicProv)
	authSvc := service.NewAuthService(db, rdb, log, jm, service.AuthConfig{AgentBotUsername: cfg.AgentBotUsername})
	followSvc := service.NewFollowService(db, log)
	danmakuSvc := service.NewDanmakuService(db, rdb, log, sens)
	userSvc := service.NewUserService(db, log)
	videoSvc := service.NewVideoService(db, rdb, log)
	dmSvc := service.NewDmService(db, rdb, log)
	favoriteSvc := service.NewFavoriteService(db, rdb, log, userProv, videoProv)
	articleSvc := service.NewArticleService(db, rdb, log, userSvc)
	dynamicSvc := service.NewDynamicService(db, rdb, log)
	engagementSvc := service.NewEngagementService(db, rdb, log, userProv, videoProv)
	viewHistorySvc := service.NewViewHistoryService(db, rdb, log)
	videoDraftSvc := service.NewVideoDraftService(db, rdb, log)
	creatorCommentSvc := service.NewCreatorCommentService(db, rdb, log)
	searchHistorySvc := service.NewSearchHistoryService(db, log)
	hotSearchSvc := service.NewHotSearchService(db, searchHot)

	deps := &handler.Dependencies{
		Cfg: cfg, DB: db, Redis: rdb, Log: log, Hub: hub, ChatHub: chatHub,
		JWT: jm, Sens: sens, OSS: ossc, MQ: mq, ES: esc, Play: pc,
		SearchHot: searchHot, DanmakuRelay: relay, IPLocate: ipLoc, Agent: agentSvc,
		RateLimiter: rl, RuntimeCfg: runtimeCfg,
		AuthSvc:    authSvc,
		FollowSvc:  followSvc,
		DanmakuSvc: danmakuSvc,
		UserSvc:    userSvc,
		SearchHistorySvc: searchHistorySvc,
		HotSearchSvc:    hotSearchSvc,
		CommentSvc: commentSvc,
		NotifSvc:   notifSvc,
		VideoSvc:   videoSvc,
		DmSvc:      dmSvc,
		FavoriteSvc: favoriteSvc,
		ArticleSvc:  articleSvc,
		DynamicSvc:  dynamicSvc,
		EngagementSvc: engagementSvc,
		ViewHistorySvc: viewHistorySvc,
		VideoDraftSvc: videoDraftSvc,
		CreatorCommentSvc: creatorCommentSvc,
	}
	api := &handler.API{Dependencies: deps}
	api.InitHotRecorder(64)

	if cfg.AppEnv == "development" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.New()
	r.Use(gin.Recovery())
	handler.RegisterRoutes(r, api, jm, cfg.AppEnv)

	srv := &http.Server{
		Addr:    cfg.HTTPAddr,
		Handler: r,
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("http server", zap.Error(err))
		}
	}()
	log.Info("mini-bili listening", zap.String("addr", cfg.HTTPAddr))

	// ── Graceful shutdown ──
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch
	log.Info("shutting down gracefully", zap.Duration("timeout", cfg.ShutdownTimeout))

	// Step 1: cancel contexts to stop accepting new work.
	cancel()
	runtimeCancel()

	// Step 2: drain HTTP connections.
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("http server forced to close", zap.Error(err))
	}

	// Step 3: wait for background goroutines with timeout.
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		log.Info("all background tasks finished")
	case <-time.After(cfg.ShutdownTimeout):
		log.Warn("shutdown timeout, forcing exit")
	}
}
