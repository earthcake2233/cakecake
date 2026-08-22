package handler

import (
	"context"

	"go.uber.org/zap"

	"cakecake/internal/search"
)

func (a *API) esIndexVideo(videoID uint64) {
	if a.SearchSvc == nil || !a.SearchSvc.Enabled() {
		return
	}
	if err := search.EnqueueSyncJob(context.Background(), a.DB, search.SyncJobEntityVideo, videoID, search.SyncJobActionUpsert); err != nil && a.Log != nil {
		a.Log.Warn("enqueue es index video", zap.Uint64("video_id", videoID), zap.Error(err))
	}
}

func (a *API) esDeleteVideo(videoID uint64) {
	if a.SearchSvc == nil || !a.SearchSvc.Enabled() {
		return
	}
	if err := search.EnqueueSyncJob(context.Background(), a.DB, search.SyncJobEntityVideo, videoID, search.SyncJobActionDelete); err != nil && a.Log != nil {
		a.Log.Warn("enqueue es delete video", zap.Uint64("video_id", videoID), zap.Error(err))
	}
}

func (a *API) esIndexArticle(articleID uint64) {
	if a.SearchSvc == nil || !a.SearchSvc.Enabled() {
		return
	}
	if err := search.EnqueueSyncJob(context.Background(), a.DB, search.SyncJobEntityArticle, articleID, search.SyncJobActionUpsert); err != nil && a.Log != nil {
		a.Log.Warn("enqueue es index article", zap.Uint64("article_id", articleID), zap.Error(err))
	}
}

func (a *API) esDeleteArticle(articleID uint64) {
	if a.SearchSvc == nil || !a.SearchSvc.Enabled() {
		return
	}
	if err := search.EnqueueSyncJob(context.Background(), a.DB, search.SyncJobEntityArticle, articleID, search.SyncJobActionDelete); err != nil && a.Log != nil {
		a.Log.Warn("enqueue es delete article", zap.Uint64("article_id", articleID), zap.Error(err))
	}
}

func (a *API) esIndexUser(userID uint64) {
	if a.SearchSvc == nil || !a.SearchSvc.Enabled() {
		return
	}
	if err := search.EnqueueSyncJob(context.Background(), a.DB, search.SyncJobEntityUser, userID, search.SyncJobActionUpsert); err != nil && a.Log != nil {
		a.Log.Warn("enqueue es index user", zap.Uint64("user_id", userID), zap.Error(err))
	}
}
