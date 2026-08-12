//go:build integration

package handler

import (
	"cakecake/internal/model/video"
	"cakecake/internal/queue"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAdminTranscodeDeadLetter_ListAndRequeue(t *testing.T) {
	api, r, _ := newTestAPI(t)
	at := admintok(t, api)
	require.NoError(t, api.DB.Create(&video.Video{ID: 20, UserID: 1, Title: "v", Status: video.StatusFailed, FailReason: "oss down"}).Error)
	rawPath := filepath.Join(t.TempDir(), "raw.mp4")
	require.NoError(t, os.WriteFile(rawPath, []byte("media"), 0o644))
	payload, err := json.Marshal(map[string]interface{}{
		"job": queue.TranscodeJob{VideoID: 20, RawPath: rawPath, RetryCount: 3}, "reason": "oss down",
	})
	require.NoError(t, err)
	dl := video.TranscodeDeadLetter{
		VideoID: 20, RetryCount: 3, Reason: "oss down",
		PayloadJSON: string(payload),
	}
	require.NoError(t, api.DB.Create(&dl).Error)

	// List pending dead letters.
	w := srveOK(t, r, areq("GET", "/api/v1/admin/transcode-dead-letters?status=pending", at, nil), http.StatusOK)
	require.Contains(t, w.Body.String(), `"video_id":20`)

	// Requeue.
	w = srveOK(t, r, areq("POST", fmt.Sprintf("/api/v1/admin/transcode-dead-letters/%d/requeue", dl.ID), at, nil), http.StatusOK)
	require.Contains(t, w.Body.String(), `"ok":true`)

	// Row is now requeued and the video is back to processing.
	w = srveOK(t, r, areq("GET", "/api/v1/admin/transcode-dead-letters?status=requeued", at, nil), http.StatusOK)
	require.Contains(t, w.Body.String(), `"video_id":20`)
	var v video.Video
	require.NoError(t, api.DB.First(&v, 20).Error)
	require.Equal(t, video.StatusProcessing, v.Status)

	// Missing dead letter -> 404.
	srveOK(t, r, areq("POST", "/api/v1/admin/transcode-dead-letters/99999/requeue", at, nil), http.StatusNotFound)
}

func TestAdminTranscodeDeadLetter_RequeueMissingSourceConflict(t *testing.T) {
	api, r, _ := newTestAPI(t)
	at := admintok(t, api)
	require.NoError(t, api.DB.Create(&video.Video{ID: 22, UserID: 1, Title: "v", Status: video.StatusFailed}).Error)
	payload, err := json.Marshal(map[string]interface{}{
		"job": queue.TranscodeJob{VideoID: 22, RawPath: filepath.Join(t.TempDir(), "gone.mp4"), RetryCount: 3}, "reason": "x",
	})
	require.NoError(t, err)
	dl := video.TranscodeDeadLetter{VideoID: 22, RetryCount: 3, Reason: "x", PayloadJSON: string(payload)}
	require.NoError(t, api.DB.Create(&dl).Error)

	w := srveOK(t, r, areq("POST", fmt.Sprintf("/api/v1/admin/transcode-dead-letters/%d/requeue", dl.ID), at, nil), http.StatusConflict)
	require.Contains(t, w.Body.String(), `"code":40023`)
	require.Contains(t, w.Body.String(), "原始文件已被清理")
	var v video.Video
	require.NoError(t, api.DB.First(&v, 22).Error)
	require.Equal(t, video.StatusFailed, v.Status, "missing source must not touch the video")
}

func TestAdminTranscodeDeadLetter_ListShape(t *testing.T) {
	api, r, _ := newTestAPI(t)
	at := admintok(t, api)
	require.NoError(t, api.DB.Create(&video.TranscodeDeadLetter{VideoID: 21, RetryCount: 1, Reason: "x", PayloadJSON: "{}"}).Error)

	w := srveOK(t, r, areq("GET", "/api/v1/admin/transcode-dead-letters", at, nil), http.StatusOK)
	var body struct {
		Code int `json:"code"`
		Data struct {
			Items []transcodeDeadLetterItem `json:"items"`
			Total int64                     `json:"total"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Equal(t, 0, body.Code)
	require.Equal(t, int64(1), body.Data.Total)
	require.Len(t, body.Data.Items, 1)
	require.Equal(t, uint64(21), body.Data.Items[0].VideoID)
}
