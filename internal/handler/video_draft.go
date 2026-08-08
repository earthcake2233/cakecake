package handler

import (
	"cakecake/internal/errcode"
	"cakecake/internal/middleware"
	"cakecake/internal/model/video"
	"cakecake/internal/pkg/coverval"
	"cakecake/internal/pkg/resp"
	"context"
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
)

const videoStatusDraft = video.StatusDraft

type videoDraftFullResponse struct {
	ID        uint64  `json:"id"`
	Status    string  `json:"status"`
	Title     string  `json:"title"`
	CoverURL  string  `json:"cover_url"`
	Duration  float64 `json:"duration"`
	CreatedAt string  `json:"created_at"`
	videoZoneFields
}

type videoDraftBriefResponse struct {
	ID       uint64 `json:"id"`
	Status   string `json:"status"`
	Title    string `json:"title"`
	CoverURL string `json:"cover_url"`
	videoZoneFields
}

type videoDraftStatusResponse struct {
	ID     uint64 `json:"id"`
	Status string `json:"status"`
}

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

func (a *API) uploadDraftCoverToOSS(ctx context.Context, v *video.Video, coverPath string) error {
	if !a.StorageSvc.Enabled() || coverPath == "" {
		return nil
	}
	ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(coverPath)), ".")
	if ext == "jpeg" {
		ext = "jpg"
	}
	key := fmt.Sprintf("covers/%d.%s", v.ID, ext)
	if err := a.StorageSvc.UploadFile(key, coverPath); err != nil {
		return err
	}
	url := a.Cfg.OSSObjectURL(key)
	return a.VideoDraftSvc.UpdateDraftField(ctx, v, "cover_url", url)
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
	dur, err = a.VideoDraftSvc.ProbeDurationSeconds(rawPath)
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

// Error implements the error interface for cover validation failures.
func (e errCoverValidation) Error() string { return "cover validation" }

type draftCreateInput struct {
	metadataOnly bool
	title        string
	desc         string
	tagsJSON     string
	zone         string
	fileFh       *multipart.FileHeader
	coverFh      *multipart.FileHeader
}

// parseDraftCreateForm parses and validates the multipart create-draft form.
// A non-zero return code means the request is invalid and should be rejected.
func (a *API) parseDraftCreateForm(c *gin.Context) (draftCreateInput, int) {
	var in draftCreateInput
	if err := c.Request.ParseMultipartForm(maxVideoBytes + (12 << 20)); err != nil {
		a.Log.Warn("parse multipart form", zap.Error(err))
		return in, errcode.CodeMultipartParseError
	}
	in.title = strings.TrimSpace(c.PostForm("title"))
	in.desc = strings.TrimSpace(c.PostForm("description"))
	in.metadataOnly = a.Cfg != nil && a.Cfg.VideoUploadDisabled
	fh, fileErr := c.FormFile("file")
	if in.metadataOnly {
		if fileErr == nil {
			return in, errcode.CodeVideoUploadDisabled
		}
		if !validateMetadataOnlyDraft(in.title, in.desc) {
			return in, errcode.CodeTitleInvalid
		}
	} else {
		if fileErr != nil {
			return in, errcode.CodeUploadMissingFile
		}
		if fh.Size > maxVideoBytes {
			return in, errcode.CodeVideoFileTooLarge
		}
		if !validateVideoDraftContent(in.title, in.desc, true) {
			return in, errcode.CodeParamError
		}
	}
	in.fileFh = fh
	in.coverFh, _ = c.FormFile("cover")
	if in.coverFh != nil {
		if code := coverval.ValidateCoverHeader(in.coverFh); code != 0 {
			return in, code
		}
	}
	tagsJSON, err := parseTagsPostForm(c.PostForm("tags"))
	if err != nil {
		return in, errcode.CodeParamError
	}
	in.tagsJSON = tagsJSON
	in.zone = normalizeVideoZone(c.PostForm("zone"))
	return in, 0
}

// createDraftRecord persists a new draft row from parsed form input.
func (a *API) createDraftRecord(ctx context.Context, uid uint64, in draftCreateInput) (*video.Video, int) {
	v := &video.Video{
		UserID:      uid,
		Title:       in.title,
		Description: in.desc,
		Status:      videoStatusDraft,
		TagsJSON:    in.tagsJSON,
		Zone:        in.zone,
	}
	if err := a.VideoDraftSvc.CreateDraft(ctx, v); err != nil {
		a.Log.Error("create draft video", zap.Error(err))
		return nil, errcode.CodeInternalError
	}
	return v, 0
}

// saveDraftCoverFileChecked saves a cover file, mapping validation errors to error codes.
func (a *API) saveDraftCoverFileChecked(coverFh *multipart.FileHeader, videoID uint64) (string, int) {
	coverPath, err := a.saveDraftCoverFile(coverFh, videoID)
	if err != nil {
		if cv, ok := err.(errCoverValidation); ok {
			return "", cv.code
		}
		return "", errcode.CodeInternalError
	}
	return coverPath, 0
}

// httpStatusForCode maps internal error codes to HTTP status for cover/file helpers.
func httpStatusForCode(code int) int {
	if code == errcode.CodeInternalError {
		return http.StatusInternalServerError
	}
	return http.StatusBadRequest
}

// refetchDraft refreshes v from the DB when possible.
func (a *API) refetchDraft(ctx context.Context, v *video.Video) {
	if tmp, _ := a.VideoDraftSvc.RefetchDraft(ctx, v.ID); tmp != nil {
		*v = *tmp
	}
}

// uploadDraftCoverAndRefetch uploads a saved cover to OSS and refreshes the draft.
func (a *API) uploadDraftCoverAndRefetch(ctx context.Context, v *video.Video, coverPath string) {
	if coverPath == "" {
		return
	}
	if err := a.uploadDraftCoverToOSS(ctx, v, coverPath); err != nil {
		a.Log.Warn("draft cover oss", zap.Error(err), zap.Uint64("video_id", v.ID))
		return
	}
	if tmp, _ := a.VideoDraftSvc.RefetchDraft(ctx, v.ID); tmp != nil {
		*v = *tmp
	}
}

func draftFullResponse(v video.Video) videoDraftFullResponse {
	out := videoDraftFullResponse{
		ID:        v.ID,
		Status:    v.Status,
		Title:     v.Title,
		CoverURL:  v.CoverURL,
		Duration:  v.DurationSec,
		CreatedAt: v.CreatedAt.Format("2006-01-02 15:04:05"),
	}
	appendVideoZoneFields(&out.videoZoneFields, v.Zone)
	return out
}

func draftBriefResponse(v video.Video) videoDraftBriefResponse {
	out := videoDraftBriefResponse{
		ID:       v.ID,
		Status:   v.Status,
		Title:    v.Title,
		CoverURL: v.CoverURL,
	}
	appendVideoZoneFields(&out.videoZoneFields, v.Zone)
	return out
}

type draftUpdateInput struct {
	isMultipart bool
	title       string
	desc        string
	tagsJSON    string
	zoneRaw     string
	jsonTags    *[]string
	fileFh      *multipart.FileHeader
	coverFh     *multipart.FileHeader
}

// parseDraftUpdateForm parses and validates an update request (JSON or multipart).
func (a *API) parseDraftUpdateForm(c *gin.Context, existing *video.Video) (draftUpdateInput, int) {
	var in draftUpdateInput
	ct := c.ContentType()
	in.isMultipart = strings.HasPrefix(ct, "multipart/form-data")
	if in.isMultipart {
		if err := c.Request.ParseMultipartForm(maxVideoBytes + (12 << 20)); err != nil {
			a.Log.Warn("parse multipart form", zap.Error(err))
			return in, errcode.CodeMultipartParseError
		}
		in.title = strings.TrimSpace(c.PostForm("title"))
		in.desc = strings.TrimSpace(c.PostForm("description"))
		in.zoneRaw = c.PostForm("zone")
		tj, err := parseTagsPostForm(c.PostForm("tags"))
		if err != nil {
			return in, errcode.CodeParamError
		}
		in.tagsJSON = tj
		in.fileFh, _ = c.FormFile("file")
		in.coverFh, _ = c.FormFile("cover")
		if in.coverFh != nil {
			if code := coverval.ValidateCoverHeader(in.coverFh); code != 0 {
				return in, code
			}
		}
		return in, 0
	}
	var req updateMyVideoJSON
	if err := c.ShouldBindJSON(&req); err != nil {
		return in, errcode.CodeParamError
	}
	if strings.TrimSpace(req.Title) != "" {
		in.title = strings.TrimSpace(req.Title)
	} else {
		in.title = strings.TrimSpace(existing.Title)
	}
	in.desc = strings.TrimSpace(req.Description)
	in.jsonTags = req.Tags
	in.zoneRaw = req.Zone
	return in, 0
}

// applyDraftFileUpdate saves a replacement draft file, removing the old one.
func (a *API) applyDraftFileUpdate(v *video.Video, fileFh *multipart.FileHeader, updates map[string]interface{}) int {
	if a.Cfg != nil && a.Cfg.VideoUploadDisabled {
		return errcode.CodeVideoUploadDisabled
	}
	if fileFh.Size > maxVideoBytes {
		return errcode.CodeVideoFileTooLarge
	}
	rawPath, dur, err := a.saveDraftVideoFile(fileFh, v.ID)
	if err != nil {
		if err.Error() == "duration exceeded" {
			return errcode.CodeVideoDurationExceeded
		}
		return errcode.CodeVideoProbeFailed
	}
	updates["draft_raw_path"] = rawPath
	updates["duration_sec"] = dur
	if oldRaw := v.DraftRawPath; oldRaw != "" && oldRaw != rawPath {
		_ = os.Remove(oldRaw)
	}
	return 0
}

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
	in, code := a.parseDraftCreateForm(c)
	if code != 0 {
		resp.Err(c, http.StatusBadRequest, code)
		return
	}
	v, code := a.createDraftRecord(c.Request.Context(), uid, in)
	if code != 0 {
		resp.Err(c, http.StatusInternalServerError, code)
		return
	}
	if in.metadataOnly {
		if in.coverFh != nil {
			coverPath, cv := a.saveDraftCoverFileChecked(in.coverFh, v.ID)
			if cv != 0 {
				_ = a.VideoDraftSvc.DeleteDraft(c.Request.Context(), v.ID)
				resp.Err(c, httpStatusForCode(cv), cv)
				return
			}
			if err := a.VideoDraftSvc.UpdateDraftField(c.Request.Context(), v, "draft_cover_path", coverPath); err != nil {
				_ = os.Remove(coverPath)
				_ = a.VideoDraftSvc.DeleteDraft(c.Request.Context(), v.ID)
				resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
				return
			}
			a.refetchDraft(c.Request.Context(), v)
			a.uploadDraftCoverAndRefetch(c.Request.Context(), v, coverPath)
		}
		resp.JSON(c, http.StatusCreated, errcode.CodeSuccess, draftFullResponse(*v))
		return
	}
	rawPath, dur, err := a.saveDraftVideoFile(in.fileFh, v.ID)
	if err != nil {
		_ = a.VideoDraftSvc.DeleteDraft(c.Request.Context(), v.ID)
		if err.Error() == "duration exceeded" {
			resp.Err(c, http.StatusBadRequest, errcode.CodeVideoDurationExceeded)
			return
		}
		a.Log.Warn("draft save video", zap.Error(err))
		resp.Err(c, http.StatusBadRequest, errcode.CodeVideoProbeFailed)
		return
	}
	updates := map[string]interface{}{"draft_raw_path": rawPath, "duration_sec": dur}
	var coverPath string
	if in.coverFh != nil {
		coverPath, code = a.saveDraftCoverFileChecked(in.coverFh, v.ID)
		if code != 0 {
			_ = os.Remove(rawPath)
			_ = a.VideoDraftSvc.DeleteDraft(c.Request.Context(), v.ID)
			resp.Err(c, httpStatusForCode(code), code)
			return
		}
		updates["draft_cover_path"] = coverPath
	}
	if err := a.VideoDraftSvc.UpdateDraft(c.Request.Context(), v, updates); err != nil {
		removeVideoDraftFiles(video.Video{DraftRawPath: rawPath, DraftCoverPath: coverPath})
		_ = a.VideoDraftSvc.DeleteDraft(c.Request.Context(), v.ID)
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	a.refetchDraft(c.Request.Context(), v)
	a.uploadDraftCoverAndRefetch(c.Request.Context(), v, coverPath)
	resp.JSON(c, http.StatusCreated, errcode.CodeSuccess, draftFullResponse(*v))
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
	v, err := a.VideoDraftSvc.GetOwnedDraft(c.Request.Context(), id, uid)
	if err != nil {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	if v.UserID != uid || v.Status != videoStatusDraft {
		resp.Err(c, http.StatusForbidden, errcode.CodeForbidden)
		return
	}
	in, code := a.parseDraftUpdateForm(c, v)
	if code != 0 {
		resp.Err(c, httpStatusForCode(code), code)
		return
	}
	hasFile := in.fileFh != nil || strings.TrimSpace(v.DraftRawPath) != ""
	if !validateVideoDraftContent(in.title, in.desc, hasFile) {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	updates := map[string]interface{}{
		"title":       in.title,
		"description": in.desc,
	}
	if in.isMultipart {
		updates["tags_json"] = in.tagsJSON
	} else if in.jsonTags != nil {
		tj, err := tagsJSONFromStringSlice(*in.jsonTags)
		if err != nil {
			resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
			return
		}
		updates["tags_json"] = tj
	}
	if z := normalizeVideoZone(in.zoneRaw); z != "" {
		updates["zone"] = z
	}
	if in.fileFh != nil {
		if code := a.applyDraftFileUpdate(v, in.fileFh, updates); code != 0 {
			resp.Err(c, httpStatusForCode(code), code)
			return
		}
	}
	var newCoverPath string
	if in.coverFh != nil {
		cp, code := a.saveDraftCoverFileChecked(in.coverFh, v.ID)
		if code != 0 {
			resp.Err(c, httpStatusForCode(code), code)
			return
		}
		newCoverPath = cp
		updates["draft_cover_path"] = cp
		if oldCover := v.DraftCoverPath; oldCover != "" && oldCover != cp {
			_ = os.Remove(oldCover)
		}
	}
	if err := a.VideoDraftSvc.UpdateDraft(c.Request.Context(), v, updates); err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	a.refetchDraft(c.Request.Context(), v)
	a.uploadDraftCoverAndRefetch(c.Request.Context(), v, newCoverPath)
	resp.OK(c, draftBriefResponse(*v))
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
	v, err := a.VideoDraftSvc.GetOwnedDraftByStatus(c.Request.Context(), id, uid, videoStatusDraft)
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
	if err := a.VideoDraftSvc.EnqueueTranscode(c.Request.Context(), v.ID, rawPath, coverPath); err != nil {
		a.Log.Error("publish transcode from draft", zap.Error(err))
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	updates := map[string]interface{}{
		"status":           video.StatusProcessing,
		"draft_raw_path":   "",
		"draft_cover_path": "",
	}
	if err := a.VideoDraftSvc.UpdateDraft(c.Request.Context(), v, updates); err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	a.Log.Info("draft published to transcode queue", zap.Uint64("video_id", v.ID))
	resp.OK(c, videoDraftStatusResponse{ID: v.ID, Status: video.StatusProcessing})
}
