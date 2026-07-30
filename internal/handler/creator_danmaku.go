package handler

import (
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"minibili/internal/errcode"
	"minibili/internal/middleware"
	"minibili/internal/pkg/resp"
	"minibili/internal/service"
)

const creatorDanmakuMaxList = 1000

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

	items := make([]gin.H, 0, len(result.Items))
	for _, d := range result.Items {
		items = append(items, gin.H{
			"id":          d.ID,
			"video_id":    d.VideoID,
			"user_id":     d.UserID,
			"username":    d.Username,
			"content":     d.Content,
			"color":       d.Color,
			"type":        d.Type,
			"type_label":  danmakuTypeLabel(d.Type),
			"video_time":  d.VideoTime,
			"play_time":   formatDanmakuPlayTime(d.VideoTime),
			"like_count":  d.LikeCount,
			"liked_by_me": d.LikedByMe,
			"created_at":  d.CreatedAt,
			"video": gin.H{
				"id":        d.VideoID,
				"title":     d.VideoTitle,
				"cover_url": d.CoverURL,
			},
		})
	}
	resp.OK(c, gin.H{
		"items": items,
		"total": result.Total,
		"limit": result.Limit,
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
	a.Hub.BroadcastJSON(d.VideoID, gin.H{
		"type":       "danmaku_deleted",
		"danmaku_id": strconv.FormatUint(did, 10),
	})
	resp.OK(c, gin.H{"id": did})
}

func danmakuTypeLabel(t string) string {
	switch strings.TrimSpace(t) {
	case "top":
		return "??"
	case "bottom":
		return "??"
	default:
		return "??"
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
