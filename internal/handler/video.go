package handler

import (
	"cakecake/internal/model/comment"
	"cakecake/internal/model/danmaku"
	"cakecake/internal/model/extra"
	"cakecake/internal/model/user"
	"cakecake/internal/model/video"
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
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"cakecake/internal/errcode"
	"cakecake/internal/middleware"
	"cakecake/internal/pkg/coverval"
	"cakecake/internal/pkg/dailyreward"
	"cakecake/internal/pkg/resp"
	vsvc "cakecake/internal/service/video"
)

func uploaderAvatarForAPI(u *user.User) string {
	return avatarURLForAPI(u)
}

// uploaderNameForAPI is the UP display name on video cards (nickname if set, else username).
func uploaderNameForAPI(u *user.User) string {
	if u == nil {
		return ""
	}
	if nick := strings.TrimSpace(u.Nickname); nick != "" && !user.IsUserAnonymized(u) {
		return nick
	}
	return user.DisplayUsername(u)
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
	dur, err := a.VideoSvc.ProbeDurationSeconds(rawPath)
	if err != nil {
		_ = os.Remove(rawPath)
		a.Log.Warn("ffprobe duration",
			zap.Error(err),
			zap.String("ffprobe", a.VideoSvc.FFprobeExe()),
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
	v := video.Video{
		UserID:       uid,
		Title:        title,
		Description:  desc,
		DurationSec:  dur,
		Status:       video.StatusProcessing,
		PlayCount:    0,
		DanmakuCount: 0,
		CommentCount: 0,
		TagsJSON:     tagsJSON,
		Zone:         zone,
	}
	if err := a.VideoSvc.CreateVideoRecord(c.Request.Context(), &v); err != nil {
		_ = os.Remove(rawPath)
		if coverPath != "" {
			_ = os.Remove(coverPath)
		}
		a.Log.Error("create video", zap.Error(err))
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	if err := a.VideoSvc.EnqueueTranscode(c.Request.Context(), v.ID, rawPath, coverPath); err != nil {
		_ = a.VideoSvc.DeleteVideoByID(c.Request.Context(), v.ID)
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
	)
	resp.JSON(c, http.StatusCreated, errcode.CodeSuccess, createVideoResponse{
		ID:        v.ID,
		Status:    v.Status,
		Title:     v.Title,
		Duration:  v.DurationSec,
		CreatedAt: v.CreatedAt.Format("2006-01-02 15:04:05"),
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
// Query: limit, cursor, zone_parent, sort=hot|time, days=1|3|7|30, arc_type=0|1 (1=recent uploads only).
// ListPublishedVideos godoc
// @Summary      List published videos
// @Description  Get paginated list of published videos
// @Tags        Videos
// @Produce     json
// @Param       page query int false "Page number" default(1)
// @Param       page_size query int false "Page size" default(20)
// @Param       zone query string false "Video zone slug"
// @Param       sort query string false "Sort field (created_at,play_count)"
// @Success     200 {object} map[string]interface{}
// @Router      /videos [get]
func (a *API) ListPublishedVideos(c *gin.Context) {
	limit := parseLimit(c, 20, 100)
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
	recentOnly := days > 0 && arcType == 1
	res, err := a.VideoSvc.ListPublishedVideos(c.Request.Context(), vsvc.VideoListOpts{
		Limit:      limit,
		SortKey:    sortKey,
		ZoneParent: zoneParent,
		Days:       days,
		RecentOnly: recentOnly,
		Cursor:     c.Query("cursor"),
	})
	if err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	items := make([]videoCardDTO, 0, len(res.Videos))
	for _, v := range res.Videos {
		pc, _ := a.Play.Display(c.Request.Context(), &v)
		items = append(items, videoCard(v, user.DisplayUsername(&user.User{Username: ""}), pc, videoEngagement{}))
	}
	resp.OK(c, zoneVideoListResponse{
		Items:          items,
		NextCursor:     res.NextCursor,
		ZoneVideoCount: res.ZoneVideoCount,
		HasMore:        res.HasMore,
	})
}
func (a *API) countZoneVideos(zoneParent string) int64 {
	return a.VideoSvc.CountZoneVideos(zoneParent)
}

func manuscriptVideoStatusToDB(st string) string {
	switch strings.TrimSpace(st) {
	case video.StatusDraft:
		return video.StatusDraft
	case video.StatusProcessing:
		return video.StatusProcessing
	case video.StatusPassed, video.StatusPublished:
		return video.StatusPublished
	case video.StatusRejected, video.StatusFailed:
		return video.StatusFailed
	default:
		return ""
	}
}

func manuscriptVideoStatusFilter(st string) (single string, multi []string) {
	switch strings.TrimSpace(st) {
	case video.StatusDraft:
		return video.StatusDraft, nil
	case video.StatusProcessing:
		return "", []string{video.StatusProcessing, video.StatusPendingReview}
	case video.StatusPassed, video.StatusPublished:
		return video.StatusPublished, nil
	case video.StatusRejected:
		return "", []string{video.StatusFailed, video.StatusRejected}
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

func (a *API) countMyVideosByStatus(uid uint64) map[string]int64 {
	result := map[string]int64{}
	for st, n := range a.VideoSvc.CountMyVideosByStatus(uid) {
		result[st] = n
	}
	return result
}

// ListMyVideos lists all statuses for the uploader (F2-b).
// Query: page, page_size, sort(time|view|fav|danmu|reply), status(all|draft|processing|passed|rejected), q(title).
func (a *API) ListMyVideos(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	page, pageSize := parsePagination(c, 10)
	sortKey := strings.TrimSpace(c.DefaultQuery("sort", "time"))
	statusQ := strings.TrimSpace(c.Query("status"))
	titleQ := strings.TrimSpace(c.Query("q"))

	f := vsvc.MyVideoFilter{UserID: uid, TitleQ: titleQ, SortKey: sortKey, Page: page, PageSize: pageSize}
	if statusQ != "" && statusQ != "all" {
		if single, multi := manuscriptVideoStatusFilter(statusQ); single != "" {
			f.Status = single
		} else if len(multi) > 0 {
			f.Statuses = multi
		}
	}
	res, err := a.VideoSvc.ListMyVideosAdvanced(c.Request.Context(), f)
	if err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	ctx := c.Request.Context()
	items := make([]myVideoItem, 0, len(res.Videos))
	for _, v := range res.Videos {
		pc, _ := a.Play.Display(ctx, &v)
		items = append(items, myVideoItem{
			ID:           v.ID,
			Title:        v.Title,
			Status:       v.Status,
			FailReason:   a.VideoSvc.HumanizeFailReason(v.FailReason),
			CoverURL:     v.CoverURL,
			Duration:     v.DurationSec,
			PlayCount:    pc,
			DanmakuCount: v.DanmakuCount,
			CommentCount: v.CommentCount,
			FavCount:     v.FavCount,
			CoinCount:    v.CoinCount,
			Tags:         videoTagsForResponse(v.TagsJSON),
			Zone:         normalizeVideoZone(v.Zone),
			CreatedAt:    v.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}
	resp.OK(c, myVideoListResponse{
		Items:      items,
		Page:       page,
		PageSize:   pageSize,
		Total:      res.Total,
		TotalPages: res.TotalPages,
		Counts:     a.countMyVideosByStatus(uid),
	})
}

// GetVideo returns detail for playback page (F3, F4).
// GetVideo godoc
// @Summary      Get video detail
// @Description  Get detailed video info by ID
// @Tags        Videos
// @Produce     json
// @Param       id path int true "Video ID"
// @Success     200 {object} map[string]interface{}
// @Router      /videos/{id} [get]
func (a *API) GetVideo(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	v, err := a.VideoSvc.GetVideoByID(c.Request.Context(), id)
	if err != nil {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	var viewer uint64
	if uid, ok := middleware.UserID(c); ok {
		viewer = uid
	}
	if v.Status != video.StatusPublished && v.UserID != viewer {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	if v.Status == video.StatusPublished {
		_ = a.Play.Incr(c.Request.Context(), v.ID)
	}
	pc, _ := a.Play.Display(c.Request.Context(), v)
	var u user.User
	uPub, _ := a.UserSvc.GetUserPublic(c.Request.Context(), v.UserID)
	if uPub != nil {
		u = user.User{ID: uPub.ID, Username: uPub.Username, AvatarURL: uPub.AvatarURL, Nickname: uPub.Nickname, Sign: uPub.Sign}
	}
	watching := 0
	if a.Hub != nil {
		watching = a.Hub.RoomSize(id)
	}
	eng := a.getVideoEngagementFlags(c.Request.Context(), viewer, v.ID)
	detail := videoDetail(*v, u, pc, watching, eng)
	if v.Status == videoStatusDraft && viewer == v.UserID {
		draftHasSource := strings.TrimSpace(v.DraftRawPath) != ""
		detail.DraftHasSource = &draftHasSource
	}
	_, followerCnt := a.getFollowCounts(c.Request.Context(), v.UserID)
	detail.UploaderFollowerCount = followerCnt
	detail.UploaderPublishedCount = a.getUploaderPublishedCount(c.Request.Context(), v.UserID)
	if viewer > 0 && v.UserID != viewer {
		detail.FollowedByMe = a.isFollowing(c.Request.Context(), viewer, v.UserID)
		prog := 0
		if a.DailyRewardSvc != nil {
			prog = a.DailyRewardSvc.CoinProgress(viewer)
		}
		max := dailyreward.ExpCoinMax
		detail.DailyCoinExpProgress = &prog
		detail.DailyCoinExpMax = &max
	} else {
		detail.FollowedByMe = false
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
// UpdateMyVideo godoc
// @Summary      Update video metadata
// @Description  Update video title, description, or other metadata
// @Tags         Videos
// @Produce      json
// @Param        id path int true "Video ID"
// @Param        body body object{} true "Video metadata to update"
// @Success      200 {object} map[string]interface{}
// @Router       /videos/{id} [put]
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
	v, err := a.VideoSvc.GetVideoByID(c.Request.Context(), id)
	if err != nil {
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
	if err := a.VideoSvc.UpdateVideo(c.Request.Context(), v, updates); err != nil {
		a.Log.Error("update my video", zap.Error(err), zap.Uint64("video_id", id))
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	if v.Status == video.StatusPublished {
		a.esIndexVideo(id)
	}
	resp.OK(c, okResponse{OK: true})
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
	v, err := a.VideoSvc.GetVideoByID(c.Request.Context(), id)
	if err != nil || v.UserID != uid {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	if v.Status != video.StatusPublished {
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
	if !a.StorageSvc.Enabled() {
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
	if err := a.StorageSvc.UploadFile(key, tmp); err != nil {
		a.Log.Error("oss cover upload", zap.Error(err))
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	url := a.Cfg.OSSObjectURL(key)
	if err := a.VideoSvc.UpdateVideo(c.Request.Context(), v, map[string]interface{}{"cover_url": url}); err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	resp.OK(c, imageURLResponse{ImageURL: url})
}

type videoPlaybackPatch struct {
	CommentsClosed  *bool `json:"comments_closed"`
	CommentsCurated *bool `json:"comments_curated"`
	DanmakuClosed   *bool `json:"danmaku_closed"`
}

// PatchVideoPlayback toggles comment area / danmaku posting for a published video (uploader only).
// PatchVideoPlayback godoc
// @Summary      Update video playback position
// @Description  Save or update the playback progress for a video
// @Tags         Videos
// @Produce      json
// @Param        id path int true "Video ID"
// @Param        body body object{position=int} true "Playback position in seconds"
// @Success      200 {object} map[string]interface{}
// @Router       /videos/{id}/playback [patch]
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
	v, err := a.VideoSvc.GetVideoByID(c.Request.Context(), id)
	if err != nil {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	if v.UserID != uid {
		resp.Err(c, http.StatusForbidden, errcode.CodeForbidden)
		return
	}
	if v.Status != video.StatusPublished {
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
	if err := a.VideoSvc.UpdateVideo(c.Request.Context(), v, updates); err != nil {
		a.Log.Error("patch video playback", zap.Error(err), zap.Uint64("video_id", id))
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	v, err = a.VideoSvc.GetVideoByID(c.Request.Context(), id)
	if err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	resp.OK(c, videoPlaybackResponse{
		CommentsClosed:  v.CommentsClosed,
		CommentsCurated: v.CommentsCurated,
		DanmakuClosed:   v.DanmakuClosed,
	})
}

type videoEngagement struct {
	LikedByMe     bool
	FavoritedByMe bool
	CoinedByMe    bool
	MyCoinAmount  int // 0, 1, or 2 coins given by viewer on this video
	InWatchLater  bool
}

func (a *API) getVideoEngagementFlags(ctx context.Context, viewer, videoID uint64) videoEngagement {
	var e videoEngagement
	if viewer == 0 || videoID == 0 {
		return e
	}
	liked := a.EngagementSvc.BatchVideoLikes(ctx, viewer, []uint64{videoID})
	e.LikedByMe = liked[videoID]
	fav := a.FavoriteSvc.BatchFavorited(ctx, viewer, []uint64{videoID})
	e.FavoritedByMe = fav[videoID]
	coinMap := a.EngagementSvc.BatchCoinedByUser(ctx, viewer, []uint64{videoID})
	if amt, ok := coinMap[videoID]; ok && amt > 0 {
		e.CoinedByMe = true
		e.MyCoinAmount = amt
		if e.MyCoinAmount < 0 {
			e.MyCoinAmount = 0
		}
		if e.MyCoinAmount > 2 {
			e.MyCoinAmount = 2
		}
	}
	wl := a.EngagementSvc.BatchWatchLater(ctx, viewer, []uint64{videoID})
	e.InWatchLater = wl[videoID]
	return e
}

type videoCardDTO struct {
	ID              uint64  `json:"id"`
	UserID          uint64  `json:"user_id"`
	Title           string  `json:"title"`
	Description     string  `json:"description"`
	CoverURL        string  `json:"cover_url"`
	PlayCount       uint64  `json:"play_count"`
	DanmakuCount    uint64  `json:"danmaku_count"`
	CommentCount    uint64  `json:"comment_count"`
	LikeCount       uint64  `json:"like_count"`
	FavCount        uint64  `json:"fav_count"`
	CoinCount       uint64  `json:"coin_count"`
	LikedByMe       bool    `json:"liked_by_me"`
	FavoritedByMe   bool    `json:"favorited_by_me"`
	CoinedByMe      bool    `json:"coined_by_me"`
	InWatchLater    bool    `json:"in_watch_later"`
	Duration        float64 `json:"duration"`
	Uploader        string  `json:"uploader"`
	CreatedAt       string  `json:"created_at"`
	CommentsClosed  bool    `json:"comments_closed"`
	CommentsCurated bool    `json:"comments_curated"`
	DanmakuClosed   bool    `json:"danmaku_closed"`
	videoZoneFields
}

type videoDetailDTO struct {
	ID                     uint64   `json:"id"`
	UserID                 uint64   `json:"user_id"`
	Title                  string   `json:"title"`
	Description            string   `json:"description"`
	PlayCount              uint64   `json:"play_count"`
	DanmakuCount           uint64   `json:"danmaku_count"`
	CommentCount           uint64   `json:"comment_count"`
	LikeCount              uint64   `json:"like_count"`
	FavCount               uint64   `json:"fav_count"`
	CoinCount              uint64   `json:"coin_count"`
	LikedByMe              bool     `json:"liked_by_me"`
	FavoritedByMe          bool     `json:"favorited_by_me"`
	CoinedByMe             bool     `json:"coined_by_me"`
	MyCoinAmount           int      `json:"my_coin_amount"`
	InWatchLater           bool     `json:"in_watch_later"`
	WatchingCount          int      `json:"watching_count"`
	Duration               float64  `json:"duration"`
	Uploader               string   `json:"uploader"`
	UploaderSign           string   `json:"uploader_sign"`
	UploaderAvatarURL      string   `json:"uploader_avatar_url"`
	CreatedAt              string   `json:"created_at"`
	VideoURL               string   `json:"video_url"`
	CoverURL               string   `json:"cover_url"`
	Status                 string   `json:"status"`
	FailReason             string   `json:"fail_reason"`
	Tags                   []string `json:"tags"`
	CommentsClosed         bool     `json:"comments_closed"`
	CommentsCurated        bool     `json:"comments_curated"`
	DanmakuClosed          bool     `json:"danmaku_closed"`
	DraftHasSource         *bool    `json:"draft_has_source,omitempty"`
	UploaderFollowerCount  int64    `json:"uploader_follower_count"`
	UploaderPublishedCount int64    `json:"uploader_published_count"`
	FollowedByMe           bool     `json:"followed_by_me"`
	DailyCoinExpProgress   *int     `json:"daily_coin_exp_progress,omitempty"`
	DailyCoinExpMax        *int     `json:"daily_coin_exp_max,omitempty"`
	videoZoneFields
}

type createVideoResponse struct {
	ID        uint64  `json:"id"`
	Status    string  `json:"status"`
	Title     string  `json:"title"`
	Duration  float64 `json:"duration"`
	CreatedAt string  `json:"created_at"`
}

type zoneVideoListResponse struct {
	Items          []videoCardDTO `json:"items"`
	NextCursor     string         `json:"next_cursor"`
	ZoneVideoCount int64          `json:"zone_video_count"`
	HasMore        bool           `json:"has_more"`
}

type myVideoItem struct {
	ID           uint64   `json:"id"`
	Title        string   `json:"title"`
	Status       string   `json:"status"`
	FailReason   string   `json:"fail_reason"`
	CoverURL     string   `json:"cover_url"`
	Duration     float64  `json:"duration"`
	PlayCount    uint64   `json:"play_count"`
	DanmakuCount uint64   `json:"danmaku_count"`
	CommentCount uint64   `json:"comment_count"`
	FavCount     uint64   `json:"fav_count"`
	CoinCount    uint64   `json:"coin_count"`
	Tags         []string `json:"tags"`
	Zone         string   `json:"zone"`
	CreatedAt    string   `json:"created_at"`
}

type myVideoListResponse struct {
	Items      []myVideoItem    `json:"items"`
	Page       int              `json:"page"`
	PageSize   int              `json:"page_size"`
	Total      int64            `json:"total"`
	TotalPages int              `json:"total_pages"`
	Counts     map[string]int64 `json:"counts"`
}

type videoPlaybackResponse struct {
	CommentsClosed  bool `json:"comments_closed"`
	CommentsCurated bool `json:"comments_curated"`
	DanmakuClosed   bool `json:"danmaku_closed"`
}

func videoCard(v video.Video, up string, play uint64, eng videoEngagement) videoCardDTO {
	m := videoCardDTO{
		ID:              v.ID,
		UserID:          v.UserID,
		Title:           v.Title,
		Description:     v.Description,
		CoverURL:        v.CoverURL,
		PlayCount:       play,
		DanmakuCount:    v.DanmakuCount,
		CommentCount:    v.CommentCount,
		LikeCount:       v.LikeCount,
		FavCount:        v.FavCount,
		CoinCount:       v.CoinCount,
		LikedByMe:       eng.LikedByMe,
		FavoritedByMe:   eng.FavoritedByMe,
		CoinedByMe:      eng.CoinedByMe,
		InWatchLater:    eng.InWatchLater,
		Duration:        v.DurationSec,
		Uploader:        up,
		CreatedAt:       v.CreatedAt.Format("2006-01-02 15:04:05"),
		CommentsClosed:  v.CommentsClosed,
		CommentsCurated: v.CommentsCurated,
		DanmakuClosed:   v.DanmakuClosed,
	}
	appendVideoZoneFields(&m.videoZoneFields, v.Zone)
	return m
}

func videoDetail(v video.Video, u user.User, play uint64, watching int, eng videoEngagement) videoDetailDTO {
	m := videoDetailDTO{
		ID:                v.ID,
		UserID:            v.UserID,
		Title:             v.Title,
		Description:       v.Description,
		PlayCount:         play,
		DanmakuCount:      v.DanmakuCount,
		CommentCount:      v.CommentCount,
		LikeCount:         v.LikeCount,
		FavCount:          v.FavCount,
		CoinCount:         v.CoinCount,
		LikedByMe:         eng.LikedByMe,
		FavoritedByMe:     eng.FavoritedByMe,
		CoinedByMe:        eng.CoinedByMe,
		MyCoinAmount:      eng.MyCoinAmount,
		InWatchLater:      eng.InWatchLater,
		WatchingCount:     watching,
		Duration:          v.DurationSec,
		Uploader:          user.DisplayUsername(&u),
		UploaderSign:      strings.TrimSpace(u.Sign),
		UploaderAvatarURL: uploaderAvatarForAPI(&u),
		CreatedAt:         v.CreatedAt.Format("2006-01-02 15:04:05"),
		VideoURL:          v.VideoURL,
		CoverURL:          v.CoverURL,
		Status:            v.Status,
		FailReason:        vsvc.HumanizeFailReason(v.FailReason),
		Tags:              videoTagsForResponse(v.TagsJSON),
		CommentsClosed:    v.CommentsClosed,
		CommentsCurated:   v.CommentsCurated,
		DanmakuClosed:     v.DanmakuClosed,
	}
	appendVideoZoneFields(&m.videoZoneFields, v.Zone)
	return m
}

// deleteVideoCascade removes one video and its comments, likes, danmaku (same package as account deletion).
func deleteVideoCascade(tx *gorm.DB, videoID uint64) error {
	var cids []uint64
	if err := tx.Model(&comment.Comment{}).Where("video_id = ?", videoID).Pluck("id", &cids).Error; err != nil {
		return err
	}
	if len(cids) > 0 {
		if err := tx.Where("comment_id IN ?", cids).Delete(&comment.CommentLike{}).Error; err != nil {
			return err
		}
		if err := tx.Where("comment_id IN ?", cids).Delete(&comment.CommentDislike{}).Error; err != nil {
			return err
		}
		if err := tx.Where("id IN ?", cids).Delete(&comment.Comment{}).Error; err != nil {
			return err
		}
	}
	if err := tx.Where("video_id = ?", videoID).Delete(&video.VideoLike{}).Error; err != nil {
		return err
	}
	if err := tx.Where("video_id = ?", videoID).Delete(&video.VideoFavorite{}).Error; err != nil {
		return err
	}
	if err := tx.Where("video_id = ?", videoID).Delete(&video.VideoCoin{}).Error; err != nil {
		return err
	}
	if err := tx.Where("video_id = ?", videoID).Delete(&video.WatchLater{}).Error; err != nil {
		return err
	}
	if err := tx.Where("video_id = ?", videoID).Delete(&extra.VideoViewHistory{}).Error; err != nil {
		return err
	}
	var dmIDs []uint64
	if err := tx.Model(&danmaku.Danmaku{}).Where("video_id = ?", videoID).Pluck("id", &dmIDs).Error; err != nil {
		return err
	}
	if len(dmIDs) > 0 {
		if err := tx.Where("danmaku_id IN ?", dmIDs).Delete(&danmaku.DanmakuLike{}).Error; err != nil {
			return err
		}
	}
	if err := tx.Where("video_id = ?", videoID).Delete(&danmaku.Danmaku{}).Error; err != nil {
		return err
	}
	return tx.Where("id = ?", videoID).Delete(&video.Video{}).Error
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
	v, err := a.VideoSvc.GetVideoByID(c.Request.Context(), id)
	if err != nil {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	if v.UserID != uid {
		resp.Err(c, http.StatusForbidden, errcode.CodeForbidden)
		return
	}
	removeVideoDraftFiles(*v)
	if err := a.VideoSvc.DeleteVideoWithCascade(c.Request.Context(), id, nil); err != nil {
		a.Log.Error("delete my video", zap.Error(err), zap.Uint64("video_id", id))
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	a.StorageSvc.PurgeVideo(*v)
	a.esDeleteVideo(id)
	resp.OK(c, okResponse{OK: true})
}
