//go:build integration

package handler

import (
	"bytes"
	"cakecake/internal/model/user"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func doReq(r *gin.Engine, method, path, token, contentType string, body io.Reader) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, body)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func doJSON(r *gin.Engine, method, path, token string, body interface{}) *httptest.ResponseRecorder {
	var rd io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			panic(err)
		}
		rd = bytes.NewReader(b)
	}
	return doReq(r, method, path, token, "application/json", rd)
}

func doMultipart(r *gin.Engine, method, path, token string, fields map[string]string) *httptest.ResponseRecorder {
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	for k, v := range fields {
		_ = mw.WriteField(k, v)
	}
	_ = mw.Close()
	return doReq(r, method, path, token, mw.FormDataContentType(), &buf)
}

func decodeBody(t *testing.T, w *httptest.ResponseRecorder, out interface{}) {
	t.Helper()
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), out))
}

func seedUserRow(t *testing.T, db *gorm.DB, id uint64, username string) {
	t.Helper()
	require.NoError(t, db.Create(&user.User{
		ID: id, Username: username, PasswordHash: "x", CoinBalanceTenths: 230,
	}).Error)
}
