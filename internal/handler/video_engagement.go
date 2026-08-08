package handler

import (
	"cakecake/internal/errcode"
	"cakecake/internal/middleware"
	"cakecake/internal/pkg/resp"
	"cakecake/internal/pkg/usercoin"
	"cakecake/internal/service/engagement"
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type watchLaterToggleResponse struct {
	InWatchLater bool `json:"in_watch_later"`
}

type watchLaterListResponse struct {
	Items    []engagement.WatchLaterVideoItem `json:"items"`
	Total    int64                            `json:"total"`
	MaxLimit int                              `json:"max_limit"`
}

type coinVideoItemDTO struct {
	ID                uint64  `json:"id"`
	Title             string  `json:"title"`
	CoverURL          string  `json:"cover_url"`
	PlayCount         uint64  `json:"play_count"`
	DanmakuCount      uint64  `json:"danmaku_count"`
	CommentCount      uint64  `json:"comment_count"`
	Duration          float64 `json:"duration"`
	Uploader          string  `json:"uploader"`
	UploaderAvatarURL string  `json:"uploader_avatar_url"`
	CreatedAt         string  `json:"created_at"`
	CoinedAt          string  `json:"coined_at"`
}

type coinListResponse struct {
	Items    []coinVideoItemDTO `json:"items"`
	Total    int64              `json:"total"`
	MaxLimit int                `json:"max_limit"`
}

type msgOKResponse struct {
	Msg string `json:"msg"`
}

type videoCoinJSON struct {
	Amount int `json:"amount"`
}

// PostVideoCoin adds 1 or 2 coins from the current user (max 2 per video; second visit adds 1 only).
// PostVideoCoin godoc
// @Summary      Coin a video
// @Description  Award coins to a video (1 or 2 coins)
// @Tags         Videos
// @Produce      json
// @Param        id path int true "Video ID"
// @Param        body body object{coins=int} true "Number of coins (1 or 2)"
// @Success      200 {object} map[string]interface{}
// @Router       /videos/{id}/coin [post]
func (a *API) PostVideoCoin(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	vid, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || vid == 0 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	v, ok := loadPublishedVideo(c.Request.Context(), a, vid)
	if !ok {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	if v.UserID == uid {
		resp.Err(c, http.StatusBadRequest, errcode.CodeCannotCoinSelf)
		return
	}
	var body videoCoinJSON
	_ = c.ShouldBindJSON(&body)
	amount := body.Amount
	if amount != 1 && amount != 2 {
		amount = 1
	}
	result, err := a.EngagementSvc.PostVideoCoin(c.Request.Context(), uid, vid, v.UserID, amount)
	if err != nil {
		if errors.Is(err, usercoin.ErrInsufficientCoins) {
			resp.Err(c, http.StatusBadRequest, errcode.CodeInsufficientCoins)
			return
		}
		a.Log.Error("post video coin", zap.Error(err))
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	if result == nil {
		resp.Err(c, http.StatusBadRequest, errcode.CodeAlreadyCoined)
		return
	}
	resp.OK(c, articleCoinResponse{
		Coined:               true,
		CoinCount:            result.CoinCount,
		Amount:               result.Amount,
		MyCoinAmount:         result.MyCoinAmount,
		CoinedByMe:           true,
		CoinBalance:          result.CoinBalance,
		DailyCoinExpProgress: result.DailyProgress,
		DailyCoinExpMax:      result.DailyMax,
	})
}

// ToggleWatchLater toggles the current user's watch-later entry for a published video.
// ToggleWatchLater godoc
// @Summary      Add/remove watch later
// @Description  Toggle watch later status for a video
// @Tags         Videos
// @Produce      json
// @Param        id path int true "Video ID"
// @Success      200 {object} map[string]interface{}
// @Router       /videos/{id}/watch-later [post]
func (a *API) ToggleWatchLater(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	vid, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || vid == 0 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	if _, ok := loadPublishedVideo(c.Request.Context(), a, vid); !ok {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	added, err := a.EngagementSvc.ToggleWatchLater(c.Request.Context(), uid, vid)
	if err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	resp.OK(c, watchLaterToggleResponse{InWatchLater: added})
}

const watchLaterMaxItems = 100

// ListMyWatchLater returns the caller's watch-later queue (newest first).
// ListMyWatchLater godoc
// @Summary      List watch later
// @Description  Get paginated list of videos saved for later
// @Tags         Videos
// @Produce      json
// @Param        page query int false "Page number" default(1)
// @Param        page_size query int false "Page size" default(20)
// @Success      200 {object} map[string]interface{}
// @Router       /users/me/watch-later [get]
func (a *API) ListMyWatchLater(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	page, pageSize := parsePagination(c, 20)
	items, total, err := a.EngagementSvc.ListWatchLaterWithVideos(c.Request.Context(), uid, page, pageSize)
	if err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	resp.OK(c, watchLaterListResponse{Items: items, Total: total, MaxLimit: watchLaterMaxItems})
}

// ClearMyWatchLater removes all watch-later entries for the current user.
// ClearMyWatchLater godoc
// @Summary      Clear watch later
// @Description  Remove all videos from watch later list
// @Tags         Videos
// @Produce      json
// @Success      200 {object} map[string]interface{}
// @Router       /users/me/watch-later [delete]
func (a *API) ClearMyWatchLater(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	if err := a.EngagementSvc.ClearWatchLater(c.Request.Context(), uid); err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	resp.OK(c, msgOKResponse{Msg: "ok"})
}

// ClearWatchedWatchLater removes watched entries from the user's watch-later queue.
// ClearWatchedWatchLater godoc
// @Summary      Clear watched from watch later
// @Description  Remove already-watched videos from watch later
// @Tags         Videos
// @Produce      json
// @Success      200 {object} map[string]interface{}
// @Router       /users/me/watch-later/watched [delete]
func (a *API) ClearWatchedWatchLater(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	if err := a.EngagementSvc.ClearWatchedWatchLater(c.Request.Context(), uid); err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	resp.OK(c, msgOKResponse{Msg: "ok"})
}

// MarkWatchLaterWatched marks a watch-later item as watched.
// MarkWatchLaterWatched godoc
// @Summary      Mark watch later as watched
// @Description  Mark a specific watch later item as watched
// @Tags         Videos
// @Produce      json
// @Param        id path int true "Video ID"
// @Success      200 {object} map[string]interface{}
// @Router       /users/me/watch-later/{id}/watched [post]
func (a *API) MarkWatchLaterWatched(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	vid, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || vid == 0 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	if err := a.EngagementSvc.MarkWatchLaterWatched(c.Request.Context(), uid, vid); err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	resp.OK(c, msgOKResponse{Msg: "ok"})
}

func (a *API) coinRecentListItems(ctx context.Context, ownerID uint64, limit int) ([]coinVideoItemDTO, int64, error) {
	items, total, err := a.EngagementSvc.ListUserCoinedVideos(ctx, ownerID, limit)
	if err != nil {
		return nil, 0, err
	}
	result := make([]coinVideoItemDTO, 0, len(items))
	for _, item := range items {
		result = append(result, coinVideoItemDTO{
			ID: item.ID, Title: item.Title, CoverURL: item.CoverURL,
			PlayCount: item.PlayCount, DanmakuCount: item.DanmakuCount,
			CommentCount: item.CommentCount, Duration: item.Duration,
			Uploader: item.UploaderName, UploaderAvatarURL: item.UploaderAvatar,
			CreatedAt: item.CreatedAt, CoinedAt: item.CoinedAt,
		})
	}
	return result, total, nil
}

// ListUserRecentCoinVideos returns videos the user recently coined (owner-only).
func (a *API) ListUserRecentCoinVideos(c *gin.Context) {
	ownerID, err := strconv.ParseUint(c.Param("userId"), 10, 64)
	if err != nil || ownerID == 0 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	up, err := a.UserSvc.GetUserPublic(c.Request.Context(), ownerID)
	if err != nil {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	viewer, viewerOK := middleware.UserID(c)
	isOwner := viewerOK && viewer == ownerID
	if !isOwner && !up.PrivacyPublicRecentCoins {
		resp.OK(c, coinListResponse{Items: []coinVideoItemDTO{}, Total: 0})
		return
	}
	limit := parseLimit(c, 20, 50)
	items, total, err := a.coinRecentListItems(c, ownerID, limit)
	if err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	resp.OK(c, coinListResponse{Items: items, Total: total, MaxLimit: watchLaterMaxItems})
}
