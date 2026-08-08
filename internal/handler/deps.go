package handler

import (
	"sync"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"cakecake/internal/client"
	"cakecake/internal/config"
	"cakecake/internal/middleware"
	"cakecake/internal/pkg/iplocate"
	"cakecake/internal/pkg/jwttoken"
	"cakecake/internal/pkg/sensitive"
	"cakecake/internal/service/danmaku"
	"cakecake/internal/service/hotsearch"
	"cakecake/internal/service/playcount"
	"cakecake/internal/ws"
)

// Dependencies are shared across HTTP handlers.
type Dependencies struct {
	Cfg          *config.C
	Redis        *redis.Client
	Log          *zap.Logger
	Hub          *ws.Hub
	ChatHub      *ws.ChatHub
	JWT          *jwttoken.Manager
	Sens         *sensitive.Filter
	Play         *playcount.PlayCounter
	SearchHot    *hotsearch.SearchHotRecorder
	DanmakuRelay *danmaku.DanmakuRelay
	IPLocate     *iplocate.Searcher
	RuntimeCfg   *config.RuntimeConfig
	RateLimiter  *middleware.RateLimiter

	// DB is a test-only seeding seam. Production handlers must not access the
	// database directly; all business data goes through the client contracts below.
	DB *gorm.DB

	// Domain service contracts (see internal/client). Handlers depend only on
	// these interfaces; concrete implementations are injected in cmd/cakecake.
	// Swapping to real gRPC clients only changes the wiring, not the handlers.
	SearchSvc         client.SearchSvc
	StorageSvc        client.StorageSvc
	DailyRewardSvc    client.DailyRewardSvc
	Agent             client.Agent
	VideoSvc          client.VideoSvc
	BannerSvc         client.BannerSvc
	DmSvc             client.DmSvc
	FavoriteSvc       client.FavoriteSvc
	ArticleSvc        client.ArticleSvc
	DynamicSvc        client.DynamicSvc
	EngagementSvc     client.EngagementSvc
	ViewHistorySvc    client.ViewHistorySvc
	VideoDraftSvc     client.VideoDraftSvc
	CreatorCommentSvc client.CreatorCommentSvc
	AuthSvc           client.AuthSvc
	FollowSvc         client.FollowSvc
	DanmakuSvc        client.DanmakuSvc
	CommentSvc        client.CommentSvc
	NotifSvc          client.NotifSvc
	UserSvc           client.UserSvc
	SearchHistorySvc  client.SearchHistorySvc
	HotSearchSvc      client.HotSearchSvc

	// hotRecCh buffers SearchHot.Record requests (async, best-effort).
	hotRecCh chan<- hotRecordReq

	// agentRunMu guards agentRunLocks (per-user generation serialization).
	agentRunMu    sync.Mutex
	agentRunLocks map[uint64]*sync.Mutex
}

// API exposes HTTP handlers.
type API struct {
	*Dependencies
}
