package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"minibili/internal/model"
)

func registerLogin(t *testing.T, r *gin.Engine, username string) string {
	t.Helper()
	body := map[string]string{"username": username, "password": "password12"}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(b))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	require.Equal(t, http.StatusOK, w2.Code)
	var out struct {
		Code int `json:"code"`
		Data struct {
			AccessToken string `json:"access_token"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w2.Body.Bytes(), &out))
	require.NotEmpty(t, out.Data.AccessToken)
	return out.Data.AccessToken
}

func createTestVideo(t *testing.T, api *API) uint64 {
	t.Helper()
	v := model.Video{Title: "test video", Description: "desc", Status: "published", UserID: 1}
	require.NoError(t, api.DB.Create(&v).Error)
	return v.ID
}

func TestPostDanmaku_Unauthorized(t *testing.T) {
	_, r, _ := newTestAPI(t)
	body := map[string]string{"content": "hello", "color": "#FFFFFF", "type": "scroll", "video_time": "0"}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/videos/1/danmaku", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestPostDanmaku_Validation(t *testing.T) {
	api, r, _ := newTestAPI(t)
	tok := registerLogin(t, r, "dmuser")
	vid := createTestVideo(t, api)
	u := "/api/v1/videos/" + formatUint(vid) + "/danmaku"

	tests := []struct {
		name   string
		body   map[string]interface{}
		status int
	}{
		{"empty content", map[string]interface{}{"content": "", "color": "#FFFFFF", "type": "scroll", "video_time": 0}, http.StatusBadRequest},
		{"invalid type", map[string]interface{}{"content": "hi", "color": "#FFFFFF", "type": "bogus", "video_time": 0}, http.StatusBadRequest},
		{"invalid color", map[string]interface{}{"content": "hi", "color": "red", "type": "scroll", "video_time": 0}, http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodPost, u, bytes.NewReader(b))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+tok)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			require.Equal(t, tt.status, w.Code, w.Body.String())
		})
	}
}

func TestPostDanmaku_Success(t *testing.T) {
	api, r, _ := newTestAPI(t)
	tok := registerLogin(t, r, "dmuser2")
	vid := createTestVideo(t, api)
	u := "/api/v1/videos/" + formatUint(vid) + "/danmaku"

	body := map[string]interface{}{
		"content": "hello world", "color": "#00FF00", "type": "scroll",
		"font_size": "md", "video_time": 1.5,
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, u, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var out struct {
		Code int `json:"code"`
		Data struct {
			ID      uint64 `json:"id"`
			Content string `json:"content"`
			Color   string `json:"color"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &out))
	require.Equal(t, 0, out.Code)
	require.NotZero(t, out.Data.ID)
	require.Equal(t, "hello world", out.Data.Content)
	require.Equal(t, "#00FF00", out.Data.Color)
}

func TestPostDanmaku_Sensitive(t *testing.T) {
	api, r, _ := newTestAPI(t)
	tok := registerLogin(t, r, "dmuser3")
	vid := createTestVideo(t, api)
	u := "/api/v1/videos/" + formatUint(vid) + "/danmaku"

	body := map[string]interface{}{"content": "badword", "color": "#FFFFFF", "type": "scroll", "video_time": 0}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, u, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPostDanmaku_DanmakuClosed(t *testing.T) {
	api, r, _ := newTestAPI(t)
	tok := registerLogin(t, r, "dmuser4")
	v := model.Video{Title: "closed", Description: "desc", Status: "published", UserID: 1, DanmakuClosed: true}
	require.NoError(t, api.DB.Create(&v).Error)
	u := "/api/v1/videos/" + formatUint(v.ID) + "/danmaku"

	body := map[string]interface{}{"content": "hi", "color": "#FFFFFF", "type": "scroll", "video_time": 0}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, u, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusForbidden, w.Code)
}

func formatUint(n uint64) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0'+n%10)
		n /= 10
	}
	return string(buf[i:])
}