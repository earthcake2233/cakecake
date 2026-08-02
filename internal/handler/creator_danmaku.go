package handler

import (
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"cakecake/internal/errcode"
	"cakecake/internal/middleware"
	"cakecake/internal/pkg/resp"
	"cakecake/internal/service"
)

const creatorDanmakuMaxList = 1000

type danmakuDeletedEvent struct {
	Type      string `json:"type"`
	DanmakuID string `json:"danmaku_id"`
}

type danmakuDeleteResponse struct {
	ID uint64 `json:"id"`
}

// ListCreatorDanmakus lists danmaku on the authenticated uploader's videos (???? ? ????).
func (a *API) ListCreatorDanmakus(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	limit := queryIntDefault(c.Query("limit"), creatorDanmakuMaxList)
	if limit < 1 {
		limit = creatorDanmakuMaxList
	}
	if limit > creatorDanmakuMaxList {
		limit = creatorDanmakuMaxList
	}
	keyword := strings.TrimSpace(c.Query("q"))
	typeFilter := strings.TrimSpace(c.Query("type"))
	var filterVideoID uint64
	if v := strings.TrimSpace(c.Query("video_id")); v != "" {
		n, err := strconv.ParseUint(v, 10, 64)
		if err != nil || n == 0 {
			resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
			return
		}
		filterVideoID = n
	}
	viewerID, _ := middleware.UserID(c)

	result, err := a.DanmakuSvc.ListCreatorDanmakus(c.Request.Context(), uid, limit, keyword, typeFilter, filterVideoID, viewerID)
	if err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}

	type creatorDanmakuVideo struct {
		ID       uint64 `json:"id"`
		Title    string `json:"title"`
		CoverURL string `json:"cover_url"`
	}
	type creatorDanmakuItem struct {
		ID        uint64              `json:"id"`
		VideoID   uint64              `json:"video_id"`
		UserID    uint64              `json:"user_id"`
		Username  string              `json:"username"`
		Content   string              `json:"content"`
		Color     string              `json:"color"`
		Type      string              `json:"type"`
		TypeLabel string              `json:"type_label"`
		VideoTime float64             `json:"video_time"`
		PlayTime  string              `json:"play_time"`
		LikeCount int64               `json:"like_count"`
		LikedByMe bool                `json:"liked_by_me"`
		CreatedAt string              `json:"created_at"`
		Video     creatorDanmakuVideo `json:"video"`
	}
	type creatorDanmakuListResponse struct {
		Items []creatorDanmakuItem `json:"items"`
		Total int64                `json:"total"`
		Limit int                  `json:"limit"`
	}
	items := make([]creatorDanmakuItem, 0, len(result.Items))
	for _, d := range result.Items {
		items = append(items, creatorDanmakuItem{
			ID:        d.ID,
			VideoID:   d.VideoID,
			UserID:    d.UserID,
			Username:  d.Username,
			Content:   d.Content,
			Color:     d.Color,
			Type:      d.Type,
			TypeLabel: danmakuTypeLabel(d.Type),
			VideoTime: d.VideoTime,
			PlayTime:  formatDanmakuPlayTime(d.VideoTime),
			LikeCount: d.LikeCount,
			LikedByMe: d.LikedByMe,
			CreatedAt: d.CreatedAt,
			Video: creatorDanmakuVideo{
				ID:       d.VideoID,
				Title:    d.VideoTitle,
				CoverURL: d.CoverURL,
			},
		})
	}
	resp.OK(c, creatorDanmakuListResponse{
		Items: items,
		Total: result.Total,
		Limit: result.Limit,
	})
}

// DeleteDanmaku removes one danmaku on the uploader's video (????).
func (a *API) DeleteDanmaku(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	did, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || did == 0 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	d, err := a.DanmakuSvc.DeleteCreatorDanmaku(c.Request.Context(), uid, did)
	if err != nil {
		if err == service.ErrForbidden {
			resp.Err(c, http.StatusForbidden, errcode.CodeForbidden)
		} else {
			resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		}
		return
	}
	a.Hub.BroadcastJSON(d.VideoID, danmakuDeletedEvent{
		Type:      "danmaku_deleted",
		DanmakuID: strconv.FormatUint(did, 10),
	})
	resp.OK(c, danmakuDeleteResponse{ID: did})
}

func danmakuTypeLabel(t string) string {
	switch strings.TrimSpace(t) {
	case "top":
		return "顶部"
	case "bottom":
		return "底部"
	default:
		return "普通"
	}
}

func formatDanmakuPlayTime(sec float64) string {
	if sec < 0 || math.IsNaN(sec) || math.IsInf(sec, 0) {
		sec = 0
	}
	s := int(sec)
	if s < 0 {
		s = 0
	}
	return fmt.Sprintf("%02d:%02d", s/60, s%60)
}
