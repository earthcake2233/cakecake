package handler

import (
	"cakecake/internal/model/article"
	"cakecake/internal/model/extra"
	"cakecake/internal/model/user"
	vmodel "cakecake/internal/model/video"
	"context"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"cakecake/internal/errcode"
	"cakecake/internal/middleware"
	"cakecake/internal/pkg/resp"
)

type viewHistoryItemDTO struct {
	MediaType         string  `json:"media_type"`
	VideoID           uint64  `json:"video_id"`
	ArticleID         uint64  `json:"article_id"`
	Title             string  `json:"title"`
	CoverURL          string  `json:"cover_url"`
	DurationSec       float64 `json:"duration_sec"`
	ProgressSec       float64 `json:"progress_sec"`
	Device            string  `json:"device"`
	ViewedAt          string  `json:"viewed_at"`
	ViewedTime        string  `json:"viewed_time"`
	UploaderID        uint64  `json:"uploader_id"`
	UploaderName      string  `json:"uploader_name,omitempty"`
	UploaderAvatarURL string  `json:"uploader_avatar_url,omitempty"`
	Category          string  `json:"category"`
}

type viewHistoryRecordResponse struct {
	Recorded bool `json:"recorded"`
	Paused   bool `json:"paused,omitempty"`
}

type viewHistoryListResponse struct {
	Items  []viewHistoryItemDTO `json:"items"`
	Paused bool                 `json:"paused"`
}

type viewHistoryClearedResponse struct {
	Cleared bool `json:"cleared"`
}

type pausedResponse struct {
	Paused bool `json:"paused"`
}

const viewHistoryMaxItems = 500

type viewHistoryPostJSON struct {
	ProgressSec float64 `json:"progress_sec"`
	DurationSec float64 `json:"duration_sec"`
	Device      string  `json:"device"`
}

type viewHistorySettingsJSON struct {
	Paused bool `json:"paused"`
}

// PostVideoViewHistory upserts watch progress for the account history page.
func (a *API) PostVideoViewHistory(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	paused, err := a.ViewHistorySvc.GetUserViewHistoryPaused(context.Background(), uid)
	if err != nil {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	if paused {
		resp.OK(c, viewHistoryRecordResponse{Recorded: false, Paused: true})
		return
	}
	vid, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || vid == 0 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	v, ok := loadPublishedVideo(a, vid)
	if !ok {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	var body viewHistoryPostJSON
	_ = c.ShouldBindJSON(&body)
	device := strings.TrimSpace(body.Device)
	if device != "mobile" {
		device = "web"
	}
	prog := body.ProgressSec
	if prog < 0 {
		prog = 0
	}
	dur := body.DurationSec
	if dur <= 0 {
		dur = v.DurationSec
	}
	if dur > 0 && prog > dur {
		prog = dur
	}
	if err := a.ViewHistorySvc.RecordVideoViewHistoryWithProgress(context.Background(), uid, vid, prog, dur, device, time.Now()); err != nil {
		a.Log.Error("record view history", zap.Error(err))
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	a.ViewHistorySvc.TrimViewHistoryCombined(context.Background(), uid, viewHistoryMaxItems)
	resp.OK(c, viewHistoryRecordResponse{Recorded: true})
}

// RecordArticleViewHistory upserts read history for a published article.
func (a *API) RecordArticleViewHistory(uid, articleID uint64, device string) {
	a.ViewHistorySvc.RecordArticleViewHistory(context.Background(), uid, articleID, device)
}

// ListMyViewHistory returns watch history for the personal-center page.
// ListMyViewHistory godoc
// @Summary      Get view history
// @Description  Get paginated video viewing history
// @Tags         Users
// @Produce      json
// @Param        q query string false "Search keyword"
// @Success      200 {object} map[string]interface{}
// @Router       /users/me/view-history [get]
func (a *API) ListMyViewHistory(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	paused, err := a.ViewHistorySvc.GetViewHistorySettings(context.Background(), uid)
	if err != nil {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	keyword := strings.TrimSpace(c.Query("q"))

	vrows, err := a.ViewHistorySvc.ListVideoViewHistory(context.Background(), uid, keyword)
	if err != nil {
		a.Log.Error("list video view history", zap.Error(err))
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	arows, err := a.ViewHistorySvc.ListArticleViewHistory(context.Background(), uid, keyword)
	if err != nil {
		a.Log.Error("list article view history", zap.Error(err))
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	items := append(a.viewHistoryVideoItems(vrows), a.viewHistoryArticleItems(arows)...)
	sort.Slice(items, func(i, j int) bool {
		ti := items[i].ViewedAt
		tj := items[j].ViewedAt
		return ti > tj
	})
	limit := viewHistoryMaxItems
	if len(items) > limit {
		items = items[:limit]
	}
	resp.OK(c, viewHistoryListResponse{Items: items, Paused: paused})
}

func (a *API) viewHistoryVideoItems(rows []extra.VideoViewHistory) []viewHistoryItemDTO {
	if len(rows) == 0 {
		return []viewHistoryItemDTO{}
	}
	vids := make([]uint64, 0, len(rows))
	for i := range rows {
		vids = append(vids, rows[i].VideoID)
	}
	videos, err := a.ViewHistorySvc.BatchFetchVideosByIDs(context.Background(), vids)
	if err != nil {
		return []viewHistoryItemDTO{}
	}
	uids := make([]uint64, 0, len(videos))
	for _, v := range videos {
		uids = append(uids, v.UserID)
	}
	users, _ := a.ViewHistorySvc.BatchFetchUsersByIDs(context.Background(), uids)
	items := make([]viewHistoryItemDTO, 0, len(rows))
	for i := range rows {
		h := rows[i]
		video, ok := videos[h.VideoID]
		if !ok || video.Status != vmodel.StatusPublished {
			continue
		}
		u := users[video.UserID]
		items = append(items, a.formatVideoHistoryItem(h, video, u))
	}
	return items
}

func (a *API) formatVideoHistoryItem(h extra.VideoViewHistory, video vmodel.Video, u user.User) viewHistoryItemDTO {
	item := viewHistoryItemDTO{
		MediaType:   "video",
		VideoID:     video.ID,
		ArticleID:   0,
		Title:       video.Title,
		CoverURL:    video.CoverURL,
		DurationSec: video.DurationSec,
		ProgressSec: h.ProgressSec,
		Device:      h.Device,
		ViewedAt:    h.ViewedAt.Format("2006-01-02 15:04:05"),
		ViewedTime:  h.ViewedAt.Format("15:04"),
		UploaderID:  video.UserID,
		Category:    video.Zone,
	}
	if u.ID > 0 {
		item.UploaderName = user.DisplayUsername(&u)
		item.UploaderAvatarURL = uploaderAvatarForAPI(&u)
	}
	return item
}

func (a *API) viewHistoryArticleItems(rows []extra.ArticleViewHistory) []viewHistoryItemDTO {
	if len(rows) == 0 {
		return []viewHistoryItemDTO{}
	}
	aids := make([]uint64, 0, len(rows))
	for i := range rows {
		aids = append(aids, rows[i].ArticleID)
	}
	articles, err := a.ViewHistorySvc.BatchFetchArticlesByIDs(context.Background(), aids)
	if err != nil {
		return []viewHistoryItemDTO{}
	}
	uids := make([]uint64, 0, len(articles))
	for _, art := range articles {
		uids = append(uids, art.UserID)
	}
	users, _ := a.ViewHistorySvc.BatchFetchUsersByIDs(context.Background(), uids)
	items := make([]viewHistoryItemDTO, 0, len(rows))
	for i := range rows {
		h := rows[i]
		art, ok := articles[h.ArticleID]
		if !ok || art.Status != article.StatusPublished {
			continue
		}
		u := users[art.UserID]
		items = append(items, viewHistoryItemDTO{
			MediaType:         "article",
			VideoID:           0,
			ArticleID:         art.ID,
			Title:             art.Title,
			CoverURL:          art.CoverURL,
			DurationSec:       0,
			ProgressSec:       0,
			Device:            h.Device,
			ViewedAt:          h.ViewedAt.Format("2006-01-02 15:04:05"),
			ViewedTime:        h.ViewedAt.Format("15:04"),
			UploaderID:        art.UserID,
			UploaderName:      user.DisplayUsername(&u),
			UploaderAvatarURL: uploaderAvatarForAPI(&u),
			Category:          "专栏",
		})
	}
	return items
}

// DeleteMyViewHistoryEntry removes one video history row.
func (a *API) DeleteMyViewHistoryEntry(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	vid, err := strconv.ParseUint(c.Param("videoId"), 10, 64)
	if err != nil || vid == 0 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	if err := a.ViewHistorySvc.DeleteVideoHistoryByVideo(context.Background(), uid, vid); err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	resp.OK(c, deletedResponse{Deleted: true})
}

// DeleteMyArticleViewHistoryEntry removes one article history row.
func (a *API) DeleteMyArticleViewHistoryEntry(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	aid, err := strconv.ParseUint(c.Param("articleId"), 10, 64)
	if err != nil || aid == 0 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	if err := a.ViewHistorySvc.DeleteArticleHistoryByArticle(context.Background(), uid, aid); err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	resp.OK(c, deletedResponse{Deleted: true})
}

// ClearMyViewHistory removes all history for the user.
func (a *API) ClearMyViewHistory(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	if err := a.ViewHistorySvc.ClearViewHistory(context.Background(), uid); err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	if err := a.ViewHistorySvc.ClearArticleViewHistory(context.Background(), uid); err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	resp.OK(c, viewHistoryClearedResponse{Cleared: true})
}

// GetMyViewHistorySettings returns whether history recording is paused.
// GetMyViewHistorySettings godoc
// @Summary      Get view history settings
// @Description  Get privacy/sharing settings for view history
// @Tags         Users
// @Produce      json
// @Success      200 {object} map[string]interface{}
// @Router       /users/me/view-history/settings [get]
func (a *API) GetMyViewHistorySettings(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	paused, err := a.ViewHistorySvc.GetViewHistorySettings(context.Background(), uid)
	if err != nil {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	resp.OK(c, pausedResponse{Paused: paused})
}

// PutMyViewHistorySettings toggles pause for history recording.
func (a *API) PutMyViewHistorySettings(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	var body viewHistorySettingsJSON
	if err := c.ShouldBindJSON(&body); err != nil {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	if err := a.ViewHistorySvc.UpdateViewHistorySettings(context.Background(), uid, body.Paused); err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	resp.OK(c, pausedResponse{Paused: body.Paused})
}
