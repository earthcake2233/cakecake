package handler

import (
	"cakecake/internal/errcode"
	"cakecake/internal/middleware"
	"cakecake/internal/model/comment"
	"cakecake/internal/model/danmaku"
	"cakecake/internal/model/extra"
	"cakecake/internal/model/video"
	"cakecake/internal/pkg/coverval"
	"cakecake/internal/pkg/resp"
	vsvc "cakecake/internal/service/video"
	"fmt"
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
)

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
