package handler

import (
	"bytes"
	"cakecake/internal/model/video"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func covReq(t *testing.T, r *gin.Engine, method, path, token string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var rd io.Reader
	var ct string
	switch b := body.(type) {
	case nil:
		rd = nil
	case string:
		rd = strings.NewReader(b)
		ct = "application/json"
	default:
		raw, err := json.Marshal(body)
		require.NoError(t, err)
		rd = bytes.NewReader(raw)
		ct = "application/json"
	}
	req := httptest.NewRequest(method, path, rd)
	if ct != "" {
		req.Header.Set("Content-Type", ct)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func covOK(t *testing.T, w *httptest.ResponseRecorder, want int) {
	t.Helper()
	require.Equal(t, want, w.Code, w.Body.String())
}

func covRegister(t *testing.T, r *gin.Engine, username, password string) (token string, uid uint64) {
	t.Helper()
	w := covReq(t, r, "POST", "/api/v1/users", "", map[string]string{"username": username, "password": password})
	covOK(t, w, http.StatusCreated)
	lw := covReq(t, r, "POST", "/api/v1/auth/login", "", map[string]string{"username": username, "password": password})
	covOK(t, lw, http.StatusOK)
	var out struct {
		Data struct {
			AccessToken string `json:"access_token"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(lw.Body.Bytes(), &out))
	require.NotEmpty(t, out.Data.AccessToken)
	me := covReq(t, r, "GET", "/api/v1/users/me", out.Data.AccessToken, nil)
	covOK(t, me, http.StatusOK)
	var meOut struct {
		Data struct {
			UserID uint64 `json:"user_id"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(me.Body.Bytes(), &meOut))
	return out.Data.AccessToken, meOut.Data.UserID
}

func covSeedVideo(t *testing.T, api *API, uid uint64, title string, status string) uint64 {
	t.Helper()
	v := video.Video{
		UserID:      uid,
		Title:       title,
		Description: "desc",
		Status:      status,
		DurationSec: 100,
		VideoURL:    "https://cdn.example.com/videos/1.mp4",
		CoverURL:    "https://cdn.example.com/covers/1.jpg",
		PlayCount:   10,
		Zone:        "动画",
	}
	require.NoError(t, api.DB.Create(&v).Error)
	return v.ID
}

func newMultipartWriter(t *testing.T, buf *bytes.Buffer, fields map[string]string) string {
	t.Helper()
	mw := multipart.NewWriter(buf)
	for k, v := range fields {
		require.NoError(t, mw.WriteField(k, v))
	}
	require.NoError(t, mw.Close())
	return mw.FormDataContentType()
}

func u64s(v uint64) string {
	return fmt.Sprintf("%d", v)
}

func (f *fakeOSSBackend) UploadFile(key, localPath string) error {
	f.uploads = append(f.uploads, key)
	return f.err
}

func (f *fakeOSSBackend) UploadReader(key string, r io.Reader) error {
	f.uploads = append(f.uploads, key)
	return f.err
}

func (f *fakeOSSBackend) DeleteObject(key string) error {
	f.deletes = append(f.deletes, key)
	return f.err
}

func (f *fakeOSSBackend) DeleteObjects(keys []string) error {
	f.deletes = append(f.deletes, keys...)
	return f.err
}
