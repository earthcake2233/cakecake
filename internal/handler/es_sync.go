package handler

import (
	"context"
	"time"

	"go.uber.org/zap"
)

func (a *API) esIndexVideo(videoID uint64) {
	if a.ES == nil || !a.ES.Enabled() {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err := a.ES.IndexVideoFromDB(ctx, a.DB, videoID); err != nil {
			a.Log.Warn("elasticsearch index video", zap.Uint64("video_id", videoID), zap.Error(err))
		}
	}()
}

func (a *API) esDeleteVideo(videoID uint64) {
	if a.ES == nil || !a.ES.Enabled() {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := a.ES.DeleteVideo(ctx, videoID); err != nil {
			a.Log.Warn("elasticsearch delete video", zap.Uint64("video_id", videoID), zap.Error(err))
		}
	}()
}

func (a *API) esIndexArticle(articleID uint64) {
	if a.ES == nil || !a.ES.Enabled() {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err := a.ES.IndexArticleFromDB(ctx, a.DB, articleID); err != nil {
			a.Log.Warn("elasticsearch index article", zap.Uint64("article_id", articleID), zap.Error(err))
		}
	}()
}

func (a *API) esDeleteArticle(articleID uint64) {
	if a.ES == nil || !a.ES.Enabled() {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := a.ES.DeleteArticle(ctx, articleID); err != nil {
			a.Log.Warn("elasticsearch delete article", zap.Uint64("article_id", articleID), zap.Error(err))
		}
	}()
}

func (a *API) esIndexUser(userID uint64) {
	if a.ES == nil || !a.ES.Enabled() {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err := a.ES.IndexUserFromDB(ctx, a.DB, userID); err != nil {
			a.Log.Warn("elasticsearch index user", zap.Uint64("user_id", userID), zap.Error(err))
		}
	}()
}
