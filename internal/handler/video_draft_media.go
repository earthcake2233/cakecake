package handler

import (
	"cakecake/internal/errcode"
	"cakecake/internal/middleware"
	"cakecake/internal/model/video"
	"cakecake/internal/pkg/resp"
	vsvc "cakecake/internal/service/video"
	"errors"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func videoStatusAllowsMediaReplace(st string) bool {
	switch st {
	case video.StatusFailed, video.StatusRejected:
		return true
	default:
		return false
	}
}

// ReplaceVideoMedia replaces the source media for failed/rejected videos: the
// client stages a new raw (and optional cover) via the draft upload ticket,
// then submits the object keys here. The outbox row and the video update
// commit atomically; old objects are purged only after the new job is durable.
// ReplaceVideoMedia godoc
// @Summary      Replace video media
// @Description  Replace failed/rejected video media and re-queue transcoding
// @Tags         Videos
// @Accept       json
// @Produce      json
// @Param        id path int true "Video ID"
// @Param        body body object{} true "Title, description, tags, zone, raw_key, cover_key"
// @Success      200 {object} map[string]interface{}
// @Router       /videos/{id}/replace-media [post]
func (a *API) ReplaceVideoMedia(c *gin.Context) {
	if a.rejectVideoUploadDisabled(c) {
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
	v, err := a.VideoDraftSvc.GetDraftByID(c.Request.Context(), id)
	if err != nil {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	if v.UserID != uid || !videoStatusAllowsMediaReplace(v.Status) {
		resp.Err(c, http.StatusForbidden, errcode.CodeForbidden)
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
	if utf8.RuneCountInString(in.Title) < 1 || utf8.RuneCountInString(in.Title) > 80 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeTitleInvalid)
		return
	}
	if utf8.RuneCountInString(in.Description) > 2000 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeIntroTooLong)
		return
	}
	media, verr := a.VideoDraftSvc.ValidateDraftMedia(c.Request.Context(), uid, in.RawKey, in.CoverKey)
	if verr != nil {
		hs, code := mediaErrorStatus(verr)
		resp.Err(c, hs, code)
		return
	}
	tagsJSON, terr := tagsJSONFromStringSlice(in.Tags)
	if terr != nil {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	old := *v
	if err := a.VideoDraftSvc.ReplaceMedia(c.Request.Context(), v, vsvc.ReplaceMediaOpts{
		Title:       in.Title,
		Description: in.Description,
		TagsJSON:    tagsJSON,
		Zone:        normalizeVideoZone(in.Zone),
		RawKey:      media.RawKey,
		CoverKey:    media.CoverKey,
		DurationSec: vsvc.NormalizeDurationHint(in.Duration),
	}); err != nil {
		if errors.Is(err, vsvc.ErrTranscodeQueueFull) {
			resp.Err(c, http.StatusServiceUnavailable, errcode.CodeTranscodeQueueFull)
			return
		}
		a.Log.Error("replace video media", zap.Uint64("video_id", id), zap.Error(err))
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	// Old media is removed only after the new job is durable, so a failure
	// never leaves the video without a recoverable source.
	a.StorageSvc.PurgeVideo(old)
	a.purgeDraftMedia(old)
	a.esDeleteVideo(id)
	a.Log.Info("video media replaced and queued for transcode", zap.Uint64("video_id", v.ID))
	resp.OK(c, videoDraftStatusResponse{ID: v.ID, Status: video.StatusProcessing})
}

// GetMyVideoDraftSource streams the draft raw media for the uploader preview:
// OSS-keyed drafts redirect to a short-lived presigned GET URL; legacy
// local-path drafts are served from disk.
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
	v, err := a.VideoDraftSvc.GetOwnedDraftByStatus(c.Request.Context(), id, uid, videoStatusDraft)
	if err != nil {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	rawRef := strings.TrimSpace(v.DraftRawKey)
	if rawRef == "" {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	if isObjectKeyReference(rawRef) {
		if !a.StorageSvc.Enabled() {
			resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
			return
		}
		u, perr := a.StorageSvc.PresignGet(rawRef, 5*time.Minute)
		if perr != nil {
			a.Log.Warn("presign draft source", zap.Uint64("video_id", id), zap.Error(perr))
			resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
			return
		}
		c.Redirect(http.StatusTemporaryRedirect, u)
		return
	}
	if _, err := os.Stat(rawRef); err != nil {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	c.Header("Content-Type", "application/octet-stream")
	c.Header("Accept-Ranges", "bytes")
	c.File(rawRef)
}
