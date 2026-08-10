package handler

import (
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"cakecake/internal/logger"
	"cakecake/internal/middleware"
	"cakecake/internal/pkg/jwttoken"
)

// RegisterRoutes wires HTTP and WebSocket routes.
func RegisterRoutes(r *gin.Engine, a *API, jwtm *jwttoken.Manager, appEnv string) {
	r.Use(corsMiddleware)
	r.Use(logger.GinMiddleware(a.Log))
	if a.RateLimiter != nil {
		r.Use(a.RateLimiter.RateLimit())
	}

	r.GET("/api/v1/health", a.Health)
	// Prometheus metrics (default registry: LLM/agent observability).
	// Optional bearer token keeps user/cost labels private on shared hosts.
	metricsToken := ""
	if a != nil && a.Cfg != nil {
		metricsToken = a.Cfg.MetricsToken
	}
	r.GET("/metrics", metricsAuth(metricsToken), gin.WrapH(promhttp.Handler()))

	// Swagger documentation
	RegisterSwaggerRoutes(r, appEnv)

	pub := r.Group("/api/v1")

	admin := r.Group("/api/v1/admin")
	admin.Use(middleware.AdminJWTAuth(jwtm))

	authd := r.Group("/api/v1")
	authd.Use(middleware.JWTAuth(jwtm))

	registerUserRoutes(r, pub, admin, authd, a, jwtm)
	registerVideoRoutes(r, pub, admin, authd, a, jwtm)
	registerArticleRoutes(r, pub, admin, authd, a, jwtm)
	registerDynamicRoutes(r, pub, admin, authd, a, jwtm)
	registerCommentRoutes(r, pub, admin, authd, a, jwtm)
	registerDanmakuRoutes(r, pub, admin, authd, a, jwtm)
	registerNotificationRoutes(r, pub, admin, authd, a, jwtm)
	registerDmRoutes(r, pub, admin, authd, a, jwtm)
	registerSearchRoutes(r, pub, admin, authd, a, jwtm)
	registerBannerRoutes(r, pub, admin, authd, a, jwtm)
	registerHotsearchRoutes(r, pub, admin, authd, a, jwtm)
	registerAdminRoutes(r, pub, admin, authd, a, jwtm)
	registerStatsRoutes(r, pub, admin, authd, a, jwtm)
	registerWsRoutes(r, pub, admin, authd, a, jwtm)
}

// metricsAuth guards the Prometheus scrape endpoint with a constant-time
// bearer-token comparison. An empty token keeps the endpoint open for local
// development; production must set METRICS_TOKEN.
func metricsAuth(token string) gin.HandlerFunc {
	if token == "" {
		return func(c *gin.Context) { c.Next() }
	}
	const prefix = "Bearer "
	return func(c *gin.Context) {
		auth := strings.TrimSpace(c.GetHeader("Authorization"))
		if len(auth) <= len(prefix) || !strings.EqualFold(auth[:len(prefix)], prefix) {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		got := strings.TrimSpace(auth[len(prefix):])
		if subtle.ConstantTimeCompare([]byte(got), []byte(token)) != 1 {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		c.Next()
	}
}

func corsMiddleware(c *gin.Context) {
	h := c.Writer.Header()
	h.Set("Access-Control-Allow-Origin", "*")
	h.Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
	h.Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
	if c.Request.Method == http.MethodOptions {
		c.AbortWithStatus(http.StatusNoContent)
		return
	}
	c.Next()
}

func registerUserRoutes(r *gin.Engine, pub, admin, authd *gin.RouterGroup, a *API, jwtm *jwttoken.Manager) {
	pub.POST("/users", a.Register)
	pub.POST("/auth/login", a.Login)
	pub.POST("/auth/refresh", a.Refresh)
	pub.GET("/space/:userId", middleware.OptionalJWT(jwtm), a.GetUserPublic)
	pub.GET("/space/:userId/recent-coins", middleware.OptionalJWT(jwtm), a.ListUserRecentCoinVideos)
	pub.GET("/space/:userId/following", middleware.OptionalJWT(jwtm), a.ListUserFollowing)
	pub.GET("/space/:userId/followers", middleware.OptionalJWT(jwtm), a.ListUserFollowers)
	pub.GET("/space/:userId/article-favorites", middleware.OptionalJWT(jwtm), a.ListUserArticleFavorites)
	authd.GET("/users/me", a.GetMe)
	authd.GET("/users/me/daily-rewards", a.GetMeDailyRewards)
	authd.GET("/users/me/coin-ledger", a.ListMeCoinLedger)
	authd.GET("/users/me/view-history", a.ListMyViewHistory)
	authd.DELETE("/users/me/view-history", a.ClearMyViewHistory)
	authd.DELETE("/users/me/view-history/:videoId", a.DeleteMyViewHistoryEntry)
	authd.DELETE("/users/me/view-history/articles/:articleId", a.DeleteMyArticleViewHistoryEntry)
	authd.GET("/users/me/view-history/settings", a.GetMyViewHistorySettings)
	authd.PUT("/users/me/view-history/settings", a.PutMyViewHistorySettings)
	authd.POST("/videos/:id/view-history", a.PostVideoViewHistory)
	authd.POST("/users/me/daily-rewards/watch", a.PostMeDailyRewardWatch)
	authd.PUT("/users/me", a.UpdateMeUsername)
	authd.PUT("/users/me/profile", a.UpdateMeProfile)
	authd.PUT("/users/me/announcement", a.UpdateMeAnnouncement)
	authd.GET("/users/me/space-privacy", a.GetMeSpacePrivacy)
	authd.PUT("/users/me/space-privacy", a.UpdateMeSpacePrivacy)
	authd.POST("/users/:userId/follow", a.ToggleFollowUser)
	authd.POST("/users/:userId/block", a.BlockUser)
	authd.GET("/users/me/follow-groups", a.ListMyFollowGroups)
	authd.POST("/users/me/follow-groups", a.CreateFollowGroup)
	authd.PUT("/users/me/follow-groups/:groupId", a.UpdateFollowGroup)
	authd.DELETE("/users/me/follow-groups/:groupId", a.DeleteFollowGroup)
	authd.GET("/users/me/following/:followeeId/groups", a.ListFolloweeGroupIDs)
	authd.POST("/users/me/follow-groups/:groupId/members", a.AddFollowGroupMember)
	authd.DELETE("/users/me/follow-groups/:groupId/members/:followeeId", a.RemoveFollowGroupMember)
	authd.PUT("/users/me/password", a.UpdateMePassword)
	authd.POST("/users/me/deletion/request", a.RequestAccountDeletion)
	authd.POST("/users/me/deletion/revoke", a.RevokeAccountDeletion)
	authd.POST("/users/me/avatar", a.UpdateMeAvatar)
	authd.GET("/users/me/article-favorites", a.ListMyArticleFavorites)
}

func registerVideoRoutes(r *gin.Engine, pub, admin, authd *gin.RouterGroup, a *API, jwtm *jwttoken.Manager) {
	pub.GET("/videos", middleware.OptionalJWT(jwtm), a.ListPublishedVideos)
	pub.GET("/space/:userId/videos", middleware.OptionalJWT(jwtm), a.ListUserPublishedVideos)
	pub.GET("/space/:userId/favorites", middleware.OptionalJWT(jwtm), a.ListUserFavorites)
	pub.GET("/space/:userId/favorite-folders", middleware.OptionalJWT(jwtm), a.ListUserFavoriteFolders)
	pub.GET("/videos/:id", middleware.OptionalJWT(jwtm), a.GetVideo)
	admin.GET("/videos", a.AdminListVideos)
	admin.GET("/videos/:id", a.AdminGetVideo)
	admin.POST("/videos/:id/approve", a.AdminApproveVideo)
	admin.POST("/videos/:id/reject", a.AdminRejectVideo)
	admin.POST("/videos/:id/delete", a.AdminDeleteVideo)
	admin.DELETE("/videos/:id", a.AdminDeleteVideo)
	authd.GET("/users/me/videos", a.ListMyVideos)
	authd.DELETE("/videos/:id", a.DeleteMyVideo)
	authd.POST("/videos", a.UploadVideo)
	authd.POST("/videos/draft", a.SaveVideoDraft)
	authd.PUT("/videos/:id/draft", a.UpdateVideoDraft)
	authd.POST("/videos/:id/draft", a.UpdateVideoDraft)
	authd.POST("/videos/:id/publish", a.PublishVideoDraft)
	authd.POST("/videos/:id/replace-media", a.ReplaceVideoMedia)
	authd.GET("/users/me/videos/:id/draft-source", a.GetMyVideoDraftSource)
	authd.PUT("/videos/:id/cover", a.UpdateVideoCover)
	authd.PUT("/videos/:id", a.UpdateMyVideo)
	authd.PATCH("/videos/:id/playback", a.PatchVideoPlayback)
	authd.POST("/videos/:id/like", a.ToggleVideoLike)
	authd.POST("/videos/:id/favorite", a.ToggleVideoFavorite)
	authd.GET("/videos/:id/favorite-picker", a.GetVideoFavoritePicker)
	authd.PUT("/videos/:id/favorite-folders", a.SetVideoFavoriteFolders)
	authd.PUT("/videos/:id/favorite-folders/move", a.MoveVideoFavoriteFolder)
	authd.POST("/videos/:id/favorite-folders/:folderId", a.AddVideoToFavoriteFolder)
	authd.DELETE("/videos/:id/favorite-folders/:folderId", a.RemoveVideoFromFavoriteFolder)
	authd.POST("/videos/:id/coin", a.PostVideoCoin)
	authd.POST("/videos/:id/watch-later", a.ToggleWatchLater)
	authd.GET("/users/me/favorites", a.ListMyFavorites)
	authd.GET("/users/me/favorite-folders", a.ListMyFavoriteFolders)
	authd.POST("/users/me/favorite-folders", a.CreateFavoriteFolder)
	authd.PUT("/users/me/favorite-folders/:folderId", a.UpdateFavoriteFolder)
	authd.DELETE("/users/me/favorite-folders/:folderId", a.DeleteFavoriteFolder)
	authd.DELETE("/users/me/favorite-folders/:folderId/invalid-favorites", a.ClearInvalidFavoritesInFolder)
	authd.POST("/users/me/favorite-folders/:folderId/batch-remove", a.BatchRemoveVideosFromFavoriteFolder)
	authd.GET("/users/me/watch-later", a.ListMyWatchLater)
	authd.DELETE("/users/me/watch-later", a.ClearMyWatchLater)
	authd.DELETE("/users/me/watch-later/watched", a.ClearWatchedWatchLater)
	authd.POST("/users/me/watch-later/:id/watched", a.MarkWatchLaterWatched)
}

func registerArticleRoutes(r *gin.Engine, pub, admin, authd *gin.RouterGroup, a *API, jwtm *jwttoken.Manager) {
	pub.GET("/articles/:id", middleware.OptionalJWT(jwtm), a.GetArticle)
	pub.GET("/space/:userId/articles", middleware.OptionalJWT(jwtm), a.ListUserPublishedArticles)
	admin.GET("/articles", a.AdminListArticles)
	admin.GET("/articles/:id", a.AdminGetArticle)
	admin.POST("/articles/:id/approve", a.AdminApproveArticle)
	admin.POST("/articles/:id/reject", a.AdminRejectArticle)
	admin.POST("/articles/:id/delete", a.AdminDeleteArticle)
	admin.DELETE("/articles/:id", a.AdminDeleteArticle)
	authd.POST("/articles", a.PostArticle)
	authd.GET("/users/me/articles", a.ListMyArticles)
	authd.GET("/users/me/articles/:id", a.GetMyArticle)
	authd.PUT("/users/me/articles/:id", a.PutMyArticle)
	authd.PATCH("/users/me/articles/:id/playback", a.PatchArticlePlayback)
	authd.PUT("/users/me/articles/:id/cover", a.UpdateArticleCover)
	authd.DELETE("/users/me/articles/:id", a.DeleteMyArticle)
	authd.POST("/articles/:id/view", a.PostArticleView)
	authd.POST("/articles/:id/favorite", a.ToggleArticleFavorite)
	authd.POST("/articles/:id/coin", a.PostArticleCoin)
	authd.POST("/article-comments/:id/like", a.ToggleArticleCommentLike)
	authd.POST("/article-comments/:id/dislike", a.ToggleArticleCommentDislike)
	authd.POST("/article-comments/:id/pin", a.PinArticleComment)
	authd.POST("/article-comments/:id/approve", a.ApproveArticleComment)
	authd.POST("/article-comments/:id/ignore-curated", a.IgnoreCuratedArticleComment)
	authd.DELETE("/article-comments/:id", a.DeleteArticleComment)
}

func registerDynamicRoutes(r *gin.Engine, pub, admin, authd *gin.RouterGroup, a *API, jwtm *jwttoken.Manager) {
	pub.GET("/space/:userId/dynamics", middleware.OptionalJWT(jwtm), a.ListUserPublishedDynamics)
	pub.GET("/user-dynamics/:id", middleware.OptionalJWT(jwtm), a.GetUserDynamic)
	admin.GET("/dynamics", a.AdminListDynamics)
	admin.GET("/dynamics/:id", a.AdminGetDynamic)
	admin.POST("/dynamics/:id/delete", a.AdminDeleteDynamic)
	admin.DELETE("/dynamics/:id", a.AdminDeleteDynamic)
	authd.GET("/users/me/dynamics", a.ListMyDynamics)
	authd.POST("/users/me/dynamics", a.PostUserDynamic)
	authd.PUT("/users/me/dynamics/:id", a.PutMyUserDynamic)
	authd.DELETE("/users/me/dynamics/:id", a.DeleteMyDynamic)
	authd.PATCH("/users/me/dynamics/:id/playback", a.PatchUserDynamicPlayback)
	authd.POST("/user-dynamics/:id/like", a.ToggleDynamicLike)
	authd.DELETE("/dynamic-comments/:id", a.DeleteDynamicComment)
	authd.POST("/dynamic-comments/:id/like", a.ToggleDynamicCommentLike)
	authd.POST("/dynamic-comments/:id/dislike", a.ToggleDynamicCommentDislike)
	authd.POST("/dynamic-comments/:id/approve", a.ApproveDynamicComment)
	authd.POST("/dynamic-comments/:id/ignore-curated", a.IgnoreCuratedDynamicComment)
}

func registerCommentRoutes(r *gin.Engine, pub, admin, authd *gin.RouterGroup, a *API, jwtm *jwttoken.Manager) {
	pub.GET("/videos/:id/comments", middleware.OptionalJWT(jwtm), a.ListComments)
	pub.GET("/articles/:id/comments", middleware.OptionalJWT(jwtm), a.ListArticleComments)
	pub.GET("/user-dynamics/:id/comments", middleware.OptionalJWT(jwtm), a.ListDynamicComments)
	authd.POST("/user-dynamics/:id/comments", a.PostDynamicComment)
	authd.POST("/articles/:id/comments", a.PostArticleComment)
	authd.GET("/users/me/creator/comments", a.ListCreatorComments)
	authd.POST("/videos/:id/comments", a.PostComment)
	authd.DELETE("/comments/:id", a.DeleteComment)
	authd.POST("/comments/:id/pin", a.PinComment)
	authd.POST("/comments/:id/approve", a.ApproveComment)
	authd.POST("/comments/:id/ignore-curated", a.IgnoreCuratedComment)
	authd.POST("/comments/:id/like", a.ToggleLike)
	authd.POST("/comments/:id/dislike", a.ToggleDislike)
}

func registerDanmakuRoutes(r *gin.Engine, pub, admin, authd *gin.RouterGroup, a *API, jwtm *jwttoken.Manager) {
	authd.GET("/users/me/creator/danmakus", a.ListCreatorDanmakus)
	authd.DELETE("/danmakus/:id", a.DeleteDanmaku)
	authd.POST("/danmakus/:id/like", a.ToggleDanmakuLike)
	authd.POST("/videos/:id/danmaku", a.PostDanmaku)
}

func registerNotificationRoutes(r *gin.Engine, pub, admin, authd *gin.RouterGroup, a *API, jwtm *jwttoken.Manager) {
	authd.GET("/notifications/unread-summary", a.UnreadSummary)
	authd.GET("/notifications", a.ListNotifications)
	authd.GET("/notifications/:id/like-likers", a.ListNotificationLikeLikers)
	authd.PATCH("/notifications/read-by-category", a.MarkNotificationCategoryRead)
	authd.PATCH("/notifications/read-batch", a.MarkNotificationsReadBatch)
	authd.PATCH("/notifications/:id/read", a.MarkNotificationRead)
	authd.POST("/notifications/:id/mute-likes", a.MuteLikeNotification)
	authd.POST("/notifications/:id/comment-like", a.ToggleNotificationCommentLike)
	authd.POST("/notifications/:id/comment-reply", a.PostNotificationCommentReply)
	authd.DELETE("/notifications/:id", a.DeleteNotification)
}

func registerDmRoutes(r *gin.Engine, pub, admin, authd *gin.RouterGroup, a *API, jwtm *jwttoken.Manager) {
	authd.GET("/dm/conversations", a.ListDmConversations)
	authd.POST("/dm/conversations", a.CreateDmConversation)
	authd.DELETE("/dm/conversations/:id", a.DeleteDmConversation)
	authd.POST("/dm/conversations/:id/reset", a.ResetDmAgentConversation)
	authd.PATCH("/dm/conversations/:id/settings", a.PatchDmConversationSettings)
	authd.GET("/dm/conversations/:id/messages", a.ListDmMessages)
	authd.POST("/dm/conversations/:id/messages", a.PostDmMessage)
	authd.POST("/dm/agent/feedback", a.PostAgentFeedback)
}

func registerSearchRoutes(r *gin.Engine, pub, admin, authd *gin.RouterGroup, a *API, jwtm *jwttoken.Manager) {
	pub.GET("/search", a.SearchAll)
	pub.GET("/search/suggest", middleware.OptionalJWT(jwtm), a.SearchSuggest)
	authd.GET("/users/me/search-history", a.GetMySearchHistory)
	authd.PUT("/users/me/search-history", a.PutMySearchHistory)
	authd.POST("/users/me/search-history", a.PostMySearchHistory)
}

func registerBannerRoutes(r *gin.Engine, pub, admin, authd *gin.RouterGroup, a *API, jwtm *jwttoken.Manager) {
	pub.GET("/home-banners", a.ListHomeBanners)
	admin.GET("/home-banners", a.AdminListBanners)
	admin.POST("/home-banners", a.AdminCreateBanner)
	admin.POST("/home-banners/upload-image", a.AdminUploadBannerImage)
	admin.POST("/home-banners/:id/image", a.AdminUploadBannerImageByID)
	admin.PUT("/home-banners/:id", a.AdminUpdateBanner)
	admin.DELETE("/home-banners/:id", a.AdminDeleteBanner)
}

func registerHotsearchRoutes(r *gin.Engine, pub, admin, authd *gin.RouterGroup, a *API, jwtm *jwttoken.Manager) {
	pub.GET("/hot-search", a.HotSearchList)
	admin.GET("/hot-search/ops", a.AdminListHotSearchOps)
	admin.GET("/hot-search/dashboard", a.AdminHotSearchDashboard)
	admin.GET("/hot-search/preview", a.AdminPreviewHotSearch)
	admin.POST("/hot-search/ops", a.AdminCreateHotSearchOp)
	admin.POST("/hot-search/quick-op", a.AdminQuickHotSearchOp)
	admin.POST("/hot-search/reorder", a.AdminReorderHotSearch)
	admin.POST("/hot-search/display-order/reset", a.AdminResetHotSearchDisplayOrder)
	admin.POST("/hot-search/redis/remove", a.AdminRemoveHotSearchRedis)
	admin.POST("/hot-search/redis/boost", a.AdminBoostHotSearchRedis)
	admin.PUT("/hot-search/ops/:id", a.AdminUpdateHotSearchOp)
	admin.DELETE("/hot-search/ops/:id", a.AdminDeleteHotSearchOp)
}

func registerAdminRoutes(r *gin.Engine, pub, admin, authd *gin.RouterGroup, a *API, jwtm *jwttoken.Manager) {
	pub.POST("/admin/auth/login", a.AdminLogin)
	pub.POST("/admin/auth/refresh", a.AdminRefresh)
	admin.GET("/me", a.AdminMe)
	admin.GET("/agent-settings", a.AdminGetAgentSettings)
	admin.GET("/agent-feedbacks", a.AdminListAgentFeedbacks)
	admin.PUT("/agent-settings", a.AdminPutAgentSettings)
	admin.POST("/agent-settings/avatar", a.AdminUploadAgentAvatar)
	admin.GET("/agent-profiles", a.AdminListAgentProfiles)
	admin.POST("/agent-profiles", a.AdminCreateAgentProfile)
	admin.PUT("/agent-profiles/:id", a.AdminUpdateAgentProfile)
	admin.DELETE("/agent-profiles/:id", a.AdminDeleteAgentProfile)
	admin.POST("/agent-profiles/:id/avatar", a.AdminUploadAgentProfileAvatar)
	admin.GET("/system-configs", a.AdminListSystemConfigs)
	admin.PUT("/system-configs", a.AdminUpdateSystemConfig)
}

func registerStatsRoutes(r *gin.Engine, pub, admin, authd *gin.RouterGroup, a *API, jwtm *jwttoken.Manager) {
	pub.GET("/stats/home", a.HomeStats)
}

func registerWsRoutes(r *gin.Engine, pub, admin, authd *gin.RouterGroup, a *API, jwtm *jwttoken.Manager) {
	r.GET("/api/v1/ws/danmaku", a.ServeDanmaku)
	r.GET("/api/v1/ws/chat", a.ServeChat)
}
