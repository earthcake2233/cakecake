package handler

import (
	"cakecake/internal/errcode"
	"cakecake/internal/middleware"
	"cakecake/internal/model/video"
	"cakecake/internal/pkg/coverval"
	"cakecake/internal/pkg/resp"
	vsvc "cakecake/internal/service/video"
	"errors"
	"net/http"
	"os"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// parseReplaceMediaForm parses and validates the replace-media multipart form.
func (a *API) parseReplaceMediaForm(c *gin.Context) (draftCreateInput, int) {
	var in draftCreateInput
	if err := c.Request.ParseMultipartForm(maxVideoBytes + (12 << 20)); err != nil {
		a.Log.Warn("parse multipart form", zap.Error(err))
		return in, errcode.CodeMultipartParseError
	}
	in.title = strings.TrimSpace(c.PostForm("title"))
	in.desc = strings.TrimSpace(c.PostForm("description"))
	if utf8.RuneCountInString(in.title) < 1 || utf8.RuneCountInString(in.title) > 80 {
		return in, errcode.CodeTitleInvalid
	}
	if utf8.RuneCountInString(in.desc) > 2000 {
		return in, errcode.CodeIntroTooLong
	}
	fh, err := c.FormFile("file")
	if err != nil {
		return in, errcode.CodeUploadMissingFile
	}
	if fh.Size > maxVideoBytes {
		return in, errcode.CodeVideoFileTooLarge
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
	v, err := a.VideoDraftSvc.GetDraftByID(c.Request.Context(), id)
	if err != nil {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	if v.UserID != uid || !videoStatusAllowsMediaReplace(v.Status) {
		resp.Err(c, http.StatusForbidden, errcode.CodeForbidden)
		return
	}
	in, code := a.parseReplaceMediaForm(c)
	if code != 0 {
		resp.Err(c, http.StatusBadRequest, code)
		return
	}
	a.StorageSvc.PurgeVideo(*v)
	a.esDeleteVideo(id)
	removeVideoDraftFiles(*v)

	rawPath, dur, err := a.saveDraftVideoFile(in.fileFh, v.ID)
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
	if in.coverFh != nil {
		coverPath, code = a.saveDraftCoverFileChecked(in.coverFh, v.ID)
		if code != 0 {
			_ = os.Remove(rawPath)
			resp.Err(c, httpStatusForCode(code), code)
			return
		}
	}

	if err := a.VideoDraftSvc.ReplaceMedia(c.Request.Context(), v, vsvc.ReplaceMediaOpts{
		Title:       in.title,
		Description: in.desc,
		TagsJSON:    in.tagsJSON,
		Zone:        in.zone,
		RawPath:     rawPath,
		CoverPath:   coverPath,
		DurationSec: dur,
	}); err != nil {
		if errors.Is(err, vsvc.ErrReplaceMediaUpdate) {
			removeVideoDraftFiles(video.Video{DraftRawPath: rawPath, DraftCoverPath: coverPath})
		}
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	a.Log.Info("video media replaced and queued for transcode", zap.Uint64("video_id", v.ID))
	resp.OK(c, videoDraftStatusResponse{ID: v.ID, Status: video.StatusProcessing})
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
	v, err := a.VideoDraftSvc.GetOwnedDraftByStatus(c.Request.Context(), id, uid, videoStatusDraft)
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
