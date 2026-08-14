package handler

import (
	"cakecake/internal/model/video"
	vsvc "cakecake/internal/service/video"
	"context"
	"encoding/json"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"net/http"
	"os"
	"testing"
	"time"
)

func TestVideoEngagementAndComments(t *testing.T) {
	api, r, _ := newTestAPI(t)
	tokenA, uidA := covRegister(t, r, "covva", "password12")
	tokenB, _ := covRegister(t, r, "covvb", "password12")
	vid := covSeedVideo(t, api, uidA, "video one", video.StatusPublished)
	vid2 := covSeedVideo(t, api, uidA, "video two", video.StatusPublished)

	listW := covReq(t, r, "GET", "/api/v1/videos", "", nil)
	covOK(t, listW, http.StatusOK)
	var listOut struct {
		Data struct {
			Items []struct {
				Uploader string `json:"uploader"`
			} `json:"items"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(listW.Body.Bytes(), &listOut))
	require.NotEmpty(t, listOut.Data.Items)
	for _, it := range listOut.Data.Items {
		require.Equal(t, "covva", it.Uploader, "feed cards must carry the uploader name")
	}
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
	// and a fake object store so the direct-upload pipeline runs without ffmpeg.
	oss := &fakeVideoOSS{allowAll: true}
	api.VideoSvc = vsvc.NewVideoService(api.DB, api.Redis, zap.NewNop(), nil, noopMQ{}, oss)
	api.VideoDraftSvc = vsvc.NewVideoDraftService(api.DB, api.Redis, zap.NewNop(), noopMQ{}, oss)
	oldProbe := vsvc.VideoProbe
	vsvc.VideoProbe = func(context.Context, string) (float64, error) { return 12.5, nil }
	defer func() { vsvc.VideoProbe = oldProbe }()

	tokenA, uidA := covRegister(t, r, "covup", "password12")
	_ = uidA

	// Direct upload: ticket then JSON submit.
	w := covReq(t, r, "POST", "/api/v1/videos/upload-ticket", tokenA, map[string]any{
		"filename": "clip.mp4",
	})
	covOK(t, w, http.StatusOK)
	var ticket struct {
		Data struct {
			RawKey string `json:"raw_key"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &ticket))
	uploadKey := ticket.Data.RawKey
	w = covReq(t, r, "POST", "/api/v1/videos", tokenA, map[string]any{
		"title":       "uploaded video",
		"description": "desc",
		"tags":        []string{"a", "b"},
		"zone":        "动画",
		"raw_key":     uploadKey,
	})
	covOK(t, w, http.StatusCreated)

	// Draft with OSS media: draft ticket then JSON create.
	w = covReq(t, r, "POST", "/api/v1/videos/draft/upload-ticket", tokenA, map[string]any{
		"filename": "draft.mp4",
	})
	covOK(t, w, http.StatusOK)
	var draftTicket struct {
		Data struct {
			RawKey string `json:"raw_key"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &draftTicket))
	draftRawKey := draftTicket.Data.RawKey
	w = covReq(t, r, "POST", "/api/v1/videos/draft", tokenA, map[string]any{
		"title":       "draft with file",
		"description": "d",
		"raw_key":     draftRawKey,
	})
	covOK(t, w, http.StatusCreated)
	var dOut struct {
		Data struct {
			ID uint64 `json:"id"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &dOut))
	draftID := dOut.Data.ID

	// Update draft with a replacement media key.
	w = covReq(t, r, "POST", "/api/v1/videos/draft/upload-ticket", tokenA, map[string]any{
		"filename": "update.mp4",
	})
	covOK(t, w, http.StatusOK)
	var updateTicket struct {
		Data struct {
			RawKey string `json:"raw_key"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &updateTicket))
	w = covReq(t, r, "PUT", "/api/v1/videos/"+u64s(draftID)+"/draft", tokenA, map[string]any{
		"title":   "updated draft",
		"raw_key": updateTicket.Data.RawKey,
	})
	covOK(t, w, http.StatusOK)

	// Replace media on a seeded draft
	draft := video.Video{UserID: uidA, Title: "replace me", Description: "d", Status: video.StatusFailed}
	require.NoError(t, api.DB.Create(&draft).Error)
	w = covReq(t, r, "POST", "/api/v1/videos/draft/upload-ticket", tokenA, map[string]any{
		"filename": "replace.mp4",
	})
	covOK(t, w, http.StatusOK)
	var replaceTicket struct {
		Data struct {
			RawKey string `json:"raw_key"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &replaceTicket))
	w = covReq(t, r, "POST", "/api/v1/videos/"+u64s(draft.ID)+"/replace-media", tokenA, map[string]any{
		"title":       "replaced",
		"description": "d2",
		"raw_key":     replaceTicket.Data.RawKey,
	})
	covOK(t, w, http.StatusOK)

	// Publish draft (OSS-keyed draft -> outbox row + processing)
	covOK(t, covReq(t, r, "POST", "/api/v1/videos/"+u64s(draftID)+"/publish", tokenA, map[string]any{}), http.StatusOK)
}

type fakeOSSBackend struct {
	uploads []string
	deletes []string
	err     error
}

func (f *fakeOSSBackend) DownloadFile(_ string, localPath string) error {
	if f.err != nil {
		return f.err
	}
	return os.WriteFile(localPath, []byte("media"), 0o644)
}

func (f *fakeOSSBackend) Exists(string) (bool, error) {
	return true, f.err
}

func (f *fakeOSSBackend) Size(string) (int64, error) {
	return 1024, f.err
}

func (f *fakeOSSBackend) PresignPut(key string, _ time.Duration, _ string) (string, error) {
	return "https://oss.example.com/" + key, f.err
}

func (f *fakeOSSBackend) PresignGet(key string, _ time.Duration) (string, error) {
	return "https://oss.example.com/" + key + "?x-oss-process=preview", f.err
}

func (f *fakeOSSBackend) CopyObject(srcKey, dstKey string) error {
	f.uploads = append(f.uploads, dstKey)
	return f.err
}
