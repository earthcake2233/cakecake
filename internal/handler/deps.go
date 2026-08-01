package handler

import (
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"cakecake/internal/config"
	"cakecake/internal/middleware"
	"cakecake/internal/pkg/iplocate"
	"cakecake/internal/pkg/jwttoken"
	"cakecake/internal/pkg/sensitive"
	"cakecake/internal/queue"
	"cakecake/internal/search"
	"cakecake/internal/service"
	"cakecake/internal/storage"
	"cakecake/internal/ws"
)

// Dependencies are shared across HTTP handlers.
type Dependencies struct {
	Cfg          *config.C
	DB           *gorm.DB
	Redis        *redis.Client
	Log          *zap.Logger
	Hub          *ws.Hub
	ChatHub      *ws.ChatHub
	JWT          *jwttoken.Manager
	Sens         *sensitive.Filter
	OSS          *storage.OSS
	MQ           queue.TranscodePublisher
	ES           *search.Client
	Play         *service.PlayCounter
	SearchHot    *service.SearchHotRecorder
	DanmakuRelay *service.DanmakuRelay
	IPLocate     *iplocate.Searcher
	RuntimeCfg   *config.RuntimeConfig
	RateLimiter  *middleware.RateLimiter
	Agent        *service.AgentService

	// Phase 1 domain services (thin service layer over business logic).
	VideoSvc          *service.VideoService
	DmSvc             *service.DmService
	FavoriteSvc       *service.FavoriteService
	ArticleSvc        *service.ArticleService
	DynamicSvc        *service.DynamicService
	EngagementSvc     *service.EngagementService
	ViewHistorySvc    *service.ViewHistoryService
	VideoDraftSvc     *service.VideoDraftService
	CreatorCommentSvc *service.CreatorCommentService
	AuthSvc           *service.AuthService
	FollowSvc         *service.FollowService
	DanmakuSvc        *service.DanmakuService
	CommentSvc        *service.CommentService
	NotifSvc          *service.NotificationService
	UserSvc           *service.UserService
	SearchHistorySvc  *service.SearchHistoryService
	HotSearchSvc      *service.HotSearchService

	// hotRecCh buffers SearchHot.Record requests (async, best-effort).
	hotRecCh chan<- hotRecordReq
}

// API exposes HTTP handlers.
type API struct {
	*Dependencies
}
