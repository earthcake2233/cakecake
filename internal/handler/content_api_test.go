package handler

import (
	"bytes"
	"cakecake/internal/model/article"
	"cakecake/internal/model/dynamic"
	"cakecake/internal/service/storage"
	"encoding/json"
	"github.com/stretchr/testify/require"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestArticleAndDynamicFlows(t *testing.T) {
	api, r, _ := newTestAPI(t)
	tokenA, uidA := covRegister(t, r, "covaa", "password12")
	tokenB, _ := covRegister(t, r, "covab", "password12")

	// Article via API
	aw := covReq(t, r, "POST", "/api/v1/articles", tokenA, map[string]any{"title": "art one", "content": "# hello\nbody", "cover_url": "https://cdn.example.com/a.jpg"})
	covOK(t, aw, http.StatusCreated)
	var aOut struct {
		Data struct {
			ID uint64 `json:"id"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(aw.Body.Bytes(), &aOut))
	postedAID := aOut.Data.ID
	// Published article seeded for read/engagement paths.
	seedArt := article.Article{UserID: uidA, Title: "published art", BodyMD: "# body", CoverURL: "https://cdn.example.com/pa.jpg", Status: article.StatusPublished}
	require.NoError(t, api.DB.Create(&seedArt).Error)
	aid := seedArt.ID
	covOK(t, covReq(t, r, "GET", "/api/v1/articles/"+u64s(aid), tokenB, nil), http.StatusOK)
	covOK(t, covReq(t, r, "POST", "/api/v1/articles/"+u64s(aid)+"/view", tokenB, map[string]any{}), http.StatusOK)
	covOK(t, covReq(t, r, "POST", "/api/v1/articles/"+u64s(aid)+"/favorite", tokenB, map[string]any{}), http.StatusOK)
	covOK(t, covReq(t, r, "POST", "/api/v1/articles/"+u64s(aid)+"/coin", tokenB, map[string]any{"amount": 1}), http.StatusOK)
	acw := covReq(t, r, "POST", "/api/v1/articles/"+u64s(aid)+"/comments", tokenB, map[string]any{"content": "article comment"})
	covOK(t, acw, http.StatusCreated)
	var acOut struct {
		Data struct {
			ID uint64 `json:"id"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(acw.Body.Bytes(), &acOut))
	covOK(t, covReq(t, r, "GET", "/api/v1/articles/"+u64s(aid)+"/comments", tokenA, nil), http.StatusOK)
	covOK(t, covReq(t, r, "POST", "/api/v1/article-comments/"+u64s(acOut.Data.ID)+"/like", tokenA, map[string]any{}), http.StatusOK)
	covOK(t, covReq(t, r, "POST", "/api/v1/article-comments/"+u64s(acOut.Data.ID)+"/pin", tokenA, map[string]any{}), http.StatusOK)
	covOK(t, covReq(t, r, "POST", "/api/v1/article-comments/"+u64s(acOut.Data.ID)+"/approve", tokenA, map[string]any{}), http.StatusOK)
	covOK(t, covReq(t, r, "POST", "/api/v1/article-comments/"+u64s(acOut.Data.ID)+"/ignore-curated", tokenA, map[string]any{}), http.StatusOK)
	covOK(t, covReq(t, r, "DELETE", "/api/v1/article-comments/"+u64s(acOut.Data.ID), tokenA, nil), http.StatusOK)
	covOK(t, covReq(t, r, "GET", "/api/v1/users/me/articles", tokenA, nil), http.StatusOK)
	covOK(t, covReq(t, r, "GET", "/api/v1/users/me/articles/"+u64s(postedAID), tokenA, nil), http.StatusOK)
	covOK(t, covReq(t, r, "PUT", "/api/v1/users/me/articles/"+u64s(postedAID), tokenA, map[string]any{"title": "art updated"}), http.StatusOK)
	covOK(t, covReq(t, r, "DELETE", "/api/v1/users/me/articles/"+u64s(postedAID), tokenA, nil), http.StatusOK)
	covOK(t, covReq(t, r, "GET", "/api/v1/users/me/article-favorites", tokenB, nil), http.StatusOK)
	covOK(t, covReq(t, r, "GET", "/api/v1/space/"+u64s(uidA)+"/articles", "", nil), http.StatusOK)
	covOK(t, covReq(t, r, "GET", "/api/v1/space/"+u64s(uidA)+"/article-favorites", tokenB, nil), http.StatusOK)

	// Dynamic via multipart
	body := &bytes.Buffer{}
	mw := newMultipartWriter(t, body, map[string]string{"title": "dyn title", "content": "dyn content"})
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
	did := dOut.Data.ID
	covOK(t, covReq(t, r, "GET", "/api/v1/users/me/dynamics", tokenA, nil), http.StatusOK)
	covOK(t, covReq(t, r, "GET", "/api/v1/user-dynamics/"+u64s(did), tokenB, nil), http.StatusOK)
	covOK(t, covReq(t, r, "POST", "/api/v1/user-dynamics/"+u64s(did)+"/like", tokenB, map[string]any{}), http.StatusOK)
	dcw := covReq(t, r, "POST", "/api/v1/user-dynamics/"+u64s(did)+"/comments", tokenB, map[string]any{"content": "dyn comment"})
	covOK(t, dcw, http.StatusOK)
	var dcOut struct {
		Data struct {
			ID uint64 `json:"id"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(dcw.Body.Bytes(), &dcOut))
	covOK(t, covReq(t, r, "GET", "/api/v1/user-dynamics/"+u64s(did)+"/comments", tokenA, nil), http.StatusOK)
	covOK(t, covReq(t, r, "POST", "/api/v1/dynamic-comments/"+u64s(dcOut.Data.ID)+"/like", tokenA, map[string]any{}), http.StatusOK)
	covOK(t, covReq(t, r, "POST", "/api/v1/dynamic-comments/"+u64s(dcOut.Data.ID)+"/dislike", tokenA, map[string]any{}), http.StatusOK)
	covOK(t, covReq(t, r, "POST", "/api/v1/dynamic-comments/"+u64s(dcOut.Data.ID)+"/approve", tokenA, map[string]any{}), http.StatusOK)
	covOK(t, covReq(t, r, "POST", "/api/v1/dynamic-comments/"+u64s(dcOut.Data.ID)+"/ignore-curated", tokenA, map[string]any{}), http.StatusOK)
	covOK(t, covReq(t, r, "DELETE", "/api/v1/dynamic-comments/"+u64s(dcOut.Data.ID), tokenA, nil), http.StatusOK)
	covOK(t, covReq(t, r, "PATCH", "/api/v1/users/me/dynamics/"+u64s(did)+"/playback", tokenA, map[string]any{"comments_closed": true}), http.StatusOK)
	covOK(t, covReq(t, r, "GET", "/api/v1/space/"+u64s(uidA)+"/dynamics", "", nil), http.StatusOK)
	covOK(t, covReq(t, r, "DELETE", "/api/v1/users/me/dynamics/"+u64s(did), tokenA, nil), http.StatusOK)

	// Seeded dynamic for admin-dynamic-ish path (delete cascade)
	dyn := dynamic.UserDynamic{UserID: uidA, Title: "t", Content: "c"}
	require.NoError(t, api.DB.Create(&dyn).Error)
	covOK(t, covReq(t, r, "GET", "/api/v1/user-dynamics/"+u64s(dyn.ID), "", nil), http.StatusOK)
}

func TestCommentAndFolderFlows(t *testing.T) {
	api, r, _ := newTestAPI(t)
	api.StorageSvc = storage.NewStorageService(api.Cfg, nil, api.Log)
	oldBackend := storage.OSSBackendOverride
	storage.OSSBackendOverride = &fakeOSSBackend{}
	defer func() { storage.OSSBackendOverride = oldBackend }()

	tokenA, uidA := covRegister(t, r, "covmc", "password12")
	tokenB, _ := covRegister(t, r, "covmd", "password12")
	seedArt := article.Article{UserID: uidA, Title: "mc art", BodyMD: "# x", Status: article.StatusPublished}
	require.NoError(t, api.DB.Create(&seedArt).Error)
	dyn := dynamic.UserDynamic{UserID: uidA, Title: "mc dyn", Content: "c"}
	require.NoError(t, api.DB.Create(&dyn).Error)

	acw := covReq(t, r, "POST", "/api/v1/articles/"+u64s(seedArt.ID)+"/comments", tokenB, map[string]any{"content": "ac"})
	covOK(t, acw, http.StatusCreated)
	var acOut struct {
		Data struct {
			ID uint64 `json:"id"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(acw.Body.Bytes(), &acOut))
	covOK(t, covReq(t, r, "POST", "/api/v1/article-comments/"+u64s(acOut.Data.ID)+"/dislike", tokenA, map[string]any{}), http.StatusOK)
	covOK(t, covReq(t, r, "GET", "/api/v1/articles/"+u64s(seedArt.ID)+"/comments?curated=1&closed=1", tokenA, nil), http.StatusOK)
	covOK(t, covReq(t, r, "GET", "/api/v1/user-dynamics/"+u64s(dyn.ID)+"/comments?curated=1", tokenA, nil), http.StatusOK)

	// Favorite folder update with cover (multipart + fake OSS)
	fw := covReq(t, r, "POST", "/api/v1/users/me/favorite-folders", tokenB, map[string]any{"title": "mc folder"})
	covOK(t, fw, http.StatusOK)
	var fOut struct {
		Data struct {
			ID uint64 `json:"id"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(fw.Body.Bytes(), &fOut))
	body := &bytes.Buffer{}
	mw := multipart.NewWriter(body)
	require.NoError(t, mw.WriteField("title", "mc folder 2"))
	cfw, err := mw.CreateFormFile("cover", "fc.jpg")
	require.NoError(t, err)
	_, err = cfw.Write([]byte("jpeg"))
	require.NoError(t, err)
	require.NoError(t, mw.Close())
	req := httptest.NewRequest("PUT", "/api/v1/users/me/favorite-folders/"+u64s(fOut.Data.ID), body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+tokenB)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	covOK(t, w, http.StatusOK)
}
