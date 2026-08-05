// Package main cakecake API.
//
// @title           cakecake API
// @version         1.0
// @description     cakecake - 轻量级弹幕视频分享平台后端API
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
	"cakecake/internal/service/danmaku"
	"cakecake/internal/service/dm"
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

	"cakecake/internal/aigateway"
	"cakecake/internal/aigateway/toolkit"
	"cakecake/internal/config"
	"cakecake/internal/data"
	"cakecake/internal/ffmpeg"
	"cakecake/internal/handler"
	"cakecake/internal/logger"
	"cakecake/internal/middleware"
	"cakecake/internal/pkg/iplocate"
	"cakecake/internal/pkg/jwttoken"
	"cakecake/internal/pkg/sensitive"
	"cakecake/internal/queue"
	"cakecake/internal/search"
	"cakecake/internal/service"
	"cakecake/internal/service/agent"
	"cakecake/internal/service/article"
	"cakecake/internal/service/banner"
	"cakecake/internal/service/comment"
	"cakecake/internal/service/dailyreward"
	"cakecake/internal/service/dynamic"
	"cakecake/internal/service/engagement"
	"cakecake/internal/service/favorite"
	"cakecake/internal/service/follow"
	"cakecake/internal/service/hotsearch"
	"cakecake/internal/service/notification"
	"cakecake/internal/service/playcount"
	searchsvc "cakecake/internal/service/search"
	storesvc "cakecake/internal/service/storage"
	"cakecake/internal/service/user"
	"cakecake/internal/service/video"
	"cakecake/internal/service/viewhistory"
	"cakecake/internal/storage"
	"cakecake/internal/worker"
	"cakecake/internal/ws"

	_ "cakecake/docs/swagger"
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
	if err := data.SeedDemoData(db, cfg, log); err != nil {
		log.Warn("seed demo data", zap.Error(err))
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

	pc := &playcount.PlayCounter{Rdb: rdb, Store: playcount.NewPlayCountStore(db)}
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
	relay := danmaku.NewDanmakuRelay(rdb, hub, log)
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

	searchHot := &hotsearch.SearchHotRecorder{Rdb: rdb, Sens: sens}

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
	agentSvc := &agent.AgentService{
		Cfg: cfg, Store: agent.NewAgentStore(db), Redis: rdb, Gateway: agentGW, Sens: sens,
		ChatHub: chatHub, Log: log, RC: runtimeCfg,
		ToolExec: &toolkit.PlatformExecutor{DB: db, ES: esc, Sens: sens},
	}

	userProv := service.NewUserProvider(db)
	videoProv := video.NewVideoProvider(db)
	articleProv := service.NewArticleProvider(db)
	dynamicProv := service.NewDynamicProvider(db)

	notifSvc := notification.NewNotificationService(db, rdb, log, userProv)
	commentSvc := comment.NewCommentService(db, rdb, log, sens, notifSvc, userProv, videoProv, articleProv, dynamicProv)
	authSvc := user.NewAuthService(db, rdb, log, jm, user.AuthConfig{AgentBotUsername: cfg.AgentBotUsername})
	followSvc := follow.NewFollowService(db, log)
	danmakuSvc := danmaku.NewDanmakuService(db, rdb, log, sens)
	userSvc := user.NewUserService(db, log)
	searchSvc := searchsvc.NewSearchService(esc, db, rdb, log)
	dailyRewardSvc := dailyreward.NewDailyRewardService(db)
	videoSvc := video.NewVideoService(db, rdb, log, esc, mq)
	bannerSvc := banner.NewBannerService(db)
	dmSvc := dm.NewDmService(db, rdb, log)
	favoriteSvc := favorite.NewFavoriteService(db, rdb, log, userProv, videoProv)
	articleSvc := article.NewArticleService(db, rdb, log, esc)
	dynamicSvc := dynamic.NewDynamicService(db, rdb, log)
	engagementSvc := engagement.NewEngagementService(db, rdb, log, userProv, videoProv)
	viewHistorySvc := viewhistory.NewViewHistoryService(db, rdb, log)
	videoDraftSvc := video.NewVideoDraftService(db, rdb, log, mq)
	creatorCommentSvc := comment.NewCreatorCommentService(db, rdb, log)
	searchHistorySvc := searchsvc.NewSearchHistoryService(db, log)
	hotSearchSvc := hotsearch.NewHotSearchService(db, searchHot)

	deps := &handler.Dependencies{
		Cfg: cfg, DB: db, Redis: rdb, Log: log, Hub: hub, ChatHub: chatHub,
		JWT: jm, Sens: sens, StorageSvc: storesvc.NewStorageService(cfg, ossc, log), Play: pc,
		SearchHot: searchHot, DanmakuRelay: relay, IPLocate: ipLoc, Agent: agentSvc,
		RateLimiter: rl, RuntimeCfg: runtimeCfg,
		SearchSvc:         searchSvc,
		DailyRewardSvc:    dailyRewardSvc,
		AuthSvc:           authSvc,
		FollowSvc:         followSvc,
		DanmakuSvc:        danmakuSvc,
		UserSvc:           userSvc,
		SearchHistorySvc:  searchHistorySvc,
		HotSearchSvc:      hotSearchSvc,
		CommentSvc:        commentSvc,
		NotifSvc:          notifSvc,
		VideoSvc:          videoSvc,
		BannerSvc:         bannerSvc,
		DmSvc:             dmSvc,
		FavoriteSvc:       favoriteSvc,
		ArticleSvc:        articleSvc,
		DynamicSvc:        dynamicSvc,
		EngagementSvc:     engagementSvc,
		ViewHistorySvc:    viewHistorySvc,
		VideoDraftSvc:     videoDraftSvc,
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
	log.Info("cakecake listening", zap.String("addr", cfg.HTTPAddr))

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
