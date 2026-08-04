package handler

import (
	"cakecake/internal/service/danmaku"
	"cakecake/internal/service/dm"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"cakecake/internal/config"
	"cakecake/internal/middleware"
	"cakecake/internal/pkg/iplocate"
	"cakecake/internal/pkg/jwttoken"
	"cakecake/internal/pkg/sensitive"
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
	"cakecake/internal/service/storage"
	"cakecake/internal/service/user"
	"cakecake/internal/service/video"
	"cakecake/internal/service/viewhistory"
	"cakecake/internal/ws"
)

// Dependencies are shared across HTTP handlers.
type Dependencies struct {
	Cfg            *config.C
	DB             *gorm.DB
	Redis          *redis.Client
	Log            *zap.Logger
	Hub            *ws.Hub
	ChatHub        *ws.ChatHub
	JWT            *jwttoken.Manager
	Sens           *sensitive.Filter
	Play           *playcount.PlayCounter
	SearchHot      *hotsearch.SearchHotRecorder
	SearchSvc      *searchsvc.SearchService
	StorageSvc     *storage.StorageService
	DailyRewardSvc *dailyreward.DailyRewardService
	DanmakuRelay   *danmaku.DanmakuRelay
	IPLocate       *iplocate.Searcher
	RuntimeCfg     *config.RuntimeConfig
	RateLimiter    *middleware.RateLimiter
	Agent          *agent.AgentService

	// Phase 1 domain services (thin service layer over business logic).
	VideoSvc          *video.VideoService
	BannerSvc         *banner.BannerService
	DmSvc             *dm.DmService
	FavoriteSvc       *favorite.FavoriteService
	ArticleSvc        *article.ArticleService
	DynamicSvc        *dynamic.DynamicService
	EngagementSvc     *engagement.EngagementService
	ViewHistorySvc    *viewhistory.ViewHistoryService
	VideoDraftSvc     *video.VideoDraftService
	CreatorCommentSvc *comment.CreatorCommentService
	AuthSvc           *user.AuthService
	FollowSvc         *follow.FollowService
	DanmakuSvc        *danmaku.DanmakuService
	CommentSvc        *comment.CommentService
	NotifSvc          *notification.NotificationService
	UserSvc           *user.UserService
	SearchHistorySvc  *searchsvc.SearchHistoryService
	HotSearchSvc      *hotsearch.HotSearchService

	// hotRecCh buffers SearchHot.Record requests (async, best-effort).
	hotRecCh chan<- hotRecordReq
}

// API exposes HTTP handlers.
type API struct {
	*Dependencies
}
