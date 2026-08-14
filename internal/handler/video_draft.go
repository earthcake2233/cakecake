package handler

import (
	"cakecake/internal/errcode"
	"cakecake/internal/middleware"
	"cakecake/internal/model/video"
	"cakecake/internal/pkg/resp"
	vsvc "cakecake/internal/service/video"
	"context"
	"errors"
	"fmt"
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

// draftCreateInput is the JSON create-draft payload. Media is referenced by
// OSS object keys staged through the draft upload ticket endpoint.
type draftCreateInput struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
	Zone        string   `json:"zone"`
	RawKey      string   `json:"raw_key"`
	CoverKey    string   `json:"cover_key,omitempty"`
	Duration    float64  `json:"duration,omitempty"`
}

// draftUpdateInput is the JSON update-draft payload; raw_key/cover_key are
// optional media replacements.
type draftUpdateInput struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Tags        []string `json:"tags,omitempty"`
	Zone        string   `json:"zone,omitempty"`
	RawKey      string   `json:"raw_key,omitempty"`
	CoverKey    string   `json:"cover_key,omitempty"`
	Duration    float64  `json:"duration,omitempty"`
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

// isObjectKeyReference reports whether a stored draft source reference is an
// OSS object key (new drafts) rather than a legacy absolute local path.
func isObjectKeyReference(ref string) bool {
	for _, p := range []string{"drafts/", "uploads/", "raws/", "covers/"} {
		if strings.HasPrefix(ref, p) {
			return true
		}
	}
	return false
}

func coverKeyExt(key string) string {
	ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(strings.TrimSpace(key))), ".")
	if ext == "" {
		ext = "jpg"
	}
	if ext == "jpeg" {
		ext = "jpg"
	}
	return ext
}

// mediaErrorStatus maps draft/direct media validation errors to HTTP status +
// error code pairs.
func mediaErrorStatus(err error) (int, int) {
	switch {
	case errors.Is(err, vsvc.ErrDraftMediaUnavailable):
		return http.StatusServiceUnavailable, errcode.CodeDirectUploadUnavailable
	case errors.Is(err, vsvc.ErrTranscodeQueueFull):
		return http.StatusServiceUnavailable, errcode.CodeTranscodeQueueFull
	case errors.Is(err, vsvc.ErrDraftMediaEmpty):
		return http.StatusBadRequest, errcode.CodeUploadMissingFile
	case errors.Is(err, vsvc.ErrDraftMediaInvalidKey):
		return http.StatusBadRequest, errcode.CodeParamError
	case errors.Is(err, vsvc.ErrDraftMediaMissing):
		return http.StatusBadRequest, errcode.CodeDirectUploadSourceMissing
	case errors.Is(err, vsvc.ErrDraftMediaTooLarge):
		return http.StatusBadRequest, errcode.CodeVideoFileTooLarge
	default:
		return http.StatusInternalServerError, errcode.CodeInternalError
	}
}

// copyDraftCoverToPublic copies the private draft cover object to the public
// covers/{id}.{ext} key and records its URL. Server-side copy keeps the bytes
// off the API server and leaves the draft object private.
func (a *API) copyDraftCoverToPublic(ctx context.Context, v *video.Video, coverKey string) error {
	coverKey = strings.TrimSpace(coverKey)
	if coverKey == "" || !a.StorageSvc.Enabled() {
		return nil
	}
	dst := fmt.Sprintf("covers/%d.%s", v.ID, coverKeyExt(coverKey))
	if err := a.StorageSvc.CopyObject(coverKey, dst); err != nil {
		return err
	}
	return a.VideoDraftSvc.UpdateDraftField(ctx, v, "cover_url", a.Cfg.OSSObjectURL(dst))
}

// purgeDraftMedia removes the previous draft media (OSS object or legacy local
// file) once a draft has been replaced or deleted. Called only after the new
// media has been durably recorded.
func (a *API) purgeDraftMedia(v video.Video) {
	for _, ref := range []string{v.DraftRawKey, v.DraftCoverKey} {
		ref = strings.TrimSpace(ref)
		if ref == "" {
			continue
		}
		if isObjectKeyReference(ref) {
			if a.StorageSvc.Enabled() {
				if err := a.StorageSvc.DeleteObject(ref); err != nil {
					a.Log.Warn("delete draft object", zap.String("key", ref), zap.Error(err))
				}
			}
			continue
		}
		_ = os.Remove(ref)
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

// refetchDraft refreshes v from the DB when possible.
func (a *API) refetchDraft(ctx context.Context, v *video.Video) {
	if tmp, _ := a.VideoDraftSvc.RefetchDraft(ctx, v.ID); tmp != nil {
		*v = *tmp
	}
}

// CreateVideoDraftUploadTicket issues presigned PUT URLs so the browser can
// stage draft media (raw video + optional cover) straight to OSS.
// CreateVideoDraftUploadTicket godoc
// @Summary      Create draft upload ticket
// @Description  Get presigned upload URLs for draft media (direct-to-OSS)
// @Tags         Videos
// @Accept       json
// @Produce      json
// @Param        body body object{} true "filename and optional cover_filename"
// @Success      200 {object} map[string]interface{}
// @Router       /videos/draft/upload-ticket [post]
func (a *API) CreateVideoDraftUploadTicket(c *gin.Context) {
	if a.rejectVideoUploadDisabled(c) {
		return
	}
	uid, ok := middleware.UserID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	var req createUploadTicketRequest
	if err := c.ShouldBindJSON(&req); err != nil ||
		(strings.TrimSpace(req.Filename) == "" && strings.TrimSpace(req.CoverFilename) == "") {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	ticket, err := a.VideoDraftSvc.CreateDraftUploadTicket(c.Request.Context(), uid, strings.TrimSpace(req.Filename), strings.TrimSpace(req.CoverFilename), strings.TrimSpace(req.ContentType), strings.TrimSpace(req.CoverContentType))
	if err != nil {
		if errors.Is(err, vsvc.ErrDraftMediaUnavailable) {
			resp.Err(c, http.StatusServiceUnavailable, errcode.CodeDirectUploadUnavailable)
			return
		}
		a.Log.Error("create draft upload ticket", zap.Uint64("uid", uid), zap.Error(err))
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	resp.JSON(c, http.StatusOK, errcode.CodeSuccess, ticket)
}

// SaveVideoDraft creates a draft. Metadata-only drafts are allowed when
// VIDEO_UPLOAD_DISABLED; otherwise raw_key from the draft ticket is required.
// SaveVideoDraft godoc
// @Summary      Save video draft
// @Description  Save a new video as draft without publishing
// @Tags         Videos
// @Accept       json
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
	var in draftCreateInput
	if err := c.ShouldBindJSON(&in); err != nil {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	in.Title = strings.TrimSpace(in.Title)
	in.Description = strings.TrimSpace(in.Description)
	in.RawKey = strings.TrimSpace(in.RawKey)
	in.CoverKey = strings.TrimSpace(in.CoverKey)
	disabled := a.Cfg != nil && a.Cfg.VideoUploadDisabled
	if disabled && (in.RawKey != "" || in.CoverKey != "") {
		resp.Err(c, http.StatusBadRequest, errcode.CodeVideoUploadDisabled)
		return
	}
	if in.RawKey == "" {
		if disabled {
			if !validateMetadataOnlyDraft(in.Title, in.Description) {
				resp.Err(c, http.StatusBadRequest, errcode.CodeTitleInvalid)
				return
			}
		} else {
			resp.Err(c, http.StatusBadRequest, errcode.CodeUploadMissingFile)
			return
		}
	} else if !validateVideoDraftContent(in.Title, in.Description, true) {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	tagsJSON, err := tagsJSONFromStringSlice(in.Tags)
	if err != nil {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	var media vsvc.DraftMedia
	if in.RawKey != "" {
		media, err = a.VideoDraftSvc.ValidateDraftMedia(c.Request.Context(), uid, in.RawKey, in.CoverKey)
		if err != nil {
			hs, code := mediaErrorStatus(err)
			resp.Err(c, hs, code)
			return
		}
	}
	v := &video.Video{
		UserID:        uid,
		Title:         in.Title,
		Description:   in.Description,
		Status:        videoStatusDraft,
		TagsJSON:      tagsJSON,
		Zone:          normalizeVideoZone(in.Zone),
		DraftRawKey:   media.RawKey,
		DraftCoverKey: media.CoverKey,
		DurationSec:   vsvc.NormalizeDurationHint(in.Duration),
	}
	if err := a.VideoDraftSvc.CreateDraft(c.Request.Context(), v); err != nil {
		a.Log.Error("create draft video", zap.Error(err))
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	if media.CoverKey != "" {
		if err := a.copyDraftCoverToPublic(c.Request.Context(), v, media.CoverKey); err != nil {
			a.Log.Warn("draft cover oss copy", zap.Uint64("video_id", v.ID), zap.Error(err))
			_ = a.VideoDraftSvc.DeleteDraft(c.Request.Context(), v.ID)
			a.purgeDraftMedia(*v)
			resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
			return
		}
		a.refetchDraft(c.Request.Context(), v)
	}
	resp.JSON(c, http.StatusCreated, errcode.CodeSuccess, draftFullResponse(*v))
}

// UpdateVideoDraft updates metadata and optionally replaces draft media
// (raw_key/cover_key from the draft ticket).
// UpdateVideoDraft godoc
// @Summary      Update video draft
// @Description  Update an existing video draft
// @Tags         Videos
// @Accept       json
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
	var in draftUpdateInput
	if err := c.ShouldBindJSON(&in); err != nil {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	in.Title = strings.TrimSpace(in.Title)
	in.Description = strings.TrimSpace(in.Description)
	in.RawKey = strings.TrimSpace(in.RawKey)
	in.CoverKey = strings.TrimSpace(in.CoverKey)
	disabled := a.Cfg != nil && a.Cfg.VideoUploadDisabled
	if disabled && (in.RawKey != "" || in.CoverKey != "") {
		resp.Err(c, http.StatusBadRequest, errcode.CodeVideoUploadDisabled)
		return
	}
	title := in.Title
	if title == "" {
		title = strings.TrimSpace(v.Title)
	}
	desc := in.Description
	var media *vsvc.DraftMedia
	if in.RawKey != "" {
		m, verr := a.VideoDraftSvc.ValidateDraftMedia(c.Request.Context(), uid, in.RawKey, in.CoverKey)
		if verr != nil {
			hs, code := mediaErrorStatus(verr)
			resp.Err(c, hs, code)
			return
		}
		media = &m
	} else if in.CoverKey != "" {
		if verr := a.VideoDraftSvc.ValidateDraftCover(c.Request.Context(), uid, in.CoverKey); verr != nil {
			hs, code := mediaErrorStatus(verr)
			resp.Err(c, hs, code)
			return
		}
	}
	hasFile := media != nil || strings.TrimSpace(v.DraftRawKey) != ""
	if !validateVideoDraftContent(title, desc, hasFile) {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	updates := map[string]interface{}{
		"title":       title,
		"description": desc,
	}
	if in.Tags != nil {
		tj, terr := tagsJSONFromStringSlice(in.Tags)
		if terr != nil {
			resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
			return
		}
		updates["tags_json"] = tj
	}
	if z := normalizeVideoZone(in.Zone); z != "" {
		updates["zone"] = z
	}
	oldRaw, oldCover := strings.TrimSpace(v.DraftRawKey), strings.TrimSpace(v.DraftCoverKey)
	if media != nil {
		updates["draft_raw_key"] = media.RawKey
		updates["draft_cover_key"] = media.CoverKey
		updates["duration_sec"] = vsvc.NormalizeDurationHint(in.Duration)
	}
	if err := a.VideoDraftSvc.UpdateDraft(c.Request.Context(), v, updates); err != nil {
		a.Log.Error("update draft", zap.Uint64("video_id", id), zap.Error(err))
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	newCoverKey := ""
	if media != nil {
		newCoverKey = media.CoverKey
		a.purgeDraftMedia(video.Video{DraftRawKey: oldRaw, DraftCoverKey: oldCover})
	} else {
		newCoverKey = in.CoverKey
		if newCoverKey != "" && newCoverKey != oldCover {
			a.purgeDraftMedia(video.Video{DraftCoverKey: oldCover})
		}
	}
	if newCoverKey != "" {
		if cerr := a.copyDraftCoverToPublic(c.Request.Context(), v, newCoverKey); cerr != nil {
			a.Log.Warn("draft cover oss copy", zap.Uint64("video_id", id), zap.Error(cerr))
			resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
			return
		}
	}
	a.refetchDraft(c.Request.Context(), v)
	resp.OK(c, draftBriefResponse(*v))
}

// PublishVideoDraft submits a draft for transcoding. OSS-keyed drafts are
// validated and committed atomically (outbox row + status update).
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
	rawRef := strings.TrimSpace(v.DraftRawKey)
	coverRef := strings.TrimSpace(v.DraftCoverKey)
	if rawRef == "" {
		resp.Err(c, http.StatusBadRequest, errcode.CodeUploadMissingFile)
		return
	}
	if isObjectKeyReference(rawRef) {
		media, verr := a.VideoDraftSvc.ValidateDraftMedia(c.Request.Context(), uid, rawRef, coverRef)
		if verr != nil {
			hs, code := mediaErrorStatus(verr)
			resp.Err(c, hs, code)
			return
		}
		media.DurationSec = vsvc.NormalizeDurationHint(v.DurationSec)
		if serr := a.VideoDraftSvc.SubmitDraft(c.Request.Context(), v, media); serr != nil {
			if errors.Is(serr, vsvc.ErrTranscodeQueueFull) {
				resp.Err(c, http.StatusServiceUnavailable, errcode.CodeTranscodeQueueFull)
				return
			}
			a.Log.Error("publish draft from oss", zap.Uint64("video_id", id), zap.Error(serr))
			resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
			return
		}
		a.Log.Info("draft published to transcode outbox", zap.Uint64("video_id", v.ID))
		resp.OK(c, videoDraftStatusResponse{ID: v.ID, Status: video.StatusProcessing})
		return
	}
	// Legacy local-path draft (pre OSS migration): keep the single-host path
	// working until the draft is resubmitted.
	if _, err := os.Stat(rawRef); err != nil {
		resp.Err(c, http.StatusBadRequest, errcode.CodeUploadMissingFile)
		return
	}
	if err := a.VideoDraftSvc.EnqueueTranscode(c.Request.Context(), v.ID, rawRef, coverRef); err != nil {
		if errors.Is(err, vsvc.ErrTranscodeQueueFull) {
			resp.Err(c, http.StatusServiceUnavailable, errcode.CodeTranscodeQueueFull)
			return
		}
		a.Log.Error("publish transcode from draft", zap.Error(err))
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	if err := a.VideoDraftSvc.UpdateDraft(c.Request.Context(), v, map[string]interface{}{
		"status":          video.StatusProcessing,
		"draft_raw_key":   "",
		"draft_cover_key": "",
	}); err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	a.Log.Info("draft published to transcode queue", zap.Uint64("video_id", v.ID))
	resp.OK(c, videoDraftStatusResponse{ID: v.ID, Status: video.StatusProcessing})
}
