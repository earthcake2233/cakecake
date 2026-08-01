package handler

import (
	"cakecake/internal/model/video"
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"cakecake/internal/errcode"
	"cakecake/internal/ffmpeg"
	"cakecake/internal/middleware"
	"cakecake/internal/pkg/coverval"
	"cakecake/internal/pkg/resp"
	"cakecake/internal/worker"
)

const videoStatusDraft = video.StatusDraft

func videoDraftDir(cfgTemp string) string {
	return filepath.Join(cfgTemp, "drafts")
}

func videoDraftRawPath(dir string, videoID uint64, ext string) string {
	ext = strings.TrimPrefix(strings.ToLower(ext), ".")
	if ext == "" {
		ext = "bin"
	}
	return filepath.Join(dir, fmt.Sprintf("%d.%s", videoID, ext))
}

func videoDraftCoverPath(dir string, videoID uint64, ext string) string {
	ext = strings.TrimPrefix(strings.ToLower(ext), ".")
	if ext == "" {
		ext = "jpg"
	}
	if ext == "jpeg" {
		ext = "jpg"
	}
	return filepath.Join(dir, fmt.Sprintf("%d_cover.%s", videoID, ext))
}

func validateVideoDraftContent(title, desc string, hasFile bool) bool {
	title = strings.TrimSpace(title)
	desc = strings.TrimSpace(desc)
	if title == "" && desc == "" && !hasFile {
		return false
	}
	if title != "" && (utf8.RuneCountInString(title) < 1 || utf8.RuneCountInString(title) > 80) {
		return false
	}
	if utf8.RuneCountInString(desc) > 2000 {
		return false
	}
	return true
}

func validateMetadataOnlyDraft(title, desc string) bool {
	title = strings.TrimSpace(title)
	desc = strings.TrimSpace(desc)
	if utf8.RuneCountInString(title) < 1 || utf8.RuneCountInString(title) > 80 {
		return false
	}
	if utf8.RuneCountInString(desc) > 2000 {
		return false
	}
	return true
}

func removeVideoDraftFiles(v video.Video) {
	if p := strings.TrimSpace(v.DraftRawPath); p != "" {
		_ = os.Remove(p)
	}
	if p := strings.TrimSpace(v.DraftCoverPath); p != "" {
		_ = os.Remove(p)
	}
}

func (a *API) uploadDraftCoverToOSS(v *video.Video, coverPath string) error {
	if a.OSS == nil || coverPath == "" {
		return nil
	}
	ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(coverPath)), ".")
	if ext == "jpeg" {
		ext = "jpg"
	}
	key := fmt.Sprintf("covers/%d.%s", v.ID, ext)
	if err := a.OSS.UploadFile(key, coverPath); err != nil {
		return err
	}
	url := a.Cfg.OSSObjectURL(key)
	return a.VideoDraftSvc.UpdateDraftField(context.Background(), v, "cover_url", url)
}

func (a *API) saveDraftVideoFile(fh *multipart.FileHeader, videoID uint64) (rawPath string, dur float64, err error) {
	if err = os.MkdirAll(videoDraftDir(a.Cfg.TempUploadDir), 0o755); err != nil {
		return "", 0, err
	}
	ext := filepath.Ext(fh.Filename)
	rawPath = videoDraftRawPath(videoDraftDir(a.Cfg.TempUploadDir), videoID, ext)
	if err = saveUploadedFile(fh, rawPath); err != nil {
		return "", 0, err
	}
	dur, err = ffmpeg.ProbeDurationSeconds(rawPath)
	if err != nil {
		_ = os.Remove(rawPath)
		return "", 0, err
	}
	if dur > maxDurationSec {
		_ = os.Remove(rawPath)
		return "", 0, fmt.Errorf("duration exceeded")
	}
	return rawPath, dur, nil
}

func (a *API) saveDraftCoverFile(coverFh *multipart.FileHeader, videoID uint64) (coverPath string, err error) {
	if code := coverval.ValidateCoverHeader(coverFh); code != 0 {
		return "", errCoverValidation{code: code}
	}
	if err = os.MkdirAll(videoDraftDir(a.Cfg.TempUploadDir), 0o755); err != nil {
		return "", err
	}
	ext := filepath.Ext(coverFh.Filename)
	coverPath = videoDraftCoverPath(videoDraftDir(a.Cfg.TempUploadDir), videoID, ext)
	if err = saveUploadedFile(coverFh, coverPath); err != nil {
		return "", err
	}
	return coverPath, nil
}

type errCoverValidation struct{ code int }

func (e errCoverValidation) Error() string { return "cover validation" }

// SaveVideoDraft creates a draft video (multipart: file required unless VIDEO_UPLOAD_DISABLED).
// SaveVideoDraft godoc
// @Summary      Save video draft
// @Description  Save a new video as draft without publishing
// @Tags         Videos
// @Produce      json
// @Param        body body object{} true "Draft data"
// @Success      200 {object} map[string]interface{}
// @Router       /videos/draft [post]
func (a *API) SaveVideoDraft(c *gin.Context) {
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
	fh, fileErr := c.FormFile("file")
	metadataOnly := a.Cfg != nil && a.Cfg.VideoUploadDisabled
	if metadataOnly {
		if fileErr == nil {
			resp.Err(c, http.StatusBadRequest, errcode.CodeVideoUploadDisabled)
			return
		}
		if !validateMetadataOnlyDraft(title, desc) {
			resp.Err(c, http.StatusBadRequest, errcode.CodeTitleInvalid)
			return
		}
	} else {
		if fileErr != nil {
			resp.Err(c, http.StatusBadRequest, errcode.CodeUploadMissingFile)
			return
		}
		if fh.Size > maxVideoBytes {
			resp.Err(c, http.StatusBadRequest, errcode.CodeVideoFileTooLarge)
			return
		}
		if !validateVideoDraftContent(title, desc, true) {
			resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
			return
		}
	}
	coverFh, _ := c.FormFile("cover")
	if coverFh != nil {
		if code := coverval.ValidateCoverHeader(coverFh); code != 0 {
			resp.Err(c, http.StatusBadRequest, code)
			return
		}
	}
	tagsJSON, err := parseTagsPostForm(c.PostForm("tags"))
	if err != nil {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	zone := normalizeVideoZone(c.PostForm("zone"))
	v := video.Video{
		UserID:      uid,
		Title:       title,
		Description: desc,
		Status:      videoStatusDraft,
		TagsJSON:    tagsJSON,
		Zone:        zone,
	}
	if err := a.VideoDraftSvc.CreateDraft(context.Background(), &v); err != nil {
		a.Log.Error("create draft video", zap.Error(err))
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	if metadataOnly {
		var coverPath string
		if coverFh != nil {
			coverPath, err = a.saveDraftCoverFile(coverFh, v.ID)
			if err != nil {
				_ = a.VideoDraftSvc.DeleteDraft(context.Background(), v.ID)
				if cv, ok := err.(errCoverValidation); ok {
					resp.Err(c, http.StatusBadRequest, cv.code)
					return
				}
				resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
				return
			}
			if err := a.VideoDraftSvc.UpdateDraftField(context.Background(), &v, "draft_cover_path", coverPath); err != nil {
				_ = os.Remove(coverPath)
				_ = a.VideoDraftSvc.DeleteDraft(context.Background(), v.ID)
				resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
				return
			}
			if tmp, _ := a.VideoDraftSvc.RefetchDraft(context.Background(), v.ID); tmp != nil {
				v = *tmp
			}
			if err := a.uploadDraftCoverToOSS(&v, coverPath); err != nil {
				a.Log.Warn("draft cover oss", zap.Error(err), zap.Uint64("video_id", v.ID))
			} else {
				if tmp, _ := a.VideoDraftSvc.RefetchDraft(context.Background(), v.ID); tmp != nil {
					v = *tmp
				}
			}
		}
		out := gin.H{
			"id":         v.ID,
			"status":     v.Status,
			"title":      v.Title,
			"cover_url":  v.CoverURL,
			"duration":   v.DurationSec,
			"created_at": v.CreatedAt.Format("2006-01-02 15:04:05"),
		}
		appendVideoZoneFields(out, v.Zone)
		resp.JSON(c, http.StatusCreated, errcode.CodeSuccess, out)
		return
	}
	rawPath, dur, err := a.saveDraftVideoFile(fh, v.ID)
	if err != nil {
		_ = a.VideoDraftSvc.DeleteDraft(context.Background(), v.ID)
		if err.Error() == "duration exceeded" {
			resp.Err(c, http.StatusBadRequest, errcode.CodeVideoDurationExceeded)
			return
		}
		a.Log.Warn("draft save video", zap.Error(err))
		resp.Err(c, http.StatusBadRequest, errcode.CodeVideoProbeFailed)
		return
	}
	updates := map[string]interface{}{
		"draft_raw_path": rawPath,
		"duration_sec":   dur,
	}
	var coverPath string
	if coverFh != nil {
		coverPath, err = a.saveDraftCoverFile(coverFh, v.ID)
		if err != nil {
			_ = os.Remove(rawPath)
			_ = a.VideoDraftSvc.DeleteDraft(context.Background(), v.ID)
			if cv, ok := err.(errCoverValidation); ok {
				resp.Err(c, http.StatusBadRequest, cv.code)
				return
			}
			resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
			return
		}
		updates["draft_cover_path"] = coverPath
	}
	if err := a.VideoDraftSvc.UpdateDraft(context.Background(), &v, updates); err != nil {
		removeVideoDraftFiles(video.Video{DraftRawPath: rawPath, DraftCoverPath: coverPath})
		_ = a.VideoDraftSvc.DeleteDraft(context.Background(), v.ID)
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	if tmp, _ := a.VideoDraftSvc.RefetchDraft(context.Background(), v.ID); tmp != nil {
		v = *tmp
	}
	if coverPath != "" {
		if err := a.uploadDraftCoverToOSS(&v, coverPath); err != nil {
			a.Log.Warn("draft cover oss", zap.Error(err), zap.Uint64("video_id", v.ID))
		} else {
			if tmp, _ := a.VideoDraftSvc.RefetchDraft(context.Background(), v.ID); tmp != nil {
				v = *tmp
			}
		}
	}
	out := gin.H{
		"id":         v.ID,
		"status":     v.Status,
		"title":      v.Title,
		"cover_url":  v.CoverURL,
		"duration":   v.DurationSec,
		"created_at": v.CreatedAt.Format("2006-01-02 15:04:05"),
	}
	appendVideoZoneFields(out, v.Zone)
	resp.JSON(c, http.StatusCreated, errcode.CodeSuccess, out)
}

// UpdateVideoDraft updates metadata and optionally replaces file/cover on a draft.
// UpdateVideoDraft godoc
// @Summary      Update video draft
// @Description  Update an existing video draft
// @Tags         Videos
// @Produce      json
// @Param        id path int true "Video ID"
// @Param        body body object{} true "Updated draft data"
// @Success      200 {object} map[string]interface{}
// @Router       /videos/{id}/draft [put]
func (a *API) UpdateVideoDraft(c *gin.Context) {
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
	v, err := a.VideoDraftSvc.GetOwnedDraft(context.Background(), id, uid)
	if err != nil {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	if v.UserID != uid || v.Status != videoStatusDraft {
		resp.Err(c, http.StatusForbidden, errcode.CodeForbidden)
		return
	}
	ct := c.ContentType()
	isMultipart := strings.HasPrefix(ct, "multipart/form-data")
	var title, desc, tagsJSON, zoneRaw string
	var coverFh, fileFh *multipart.FileHeader
	var jsonTags *[]string

	if isMultipart {
		if err := c.Request.ParseMultipartForm(maxVideoBytes + (12 << 20)); err != nil {
			resp.Err(c, http.StatusBadRequest, errcode.CodeMultipartParseError)
			return
		}
		title = strings.TrimSpace(c.PostForm("title"))
		desc = strings.TrimSpace(c.PostForm("description"))
		zoneRaw = c.PostForm("zone")
		tj, err := parseTagsPostForm(c.PostForm("tags"))
		if err != nil {
			resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
			return
		}
		tagsJSON = tj
		fileFh, _ = c.FormFile("file")
		coverFh, _ = c.FormFile("cover")
		if coverFh != nil {
			if code := coverval.ValidateCoverHeader(coverFh); code != 0 {
				resp.Err(c, http.StatusBadRequest, code)
				return
			}
		}
	} else {
		var req updateMyVideoJSON
		if err := c.ShouldBindJSON(&req); err != nil {
			resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
			return
		}
		if strings.TrimSpace(req.Title) != "" {
			title = strings.TrimSpace(req.Title)
		} else {
			title = strings.TrimSpace(v.Title)
		}
		desc = strings.TrimSpace(req.Description)
		jsonTags = req.Tags
		zoneRaw = req.Zone
	}

	hasFile := fileFh != nil || strings.TrimSpace(v.DraftRawPath) != ""
	if !validateVideoDraftContent(title, desc, hasFile) {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}

	updates := map[string]interface{}{
		"title":       title,
		"description": desc,
	}
	if isMultipart {
		updates["tags_json"] = tagsJSON
	} else if jsonTags != nil {
		tj, err := tagsJSONFromStringSlice(*jsonTags)
		if err != nil {
			resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
			return
		}
		updates["tags_json"] = tj
	}
	if z := normalizeVideoZone(zoneRaw); z != "" {
		updates["zone"] = z
	}

	oldRaw := v.DraftRawPath
	oldCover := v.DraftCoverPath

	if fileFh != nil {
		if a.Cfg != nil && a.Cfg.VideoUploadDisabled {
			resp.Err(c, http.StatusBadRequest, errcode.CodeVideoUploadDisabled)
			return
		}
		if fileFh.Size > maxVideoBytes {
			resp.Err(c, http.StatusBadRequest, errcode.CodeVideoFileTooLarge)
			return
		}
		rawPath, dur, err := a.saveDraftVideoFile(fileFh, v.ID)
		if err != nil {
			if err.Error() == "duration exceeded" {
				resp.Err(c, http.StatusBadRequest, errcode.CodeVideoDurationExceeded)
				return
			}
			resp.Err(c, http.StatusBadRequest, errcode.CodeVideoProbeFailed)
			return
		}
		updates["draft_raw_path"] = rawPath
		updates["duration_sec"] = dur
		if oldRaw != "" && oldRaw != rawPath {
			_ = os.Remove(oldRaw)
		}
	}

	var newCoverPath string
	if coverFh != nil {
		cp, err := a.saveDraftCoverFile(coverFh, v.ID)
		if err != nil {
			if cv, ok := err.(errCoverValidation); ok {
				resp.Err(c, http.StatusBadRequest, cv.code)
				return
			}
			resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
			return
		}
		newCoverPath = cp
		updates["draft_cover_path"] = cp
		if oldCover != "" && oldCover != cp {
			_ = os.Remove(oldCover)
		}
	}

	if err := a.VideoDraftSvc.UpdateDraft(context.Background(), v, updates); err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	var rfErr error
	v, rfErr = a.VideoDraftSvc.RefetchDraft(context.Background(), id)
	_ = rfErr
	if newCoverPath != "" {
		if err := a.uploadDraftCoverToOSS(v, newCoverPath); err != nil {
			a.Log.Warn("draft cover oss update", zap.Error(err))
		} else {
			if tmp, _ := a.VideoDraftSvc.RefetchDraft(context.Background(), id); tmp != nil {
				v = tmp
			}
		}
	}
	out := gin.H{
		"id":        v.ID,
		"status":    v.Status,
		"title":     v.Title,
		"cover_url": v.CoverURL,
	}
	appendVideoZoneFields(out, v.Zone)
	resp.OK(c, out)
}

// PublishVideoDraft submits a draft for transcoding (F2).
// PublishVideoDraft godoc
// @Summary      Publish video draft
// @Description  Publish a draft video to make it visible
// @Tags         Videos
// @Produce      json
// @Param        id path int true "Video ID"
// @Success      200 {object} map[string]interface{}
// @Router       /videos/{id}/publish [post]
func (a *API) PublishVideoDraft(c *gin.Context) {
	if a.Cfg != nil && a.Cfg.VideoUploadDisabled {
		resp.Err(c, http.StatusBadRequest, errcode.CodeVideoUploadDisabled)
		return
	}
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
	v, err := a.VideoDraftSvc.GetOwnedDraftByStatus(context.Background(), id, uid, videoStatusDraft)
	if err != nil {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	if v.UserID != uid || v.Status != videoStatusDraft {
		resp.Err(c, http.StatusForbidden, errcode.CodeForbidden)
		return
	}
	title := strings.TrimSpace(v.Title)
	desc := strings.TrimSpace(v.Description)
	if utf8.RuneCountInString(title) < 1 || utf8.RuneCountInString(title) > 80 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeTitleInvalid)
		return
	}
	if utf8.RuneCountInString(desc) > 2000 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeIntroTooLong)
		return
	}
	rawPath := strings.TrimSpace(v.DraftRawPath)
	if rawPath == "" {
		resp.Err(c, http.StatusBadRequest, errcode.CodeUploadMissingFile)
		return
	}
	if _, err := os.Stat(rawPath); err != nil {
		resp.Err(c, http.StatusBadRequest, errcode.CodeUploadMissingFile)
		return
	}
	coverPath := strings.TrimSpace(v.DraftCoverPath)
	job := worker.TranscodeJob{VideoID: v.ID, RawPath: rawPath, CoverPath: coverPath, RetryCount: 0}
	body, _ := json.Marshal(job)
	if err := a.MQ.PublishTranscode(context.Background(), body); err != nil {
		a.Log.Error("publish transcode from draft", zap.Error(err))
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	updates := map[string]interface{}{
		"status":           video.StatusProcessing,
		"draft_raw_path":   "",
		"draft_cover_path": "",
	}
	if err := a.VideoDraftSvc.UpdateDraft(context.Background(), v, updates); err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	a.Log.Info("draft published to transcode queue", zap.Uint64("video_id", v.ID))
	resp.OK(c, gin.H{"id": v.ID, "status": video.StatusProcessing})
}

func videoStatusAllowsMediaReplace(st string) bool {
	switch st {
	case video.StatusFailed, video.StatusRejected:
		return true
	default:
		return false
	}
}

// ReplaceVideoMedia replaces the source file for failed/rejected videos: purge OSS, re-queue transcode.
// ReplaceVideoMedia godoc
// @Summary      Replace video media
// @Description  Upload a new media file to replace existing video
// @Tags         Videos
// @Accept       multipart/form-data
// @Produce      json
// @Param        id path int true "Video ID"
// @Param        file formData file true "New video file"
// @Success      200 {object} map[string]interface{}
// @Router       /videos/{id}/replace-media [post]
func (a *API) ReplaceVideoMedia(c *gin.Context) {
	if a.Cfg != nil && a.Cfg.VideoUploadDisabled {
		resp.Err(c, http.StatusBadRequest, errcode.CodeVideoUploadDisabled)
		return
	}
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
	v, err := a.VideoDraftSvc.GetDraftByID(context.Background(), id)
	if err != nil {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	if v.UserID != uid || !videoStatusAllowsMediaReplace(v.Status) {
		resp.Err(c, http.StatusForbidden, errcode.CodeForbidden)
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
	tagsJSON, err := parseTagsPostForm(c.PostForm("tags"))
	if err != nil {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}

	purgeVideoOSSObjects(a.Cfg, a.OSS, a.Log, *v)
	a.esDeleteVideo(id)
	removeVideoDraftFiles(*v)

	rawPath, dur, err := a.saveDraftVideoFile(fh, v.ID)
	if err != nil {
		if err.Error() == "duration exceeded" {
			resp.Err(c, http.StatusBadRequest, errcode.CodeVideoDurationExceeded)
			return
		}
		a.Log.Warn("replace video save file", zap.Error(err))
		resp.Err(c, http.StatusBadRequest, errcode.CodeVideoProbeFailed)
		return
	}

	var coverPath string
	if coverFh != nil {
		coverPath, err = a.saveDraftCoverFile(coverFh, v.ID)
		if err != nil {
			_ = os.Remove(rawPath)
			if cv, ok := err.(errCoverValidation); ok {
				resp.Err(c, http.StatusBadRequest, cv.code)
				return
			}
			resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
			return
		}
	}

	updates := map[string]interface{}{
		"title":            title,
		"description":      desc,
		"tags_json":        tagsJSON,
		"status":           video.StatusProcessing,
		"fail_reason":      "",
		"video_url":        "",
		"cover_url":        "",
		"duration_sec":     dur,
		"draft_raw_path":   rawPath,
		"draft_cover_path": coverPath,
	}
	if z := normalizeVideoZone(c.PostForm("zone")); z != "" {
		updates["zone"] = z
	}
	if err := a.VideoDraftSvc.UpdateDraft(context.Background(), v, updates); err != nil {
		removeVideoDraftFiles(video.Video{DraftRawPath: rawPath, DraftCoverPath: coverPath})
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}

	job := worker.TranscodeJob{VideoID: v.ID, RawPath: rawPath, CoverPath: coverPath, RetryCount: 0}
	body, _ := json.Marshal(job)
	if err := a.MQ.PublishTranscode(context.Background(), body); err != nil {
		a.Log.Error("publish transcode after replace", zap.Error(err))
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	if err := a.VideoDraftSvc.UpdateDraft(context.Background(), v, map[string]interface{}{
		"draft_raw_path":   "",
		"draft_cover_path": "",
	}); err != nil {
		a.Log.Error("clear draft paths after replace", zap.Error(err))
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	a.Log.Info("video media replaced and queued for transcode", zap.Uint64("video_id", v.ID))
	resp.OK(c, gin.H{"id": v.ID, "status": video.StatusProcessing})
}

// GetMyVideoDraftSource streams the draft raw file for the uploader preview.
// GetMyVideoDraftSource godoc
// @Summary      Get draft source
// @Description  Get the original uploaded source info for a draft video
// @Tags         Videos
// @Produce      json
// @Param        id path int true "Video ID"
// @Success      200 {object} map[string]interface{}
// @Router       /users/me/videos/{id}/draft-source [get]
func (a *API) GetMyVideoDraftSource(c *gin.Context) {
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
	v, err := a.VideoDraftSvc.GetOwnedDraftByStatus(context.Background(), id, uid, videoStatusDraft)
	if err != nil {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}

	rawPath := strings.TrimSpace(v.DraftRawPath)
	if rawPath == "" {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	if _, err := os.Stat(rawPath); err != nil {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	c.Header("Content-Type", "application/octet-stream")
	c.Header("Accept-Ranges", "bytes")
	c.File(rawPath)
}
