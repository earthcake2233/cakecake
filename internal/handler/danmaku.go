package handler

import (
	"context"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"minibili/internal/data"
	"minibili/internal/errcode"
	"minibili/internal/middleware"
	"minibili/internal/model"
	"minibili/internal/pkg/resp"
	"minibili/internal/pkg/sensitive"
)

var danmakuColorHexRe = regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)

var allowedDanmakuTypes = map[string]struct{}{
	"scroll": {}, "top": {}, "bottom": {},
}

func normalizeDanmakuFontSize(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "sm", "small":
		return "sm"
	case "lg", "large":
		return "lg"
	default:
		return "md"
	}
}

func danmakuFontSizeField(d model.Danmaku) string {
	if fs := strings.TrimSpace(d.FontSize); fs != "" {
		return normalizeDanmakuFontSize(fs)
	}
	return "md"
}

type danmakuPost struct {
	Content   string  `json:"content"`
	Color     string  `json:"color"`
	Type      string  `json:"type"`
	FontSize  string  `json:"font_size"`
	VideoTime float64 `json:"video_time"`
}

// PostDanmaku persists and broadcasts a danmaku (F5, S-007, S-014).
func (a *API) PostDanmaku(c *gin.Context) {
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
	var v model.Video
	if err := a.DB.First(&v, vid).Error; err != nil || v.Status != "published" {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	if v.DanmakuClosed {
		resp.Err(c, http.StatusForbidden, errcode.CodeDanmakuClosed)
		return
	}
	var req danmakuPost
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	if _, ok := allowedDanmakuTypes[strings.TrimSpace(req.Type)]; !ok {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	content := strings.TrimSpace(req.Content)
	if n := utf8.RuneCountInString(content); n < 1 || n > 100 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}

	ctx := context.Background()
	key := data.DanmakuCooldownKey(uid, vid)
	okSet, err := a.Redis.SetNX(ctx, key, "1", 5*time.Second).Result()
	if err != nil {
		a.Log.Error("redis cooldown", zap.Error(err))
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	if !okSet {
		resp.Err(c, http.StatusBadRequest, errcode.CodeDanmakuCooldown)
		return
	}

	// S-014：空或未提供 color → #FFFFFF；否则须匹配 ^#[0-9A-Fa-f]{6}$，存储大写
	colorRaw := strings.TrimSpace(req.Color)
	var color string
	if colorRaw == "" {
		color = "#FFFFFF"
	} else if !danmakuColorHexRe.MatchString(colorRaw) {
		_, _ = a.Redis.Del(ctx, key).Result()
		resp.Err(c, http.StatusBadRequest, errcode.CodeInvalidColor)
		return
	} else {
		color = strings.ToUpper(colorRaw)
	}

	if err := a.Sens.Check(content); err != nil {
		_, _ = a.Redis.Del(ctx, key).Result()
		if _, ok := err.(sensitive.ErrBlocked); ok {
			resp.Err(c, http.StatusBadRequest, errcode.CodeDanmakuSensitive)
			return
		}
		a.Log.Error("sensitive check", zap.Error(err))
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}

	fontSize := normalizeDanmakuFontSize(req.FontSize)
	d := model.Danmaku{
		VideoID:   vid,
		UserID:    uid,
		Content:   content,
		Color:     color,
		Type:      strings.TrimSpace(req.Type),
		FontSize:  fontSize,
		VideoTime: req.VideoTime,
	}
	if err := a.DB.Create(&d).Error; err != nil {
		_, _ = a.Redis.Del(ctx, key).Result()
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	_ = a.DB.Model(&model.Video{}).Where("id = ?", vid).UpdateColumn("danmaku_count", gorm.Expr("danmaku_count + ?", 1)).Error
	var u model.User
	_ = a.DB.First(&u, uid).Error
	payload := gin.H{
		"type": "danmaku",
		"data": gin.H{
			"id":         d.ID,
			"content":    d.Content,
			"color":      d.Color,
			"type":       d.Type,
			"font_size":  fontSize,
			"video_time": d.VideoTime,
			"user":       model.DisplayUsername(&u),
			"created_at": d.CreatedAt.Format("2006-01-02 15:04:05"),
		},
	}
	if a.DanmakuRelay != nil {
		if err := a.DanmakuRelay.Publish(ctx, vid, payload); err != nil {
			a.Log.Error("danmaku relay publish", zap.Error(err))
		}
	} else {
		a.Hub.BroadcastJSON(vid, payload)
	}
	resp.OK(c, gin.H{
		"id":         d.ID,
		"content":    d.Content,
		"color":      d.Color,
		"type":       d.Type,
		"font_size":  fontSize,
		"video_time": d.VideoTime,
		"user":       model.DisplayUsername(&u),
		"created_at": d.CreatedAt.Format("2006-01-02 15:04:05"),
	})
}

// ToggleDanmakuLike toggles like on a danmaku.
func (a *API) ToggleDanmakuLike(c *gin.Context) {
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
	var like model.DanmakuLike
	res := a.DB.Where("user_id = ? AND danmaku_id = ?", uid, did).Limit(1).Find(&like)
	if res.Error != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	if res.RowsAffected == 0 {
		like = model.DanmakuLike{UserID: uid, DanmakuID: did}
		if err := a.DB.Create(&like).Error; err != nil {
			resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
			return
		}
		_ = a.DB.Model(&d).UpdateColumn("like_count", gorm.Expr("like_count + ?", 1)).Error
		var dm model.Danmaku
		_ = a.DB.First(&dm, did).Error
		resp.OK(c, gin.H{"liked": true, "like_count": dm.LikeCount})
		return
	}
	if err := a.DB.Delete(&like).Error; err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	_ = a.DB.Model(&d).UpdateColumn("like_count", gorm.Expr("GREATEST(like_count - ?, 0)", 1)).Error
	var dm model.Danmaku
	_ = a.DB.First(&dm, did).Error
	resp.OK(c, gin.H{"liked": false, "like_count": dm.LikeCount})
}
