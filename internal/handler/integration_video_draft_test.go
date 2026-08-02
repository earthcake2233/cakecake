//go:build integration

package handler

import (
	"cakecake/internal/model/video"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestVideoDraft_SaveMetadataOnly(t *testing.T) {
	api, r, jm := newTestAPI(t)
	api.Cfg.VideoUploadDisabled = true
	access, _, _, err := jm.IssuePair(1)
	require.NoError(t, err)
	seedUserRow(t, api.DB, 1, "creator")

	// Missing title -> bad request.
	w := doMultipart(r, "POST", "/api/v1/videos/draft", access, map[string]string{"description": "desc"})
	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())

	// Valid metadata-only draft.
	w = doMultipart(r, "POST", "/api/v1/videos/draft", access, map[string]string{
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

	// Normal mode requires a file.
	w := doMultipart(r, "POST", "/api/v1/videos/draft", access, map[string]string{"title": "x"})
	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())

	// Unauthorized.
	w = doMultipart(r, "POST", "/api/v1/videos/draft", "", map[string]string{"title": "x"})
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
	access, _, _, err := jm.IssuePair(1)
	require.NoError(t, err)
	seedUserRow(t, api.DB, 1, "creator")

	raw := filepath.Join(t.TempDir(), "raw.mp4")
	require.NoError(t, os.WriteFile(raw, []byte("fake"), 0o600))
	require.NoError(t, api.DB.Create(&video.Video{
		ID: 10, UserID: 1, Title: "ready", Status: video.StatusDraft,
		DraftRawPath: raw,
	}).Error)

	w := doJSON(r, "POST", "/api/v1/videos/10/publish", access, nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var v video.Video
	require.NoError(t, api.DB.First(&v, 10).Error)
	require.Equal(t, video.StatusProcessing, v.Status)

	// Draft without raw path -> bad request.
	require.NoError(t, api.DB.Create(&video.Video{ID: 11, UserID: 1, Title: "empty", Status: video.StatusDraft}).Error)
	w = doJSON(r, "POST", "/api/v1/videos/11/publish", access, nil)
	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())

	// Missing draft -> not found.
	w = doJSON(r, "POST", "/api/v1/videos/999/publish", access, nil)
	require.Equal(t, http.StatusNotFound, w.Code, w.Body.String())
}

func TestVideoDraft_ReplaceAndSource(t *testing.T) {
	api, r, jm := newTestAPI(t)
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

	// Draft source: raw path missing -> not found.
	w = doReq(r, "GET", "/api/v1/users/me/videos/10/draft-source", access, "", nil)
	require.Equal(t, http.StatusNotFound, w.Code, w.Body.String())

	// Draft source with raw file -> 200.
	raw := filepath.Join(t.TempDir(), "raw.mp4")
	require.NoError(t, os.WriteFile(raw, []byte("data"), 0o600))
	require.NoError(t, api.DB.Model(&video.Video{}).Where("id = ?", 10).Update("draft_raw_path", raw).Error)
	w = doReq(r, "GET", "/api/v1/users/me/videos/10/draft-source", access, "", nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.Equal(t, "data", w.Body.String())

	// Replace media on failed video without file -> upload missing file.
	require.NoError(t, api.DB.Model(&video.Video{}).Where("id = ?", 10).Updates(map[string]interface{}{
		"status":         video.StatusFailed,
		"draft_raw_path": "",
	}).Error)
	w = doJSON(r, "POST", "/api/v1/videos/10/replace-media", access, nil)
	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
}
