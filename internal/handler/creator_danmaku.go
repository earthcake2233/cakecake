package handler

import (
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"minibili/internal/errcode"
	"minibili/internal/middleware"
	"minibili/internal/model"
	"minibili/internal/pkg/resp"
)

const creatorDanmakuMaxList = 1000

// ListCreatorDanmakus lists danmaku on the authenticated uploader's videos (创作中心 · 弹幕管理).
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

	base := a.DB.Model(&model.Danmaku{}).
		Joins("INNER JOIN videos ON videos.id = danmakus.video_id AND videos.user_id = ?", uid)
	if filterVideoID > 0 {
		var owned model.Video
		if err := a.DB.Where("id = ? AND user_id = ?", filterVideoID, uid).First(&owned).Error; err != nil {
			resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
			return
		}
		base = base.Where("danmakus.video_id = ?", filterVideoID)
	}
	if keyword != "" {
		like := "%" + keyword + "%"
		base = base.Where("danmakus.content LIKE ?", like)
	}
	switch typeFilter {
	case "scroll", "top", "bottom":
		base = base.Where("danmakus.type = ?", typeFilter)
	}

	var total int64
	if err := base.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}

	var list []model.Danmaku
	if err := base.Order("danmakus.id DESC").Limit(limit).Find(&list).Error; err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}

	videoIDs := make([]uint64, 0, len(list))
	userIDs := make([]uint64, 0, len(list))
	for _, d := range list {
		videoIDs = append(videoIDs, d.VideoID)
		userIDs = append(userIDs, d.UserID)
	}
	videos := map[uint64]model.Video{}
	if len(videoIDs) > 0 {
		var vlist []model.Video
		_ = a.DB.Where("id IN ?", videoIDs).Find(&vlist).Error
		for i := range vlist {
			videos[vlist[i].ID] = vlist[i]
		}
	}
	names := map[uint64]string{}
	if len(userIDs) > 0 {
		var users []model.User
		_ = a.DB.Where("id IN ?", userIDs).Find(&users).Error
		for i := range users {
			names[users[i].ID] = model.DisplayUsername(&users[i])
		}
	}

	danmakuIDs := make([]uint64, 0, len(list))
	for _, d := range list {
		danmakuIDs = append(danmakuIDs, d.ID)
	}
	likedByViewer := map[uint64]bool{}
	if viewerID, ok := middleware.UserID(c); ok && viewerID > 0 && len(danmakuIDs) > 0 {
		var likes []model.DanmakuLike
		_ = a.DB.Where("user_id = ? AND danmaku_id IN ?", viewerID, danmakuIDs).Find(&likes).Error
		for _, lk := range likes {
			likedByViewer[lk.DanmakuID] = true
		}
	}

	items := make([]gin.H, 0, len(list))
	for _, d := range list {
		v := videos[d.VideoID]
		items = append(items, gin.H{
			"id":          d.ID,
			"video_id":    d.VideoID,
			"user_id":     d.UserID,
			"username":    names[d.UserID],
			"content":     d.Content,
			"color":       d.Color,
			"type":        d.Type,
			"type_label":  danmakuTypeLabel(d.Type),
			"video_time":  d.VideoTime,
			"play_time":    formatDanmakuPlayTime(d.VideoTime),
			"like_count":   d.LikeCount,
			"liked_by_me":  likedByViewer[d.ID],
			"created_at":   d.CreatedAt.Format("2006-01-02 15:04:05"),
			"video": gin.H{
				"id":        v.ID,
				"title":     v.Title,
				"cover_url": v.CoverURL,
			},
		})
	}
	if total > int64(limit) {
		total = int64(limit)
	}
	resp.OK(c, gin.H{
		"items": items,
		"total": total,
		"limit": limit,
	})
}

// DeleteDanmaku removes one danmaku on the uploader's video (创作中心).
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
	var d model.Danmaku
	if err := a.DB.First(&d, did).Error; err != nil {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	var v model.Video
	if err := a.DB.First(&v, d.VideoID).Error; err != nil {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	if v.UserID != uid {
		resp.Err(c, http.StatusForbidden, errcode.CodeForbidden)
		return
	}
	tx := a.DB.Begin()
	if err := tx.Where("danmaku_id = ?", did).Delete(&model.DanmakuLike{}).Error; err != nil {
		tx.Rollback()
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	if err := tx.Delete(&d).Error; err != nil {
		tx.Rollback()
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	if err := tx.Model(&model.Video{}).Where("id = ?", v.ID).
		UpdateColumn("danmaku_count", gorm.Expr("GREATEST(danmaku_count - ?, 0)", 1)).Error; err != nil {
		tx.Rollback()
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	if err := tx.Commit().Error; err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	a.Hub.BroadcastJSON(v.ID, gin.H{
		"type":       "danmaku_deleted",
		"danmaku_id": strconv.FormatUint(did, 10),
	})
	resp.OK(c, gin.H{"id": did})
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
