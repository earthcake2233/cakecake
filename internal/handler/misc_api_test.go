package handler

import (
	"bytes"
	"cakecake/internal/model/article"
	"cakecake/internal/model/dynamic"
	"cakecake/internal/model/video"
	"encoding/json"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMiscFlows(t *testing.T) {
	api, r, _ := newTestAPI(t)
	tokenA, uidA := covRegister(t, r, "covea", "password12")
	tokenB, _ := covRegister(t, r, "coveb", "password12")
	vid := covSeedVideo(t, api, uidA, "extra video", video.StatusPublished)

	// Reply + dislike comment flows
	cw := covReq(t, r, "POST", "/api/v1/videos/"+u64s(vid)+"/comments", tokenB, map[string]any{"content": "parent comment"})
	covOK(t, cw, http.StatusCreated)
	var cOut struct {
		Data struct {
			ID uint64 `json:"id"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(cw.Body.Bytes(), &cOut))
	rw := covReq(t, r, "POST", "/api/v1/videos/"+u64s(vid)+"/comments", tokenA, map[string]any{"content": "a reply", "parent_id": cOut.Data.ID})
	covOK(t, rw, http.StatusCreated)
	var rOut struct {
		Data struct {
			ID uint64 `json:"id"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rw.Body.Bytes(), &rOut))
	covOK(t, covReq(t, r, "POST", "/api/v1/comments/"+u64s(rOut.Data.ID)+"/dislike", tokenB, map[string]any{}), http.StatusOK)
	covOK(t, covReq(t, r, "GET", "/api/v1/videos/"+u64s(vid)+"/comments?page=1&page_size=10&sort=hot", tokenA, nil), http.StatusOK)
	covOK(t, covReq(t, r, "DELETE", "/api/v1/comments/"+u64s(rOut.Data.ID), tokenA, nil), http.StatusOK)
	covOK(t, covReq(t, r, "DELETE", "/api/v1/comments/"+u64s(cOut.Data.ID), tokenB, nil), http.StatusOK)

	// Favorite folder update + delete
	fw := covReq(t, r, "POST", "/api/v1/users/me/favorite-folders", tokenB, map[string]any{"title": "folder x"})
	covOK(t, fw, http.StatusOK)
	var fOut struct {
		Data struct {
			ID uint64 `json:"id"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(fw.Body.Bytes(), &fOut))
	covOK(t, covReq(t, r, "PUT", "/api/v1/users/me/favorite-folders/"+u64s(fOut.Data.ID), tokenB, map[string]any{"title": "folder y"}), http.StatusOK)
	covOK(t, covReq(t, r, "DELETE", "/api/v1/users/me/favorite-folders/"+u64s(fOut.Data.ID), tokenB, nil), http.StatusOK)

	// Article playback patch + my article update
	aw := covReq(t, r, "POST", "/api/v1/articles", tokenA, map[string]any{"title": "extra art", "content": "# body"})
	covOK(t, aw, http.StatusCreated)
	var aOut struct {
		Data struct {
			ID uint64 `json:"id"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(aw.Body.Bytes(), &aOut))
	seedPubArt := article.Article{UserID: uidA, Title: "pub art", BodyMD: "# x", Status: article.StatusPublished}
	require.NoError(t, api.DB.Create(&seedPubArt).Error)
	covOK(t, covReq(t, r, "PATCH", "/api/v1/users/me/articles/"+u64s(seedPubArt.ID)+"/playback", tokenA, map[string]any{"comments_closed": true}), http.StatusOK)

	// Dynamic update via multipart
	body := &bytes.Buffer{}
	mw := newMultipartWriter(t, body, map[string]string{"title": "dyn x", "content": "dyn content"})
	req := httptest.NewRequest("POST", "/api/v1/users/me/dynamics", body)
	req.Header.Set("Content-Type", mw)
	req.Header.Set("Authorization", "Bearer "+tokenA)
	dw := httptest.NewRecorder()
	r.ServeHTTP(dw, req)
	covOK(t, dw, http.StatusOK)
	var dOut struct {
		Data struct {
			ID uint64 `json:"id"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(dw.Body.Bytes(), &dOut))
	body2 := &bytes.Buffer{}
	mw2 := newMultipartWriter(t, body2, map[string]string{"title": "updated dyn", "content": "new content"})
	req = httptest.NewRequest("PUT", "/api/v1/users/me/dynamics/"+u64s(dOut.Data.ID), body2)
	req.Header.Set("Content-Type", mw2)
	req.Header.Set("Authorization", "Bearer "+tokenA)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	covOK(t, w, http.StatusOK)

	// Notifications batch-read (empty) + misc
	covOK(t, covReq(t, r, "GET", "/api/v1/users/me/creator/danmakus", tokenA, nil), http.StatusOK)
	covOK(t, covReq(t, r, "POST", "/api/v1/danmakus/1/like", tokenA, map[string]any{}), http.StatusNotFound)
	covOK(t, covReq(t, r, "DELETE", "/api/v1/danmakus/1", tokenA, nil), http.StatusNotFound)
}

func TestParamsAndErrorPaths(t *testing.T) {
	api, r, _ := newTestAPI(t)
	tokenA, uidA := covRegister(t, r, "covpa", "password12")
	tokenB, _ := covRegister(t, r, "covpb", "password12")
	vid := covSeedVideo(t, api, uidA, "params video", video.StatusPublished)
	vid2 := covSeedVideo(t, api, uidA, "params video 2", video.StatusPublished)
	seedArt := article.Article{UserID: uidA, Title: "params art", BodyMD: "# x", Status: article.StatusPublished}
	require.NoError(t, api.DB.Create(&seedArt).Error)
	dyn := dynamic.UserDynamic{UserID: uidA, Title: "params dyn", Content: "c"}
	require.NoError(t, api.DB.Create(&dyn).Error)

	// List parameter variants
	covOK(t, covReq(t, r, "GET", "/api/v1/videos?sort=time&page=1&page_size=5&cursor=", "", nil), http.StatusOK)
	covOK(t, covReq(t, r, "GET", "/api/v1/users/me/videos?status=published&sort=hot&page=1", tokenA, nil), http.StatusOK)
	covOK(t, covReq(t, r, "GET", "/api/v1/users/me/videos?status=draft", tokenA, nil), http.StatusOK)
	covOK(t, covReq(t, r, "GET", "/api/v1/users/me/videos?status=processing", tokenA, nil), http.StatusOK)
	covOK(t, covReq(t, r, "GET", "/api/v1/videos/"+u64s(vid)+"?t=5", tokenB, nil), http.StatusOK)

	// Favorites with folder filter + public space lists
	fw := covReq(t, r, "POST", "/api/v1/users/me/favorite-folders", tokenB, map[string]any{"title": "pfolder"})
	covOK(t, fw, http.StatusOK)
	var fOut struct {
		Data struct {
			ID uint64 `json:"id"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(fw.Body.Bytes(), &fOut))
	covOK(t, covReq(t, r, "PUT", "/api/v1/videos/"+u64s(vid)+"/favorite-folders", tokenB, map[string]any{"folder_ids": []uint64{fOut.Data.ID}}), http.StatusOK)
	covOK(t, covReq(t, r, "GET", "/api/v1/users/me/favorites?folder_id="+u64s(fOut.Data.ID), tokenB, nil), http.StatusOK)
	covOK(t, covReq(t, r, "GET", "/api/v1/space/"+u64s(uidA)+"/favorite-folders?public=1", "", nil), http.StatusOK)
	covOK(t, covReq(t, r, "GET", "/api/v1/users/me/favorite-folders?default=1", tokenB, nil), http.StatusOK)

	// Article + dynamic comment replies and list params
	acw := covReq(t, r, "POST", "/api/v1/articles/"+u64s(seedArt.ID)+"/comments", tokenB, map[string]any{"content": "art parent"})
	covOK(t, acw, http.StatusCreated)
	var acOut struct {
		Data struct {
			ID uint64 `json:"id"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(acw.Body.Bytes(), &acOut))
	covOK(t, covReq(t, r, "POST", "/api/v1/articles/"+u64s(seedArt.ID)+"/comments", tokenA, map[string]any{"content": "art reply", "parent_id": acOut.Data.ID}), http.StatusCreated)
	covOK(t, covReq(t, r, "GET", "/api/v1/articles/"+u64s(seedArt.ID)+"/comments?page=1&page_size=10", tokenA, nil), http.StatusOK)
	dcw := covReq(t, r, "POST", "/api/v1/user-dynamics/"+u64s(dyn.ID)+"/comments", tokenB, map[string]any{"content": "dyn parent"})
	covOK(t, dcw, http.StatusOK)
	var dcOut struct {
		Data struct {
			ID uint64 `json:"id"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(dcw.Body.Bytes(), &dcOut))
	covOK(t, covReq(t, r, "POST", "/api/v1/user-dynamics/"+u64s(dyn.ID)+"/comments", tokenA, map[string]any{"content": "dyn reply", "parent_id": dcOut.Data.ID}), http.StatusOK)
	covOK(t, covReq(t, r, "GET", "/api/v1/user-dynamics/"+u64s(dyn.ID)+"/comments?page=1", tokenA, nil), http.StatusOK)

	// Search params (ES disabled -> graceful path with params)
	covOK(t, covReq(t, r, "GET", "/api/v1/search?keyword=video&type=video&sort=time&highlight=1", tokenB, nil), http.StatusOK)
	covOK(t, covReq(t, r, "GET", "/api/v1/search?keyword=video&order=hot&duration=1&zone=动画", tokenB, nil), http.StatusOK)
	covOK(t, covReq(t, r, "GET", "/api/v1/search/suggest?term=", tokenB, nil), http.StatusOK)

	// Account deletion revoke
	covOK(t, covReq(t, r, "POST", "/api/v1/users/me/deletion/request", tokenA, map[string]any{"password": "password12"}), http.StatusOK)
	covOK(t, covReq(t, r, "POST", "/api/v1/users/me/deletion/revoke", tokenA, map[string]any{}), http.StatusOK)

	// Watch later toggle off + my favorites with video
	covOK(t, covReq(t, r, "POST", "/api/v1/videos/"+u64s(vid2)+"/watch-later", tokenB, map[string]any{}), http.StatusOK)
	covOK(t, covReq(t, r, "GET", "/api/v1/users/me/watch-later", tokenB, nil), http.StatusOK)
	covOK(t, covReq(t, r, "POST", "/api/v1/videos/"+u64s(vid)+"/favorite", tokenB, map[string]any{}), http.StatusOK)
	covOK(t, covReq(t, r, "GET", "/api/v1/hot-search?limit=5", "", nil), http.StatusOK)

	// Danmaku error path: bad color
	covOK(t, covReq(t, r, "POST", "/api/v1/videos/"+u64s(vid)+"/danmaku", tokenB, map[string]any{"content": "hi", "video_time": 1, "type": "scroll", "color": "notacolor"}), http.StatusBadRequest)

	// Error paths
	covOK(t, covReq(t, r, "PUT", "/api/v1/users/me", tokenA, map[string]any{"username": ""}), http.StatusBadRequest)
	covOK(t, covReq(t, r, "POST", "/api/v1/videos/"+u64s(vid)+"/coin", tokenB, map[string]any{"amount": 0}), http.StatusOK)
	covOK(t, covReq(t, r, "GET", "/api/v1/articles/99999", "", nil), http.StatusNotFound)
	covOK(t, covReq(t, r, "GET", "/api/v1/user-dynamics/99999", "", nil), http.StatusNotFound)
	covOK(t, covReq(t, r, "GET", "/api/v1/videos/99999/comments", "", nil), http.StatusNotFound)
	covOK(t, covReq(t, r, "POST", "/api/v1/videos/"+u64s(vid)+"/comments", tokenB, map[string]any{"content": ""}), http.StatusBadRequest)
	covOK(t, covReq(t, r, "POST", "/api/v1/users/me/deletion/revoke", tokenA, map[string]any{}), http.StatusBadRequest)
	covOK(t, covReq(t, r, "POST", "/api/v1/users/me/dynamics", tokenA, map[string]any{"title": "x"}), http.StatusBadRequest)
	covOK(t, covReq(t, r, "DELETE", "/api/v1/dynamic-comments/99999", tokenA, nil), http.StatusNotFound)

	// Owner reads a processing video detail
	pv := video.Video{UserID: uidA, Title: "processing vid", Description: "d", Status: video.StatusProcessing, DurationSec: 1}
	require.NoError(t, api.DB.Create(&pv).Error)
	covOK(t, covReq(t, r, "GET", "/api/v1/videos/"+u64s(pv.ID), tokenA, nil), http.StatusOK)

	// Coin ledger with pagination
	covOK(t, covReq(t, r, "GET", "/api/v1/users/me/coin-ledger?page=1&page_size=5", tokenA, nil), http.StatusOK)

	// Creator danmaku list with params
	covOK(t, covReq(t, r, "GET", "/api/v1/users/me/creator/danmakus?page=1&page_size=10&keyword=x", tokenA, nil), http.StatusOK)

	// Unlike + history keyword + privacy toggles
	covOK(t, covReq(t, r, "POST", "/api/v1/videos/"+u64s(vid)+"/like", tokenB, map[string]any{}), http.StatusOK)
	covOK(t, covReq(t, r, "GET", "/api/v1/users/me/view-history?q=video", tokenB, nil), http.StatusOK)
	covOK(t, covReq(t, r, "GET", "/api/v1/space/"+u64s(uidA)+"/following?page=1", tokenB, nil), http.StatusForbidden)
	covOK(t, covReq(t, r, "PUT", "/api/v1/users/me/space-privacy", tokenB, map[string]any{"public_following": false, "public_fans": false, "public_favorites": false, "public_birthday": true}), http.StatusOK)
}
