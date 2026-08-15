package handler

import (
	"cakecake/internal/errcode"
	"cakecake/internal/middleware"
	"cakecake/internal/pkg/resp"
	vsvc "cakecake/internal/service/video"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"strings"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

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

// rejectVideoUploadDisabled returns true (and responds 40022) when the global
// video upload switch is disabled. Every upload entry point must call this,
// including the direct-upload ticket/submit endpoints.
func (a *API) rejectVideoUploadDisabled(c *gin.Context) bool {
	if a.Cfg != nil && a.Cfg.VideoUploadDisabled {
		resp.Err(c, http.StatusBadRequest, errcode.CodeVideoUploadDisabled)
		return true
	}
	return false
}

// UploadVideo is the single user-facing upload entry point: the browser
// stages the raw video (and optional cover) straight to OSS via the upload
// ticket, then submits metadata + object keys here as JSON. Server-side
// multipart upload is no longer exposed to users.
func (a *API) UploadVideo(c *gin.Context) {
	if a.rejectVideoUploadDisabled(c) {
		return
	}
	uid, ok := middleware.UserID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	if strings.HasPrefix(c.ContentType(), "application/json") {
		a.uploadVideoDirect(c, uid)
		return
	}
	resp.Err(c, http.StatusUnsupportedMediaType, errcode.CodeParamError)
}

type createUploadTicketRequest struct {
	Filename         string `json:"filename"`
	CoverFilename    string `json:"cover_filename,omitempty"`
	ContentType      string `json:"content_type,omitempty"`
	CoverContentType string `json:"cover_content_type,omitempty"`
}

// CreateVideoUploadTicket issues presigned PUT URLs so the browser can upload
// the raw video (and optional cover) straight to OSS without proxying through
// the API server.
func (a *API) CreateVideoUploadTicket(c *gin.Context) {
	if a.rejectVideoUploadDisabled(c) {
		return
	}
	uid, ok := middleware.UserID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	var req createUploadTicketRequest
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Filename) == "" {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	ticket, err := a.VideoSvc.CreateDirectUploadTicket(c.Request.Context(), uid, strings.TrimSpace(req.Filename), strings.TrimSpace(req.CoverFilename), strings.TrimSpace(req.ContentType), strings.TrimSpace(req.CoverContentType))
	if err != nil {
		if errors.Is(err, vsvc.ErrDirectUploadUnavailable) {
			resp.Err(c, http.StatusServiceUnavailable, errcode.CodeDirectUploadUnavailable)
			return
		}
		a.Log.Error("create direct upload ticket", zap.Uint64("uid", uid), zap.Error(err))
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	resp.JSON(c, http.StatusOK, errcode.CodeSuccess, ticket)
}

type directUploadSubmitRequest struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
	Zone        string   `json:"zone"`
	RawKey      string   `json:"raw_key"`
	CoverKey    string   `json:"cover_key,omitempty"`
	Duration    float64  `json:"duration,omitempty"`
}

// uploadVideoDirect completes a direct upload: the source files were already
// PUT to OSS by the browser, so this endpoint only validates metadata + the
// object, probes duration and enqueues transcoding.
func (a *API) uploadVideoDirect(c *gin.Context, uid uint64) {
	if a.rejectVideoUploadDisabled(c) {
		return
	}
	var req directUploadSubmitRequest
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
	if strings.TrimSpace(req.RawKey) == "" {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	tagsJSON, err := tagsJSONFromStringSlice(req.Tags)
	if err != nil {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	v, err := a.VideoSvc.CreateVideoFromDirectUpload(c.Request.Context(), uid, title, desc, tagsJSON, normalizeVideoZone(req.Zone), strings.TrimSpace(req.RawKey), strings.TrimSpace(req.CoverKey), req.Duration)
	if err != nil {
		switch {
		case errors.Is(err, vsvc.ErrDirectUploadUnavailable):
			resp.Err(c, http.StatusServiceUnavailable, errcode.CodeDirectUploadUnavailable)
		case errors.Is(err, vsvc.ErrDirectUploadSourceMissing):
			resp.Err(c, http.StatusBadRequest, errcode.CodeDirectUploadSourceMissing)
		case errors.Is(err, vsvc.ErrDirectUploadTooLarge):
			resp.Err(c, http.StatusBadRequest, errcode.CodeVideoFileTooLarge)
		case errors.Is(err, vsvc.ErrDirectUploadInvalidKey):
			resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		case errors.Is(err, vsvc.ErrDirectUploadInvalidCover):
			resp.Err(c, http.StatusBadRequest, errcode.CodeCoverFormat)
		case errors.Is(err, vsvc.ErrDirectUploadAlreadyClaimed), errors.Is(err, vsvc.ErrDirectUploadInProgress):
			resp.Err(c, http.StatusConflict, errcode.CodeDirectUploadConflict)
		case errors.Is(err, vsvc.ErrTranscodeQueueFull):
			resp.Err(c, http.StatusServiceUnavailable, errcode.CodeTranscodeQueueFull)
		default:
			a.Log.Error("direct upload submit", zap.Uint64("uid", uid), zap.String("raw_key", req.RawKey), zap.Error(err))
			resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		}
		return
	}
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

type createVideoResponse struct {
	ID        uint64  `json:"id"`
	Status    string  `json:"status"`
	Title     string  `json:"title"`
	Duration  float64 `json:"duration"`
	CreatedAt string  `json:"created_at"`
}
