package handler

import (
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"minibili/internal/config"
	"minibili/internal/middleware"
	"minibili/internal/pkg/iplocate"
	"minibili/internal/pkg/jwttoken"
	"minibili/internal/pkg/sensitive"
	"minibili/internal/queue"
	"minibili/internal/search"
	"minibili/internal/service"
	"minibili/internal/storage"
	"minibili/internal/ws"
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
	RateLimiter  *middleware.RateLimiter
	Agent        *service.AgentService
}

// API exposes HTTP handlers.
type API struct {
	*Dependencies
}
