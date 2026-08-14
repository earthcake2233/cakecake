//go:build integration

package handler

import (
	"cakecake/internal/model/video"
	"cakecake/internal/service/storage"
	vsvc "cakecake/internal/service/video"
	"context"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestVideoDraft_SaveMetadataOnly(t *testing.T) {
	api, r, jm := newTestAPI(t)
	api.Cfg.VideoUploadDisabled = true
	access, _, _, err := jm.IssuePair(1)
	require.NoError(t, err)
	seedUserRow(t, api.DB, 1, "creator")

	// Missing title -> bad request.
	w := doJSON(r, "POST", "/api/v1/videos/draft", access, map[string]string{"description": "desc"})
	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())

	// Valid metadata-only draft.
	w = doJSON(r, "POST", "/api/v1/videos/draft", access, map[string]string{
		"title": "my draft", "description": "desc", "zone": "动画",
	})
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	var resp struct {
		Code int `json:"code"`
		Data struct {
			ID uint64 `json:"id"`
		} `json:"data"`
	}
	decodeBody(t, w, &resp)
	require.Equal(t, 0, resp.Code)
	require.NotZero(t, resp.Data.ID)

	// A draft row exists.
	var v video.Video
	require.NoError(t, api.DB.First(&v, resp.Data.ID).Error)
	require.Equal(t, video.StatusDraft, v.Status)
	require.Equal(t, "动画", v.Zone)
}

func TestVideoDraft_SaveMissingFile(t *testing.T) {
	_, r, jm := newTestAPI(t)
	access, _, _, err := jm.IssuePair(1)
	require.NoError(t, err)

	// Normal mode requires a staged raw object key.
	w := doJSON(r, "POST", "/api/v1/videos/draft", access, map[string]string{"title": "x"})
	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())

	// Unauthorized.
	w = doJSON(r, "POST", "/api/v1/videos/draft", "", map[string]string{"title": "x"})
	require.Equal(t, http.StatusUnauthorized, w.Code, w.Body.String())
}

func TestVideoDraft_UpdateJSON(t *testing.T) {
	api, r, jm := newTestAPI(t)
	access, _, _, err := jm.IssuePair(1)
	require.NoError(t, err)
	seedUserRow(t, api.DB, 1, "creator")
	require.NoError(t, api.DB.Create(&video.Video{
		ID: 10, UserID: 1, Title: "draft", Status: video.StatusDraft,
	}).Error)

	// Update via JSON.
	w := doJSON(r, "PUT", "/api/v1/videos/10/draft", access, map[string]interface{}{
		"title": "renamed", "description": "d2", "tags": []string{"a", "b"}, "zone": "动画",
	})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var v video.Video
	require.NoError(t, api.DB.First(&v, 10).Error)
	require.Equal(t, "renamed", v.Title)
	require.Equal(t, "动画", v.Zone)

	// Update someone else's draft -> not found.
	require.NoError(t, api.DB.Create(&video.Video{ID: 11, UserID: 2, Title: "other", Status: video.StatusDraft}).Error)
	w = doJSON(r, "PUT", "/api/v1/videos/11/draft", access, map[string]interface{}{"title": "x"})
	require.Equal(t, http.StatusNotFound, w.Code, w.Body.String())

	// Bad id.
	w = doJSON(r, "PUT", "/api/v1/videos/abc/draft", access, map[string]interface{}{"title": "x"})
	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
}

func TestVideoDraft_Publish(t *testing.T) {
	api, r, jm := newTestAPI(t)
	oldProbe := vsvc.VideoProbe
	vsvc.VideoProbe = func(context.Context, string) (float64, error) { return 12.5, nil }
	defer func() { vsvc.VideoProbe = oldProbe }()
	access, _, _, err := jm.IssuePair(1)
	require.NoError(t, err)
	seedUserRow(t, api.DB, 1, "creator")

	// Legacy local-path draft still publishes (outbox row + processing).
	raw := filepath.Join(t.TempDir(), "raw.mp4")
	require.NoError(t, os.WriteFile(raw, []byte("fake"), 0o600))
	require.NoError(t, api.DB.Create(&video.Video{
		ID: 10, UserID: 1, Title: "ready", Status: video.StatusDraft,
		DraftRawKey: raw,
	}).Error)
	w := doJSON(r, "POST", "/api/v1/videos/10/publish", access, nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var v video.Video
	require.NoError(t, api.DB.First(&v, 10).Error)
	require.Equal(t, video.StatusProcessing, v.Status)

	// OSS-keyed draft: atomic outbox + processing.
	oss := &fakeVideoOSS{allowAll: true}
	api.VideoDraftSvc = vsvc.NewVideoDraftService(api.DB, api.Redis, zap.NewNop(), noopMQ{}, oss)
	require.NoError(t, api.DB.Create(&video.Video{
		ID: 12, UserID: 1, Title: "oss ready", Status: video.StatusDraft,
		DraftRawKey: "drafts/1/abc/source.mp4",
	}).Error)
	w = doJSON(r, "POST", "/api/v1/videos/12/publish", access, nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var v12 video.Video
	require.NoError(t, api.DB.First(&v12, 12).Error)
	require.Equal(t, video.StatusProcessing, v12.Status)
	var outbox int64
	require.NoError(t, api.DB.Model(&video.TranscodeOutbox{}).Where("video_id = ?", 12).Count(&outbox).Error)
	require.Equal(t, int64(1), outbox)

	// Draft without raw reference -> bad request.
	require.NoError(t, api.DB.Create(&video.Video{ID: 11, UserID: 1, Title: "empty", Status: video.StatusDraft}).Error)
	w = doJSON(r, "POST", "/api/v1/videos/11/publish", access, nil)
	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())

	// Missing draft -> not found.
	w = doJSON(r, "POST", "/api/v1/videos/999/publish", access, nil)
	require.Equal(t, http.StatusNotFound, w.Code, w.Body.String())
}

func TestVideoDraft_ReplaceAndSource(t *testing.T) {
	api, r, jm := newTestAPI(t)
	oldProbe := vsvc.VideoProbe
	vsvc.VideoProbe = func(context.Context, string) (float64, error) { return 8.0, nil }
	defer func() { vsvc.VideoProbe = oldProbe }()
	access, _, _, err := jm.IssuePair(1)
	require.NoError(t, err)
	seedUserRow(t, api.DB, 1, "creator")

	// Replace media only allowed for failed/rejected videos.
	require.NoError(t, api.DB.Create(&video.Video{ID: 10, UserID: 1, Title: "d", Status: video.StatusDraft}).Error)
	w := doJSON(r, "POST", "/api/v1/videos/10/replace-media", access, nil)
	require.Equal(t, http.StatusForbidden, w.Code, w.Body.String())
	// Missing video.
	w = doJSON(r, "POST", "/api/v1/videos/999/replace-media", access, nil)
	require.Equal(t, http.StatusNotFound, w.Code, w.Body.String())

	// Draft source: raw reference missing -> not found.
	w = doReq(r, "GET", "/api/v1/users/me/videos/10/draft-source", access, "", nil)
	require.Equal(t, http.StatusNotFound, w.Code, w.Body.String())

	// Draft source with legacy local file -> 200.
	raw := filepath.Join(t.TempDir(), "raw.mp4")
	require.NoError(t, os.WriteFile(raw, []byte("data"), 0o600))
	require.NoError(t, api.DB.Model(&video.Video{}).Where("id = ?", 10).Update("draft_raw_key", raw).Error)
	w = doReq(r, "GET", "/api/v1/users/me/videos/10/draft-source", access, "", nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.Equal(t, "data", w.Body.String())

	// Draft source with OSS key -> 307 to a presigned GET URL.
	storage.OSSBackendOverride = &fakeOSSBackend{}
	t.Cleanup(func() { storage.OSSBackendOverride = nil })
	require.NoError(t, api.DB.Model(&video.Video{}).Where("id = ?", 10).
		Updates(map[string]interface{}{"draft_raw_key": "drafts/1/xyz/source.mp4"}).Error)
	w = doReq(r, "GET", "/api/v1/users/me/videos/10/draft-source", access, "", nil)
	require.Equal(t, http.StatusTemporaryRedirect, w.Code, w.Body.String())
	require.True(t, strings.Contains(w.Header().Get("Location"), "drafts/1/xyz/source.mp4"), w.Header().Get("Location"))

	// Replace media on failed video with an OSS key.
	oss := &fakeVideoOSS{allowAll: true}
	api.VideoDraftSvc = vsvc.NewVideoDraftService(api.DB, api.Redis, zap.NewNop(), noopMQ{}, oss)
	require.NoError(t, api.DB.Model(&video.Video{}).Where("id = ?", 10).Updates(map[string]interface{}{
		"status":        video.StatusFailed,
		"draft_raw_key": "",
	}).Error)
	w = doJSON(r, "POST", "/api/v1/videos/10/replace-media", access, map[string]interface{}{
		"title":   "replaced",
		"raw_key": "drafts/1/abc/source.mp4",
	})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var v video.Video
	require.NoError(t, api.DB.First(&v, 10).Error)
	require.Equal(t, video.StatusProcessing, v.Status)
	var outbox int64
	require.NoError(t, api.DB.Model(&video.TranscodeOutbox{}).Where("video_id = ?", 10).Count(&outbox).Error)
	require.Equal(t, int64(1), outbox)

	// Replace media without raw_key -> upload missing file.
	require.NoError(t, api.DB.Model(&video.Video{}).Where("id = ?", 10).Update("status", video.StatusFailed).Error)
	w = doJSON(r, "POST", "/api/v1/videos/10/replace-media", access, map[string]interface{}{"title": "x"})
	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
}
