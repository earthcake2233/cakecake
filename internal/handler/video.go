package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"minibili/internal/errcode"
	"minibili/internal/ffmpeg"
	"minibili/internal/middleware"
	"minibili/internal/model"
	"minibili/internal/pkg/coverval"
	"minibili/internal/pkg/cursor"
	"minibili/internal/pkg/dailyreward"
	"minibili/internal/pkg/resp"
	"minibili/internal/queue"
	"minibili/internal/worker"
)

func uploaderAvatarForAPI(u *model.User) string {
	return avatarURLForAPI(u)
}

// uploaderNameForAPI is the UP display name on video cards (nickname if set, else username).
func uploaderNameForAPI(u *model.User) string {
	if u == nil {
		return ""
	}
	if nick := strings.TrimSpace(u.Nickname); nick != "" && !model.IsUserAnonymized(u) {
		return nick
	}
	return model.DisplayUsername(u)
}

func videoLikesByViewer(db *gorm.DB, viewer uint64, ids []uint64) map[uint64]bool {
	out := make(map[uint64]bool)
	if viewer == 0 || len(ids) == 0 {
		return out
	}
	var rows []model.VideoLike
	if err := db.Where("user_id = ? AND video_id IN ?", viewer, ids).Find(&rows).Error; err != nil {
		return out
	}
	for i := range rows {
		out[rows[i].VideoID] = true
	}
	return out
}

const maxVideoBytes = 500 << 20
const maxDurationSec = 30 * 60

const maxVideoTags = 10
const maxTagRunes = 32

func normalizeTagStrings(arr []string) []string {
	var out []string
	seen := map[string]struct{}{}
	for _, t := range arr {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		if utf8.RuneCountInString(t) > maxTagRunes {
			t = string([]rune(t)[:maxTagRunes])
		}
		k := strings.ToLower(t)
		if _, ok := seen[k]; ok {
			continue
		}
		seen[k] = struct{}{}
		out = append(out, t)
		if len(out) >= maxVideoTags {
			break
		}
	}
	return out
}

func tagsJSONFromStringSlice(tags []string) (string, error) {
	n := normalizeTagStrings(tags)
	b, err := json.Marshal(n)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// parseTagsPostForm reads optional multipart field "tags" as JSON string array; empty/missing => "[]".
func parseTagsPostForm(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "[]", nil
	}
	var arr []string
	if err := json.Unmarshal([]byte(raw), &arr); err != nil {
		return "", err
	}
	return tagsJSONFromStringSlice(arr)
}

func videoTagsForResponse(tagsJSON string) []string {
	tagsJSON = strings.TrimSpace(tagsJSON)
	if tagsJSON == "" {
		return []string{}
	}
	var arr []string
	if err := json.Unmarshal([]byte(tagsJSON), &arr); err != nil {
		return []string{}
	}
	return normalizeTagStrings(arr)
}

// UploadVideo handles multipart upload (F2).
func (a *API) UploadVideo(c *gin.Context) {
	if a.Cfg != nil && a.Cfg.VideoUploadDisabled {
		resp.Err(c, http.StatusBadRequest, errcode.CodeVideoUploadDisabled)
		return
	}
	uid, ok := middleware.UserID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	if err := c.Request.ParseMultipartForm(maxVideoBytes + (12 << 20)); err != nil {
		a.Log.Warn("parse multipart form", zap.Error(err))
		resp.Err(c, http.StatusBadRequest, errcode.CodeMultipartParseError)
		return
	}
	title := strings.TrimSpace(c.PostForm("title"))
	desc := strings.TrimSpace(c.PostForm("description"))
	if utf8.RuneCountInString(title) < 1 || utf8.RuneCountInString(title) > 80 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeTitleInvalid)
		return
	}
	if utf8.RuneCountInString(desc) > 2000 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeIntroTooLong)
		return
	}
	fh, err := c.FormFile("file")
	if err != nil {
		resp.Err(c, http.StatusBadRequest, errcode.CodeUploadMissingFile)
		return
	}
	if fh.Size > maxVideoBytes {
		resp.Err(c, http.StatusBadRequest, errcode.CodeVideoFileTooLarge)
		return
	}
	coverFh, _ := c.FormFile("cover")
	if coverFh != nil {
		if code := coverval.ValidateCoverHeader(coverFh); code != 0 {
			resp.Err(c, http.StatusBadRequest, code)
			return
		}
	}
	if err := os.MkdirAll(a.Cfg.TempUploadDir, 0o755); err != nil {
		a.Log.Error("mkdir temp", zap.Error(err))
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	rawName := uuid.NewString() + filepath.Ext(fh.Filename)
	rawPath := filepath.Join(a.Cfg.TempUploadDir, rawName)
	if err := saveUploadedFile(fh, rawPath); err != nil {
		a.Log.Error("save raw video", zap.Error(err))
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	dur, err := ffmpeg.ProbeDurationSeconds(rawPath)
	if err != nil {
		_ = os.Remove(rawPath)
		a.Log.Warn("ffprobe duration",
			zap.Error(err),
			zap.String("ffprobe", ffmpeg.FFprobeExe()),
			zap.String("raw_path", rawPath),
		)
		resp.Err(c, http.StatusBadRequest, errcode.CodeVideoProbeFailed)
		return
	}
	if dur > maxDurationSec {
		_ = os.Remove(rawPath)
		resp.Err(c, http.StatusBadRequest, errcode.CodeVideoDurationExceeded)
		return
	}
	tagsJSON, err := parseTagsPostForm(c.PostForm("tags"))
	if err != nil {
		_ = os.Remove(rawPath)
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	var coverPath string
	if coverFh != nil {
		cn := uuid.NewString() + filepath.Ext(coverFh.Filename)
		coverPath = filepath.Join(a.Cfg.TempUploadDir, cn)
		if err := saveUploadedFile(coverFh, coverPath); err != nil {
			_ = os.Remove(rawPath)
			a.Log.Error("save cover", zap.Error(err))
			resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
			return
		}
	}
	zone := normalizeVideoZone(c.PostForm("zone"))
	v := model.Video{
		UserID:       uid,
		Title:        title,
		Description:  desc,
		DurationSec:  dur,
		Status:       "processing",
		PlayCount:    0,
		DanmakuCount: 0,
		CommentCount: 0,
		TagsJSON:     tagsJSON,
		Zone:         zone,
	}
	if err := a.DB.Create(&v).Error; err != nil {
		_ = os.Remove(rawPath)
		if coverPath != "" {
			_ = os.Remove(coverPath)
		}
		a.Log.Error("create video", zap.Error(err))
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	job := worker.TranscodeJob{VideoID: v.ID, RawPath: rawPath, CoverPath: coverPath, RetryCount: 0}
	body, _ := json.Marshal(job)
	if err := a.MQ.PublishTranscode(context.Background(), body); err != nil {
		_ = a.DB.Where("id = ?", v.ID).Delete(&model.Video{}).Error
		_ = os.Remove(rawPath)
		if coverPath != "" {
			_ = os.Remove(coverPath)
		}
		a.Log.Error("publish transcode", zap.Error(err))
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	a.Log.Info("transcode job queued",
		zap.Uint64("video_id", v.ID),
		zap.String("queue", queue.TranscodeQueue),
	)
	resp.JSON(c, http.StatusCreated, errcode.CodeSuccess, gin.H{
		"id":         v.ID,
		"status":     v.Status,
		"title":      v.Title,
		"duration":   v.DurationSec,
		"created_at": v.CreatedAt.Format("2006-01-02 15:04:05"),
	})
}

func saveUploadedFile(fh *multipart.FileHeader, dst string) error {
	src, err := fh.Open()
	if err != nil {
		return err
	}
	defer src.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, src)
	return err
}

// ListPublishedVideos is the home feed (F10, AC-4).
// Query: limit, cursor, zone_parent, sort=hot|time, days=1|3|7|30, arc_type=0|1 (1=仅近期投稿).
func (a *API) ListPublishedVideos(c *gin.Context) {
	limit := 20
	if s := c.Query("limit"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}
	sortKey := strings.TrimSpace(c.DefaultQuery("sort", "hot"))
	zoneParent := ""
	if zp := normalizeVideoZone(strings.TrimSpace(c.Query("zone_parent"))); zp != "" {
		if p, _ := splitVideoZone(zp); p != "" {
			zoneParent = p
		}
	}
	days := 0
	if s := strings.TrimSpace(c.Query("days")); s != "" {
		if n, err := strconv.Atoi(s); err == nil {
			switch n {
			case 1, 3, 7, 30:
				days = n
			}
		}
	}
	arcType := 0
	if s := strings.TrimSpace(c.Query("arc_type")); s != "" {
		if n, err := strconv.Atoi(s); err == nil && (n == 0 || n == 1) {
			arcType = n
		}
	}
	q := a.DB.Model(&model.Video{}).Where("status = ?", "published")
	if zoneParent != "" {
		q = q.Where("zone = ? OR zone LIKE ?", zoneParent, zoneParent+"-%")
	}
	recentOnly := days > 0 && arcType == 1
	if recentOnly {
		cutoff := time.Now().AddDate(0, 0, -days)
		q = q.Where("created_at >= ?", cutoff)
	}
	useHotCursor := zoneParent == "" && sortKey != "time" && !recentOnly
	var orderClause string
	switch sortKey {
	case "time":
		orderClause = "created_at DESC, id DESC"
	default:
		orderClause = "play_count DESC, created_at DESC, danmaku_count DESC, id DESC"
	}
	cur, _ := cursor.Decode(c.Query("cursor"))
	if useHotCursor && cur != nil {
		q = q.Where(
			"(play_count < ?) OR (play_count = ? AND created_at < ?) OR (play_count = ? AND created_at = ? AND danmaku_count < ?) OR (play_count = ? AND created_at = ? AND danmaku_count = ? AND id < ?)",
			cur.PlayCount, cur.PlayCount, cur.CreatedAt,
			cur.PlayCount, cur.CreatedAt, cur.DanmakuCount,
			cur.PlayCount, cur.CreatedAt, cur.DanmakuCount, cur.ID,
		)
	}
	fetchLimit := limit + 1
	if !useHotCursor {
		fetchLimit = limit
	}
	var list []model.Video
	if err := q.Order(orderClause).Limit(fetchLimit).Find(&list).Error; err != nil {
		a.Log.Error("list videos", zap.Error(err))
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	hasMore := useHotCursor && len(list) > limit
	if hasMore {
		list = list[:limit]
	}
	var uids []uint64
	for _, v := range list {
		uids = append(uids, v.UserID)
	}
	usernames := map[uint64]string{}
	if len(uids) > 0 {
		var users []model.User
		_ = a.DB.Where("id IN ?", uids).Find(&users).Error
		for i := range users {
			usr := &users[i]
			usernames[usr.ID] = model.DisplayUsername(usr)
		}
	}
	ctx := context.Background()
	var viewer uint64
	if uid, ok := middleware.UserID(c); ok {
		viewer = uid
	}
	ids := make([]uint64, 0, len(list))
	for _, v := range list {
		ids = append(ids, v.ID)
	}
	eng := videoEngagementByViewer(a.DB, viewer, ids)
	items := make([]gin.H, 0, len(list))
	for _, v := range list {
		pc, _ := a.Play.Display(ctx, &v)
		items = append(items, videoCard(v, usernames[v.UserID], pc, eng[v.ID]))
	}
	next := ""
	if hasMore && len(list) > 0 {
		last := list[len(list)-1]
		next = cursor.Encode(cursor.VideoListC{
			PlayCount: last.PlayCount, CreatedAt: last.CreatedAt, DanmakuCount: last.DanmakuCount, ID: last.ID,
		})
	}
	payload := gin.H{"items": items, "next_cursor": next}
	if zoneParent != "" {
		payload["zone_video_count"] = a.countZoneVideos(zoneParent)
	}
	resp.OK(c, payload)
}

// countZoneVideos counts all published videos in zone_parent (including sub-zones).
func (a *API) countZoneVideos(zoneParent string) int64 {
	if zoneParent == "" {
		return 0
	}
	var n int64
	err := a.DB.Model(&model.Video{}).
		Where("status = ?", "published").
		Where("zone = ? OR zone LIKE ?", zoneParent, zoneParent+"-%").
		Count(&n).Error
	if err != nil {
		a.Log.Warn("count zone videos", zap.String("zone", zoneParent), zap.Error(err))
		return 0
	}
	return n
}

func manuscriptVideoStatusToDB(st string) string {
	switch strings.TrimSpace(st) {
	case "draft":
		return "draft"
	case "processing":
		return "processing"
	case "passed", "published":
		return "published"
	case "rejected", "failed":
		return "failed"
	default:
		return ""
	}
}

func manuscriptVideoStatusFilter(st string) (single string, multi []string) {
	switch strings.TrimSpace(st) {
	case "draft":
		return "draft", nil
	case "processing":
		return "", []string{"processing", "pending_review"}
	case "passed", "published":
		return "published", nil
	case "rejected":
		return "", []string{"failed", "rejected"}
	default:
		if db := manuscriptVideoStatusToDB(st); db != "" {
			return db, nil
		}
		return "", nil
	}
}

func orderClauseForMyVideos(sort string) string {
	switch strings.TrimSpace(sort) {
	case "view":
		return "play_count DESC, id DESC"
	case "fav":
		return "fav_count DESC, id DESC"
	case "danmu":
		return "danmaku_count DESC, id DESC"
	case "reply":
		return "comment_count DESC, id DESC"
	default:
		return "id DESC"
	}
}

func (a *API) countMyVideosByStatus(uid uint64) gin.H {
	type row struct {
		Status string
		N      int64
	}
	var rows []row
	_ = a.DB.Model(&model.Video{}).
		Select("status, COUNT(*) AS n").
		Where("user_id = ?", uid).
		Group("status").
		Scan(&rows).Error
	out := gin.H{
		"draft":      int64(0),
		"processing": int64(0),
		"passed":     int64(0),
		"rejected":   int64(0),
	}
	for _, r := range rows {
		switch r.Status {
		case "draft":
			out["draft"] = r.N
		case "processing", "pending_review":
			out["processing"] = out["processing"].(int64) + r.N
		case "published":
			out["passed"] = r.N
		case "failed", "rejected":
			out["rejected"] = out["rejected"].(int64) + r.N
		}
	}
	return out
}

// ListMyVideos lists all statuses for the uploader (F2-b).
// Query: page, page_size, sort(time|view|fav|danmu|reply), status(all|draft|processing|passed|rejected), q(title).
func (a *API) ListMyVideos(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 50 {
		pageSize = 10
	}
	sortKey := strings.TrimSpace(c.DefaultQuery("sort", "time"))
	statusQ := strings.TrimSpace(c.Query("status"))
	titleQ := strings.TrimSpace(c.Query("q"))

	base := a.DB.Model(&model.Video{}).Where("user_id = ?", uid)
	filtered := base
	if statusQ != "" && statusQ != "all" {
		if single, multi := manuscriptVideoStatusFilter(statusQ); single != "" {
			filtered = filtered.Where("status = ?", single)
		} else if len(multi) > 0 {
			filtered = filtered.Where("status IN ?", multi)
		}
	}
	if titleQ != "" {
		filtered = filtered.Where("title LIKE ?", "%"+titleQ+"%")
	}
	var total int64
	if err := filtered.Count(&total).Error; err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	totalPages := int((total + int64(pageSize) - 1) / int64(pageSize))
	if totalPages < 1 {
		totalPages = 1
	}
	if page > totalPages {
		page = totalPages
	}
	offset := (page - 1) * pageSize
	var list []model.Video
	if err := filtered.Order(orderClauseForMyVideos(sortKey)).
		Offset(offset).Limit(pageSize).Find(&list).Error; err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	ctx := context.Background()
	items := make([]gin.H, 0, len(list))
	for _, v := range list {
		pc, _ := a.Play.Display(ctx, &v)
		items = append(items, gin.H{
			"id":            v.ID,
			"title":         v.Title,
			"status":        v.Status,
			"fail_reason":   ffmpeg.HumanizeFailReason(v.FailReason),
			"cover_url":     v.CoverURL,
			"duration":      v.DurationSec,
			"play_count":    pc,
			"danmaku_count": v.DanmakuCount,
			"comment_count": v.CommentCount,
			"fav_count":     v.FavCount,
			"coin_count":    v.CoinCount,
			"tags":          videoTagsForResponse(v.TagsJSON),
			"zone":          normalizeVideoZone(v.Zone),
			"created_at":    v.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}
	resp.OK(c, gin.H{
		"items":       items,
		"page":        page,
		"page_size":   pageSize,
		"total":       total,
		"total_pages": totalPages,
		"counts":      a.countMyVideosByStatus(uid),
	})
}

// GetVideo returns detail for playback page (F3, F4).
func (a *API) GetVideo(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	var v model.Video
	if err := a.DB.First(&v, id).Error; err != nil {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	var viewer uint64
	if uid, ok := middleware.UserID(c); ok {
		viewer = uid
	}
	if v.Status != "published" && v.UserID != viewer {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	if v.Status == "published" {
		_ = a.Play.Incr(context.Background(), v.ID)
		_ = a.DB.First(&v, id).Error
	}
	pc, _ := a.Play.Display(context.Background(), &v)
	var u model.User
	_ = a.DB.First(&u, v.UserID).Error
	watching := 0
	if a.Hub != nil {
		watching = a.Hub.RoomSize(id)
	}
	eng := videoEngagementFlags(a.DB, viewer, v.ID)
	detail := videoDetail(v, u, pc, watching, eng)
	if v.Status == videoStatusDraft && viewer == v.UserID {
		detail["draft_has_source"] = strings.TrimSpace(v.DraftRawPath) != ""
	}
	_, followerCnt := userFollowCounts(a.DB, v.UserID)
	detail["uploader_follower_count"] = followerCnt
	detail["uploader_published_count"] = uploaderPublishedCount(a.DB, v.UserID)
	if viewer > 0 && v.UserID != viewer {
		detail["followed_by_me"] = userFollows(a.DB, viewer, v.UserID)
		detail["daily_coin_exp_progress"] = dailyreward.CoinProgress(a.DB, viewer)
		detail["daily_coin_exp_max"] = dailyreward.ExpCoinMax
	} else {
		detail["followed_by_me"] = false
	}
	resp.OK(c, detail)
}

type updateMyVideoJSON struct {
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Tags        *[]string `json:"tags,omitempty"`
	Zone        string    `json:"zone,omitempty"`
}

// UpdateMyVideo updates title and description for the uploader's own video (any status).
func (a *API) UpdateMyVideo(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	var v model.Video
	if err := a.DB.First(&v, id).Error; err != nil {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	if v.UserID != uid {
		resp.Err(c, http.StatusForbidden, errcode.CodeForbidden)
		return
	}
	var req updateMyVideoJSON
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	title := strings.TrimSpace(req.Title)
	desc := strings.TrimSpace(req.Description)
	if utf8.RuneCountInString(title) < 1 || utf8.RuneCountInString(title) > 80 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeTitleInvalid)
		return
	}
	if utf8.RuneCountInString(desc) > 2000 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeIntroTooLong)
		return
	}
	updates := map[string]interface{}{
		"title":       title,
		"description": desc,
	}
	if req.Tags != nil {
		tj, err := tagsJSONFromStringSlice(*req.Tags)
		if err != nil {
			resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
			return
		}
		updates["tags_json"] = tj
	}
	if z := normalizeVideoZone(req.Zone); z != "" {
		updates["zone"] = z
	}
	if err := a.DB.Model(&v).Updates(updates).Error; err != nil {
		a.Log.Error("update my video", zap.Error(err), zap.Uint64("video_id", id))
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	if v.Status == "published" {
		a.esIndexVideo(id)
	}
	resp.OK(c, gin.H{"ok": true})
}

// UpdateVideoCover replaces cover on OSS (F3).
func (a *API) UpdateVideoCover(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	var v model.Video
	if err := a.DB.First(&v, id).Error; err != nil || v.UserID != uid {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	if v.Status != "published" {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	if err := c.Request.ParseMultipartForm(12 << 20); err != nil {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	fh, err := c.FormFile("cover")
	if err != nil {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	if code := coverval.ValidateCoverHeader(fh); code != 0 {
		resp.Err(c, http.StatusBadRequest, code)
		return
	}
	if a.OSS == nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	if err := os.MkdirAll(a.Cfg.TempUploadDir, 0o755); err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	tmp := filepath.Join(a.Cfg.TempUploadDir, uuid.NewString()+filepath.Ext(fh.Filename))
	if err := saveUploadedFile(fh, tmp); err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	defer os.Remove(tmp)
	ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(fh.Filename)), ".")
	if ext == "jpeg" {
		ext = "jpg"
	}
	key := fmt.Sprintf("covers/%d.%s", v.ID, ext)
	if err := a.OSS.UploadFile(key, tmp); err != nil {
		a.Log.Error("oss cover upload", zap.Error(err))
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	url := a.Cfg.OSSObjectURL(key)
	if err := a.DB.Model(&v).Update("cover_url", url).Error; err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	resp.OK(c, gin.H{"cover_url": url})
}

type videoPlaybackPatch struct {
	CommentsClosed  *bool `json:"comments_closed"`
	CommentsCurated *bool `json:"comments_curated"`
	DanmakuClosed   *bool `json:"danmaku_closed"`
}

// PatchVideoPlayback toggles comment area / danmaku posting for a published video (uploader only).
func (a *API) PatchVideoPlayback(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	var v model.Video
	if err := a.DB.First(&v, id).Error; err != nil {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	if v.UserID != uid {
		resp.Err(c, http.StatusForbidden, errcode.CodeForbidden)
		return
	}
	if v.Status != "published" {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	var req videoPlaybackPatch
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	if req.CommentsClosed == nil && req.CommentsCurated == nil && req.DanmakuClosed == nil {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	updates := map[string]interface{}{}
	if req.CommentsClosed != nil {
		updates["comments_closed"] = *req.CommentsClosed
	}
	if req.CommentsCurated != nil {
		updates["comments_curated"] = *req.CommentsCurated
	}
	if req.DanmakuClosed != nil {
		updates["danmaku_closed"] = *req.DanmakuClosed
	}
	if err := a.DB.Model(&v).Updates(updates).Error; err != nil {
		a.Log.Error("patch video playback", zap.Error(err), zap.Uint64("video_id", id))
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	if err := a.DB.First(&v, id).Error; err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	resp.OK(c, gin.H{
		"comments_closed":  v.CommentsClosed,
		"comments_curated": v.CommentsCurated,
		"danmaku_closed":   v.DanmakuClosed,
	})
}

type videoEngagement struct {
	LikedByMe     bool
	FavoritedByMe bool
	CoinedByMe    bool
	MyCoinAmount  int // 0, 1, or 2 coins given by viewer on this video
	InWatchLater  bool
}

func videoEngagementFlags(db *gorm.DB, viewer, videoID uint64) videoEngagement {
	var e videoEngagement
	if viewer == 0 || videoID == 0 {
		return e
	}
	var cnt int64
	_ = db.Model(&model.VideoLike{}).Where("user_id = ? AND video_id = ?", viewer, videoID).Count(&cnt).Error
	e.LikedByMe = cnt > 0
	cnt = 0
	_ = db.Model(&model.VideoFavorite{}).Where("user_id = ? AND video_id = ?", viewer, videoID).Count(&cnt).Error
	e.FavoritedByMe = cnt > 0
	var coinRow model.VideoCoin
	if err := db.Where("user_id = ? AND video_id = ?", viewer, videoID).Limit(1).Find(&coinRow).Error; err == nil && coinRow.ID > 0 {
		e.CoinedByMe = true
		e.MyCoinAmount = coinRow.Amount
		if e.MyCoinAmount < 0 {
			e.MyCoinAmount = 0
		}
		if e.MyCoinAmount > 2 {
			e.MyCoinAmount = 2
		}
	}
	cnt = 0
	_ = db.Model(&model.WatchLater{}).Where("user_id = ? AND video_id = ?", viewer, videoID).Count(&cnt).Error
	e.InWatchLater = cnt > 0
	return e
}

func videoCard(v model.Video, up string, play uint64, eng videoEngagement) gin.H {
	m := gin.H{
		"id":              v.ID,
		"user_id":         v.UserID,
		"title":           v.Title,
		"description":     v.Description,
		"cover_url":       v.CoverURL,
		"play_count":      play,
		"danmaku_count":   v.DanmakuCount,
		"comment_count":   v.CommentCount,
		"like_count":      v.LikeCount,
		"fav_count":       v.FavCount,
		"coin_count":      v.CoinCount,
		"liked_by_me":     eng.LikedByMe,
		"favorited_by_me": eng.FavoritedByMe,
		"coined_by_me":    eng.CoinedByMe,
		"in_watch_later":  eng.InWatchLater,
		"duration":        v.DurationSec,
		"uploader":        up,
		"created_at":      v.CreatedAt.Format("2006-01-02 15:04:05"),
		"comments_closed":  v.CommentsClosed,
		"comments_curated": v.CommentsCurated,
		"danmaku_closed":   v.DanmakuClosed,
	}
	appendVideoZoneFields(m, v.Zone)
	return m
}

func videoDetail(v model.Video, u model.User, play uint64, watching int, eng videoEngagement) gin.H {
	m := gin.H{
		"id":                  v.ID,
		"user_id":             v.UserID,
		"title":               v.Title,
		"description":         v.Description,
		"play_count":          play,
		"danmaku_count":       v.DanmakuCount,
		"comment_count":       v.CommentCount,
		"like_count":          v.LikeCount,
		"fav_count":           v.FavCount,
		"coin_count":          v.CoinCount,
		"liked_by_me":         eng.LikedByMe,
		"favorited_by_me":     eng.FavoritedByMe,
		"coined_by_me":        eng.CoinedByMe,
		"my_coin_amount":      eng.MyCoinAmount,
		"in_watch_later":      eng.InWatchLater,
		"watching_count":      watching,
		"duration":            v.DurationSec,
		"uploader":            model.DisplayUsername(&u),
		"uploader_sign":       strings.TrimSpace(u.Sign),
		"uploader_avatar_url": uploaderAvatarForAPI(&u),
		"created_at":          v.CreatedAt.Format("2006-01-02 15:04:05"),
		"video_url":           v.VideoURL,
		"cover_url":           v.CoverURL,
		"status":              v.Status,
		"fail_reason":         ffmpeg.HumanizeFailReason(v.FailReason),
		"tags":                videoTagsForResponse(v.TagsJSON),
		"comments_closed":     v.CommentsClosed,
		"comments_curated":    v.CommentsCurated,
		"danmaku_closed":      v.DanmakuClosed,
	}
	appendVideoZoneFields(m, v.Zone)
	return m
}

// deleteVideoCascade removes one video and its comments, likes, danmaku (same package as account deletion).
func deleteVideoCascade(tx *gorm.DB, videoID uint64) error {
	var cids []uint64
	if err := tx.Model(&model.Comment{}).Where("video_id = ?", videoID).Pluck("id", &cids).Error; err != nil {
		return err
	}
	if len(cids) > 0 {
		if err := tx.Where("comment_id IN ?", cids).Delete(&model.CommentLike{}).Error; err != nil {
			return err
		}
		if err := tx.Where("comment_id IN ?", cids).Delete(&model.CommentDislike{}).Error; err != nil {
			return err
		}
		if err := tx.Where("id IN ?", cids).Delete(&model.Comment{}).Error; err != nil {
			return err
		}
	}
	if err := tx.Where("video_id = ?", videoID).Delete(&model.VideoLike{}).Error; err != nil {
		return err
	}
	if err := tx.Where("video_id = ?", videoID).Delete(&model.VideoFavorite{}).Error; err != nil {
		return err
	}
	if err := tx.Where("video_id = ?", videoID).Delete(&model.VideoCoin{}).Error; err != nil {
		return err
	}
	if err := tx.Where("video_id = ?", videoID).Delete(&model.WatchLater{}).Error; err != nil {
		return err
	}
	if err := tx.Where("video_id = ?", videoID).Delete(&model.VideoViewHistory{}).Error; err != nil {
		return err
	}
	var dmIDs []uint64
	if err := tx.Model(&model.Danmaku{}).Where("video_id = ?", videoID).Pluck("id", &dmIDs).Error; err != nil {
		return err
	}
	if len(dmIDs) > 0 {
		if err := tx.Where("danmaku_id IN ?", dmIDs).Delete(&model.DanmakuLike{}).Error; err != nil {
			return err
		}
	}
	if err := tx.Where("video_id = ?", videoID).Delete(&model.Danmaku{}).Error; err != nil {
		return err
	}
	return tx.Where("id = ?", videoID).Delete(&model.Video{}).Error
}

// DeleteMyVideo deletes the caller's own video by id (comments, likes, danmaku cascade in DB).
func (a *API) DeleteMyVideo(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	var v model.Video
	if err := a.DB.First(&v, id).Error; err != nil {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	if v.UserID != uid {
		resp.Err(c, http.StatusForbidden, errcode.CodeForbidden)
		return
	}
	removeVideoDraftFiles(v)
	if err := a.DB.Transaction(func(tx *gorm.DB) error {
		return deleteVideoCascade(tx, id)
	}); err != nil {
		a.Log.Error("delete my video", zap.Error(err), zap.Uint64("video_id", id))
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	purgeVideoOSSObjects(a.Cfg, a.OSS, a.Log, v)
	a.esDeleteVideo(id)
	resp.OK(c, gin.H{"ok": true})
}
