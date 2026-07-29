package handler

import (
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"minibili/internal/errcode"
	"minibili/internal/middleware"
	"minibili/internal/model"
	"minibili/internal/pkg/resp"
)

var danmakuColorHexRe = regexp.MustCompile("^#[0-9A-Fa-f]{6}$")

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
	var req danmakuPost
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	if _, ok := allowedDanmakuTypes[strings.TrimSpace(req.Type)]; !ok {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	colorRaw := strings.TrimSpace(req.Color)
	var color string
	if colorRaw == "" {
		color = "#FFFFFF"
	} else if !danmakuColorHexRe.MatchString(colorRaw) {
		resp.Err(c, http.StatusBadRequest, errcode.CodeInvalidColor)
		return
	} else {
		color = strings.ToUpper(colorRaw)
	}
	fontSize := normalizeDanmakuFontSize(req.FontSize)
	content := strings.TrimSpace(req.Content)

	result, svcErr := a.DanmakuSvc.PostDanmaku(c.Request.Context(), vid, uid, content, color, strings.TrimSpace(req.Type), fontSize, req.VideoTime)
	if svcErr != nil {
		code := errCodeFromSvc(svcErr)
		httpStatus := http.StatusInternalServerError
		switch code {
		case 40400:
			httpStatus = http.StatusNotFound
		case 40304:
			httpStatus = http.StatusForbidden
		case 40001, 40022, 40025:
			httpStatus = http.StatusBadRequest
		}
		resp.Err(c, httpStatus, code)
		return
	}
	d := result.Danmaku
	displayName := model.DisplayUsername(result.User)

	payload := gin.H{
		"type": "danmaku",
		"data": gin.H{
			"id":         d.ID,
			"content":    d.Content,
			"color":      d.Color,
			"type":       d.Type,
			"font_size":  fontSize,
			"video_time": d.VideoTime,
			"user":       displayName,
			"created_at": d.CreatedAt.Format("2006-01-02 15:04:05"),
		},
	}
	if a.DanmakuRelay != nil {
		if err := a.DanmakuRelay.Publish(c.Request.Context(), vid, payload); err != nil {
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
		"user":       displayName,
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
	result, svcErr := a.DanmakuSvc.ToggleDanmakuLike(c.Request.Context(), did, uid)
	if svcErr != nil {
		code := errCodeFromSvc(svcErr)
		httpStatus := http.StatusInternalServerError
		if code == 40400 {
			httpStatus = http.StatusNotFound
		}
		resp.Err(c, httpStatus, code)
		return
	}
	resp.OK(c, gin.H{"liked": result.Liked, "like_count": result.LikeCount})
}
