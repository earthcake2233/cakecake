package handler

import (
	"bytes"
	"encoding/json"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDraftMetadataOnly(t *testing.T) {
	api, r, _ := newTestAPI(t)
	api.Cfg.VideoUploadDisabled = true
	tokenA, _ := covRegister(t, r, "covda", "password12")

	body := &bytes.Buffer{}
	mw := newMultipartWriter(t, body, map[string]string{"title": "draft one", "description": "desc"})
	req := httptest.NewRequest("POST", "/api/v1/videos/draft", body)
	req.Header.Set("Content-Type", mw)
	req.Header.Set("Authorization", "Bearer "+tokenA)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	covOK(t, w, http.StatusCreated)
	var dOut struct {
		Data struct {
			ID uint64 `json:"id"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &dOut))
	did := dOut.Data.ID
	covOK(t, covReq(t, r, "PUT", "/api/v1/videos/"+u64s(did)+"/draft", tokenA, map[string]any{"title": "draft renamed"}), http.StatusOK)
	covOK(t, covReq(t, r, "POST", "/api/v1/videos/"+u64s(did)+"/draft", tokenA, map[string]any{"title": "draft again"}), http.StatusOK)
	covOK(t, covReq(t, r, "GET", "/api/v1/users/me/videos/"+u64s(did)+"/draft-source", tokenA, nil), http.StatusNotFound)
	covOK(t, covReq(t, r, "POST", "/api/v1/videos/"+u64s(did)+"/publish", tokenA, map[string]any{}), http.StatusBadRequest)
}
