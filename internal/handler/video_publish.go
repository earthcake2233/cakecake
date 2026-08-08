package handler

import (
	"cakecake/internal/errcode"
	"cakecake/internal/middleware"
	"cakecake/internal/model/video"
	"cakecake/internal/pkg/coverval"
	"cakecake/internal/pkg/resp"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

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

type createVideoResponse struct {
	ID        uint64  `json:"id"`
	Status    string  `json:"status"`
	Title     string  `json:"title"`
	Duration  float64 `json:"duration"`
	CreatedAt string  `json:"created_at"`
}
