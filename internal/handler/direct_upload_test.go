package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"cakecake/internal/errcode"
	"cakecake/internal/model/video"
	vsvc "cakecake/internal/service/video"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// fakeVideoOSS implements the service-level SourceObjectStore seam used by the
// direct-upload ticket/submit flows.
type fakeVideoOSS struct {
	exist           map[string]bool
	sizes           map[string]int64
	allowAll        bool
	putContentTypes []string
}

func (f *fakeVideoOSS) UploadFile(string, string) error { return nil }

func (f *fakeVideoOSS) DownloadFile(_ string, localPath string) error {
	return os.WriteFile(localPath, []byte("media"), 0o644)
}

func (f *fakeVideoOSS) Exists(key string) (bool, error) {
	if f.allowAll {
		return true, nil
	}
	if f.exist == nil {
		return false, nil
	}
	return f.exist[key], nil
}

func (f *fakeVideoOSS) Size(key string) (int64, error) {
	if f.allowAll {
		if f.sizes == nil {
			return 1024, nil
		}
		if v, ok := f.sizes[key]; ok {
			return v, nil
		}
		return 1024, nil
	}
	if f.sizes == nil {
		return 0, nil
	}
	return f.sizes[key], nil
}

func (f *fakeVideoOSS) PresignPut(key string, _ time.Duration, contentType string) (string, error) {
	f.putContentTypes = append(f.putContentTypes, contentType)
	return "https://oss.example.com/" + key, nil
}

func (f *fakeVideoOSS) DeleteObject(string) error { return nil }

func TestDirectUploadTicketAndSubmit(t *testing.T) {
	api, r, _ := newTestAPI(t)
	tokenA, uidA := covRegister(t, r, "covdirect", "password12")
	rawKey := fmt.Sprintf("uploads/%d/abc/source.mp4", uidA)
	coverKey := fmt.Sprintf("uploads/%d/abc/cover.png", uidA)
	oss := &fakeVideoOSS{
		exist: map[string]bool{rawKey: true},
	}
	api.VideoSvc = vsvc.NewVideoService(api.DB, api.Redis, zap.NewNop(), nil, noopMQ{}, oss)
	oldProbe := vsvc.VideoProbe
	vsvc.VideoProbe = func(context.Context, string) (float64, error) { return 12.5, nil }
	defer func() { vsvc.VideoProbe = oldProbe }()

	// Presigned ticket with raw + cover.
	w := covReq(t, r, "POST", "/api/v1/videos/upload-ticket", tokenA, map[string]any{
		"filename":           "clip.mp4",
		"cover_filename":     "cover.png",
		"content_type":       "video/mp4",
		"cover_content_type": "image/png",
	})
	covOK(t, w, http.StatusOK)
	var ticketResp struct {
		Data struct {
			RawKey   string `json:"raw_key"`
			RawURL   string `json:"raw_upload_url"`
			CoverKey string `json:"cover_key"`
			CoverURL string `json:"cover_upload_url"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &ticketResp))
	require.True(t, strings.HasPrefix(ticketResp.Data.RawKey, fmt.Sprintf("uploads/%d/", uidA)))
	require.True(t, strings.HasSuffix(ticketResp.Data.RawKey, "/source.mp4"))
	require.Contains(t, ticketResp.Data.RawURL, ticketResp.Data.RawKey)
	require.Contains(t, ticketResp.Data.CoverURL, ticketResp.Data.CoverKey)
	require.Equal(t, []string{"video/mp4", "image/png"}, oss.putContentTypes)

	// Submit direct upload: object exists -> video created and enqueued.
	w = covReq(t, r, "POST", "/api/v1/videos", tokenA, map[string]any{
		"title":       "direct upload",
		"description": "desc",
		"tags":        []string{"a", "b"},
		"zone":        "动画",
		"raw_key":     rawKey,
		"cover_key":   coverKey,
	})
	covOK(t, w, http.StatusCreated)
	var submitResp struct {
		Data struct {
			ID     uint64 `json:"id"`
			Status string `json:"status"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &submitResp))
	require.Equal(t, video.StatusProcessing, submitResp.Data.Status)

	var row video.Video
	require.NoError(t, api.DB.First(&row, submitResp.Data.ID).Error)
	require.Equal(t, "direct upload", row.Title)

	// Re-submitting the same raw_key is idempotent: same video, no new row.
	w = covReq(t, r, "POST", "/api/v1/videos", tokenA, map[string]any{
		"title":   "direct upload",
		"raw_key": rawKey,
	})
	covOK(t, w, http.StatusCreated)
	var again struct {
		Data struct {
			ID uint64 `json:"id"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &again))
	require.Equal(t, submitResp.Data.ID, again.Data.ID)
	require.Equal(t, int64(1), countVideos(t, api))

	// Keys owned by another user are rejected before any row is created.
	before := countVideos(t, api)
	w = covReq(t, r, "POST", "/api/v1/videos", tokenA, map[string]any{
		"title":   "missing",
		"raw_key": fmt.Sprintf("uploads/%d/x/source.mp4", uidA+1),
	})
	covOK(t, w, http.StatusBadRequest)
	require.Equal(t, before, countVideos(t, api))
}

func TestDirectUploadSubmit_RejectsLargeAndNormalizesZone(t *testing.T) {
	api, r, _ := newTestAPI(t)
	tokenA, uidA := covRegister(t, r, "covnorm", "password12")
	rawKey := fmt.Sprintf("uploads/%d/big/source.mp4", uidA)
	api.VideoSvc = vsvc.NewVideoService(api.DB, api.Redis, zap.NewNop(), nil, noopMQ{}, &fakeVideoOSS{
		exist: map[string]bool{rawKey: true},
		sizes: map[string]int64{rawKey: 1024},
	})
	oldProbe := vsvc.VideoProbe
	vsvc.VideoProbe = func(context.Context, string) (float64, error) { return 5, nil }
	defer func() { vsvc.VideoProbe = oldProbe }()

	// Unknown zone is normalized to empty (same rule as multipart upload).
	w := covReq(t, r, "POST", "/api/v1/videos", tokenA, map[string]any{
		"title":   "normalized zone",
		"raw_key": rawKey,
		"zone":    "not-a-zone",
	})
	covOK(t, w, http.StatusCreated)
	var created struct {
		Data struct {
			ID uint64 `json:"id"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &created))
	var row video.Video
	require.NoError(t, api.DB.First(&row, created.Data.ID).Error)
	require.Empty(t, row.Zone)

	// Oversized object is rejected before any download/row creation.
	before := countVideos(t, api)
	api.VideoSvc = vsvc.NewVideoService(api.DB, api.Redis, zap.NewNop(), nil, noopMQ{}, &fakeVideoOSS{
		exist: map[string]bool{rawKey: true},
		sizes: map[string]int64{rawKey: int64(500<<20) + 1},
	})
	w = covReq(t, r, "POST", "/api/v1/videos", tokenA, map[string]any{
		"title":   "too big",
		"raw_key": rawKey,
	})
	covOK(t, w, http.StatusBadRequest)
	require.Equal(t, before, countVideos(t, api))
}

func TestDirectUploadRespectsUploadDisabled(t *testing.T) {
	api, r, _ := newTestAPI(t)
	api.Cfg.VideoUploadDisabled = true
	tokenA, _ := covRegister(t, r, "covdisabled", "password12")

	codeOf := func(w *httptest.ResponseRecorder) int {
		var body struct {
			Code int `json:"code"`
		}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
		return body.Code
	}

	w := covReq(t, r, "POST", "/api/v1/videos/upload-ticket", tokenA, map[string]any{
		"filename": "a.mp4",
	})
	covOK(t, w, http.StatusBadRequest)
	require.Equal(t, errcode.CodeVideoUploadDisabled, codeOf(w))

	w = covReq(t, r, "POST", "/api/v1/videos", tokenA, map[string]any{
		"title":   "x",
		"raw_key": "uploads/1/x/source.mp4",
	})
	covOK(t, w, http.StatusBadRequest)
	require.Equal(t, errcode.CodeVideoUploadDisabled, codeOf(w))
}

func TestDirectUploadSubmit_StoresDurationHintWithoutProbe(t *testing.T) {
	api, r, _ := newTestAPI(t)
	tokenA, uidA := covRegister(t, r, "covlong", "password12")
	rawKey := fmt.Sprintf("uploads/%d/long/source.mp4", uidA)
	probed := false
	api.VideoSvc = vsvc.NewVideoService(api.DB, api.Redis, zap.NewNop(), nil, noopMQ{}, &fakeVideoOSS{
		exist: map[string]bool{rawKey: true},
	})
	oldProbe := vsvc.VideoProbe
	vsvc.VideoProbe = func(context.Context, string) (float64, error) {
		probed = true
		return 99, nil
	}
	defer func() { vsvc.VideoProbe = oldProbe }()

	w := covReq(t, r, "POST", "/api/v1/videos", tokenA, map[string]any{
		"title":    "long video",
		"raw_key":  rawKey,
		"duration": 30*60 + 1,
	})
	covOK(t, w, http.StatusCreated)
	var created struct {
		Data struct {
			ID uint64 `json:"id"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &created))
	var row video.Video
	require.NoError(t, api.DB.First(&row, created.Data.ID).Error)
	// Hint is clamped for display; the worker re-probes and rejects the real
	// duration. Submit itself never downloads or probes the object.
	require.Equal(t, float64(30*60), row.DurationSec)
	require.False(t, probed)
}

func countVideos(t *testing.T, api *API) int64 {
	t.Helper()
	var n int64
	require.NoError(t, api.DB.Model(&video.Video{}).Count(&n).Error)
	return n
}
