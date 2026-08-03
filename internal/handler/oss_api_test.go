package handler

import (
	"bytes"
	"cakecake/internal/model/article"
	"cakecake/internal/model/video"
	"cakecake/internal/service"
	"github.com/stretchr/testify/require"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOSSUploadPaths(t *testing.T) {
	api, r, _ := newTestAPI(t)
	api.StorageSvc = service.NewStorageService(api.Cfg, nil, api.Log)
	oldBackend := service.OSSBackendOverride
	fakeOSS := &fakeOSSBackend{}
	service.OSSBackendOverride = fakeOSS
	defer func() { service.OSSBackendOverride = oldBackend }()

	tokenA, uidA := covRegister(t, r, "covos", "password12")
	vid := covSeedVideo(t, api, uidA, "oss video", video.StatusPublished)
	seedArt := article.Article{UserID: uidA, Title: "oss art", BodyMD: "# x", Status: article.StatusPublished}
	require.NoError(t, api.DB.Create(&seedArt).Error)

	// Video cover upload
	body := &bytes.Buffer{}
	mw := multipart.NewWriter(body)
	fw, err := mw.CreateFormFile("cover", "cover.jpg")
	require.NoError(t, err)
	_, err = fw.Write([]byte("jpeg bytes"))
	require.NoError(t, err)
	require.NoError(t, mw.Close())
	req := httptest.NewRequest("PUT", "/api/v1/videos/"+u64s(vid)+"/cover", body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+tokenA)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	covOK(t, w, http.StatusOK)

	// Article cover upload
	body2 := &bytes.Buffer{}
	mw2 := multipart.NewWriter(body2)
	fw2, err := mw2.CreateFormFile("cover", "acover.jpg")
	require.NoError(t, err)
	_, err = fw2.Write([]byte("jpeg"))
	require.NoError(t, err)
	require.NoError(t, mw2.Close())
	req2 := httptest.NewRequest("PUT", "/api/v1/users/me/articles/"+u64s(seedArt.ID)+"/cover", body2)
	req2.Header.Set("Content-Type", mw2.FormDataContentType())
	req2.Header.Set("Authorization", "Bearer "+tokenA)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	covOK(t, w2, http.StatusOK)

	// Favorite folder create with cover (multipart)
	body3 := &bytes.Buffer{}
	mw3 := multipart.NewWriter(body3)
	require.NoError(t, mw3.WriteField("title", "oss folder"))
	fw3, err := mw3.CreateFormFile("cover", "fcover.jpg")
	require.NoError(t, err)
	_, err = fw3.Write([]byte("jpeg"))
	require.NoError(t, err)
	require.NoError(t, mw3.Close())
	req3 := httptest.NewRequest("POST", "/api/v1/users/me/favorite-folders", body3)
	req3.Header.Set("Content-Type", mw3.FormDataContentType())
	req3.Header.Set("Authorization", "Bearer "+tokenA)
	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, req3)
	covOK(t, w3, http.StatusOK)

	// User avatar upload
	body4 := &bytes.Buffer{}
	mw4 := multipart.NewWriter(body4)
	fw4, err := mw4.CreateFormFile("avatar", "a.png")
	require.NoError(t, err)
	_, err = fw4.Write([]byte("png"))
	require.NoError(t, err)
	require.NoError(t, mw4.Close())
	req4 := httptest.NewRequest("POST", "/api/v1/users/me/avatar", body4)
	req4.Header.Set("Content-Type", mw4.FormDataContentType())
	req4.Header.Set("Authorization", "Bearer "+tokenA)
	w4 := httptest.NewRecorder()
	r.ServeHTTP(w4, req4)
	covOK(t, w4, http.StatusOK)

	// Dynamic with image upload
	body5 := &bytes.Buffer{}
	mw5 := multipart.NewWriter(body5)
	require.NoError(t, mw5.WriteField("title", "dyn"))
	require.NoError(t, mw5.WriteField("content", "c"))
	fw5, err := mw5.CreateFormFile("images", "d.jpg")
	require.NoError(t, err)
	_, err = fw5.Write([]byte("jpeg"))
	require.NoError(t, err)
	require.NoError(t, mw5.Close())
	req5 := httptest.NewRequest("POST", "/api/v1/users/me/dynamics", body5)
	req5.Header.Set("Content-Type", mw5.FormDataContentType())
	req5.Header.Set("Authorization", "Bearer "+tokenA)
	w5 := httptest.NewRecorder()
	r.ServeHTTP(w5, req5)
	covOK(t, w5, http.StatusOK)

	// Delete video triggers OSS purge
	covOK(t, covReq(t, r, "DELETE", "/api/v1/videos/"+u64s(vid), tokenA, nil), http.StatusOK)
	require.NotEmpty(t, fakeOSS.uploads)
	require.NotEmpty(t, fakeOSS.deletes)
}
