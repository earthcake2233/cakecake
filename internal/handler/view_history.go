package handler

import (
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"minibili/internal/errcode"
	"minibili/internal/middleware"
	"minibili/internal/model"
	"minibili/internal/pkg/resp"
)

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
	var u model.User
	if err := a.DB.Select("id", "view_history_paused").First(&u, uid).Error; err != nil {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	if u.ViewHistoryPaused {
		resp.OK(c, gin.H{"recorded": false, "paused": true})
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
	now := time.Now()
	var row model.VideoViewHistory
	err = a.DB.Where("user_id = ? AND video_id = ?", uid, vid).Limit(1).Find(&row).Error
	if err != nil {
		a.Log.Error("find view history", zap.Error(err))
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	if row.ID == 0 {
		row = model.VideoViewHistory{
			UserID:      uid,
			VideoID:     vid,
			ProgressSec: prog,
			DurationSec: dur,
			Device:      device,
			ViewedAt:    now,
		}
		if err := a.DB.Create(&row).Error; err != nil {
			a.Log.Error("create view history", zap.Error(err))
			resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
			return
		}
	} else {
		updates := map[string]interface{}{
			"progress_sec": prog,
			"duration_sec": dur,
			"device":       device,
			"viewed_at":    now,
		}
		if err := a.DB.Model(&row).Updates(updates).Error; err != nil {
			a.Log.Error("update view history", zap.Error(err))
			resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
			return
		}
	}
	a.trimViewHistoryCombined(uid)
	resp.OK(c, gin.H{"recorded": true})
}

// RecordArticleViewHistory upserts read history for a published article (专栏).
func (a *API) RecordArticleViewHistory(uid, articleID uint64, device string) {
	if uid == 0 || articleID == 0 {
		return
	}
	var u model.User
	if err := a.DB.Select("id", "view_history_paused").First(&u, uid).Error; err != nil {
		return
	}
	if u.ViewHistoryPaused {
		return
	}
	if device != "mobile" {
		device = "web"
	}
	now := time.Now()
	var row model.ArticleViewHistory
	err := a.DB.Where("user_id = ? AND article_id = ?", uid, articleID).Limit(1).Find(&row).Error
	if err != nil {
		return
	}
	if row.ID == 0 {
		row = model.ArticleViewHistory{
			UserID:    uid,
			ArticleID: articleID,
			Device:    device,
			ViewedAt:  now,
		}
		_ = a.DB.Create(&row).Error
	} else {
		_ = a.DB.Model(&row).Updates(map[string]interface{}{
			"device":    device,
			"viewed_at": now,
		}).Error
	}
	a.trimViewHistoryCombined(uid)
}

func (a *API) trimViewHistoryCombined(uid uint64) {
	type dropRec struct {
		kind string
		id   uint64
		at   time.Time
	}
	var recs []dropRec
	var vrows []model.VideoViewHistory
	_ = a.DB.Where("user_id = ?", uid).Find(&vrows).Error
	for i := range vrows {
		recs = append(recs, dropRec{"video", vrows[i].ID, vrows[i].ViewedAt})
	}
	var arows []model.ArticleViewHistory
	_ = a.DB.Where("user_id = ?", uid).Find(&arows).Error
	for i := range arows {
		recs = append(recs, dropRec{"article", arows[i].ID, arows[i].ViewedAt})
	}
	if len(recs) <= viewHistoryMaxItems {
		return
	}
	sort.Slice(recs, func(i, j int) bool {
		if recs[i].at.Equal(recs[j].at) {
			return recs[i].id < recs[j].id
		}
		return recs[i].at.Before(recs[j].at)
	})
	excess := len(recs) - viewHistoryMaxItems
	for i := 0; i < excess; i++ {
		switch recs[i].kind {
		case "video":
			_ = a.DB.Delete(&model.VideoViewHistory{}, recs[i].id).Error
		case "article":
			_ = a.DB.Delete(&model.ArticleViewHistory{}, recs[i].id).Error
		}
	}
}

// ListMyViewHistory returns watch history for the personal-center page.
func (a *API) ListMyViewHistory(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	var u model.User
	if err := a.DB.Select("id", "view_history_paused").First(&u, uid).Error; err != nil {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	keyword := strings.TrimSpace(c.Query("keyword"))
	var vRows []model.VideoViewHistory
	vq := a.DB.Where("user_id = ?", uid)
	if keyword != "" {
		like := "%" + keyword + "%"
		sub := a.DB.Model(&model.Video{}).
			Select("id").
			Where("status = ? AND title LIKE ?", "published", like)
		vq = vq.Where("video_id IN (?)", sub)
	}
	if err := vq.Order("viewed_at DESC, id DESC").Limit(viewHistoryMaxItems).Find(&vRows).Error; err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	var aRows []model.ArticleViewHistory
	aq := a.DB.Where("user_id = ?", uid)
	if keyword != "" {
		like := "%" + keyword + "%"
		sub := a.DB.Model(&model.Article{}).
			Select("id").
			Where("status = ? AND title LIKE ?", articleStatusPublished, like)
		aq = aq.Where("article_id IN (?)", sub)
	}
	if err := aq.Order("viewed_at DESC, id DESC").Limit(viewHistoryMaxItems).Find(&aRows).Error; err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	items := append(a.buildViewHistoryItems(vRows), a.buildArticleViewHistoryItems(aRows)...)
	sort.Slice(items, func(i, j int) bool {
		ti, _ := items[i]["viewed_at"].(string)
		tj, _ := items[j]["viewed_at"].(string)
		return ti > tj
	})
	if len(items) > viewHistoryMaxItems {
		items = items[:viewHistoryMaxItems]
	}
	resp.OK(c, gin.H{
		"items":  items,
		"total":  len(items),
		"paused": u.ViewHistoryPaused,
	})
}

func (a *API) buildViewHistoryItems(rows []model.VideoViewHistory) []gin.H {
	if len(rows) == 0 {
		return []gin.H{}
	}
	vids := make([]uint64, 0, len(rows))
	for i := range rows {
		vids = append(vids, rows[i].VideoID)
	}
	var videos []model.Video
	_ = a.DB.Where("id IN ?", vids).Find(&videos).Error
	byID := map[uint64]model.Video{}
	uids := make([]uint64, 0, len(videos))
	for i := range videos {
		byID[videos[i].ID] = videos[i]
		uids = append(uids, videos[i].UserID)
	}
	var users []model.User
	if len(uids) > 0 {
		_ = a.DB.Where("id IN ?", uids).Find(&users).Error
	}
	userByID := map[uint64]model.User{}
	for i := range users {
		userByID[users[i].ID] = users[i]
	}
	items := make([]gin.H, 0, len(rows))
	for i := range rows {
		h := rows[i]
		v, ok := byID[h.VideoID]
		if !ok || v.Status != "published" {
			continue
		}
		u := userByID[v.UserID]
		tag := videoZoneCategoryLabel(v.Zone)
		if tag == "" {
			tags := videoTagsForResponse(v.TagsJSON)
			if len(tags) > 0 {
				tag = tags[0]
			}
		}
		items = append(items, gin.H{
			"media_type":          "video",
			"video_id":            v.ID,
			"article_id":          0,
			"title":               v.Title,
			"cover_url":           v.CoverURL,
			"duration_sec":        h.DurationSec,
			"progress_sec":        h.ProgressSec,
			"device":              h.Device,
			"viewed_at":           h.ViewedAt.Format("2006-01-02 15:04:05"),
			"viewed_time":         h.ViewedAt.Format("15:04"),
			"uploader_id":         v.UserID,
			"uploader_name":       model.DisplayUsername(&u),
			"uploader_avatar_url": uploaderAvatarForAPI(&u),
			"category":            tag,
		})
	}
	return items
}

func (a *API) buildArticleViewHistoryItems(rows []model.ArticleViewHistory) []gin.H {
	if len(rows) == 0 {
		return []gin.H{}
	}
	aids := make([]uint64, 0, len(rows))
	for i := range rows {
		aids = append(aids, rows[i].ArticleID)
	}
	var articles []model.Article
	_ = a.DB.Where("id IN ?", aids).Find(&articles).Error
	byID := map[uint64]model.Article{}
	uids := make([]uint64, 0, len(articles))
	for i := range articles {
		byID[articles[i].ID] = articles[i]
		uids = append(uids, articles[i].UserID)
	}
	var users []model.User
	if len(uids) > 0 {
		_ = a.DB.Where("id IN ?", uids).Find(&users).Error
	}
	userByID := map[uint64]model.User{}
	for i := range users {
		userByID[users[i].ID] = users[i]
	}
	items := make([]gin.H, 0, len(rows))
	for i := range rows {
		h := rows[i]
		art, ok := byID[h.ArticleID]
		if !ok || art.Status != articleStatusPublished {
			continue
		}
		u := userByID[art.UserID]
		items = append(items, gin.H{
			"media_type":          "article",
			"video_id":            0,
			"article_id":          art.ID,
			"title":               art.Title,
			"cover_url":           art.CoverURL,
			"duration_sec":        0,
			"progress_sec":        0,
			"device":              h.Device,
			"viewed_at":           h.ViewedAt.Format("2006-01-02 15:04:05"),
			"viewed_time":         h.ViewedAt.Format("15:04"),
			"uploader_id":         art.UserID,
			"uploader_name":       model.DisplayUsername(&u),
			"uploader_avatar_url": uploaderAvatarForAPI(&u),
			"category":            "专栏",
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
	if err := a.DB.Where("user_id = ? AND video_id = ?", uid, vid).
		Delete(&model.VideoViewHistory{}).Error; err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	resp.OK(c, gin.H{"deleted": true})
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
	if err := a.DB.Where("user_id = ? AND article_id = ?", uid, aid).
		Delete(&model.ArticleViewHistory{}).Error; err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	resp.OK(c, gin.H{"deleted": true})
}

// ClearMyViewHistory removes all history for the user.
func (a *API) ClearMyViewHistory(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	if err := a.DB.Where("user_id = ?", uid).Delete(&model.VideoViewHistory{}).Error; err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	if err := a.DB.Where("user_id = ?", uid).Delete(&model.ArticleViewHistory{}).Error; err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	resp.OK(c, gin.H{"cleared": true})
}

// GetMyViewHistorySettings returns whether history recording is paused.
func (a *API) GetMyViewHistorySettings(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	var u model.User
	if err := a.DB.Select("view_history_paused").First(&u, uid).Error; err != nil {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	resp.OK(c, gin.H{"paused": u.ViewHistoryPaused})
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
	if err := a.DB.Model(&model.User{}).Where("id = ?", uid).
		Update("view_history_paused", body.Paused).Error; err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	resp.OK(c, gin.H{"paused": body.Paused})
}
