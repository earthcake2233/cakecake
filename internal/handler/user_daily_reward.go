package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"cakecake/internal/errcode"
	"cakecake/internal/middleware"
	"cakecake/internal/pkg/resp"
)

// GetMeDailyRewards returns today's daily-reward task state (personal center home).
func (a *API) GetMeDailyRewards(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	if a.DailyRewardSvc == nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	_ = a.DailyRewardSvc.MarkLogin(uid)
	snap, err := a.DailyRewardSvc.BuildSnapshot(uid)
	if err != nil {
		a.Log.Error("daily rewards snapshot", zap.Error(err))
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	resp.OK(c, snap)
}

// PostMeDailyRewardWatch marks the daily watch-video task complete (e.g. after ≥60s playback).
func (a *API) PostMeDailyRewardWatch(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	if err := a.DailyRewardSvc.MarkWatch(uid); err != nil {
		a.Log.Error("daily reward watch", zap.Error(err))
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	snap, err := a.DailyRewardSvc.BuildSnapshot(uid)
	if err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	resp.OK(c, snap)
}
