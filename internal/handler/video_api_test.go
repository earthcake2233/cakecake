package handler

import (
	"bytes"
	"cakecake/internal/model/video"
	vsvc "cakecake/internal/service/video"
	"encoding/json"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestVideoEngagementAndComments(t *testing.T) {
	api, r, _ := newTestAPI(t)
	tokenA, uidA := covRegister(t, r, "covva", "password12")
	tokenB, _ := covRegister(t, r, "covvb", "password12")
	vid := covSeedVideo(t, api, uidA, "video one", video.StatusPublished)
	vid2 := covSeedVideo(t, api, uidA, "video two", video.StatusPublished)

	covOK(t, covReq(t, r, "GET", "/api/v1/videos", "", nil), http.StatusOK)
	covOK(t, covReq(t, r, "GET", "/api/v1/videos?zone=动画", "", nil), http.StatusOK)
	covOK(t, covReq(t, r, "GET", "/api/v1/videos/"+u64s(vid), "", nil), http.StatusOK)
	covOK(t, covReq(t, r, "GET", "/api/v1/videos/"+u64s(vid), tokenB, nil), http.StatusOK)
	covOK(t, covReq(t, r, "POST", "/api/v1/videos/"+u64s(vid)+"/like", tokenB, map[string]any{}), http.StatusOK)
	covOK(t, covReq(t, r, "POST", "/api/v1/videos/"+u64s(vid)+"/favorite", tokenB, map[string]any{}), http.StatusOK)
	covOK(t, covReq(t, r, "POST", "/api/v1/videos/"+u64s(vid)+"/coin", tokenB, map[string]any{"amount": 1}), http.StatusOK)
	covOK(t, covReq(t, r, "POST", "/api/v1/videos/"+u64s(vid)+"/watch-later", tokenB, map[string]any{}), http.StatusOK)

	// Favorite folders
	fw := covReq(t, r, "POST", "/api/v1/users/me/favorite-folders", tokenB, map[string]any{"title": "fav1"})
	covOK(t, fw, http.StatusOK)
	var folderOut struct {
		Data struct {
			ID uint64 `json:"id"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(fw.Body.Bytes(), &folderOut))
	fid := folderOut.Data.ID
	fw2 := covReq(t, r, "POST", "/api/v1/users/me/favorite-folders", tokenB, map[string]any{"title": "fav2"})
	covOK(t, fw2, http.StatusOK)
	var folderOut2 struct {
		Data struct {
			ID uint64 `json:"id"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(fw2.Body.Bytes(), &folderOut2))
	fid2 := folderOut2.Data.ID
	covOK(t, covReq(t, r, "GET", "/api/v1/users/me/favorite-folders", tokenB, nil), http.StatusOK)
	covOK(t, covReq(t, r, "GET", "/api/v1/videos/"+u64s(vid)+"/favorite-picker", tokenB, nil), http.StatusOK)
	covOK(t, covReq(t, r, "PUT", "/api/v1/videos/"+u64s(vid)+"/favorite-folders", tokenB, map[string]any{"folder_ids": []uint64{fid}}), http.StatusOK)
	covOK(t, covReq(t, r, "PUT", "/api/v1/videos/"+u64s(vid)+"/favorite-folders/move", tokenB, map[string]any{"from_folder_id": fid, "to_folder_id": fid2}), http.StatusOK)
	covOK(t, covReq(t, r, "POST", "/api/v1/videos/"+u64s(vid2)+"/favorite-folders/"+u64s(fid), tokenB, map[string]any{}), http.StatusOK)
	covOK(t, covReq(t, r, "DELETE", "/api/v1/videos/"+u64s(vid2)+"/favorite-folders/"+u64s(fid), tokenB, nil), http.StatusOK)
	covOK(t, covReq(t, r, "DELETE", "/api/v1/users/me/favorite-folders/"+u64s(fid)+"/invalid-favorites", tokenB, nil), http.StatusOK)
	covOK(t, covReq(t, r, "POST", "/api/v1/users/me/favorite-folders/"+u64s(fid)+"/batch-remove", tokenB, map[string]any{"video_ids": []uint64{vid}}), http.StatusOK)
	covOK(t, covReq(t, r, "GET", "/api/v1/users/me/favorites", tokenB, nil), http.StatusOK)
	covOK(t, covReq(t, r, "GET", "/api/v1/users/me/watch-later", tokenB, nil), http.StatusOK)
	covOK(t, covReq(t, r, "DELETE", "/api/v1/users/me/watch-later/watched", tokenB, nil), http.StatusOK)
	covOK(t, covReq(t, r, "POST", "/api/v1/users/me/watch-later/"+u64s(vid)+"/watched", tokenB, nil), http.StatusOK)
	covOK(t, covReq(t, r, "DELETE", "/api/v1/users/me/watch-later", tokenB, nil), http.StatusOK)
	covOK(t, covReq(t, r, "GET", "/api/v1/users/me/videos", tokenA, nil), http.StatusOK)
	covOK(t, covReq(t, r, "GET", "/api/v1/space/"+u64s(uidA), "", nil), http.StatusOK)
	covOK(t, covReq(t, r, "GET", "/api/v1/space/"+u64s(uidA)+"/videos", "", nil), http.StatusOK)
	covOK(t, covReq(t, r, "GET", "/api/v1/space/"+u64s(uidA)+"/favorites", tokenB, nil), http.StatusOK)
	covOK(t, covReq(t, r, "GET", "/api/v1/space/"+u64s(uidA)+"/favorite-folders", tokenB, nil), http.StatusOK)
	covOK(t, covReq(t, r, "GET", "/api/v1/space/"+u64s(uidA)+"/recent-coins", tokenB, nil), http.StatusOK)

	// Danmaku + comments
	covOK(t, covReq(t, r, "POST", "/api/v1/videos/"+u64s(vid)+"/danmaku", tokenB, map[string]any{"content": "hi", "video_time": 1.5, "color": "#ffffff", "type": "scroll"}), http.StatusOK)
	cw := covReq(t, r, "POST", "/api/v1/videos/"+u64s(vid)+"/comments", tokenB, map[string]any{"content": "nice video"})
	covOK(t, cw, http.StatusCreated)
	var cOut struct {
		Data struct {
			ID uint64 `json:"id"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(cw.Body.Bytes(), &cOut))
	cid := cOut.Data.ID
	covOK(t, covReq(t, r, "GET", "/api/v1/videos/"+u64s(vid)+"/comments", tokenA, nil), http.StatusOK)
	covOK(t, covReq(t, r, "POST", "/api/v1/comments/"+u64s(cid)+"/like", tokenA, map[string]any{}), http.StatusOK)
	covOK(t, covReq(t, r, "POST", "/api/v1/comments/"+u64s(cid)+"/pin", tokenA, map[string]any{}), http.StatusOK)
	covOK(t, covReq(t, r, "POST", "/api/v1/comments/"+u64s(cid)+"/approve", tokenA, map[string]any{}), http.StatusOK)
	covOK(t, covReq(t, r, "POST", "/api/v1/comments/"+u64s(cid)+"/ignore-curated", tokenA, map[string]any{}), http.StatusOK)
	covOK(t, covReq(t, r, "DELETE", "/api/v1/comments/"+u64s(cid), tokenB, nil), http.StatusOK)

	// Creator comments list
	covOK(t, covReq(t, r, "GET", "/api/v1/users/me/creator/comments", tokenA, nil), http.StatusOK)
	covOK(t, covReq(t, r, "GET", "/api/v1/users/me/creator/comments?media=article", tokenA, nil), http.StatusOK)
	covOK(t, covReq(t, r, "GET", "/api/v1/users/me/creator/comments?media=dynamic", tokenA, nil), http.StatusOK)
	covOK(t, covReq(t, r, "GET", "/api/v1/users/me/creator/comments?pending=1&sort=hot&q=x&video_id=1", tokenA, nil), http.StatusOK)

	// My video edit + delete
	covOK(t, covReq(t, r, "PUT", "/api/v1/videos/"+u64s(vid), tokenA, map[string]any{"title": "updated"}), http.StatusOK)
	covOK(t, covReq(t, r, "PATCH", "/api/v1/videos/"+u64s(vid)+"/playback", tokenA, map[string]any{"comments_closed": true}), http.StatusOK)
	covOK(t, covReq(t, r, "DELETE", "/api/v1/videos/"+u64s(vid), tokenA, nil), http.StatusOK)
}

func TestVideoUploadFlows(t *testing.T) {
	api, r, _ := newTestAPI(t)
	// Replace the video service with one wired to a no-op transcode publisher
	// and stub the media probe so the upload pipeline runs without ffmpeg.
	api.VideoSvc = vsvc.NewVideoService(api.DB, api.Redis, zap.NewNop(), nil, noopMQ{})
	oldProbe := vsvc.VideoProbe
	vsvc.VideoProbe = func(string) (float64, error) { return 12.5, nil }
	defer func() { vsvc.VideoProbe = oldProbe }()

	tokenA, uidA := covRegister(t, r, "covup", "password12")
	_ = uidA

	// UploadVideo multipart
	body := &bytes.Buffer{}
	mw := multipart.NewWriter(body)
	require.NoError(t, mw.WriteField("title", "uploaded video"))
	require.NoError(t, mw.WriteField("description", "desc"))
	require.NoError(t, mw.WriteField("tags", `["a","b"]`))
	require.NoError(t, mw.WriteField("zone", "动画"))
	fw, err := mw.CreateFormFile("file", "clip.mp4")
	require.NoError(t, err)
	_, err = fw.Write([]byte("fake mp4 bytes"))
	require.NoError(t, err)
	require.NoError(t, mw.Close())
	req := httptest.NewRequest("POST", "/api/v1/videos", body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+tokenA)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	covOK(t, w, http.StatusCreated)

	// Draft with file (multipart)
	body2 := &bytes.Buffer{}
	mw2 := multipart.NewWriter(body2)
	require.NoError(t, mw2.WriteField("title", "draft with file"))
	require.NoError(t, mw2.WriteField("description", "d"))
	fw2, err := mw2.CreateFormFile("file", "draft.mp4")
	require.NoError(t, err)
	_, err = fw2.Write([]byte("fake bytes"))
	require.NoError(t, err)
	require.NoError(t, mw2.Close())
	req2 := httptest.NewRequest("POST", "/api/v1/videos/draft", body2)
	req2.Header.Set("Content-Type", mw2.FormDataContentType())
	req2.Header.Set("Authorization", "Bearer "+tokenA)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	covOK(t, w2, http.StatusCreated)
	var dOut struct {
		Data struct {
			ID uint64 `json:"id"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w2.Body.Bytes(), &dOut))
	draftID := dOut.Data.ID

	// Update draft with a replacement file (multipart)
	body3 := &bytes.Buffer{}
	mw3 := multipart.NewWriter(body3)
	require.NoError(t, mw3.WriteField("title", "updated draft"))
	fw3, err := mw3.CreateFormFile("file", "update.mp4")
	require.NoError(t, err)
	_, err = fw3.Write([]byte("update bytes"))
	require.NoError(t, err)
	require.NoError(t, mw3.Close())
	req3 := httptest.NewRequest("PUT", "/api/v1/videos/"+u64s(draftID)+"/draft", body3)
	req3.Header.Set("Content-Type", mw3.FormDataContentType())
	req3.Header.Set("Authorization", "Bearer "+tokenA)
	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, req3)
	covOK(t, w3, http.StatusOK)

	// Replace media on a seeded draft
	draft := video.Video{UserID: uidA, Title: "replace me", Description: "d", Status: video.StatusFailed, DraftRawPath: "/tmp/x.mp4"}
	require.NoError(t, api.DB.Create(&draft).Error)
	body4 := &bytes.Buffer{}
	mw4 := multipart.NewWriter(body4)
	require.NoError(t, mw4.WriteField("title", "replaced"))
	require.NoError(t, mw4.WriteField("description", "d2"))
	fw4, err := mw4.CreateFormFile("file", "replace.mp4")
	require.NoError(t, err)
	_, err = fw4.Write([]byte("replace bytes"))
	require.NoError(t, err)
	require.NoError(t, mw4.Close())
	req4 := httptest.NewRequest("POST", "/api/v1/videos/"+u64s(draft.ID)+"/replace-media", body4)
	req4.Header.Set("Content-Type", mw4.FormDataContentType())
	req4.Header.Set("Authorization", "Bearer "+tokenA)
	w4 := httptest.NewRecorder()
	r.ServeHTTP(w4, req4)
	covOK(t, w4, http.StatusOK)

	// Publish draft (multipart draft now has a raw path -> succeeds)
	covOK(t, covReq(t, r, "POST", "/api/v1/videos/"+u64s(draftID)+"/publish", tokenA, map[string]any{}), http.StatusOK)
}

type fakeOSSBackend struct {
	uploads []string
	deletes []string
	err     error
}
