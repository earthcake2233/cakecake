//go:build integration

package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"minibili/internal/model"
	"minibili/internal/pkg/jwttoken"
	"minibili/internal/ws"
)

// ---------------------------------------------------------------------------
// Auth flow: Register -> Login -> Refresh
// ---------------------------------------------------------------------------

func TestAuthFlow_RegisterLoginRefresh(t *testing.T) {
	_, r, _ := newTestAPI(t)

	// 1. Register a user
	regBody := map[string]string{"username": "flowuser", "password": "password12"}
	b, _ := json.Marshal(regBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())

	var regResp struct {
		Code int `json:"code"`
		Data struct {
			UserID   uint64 `json:"user_id"`
			Username string `json:"username"`
			CakeID   string `json:"cake_id"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &regResp))
	require.Equal(t, 0, regResp.Code)
	require.Equal(t, "flowuser", regResp.Data.Username)
	require.Contains(t, regResp.Data.CakeID, "cake_")

	// 2. Login
	loginBody := map[string]string{"username": "flowuser", "password": "password12"}
	b2, _ := json.Marshal(loginBody)
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(b2))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	require.Equal(t, http.StatusOK, w2.Code, w2.Body.String())

	var loginResp struct {
		Code int `json:"code"`
		Data struct {
			AccessToken  string `json:"access_token"`
			RefreshToken string `json:"refresh_token"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w2.Body.Bytes(), &loginResp))
	require.Equal(t, 0, loginResp.Code)
	require.NotEmpty(t, loginResp.Data.AccessToken)
	require.NotEmpty(t, loginResp.Data.RefreshToken)

	// 3. Refresh
	refreshBody := map[string]string{"refresh_token": loginResp.Data.RefreshToken}
	b3, _ := json.Marshal(refreshBody)
	req3 := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewReader(b3))
	req3.Header.Set("Content-Type", "application/json")
	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, req3)
	require.Equal(t, http.StatusOK, w3.Code, w3.Body.String())

	var refreshResp struct {
		Code int `json:"code"`
		Data struct {
			AccessToken  string `json:"access_token"`
			RefreshToken string `json:"refresh_token"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w3.Body.Bytes(), &refreshResp))
	require.Equal(t, 0, refreshResp.Code)
	require.NotEmpty(t, refreshResp.Data.AccessToken)
	require.NotEmpty(t, refreshResp.Data.RefreshToken)
	// New tokens should differ from the original.
	require.NotEqual(t, loginResp.Data.AccessToken, refreshResp.Data.AccessToken)

	// 4. Refresh with the same token again must fail (already rotated).
	b4, _ := json.Marshal(refreshBody)
	req4 := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewReader(b4))
	req4.Header.Set("Content-Type", "application/json")
	w4 := httptest.NewRecorder()
	r.ServeHTTP(w4, req4)
	require.Equal(t, http.StatusUnauthorized, w4.Code, w4.Body.String())

	// 5. Verify the new refresh token still works.
	refreshBody2 := map[string]string{"refresh_token": refreshResp.Data.RefreshToken}
	b5, _ := json.Marshal(refreshBody2)
	req5 := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewReader(b5))
	req5.Header.Set("Content-Type", "application/json")
	w5 := httptest.NewRecorder()
	r.ServeHTTP(w5, req5)
	require.Equal(t, http.StatusOK, w5.Code, w5.Body.String())

}

func TestRegister_ValidationErrors(t *testing.T) {
	_, r, _ := newTestAPI(t)

	tests := []struct {
		name string
		body map[string]string
		want int
	}{
		{"empty payload", map[string]string{}, 400},
		{"short password", map[string]string{"username": "u1", "password": "123"}, 400},
		{"short username", map[string]string{"username": "ab", "password": "password12"}, 400},
		{"empty username", map[string]string{"username": "", "password": "password12"}, 400},
		{"whitespace username", map[string]string{"username": "  ", "password": "password12"}, 400},
		{"reserved minibili_ai", map[string]string{"username": "minibili_ai", "password": "password12"}, 400},
		{"reserved ai_ prefix", map[string]string{"username": "ai_test", "password": "password12"}, 400},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			b, _ := json.Marshal(tc.body)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/users", bytes.NewReader(b))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			require.Equal(t, tc.want, w.Code, w.Body.String())
		})
	}
}

func TestLogin_ValidationErrors(t *testing.T) {
	_, r, _ := newTestAPI(t)

	// Pre-register a user.
	regBody := map[string]string{"username": "loginuser", "password": "password12"}
	b, _ := json.Marshal(regBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	tests := []struct {
		name string
		body map[string]string
		want int
	}{
		{"nonexistent user", map[string]string{"username": "nobody", "password": "password12"}, 401},
		{"wrong password", map[string]string{"username": "loginuser", "password": "wrongpass1"}, 401},
		{"empty password", map[string]string{"username": "loginuser", "password": ""}, 401},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			b, _ := json.Marshal(tc.body)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(b))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			require.Equal(t, tc.want, w.Code, w.Body.String())
		})
	}

	// Duplicate username should raise error on second register.
	t.Run("duplicate username", func(t *testing.T) {
		b, _ := json.Marshal(regBody)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/users", bytes.NewReader(b))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
	})
}

func TestRefresh_InvalidTokens(t *testing.T) {
	_, r, jm := newTestAPI(t)

	tests := []struct {
		name  string
		token string
		want  int
	}{
		{"empty token", "", 401},
		{"garbage token", "not-a-valid-jwt-token", 401},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			body := map[string]string{"refresh_token": tc.token}
			b, _ := json.Marshal(body)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewReader(b))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			require.Equal(t, tc.want, w.Code, w.Body.String())
		})
	}

	// Access token used as refresh token.
	t.Run("access token as refresh", func(t *testing.T) {
		access, _, _, err := jm.IssuePair(1)
		require.NoError(t, err)
		body := map[string]string{"refresh_token": access}
		b, _ := json.Marshal(body)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewReader(b))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		require.Equal(t, http.StatusUnauthorized, w.Code, w.Body.String())
	})
}

// ---------------------------------------------------------------------------
// Video CRUD: ListPublishedVideos, GetVideo
// ---------------------------------------------------------------------------

func seedVideo(t *testing.T, db *gorm.DB, userID uint64, title string) uint64 {
	t.Helper()
	v := model.Video{
		UserID:      userID,
		Title:       title,
		Status:      "published",
		DurationSec: 120,
		PlayCount:   100,
		DanmakuCount: 5,
		CommentCount: 2,
	}
	err := db.Create(&v).Error
	require.NoError(t, err)
	return v.ID
}

func TestListPublishedVideos(t *testing.T) {
	api, r, _ := newTestAPI(t)

	// Register uploader.
	regBody := map[string]string{"username": "uploader", "password": "password12"}
	b, _ := json.Marshal(regBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)
	var regResp struct {
		Data struct {
			UserID uint64 `json:"user_id"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &regResp))
	uid := regResp.Data.UserID

	// Seed videos.
	seedVideo(t, api.DB, uid, "Integration Test Video 1")
	seedVideo(t, api.DB, uid, "Integration Test Video 2")

	// GET /api/v1/videos
	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/videos", nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	require.Equal(t, http.StatusOK, w2.Code, w2.Body.String())

	var listResp struct {
		Code int `json:"code"`
		Data struct {
			Items []interface{} `json:"items"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w2.Body.Bytes(), &listResp))
	require.Equal(t, 0, listResp.Code)
	require.GreaterOrEqual(t, len(listResp.Data.Items), 2, "should return at least 2 videos")
}

func TestGetVideo(t *testing.T) {
	api, r, _ := newTestAPI(t)

	// Register uploader.
	regBody := map[string]string{"username": "vidowner", "password": "password12"}
	b, _ := json.Marshal(regBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)
	var regResp struct {
		Data struct {
			UserID uint64 `json:"user_id"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &regResp))
	uid := regResp.Data.UserID

	vid := seedVideo(t, api.DB, uid, "GetVideo Test")

	// GET /api/v1/videos/:id
	req2 := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/videos/%d", vid), nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	require.Equal(t, http.StatusOK, w2.Code, w2.Body.String())

	var detailResp struct {
		Code int `json:"code"`
		Data struct {
			ID       uint64 `json:"id"`
			Title    string `json:"title"`
			Uploader string `json:"uploader"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w2.Body.Bytes(), &detailResp))
	require.Equal(t, 0, detailResp.Code)
	require.Equal(t, vid, detailResp.Data.ID)
	require.Equal(t, "GetVideo Test", detailResp.Data.Title)
	require.Equal(t, "vidowner", detailResp.Data.Uploader)

	// GET non-existent video returns 404.
	req3 := httptest.NewRequest(http.MethodGet, "/api/v1/videos/999999", nil)
	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, req3)
	require.Equal(t, http.StatusNotFound, w3.Code, w3.Body.String())
}

// ---------------------------------------------------------------------------
// Comment System: ListComments, CreateComment
// ---------------------------------------------------------------------------

func TestCommentListAndCreate(t *testing.T) {
	api, r, _ := newTestAPI(t)

	// Register users: uploader and commenter.
	for _, uname := range []string{"uploader2", "commenter"} {
		b, _ := json.Marshal(map[string]string{"username": uname, "password": "password12"})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/users", bytes.NewReader(b))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	}

	// Login as commenter to get token.
	lb, _ := json.Marshal(map[string]string{"username": "commenter", "password": "password12"})
	loginReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(lb))
	loginReq.Header.Set("Content-Type", "application/json")
	lw := httptest.NewRecorder()
	r.ServeHTTP(lw, loginReq)
	require.Equal(t, http.StatusOK, lw.Code)

	var loginResp struct {
		Data struct {
			AccessToken string `json:"access_token"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(lw.Body.Bytes(), &loginResp))
	token := loginResp.Data.AccessToken
	require.NotEmpty(t, token)

	// Seed a published video (user 1 is uploader2).
	vid := seedVideo(t, api.DB, 1, "Comment Test Video")

	// GET comments before any exist (expect empty list).
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/videos/%d/comments", vid), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var emptyList struct {
		Code int `json:"code"`
		Data struct {
			Items []interface{} `json:"items"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &emptyList))
	require.Equal(t, 0, emptyList.Code)
	require.Empty(t, emptyList.Data.Items, "comments should be empty initially")

	// POST a comment (authenticated).
	cmtBody := map[string]string{"content": "Great video!"}
	cb, _ := json.Marshal(cmtBody)
	postReq := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/videos/%d/comments", vid), bytes.NewReader(cb))
	postReq.Header.Set("Content-Type", "application/json")
	postReq.Header.Set("Authorization", "Bearer "+token)
	pw := httptest.NewRecorder()
	r.ServeHTTP(pw, postReq)
	require.Equal(t, http.StatusCreated, pw.Code, pw.Body.String())

	// POST a second comment.
	cmtBody2 := map[string]string{"content": "Nice work!"}
	cb2, _ := json.Marshal(cmtBody2)
	postReq2 := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/videos/%d/comments", vid), bytes.NewReader(cb2))
	postReq2.Header.Set("Content-Type", "application/json")
	postReq2.Header.Set("Authorization", "Bearer "+token)
	pw2 := httptest.NewRecorder()
	r.ServeHTTP(pw2, postReq2)
	require.Equal(t, http.StatusCreated, pw2.Code, pw2.Body.String())

	// GET comments now should contain 2 items.
	req3 := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/videos/%d/comments", vid), nil)
	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, req3)
	require.Equal(t, http.StatusOK, w3.Code, w3.Body.String())

	var listResp struct {
		Code int `json:"code"`
		Data struct {
			Items []interface{} `json:"items"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w3.Body.Bytes(), &listResp))
	require.Equal(t, 0, listResp.Code)
	require.Len(t, listResp.Data.Items, 2)

	// Verify comment content.
	first := listResp.Data.Items[0].(map[string]interface{})
	content, ok := first["content"].(string)
	require.True(t, ok)
	require.Contains(t, []string{"Great video!", "Nice work!"}, content)

	// POST comment without auth -> 401.
	postReq3 := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/videos/%d/comments", vid), bytes.NewReader(cb))
	postReq3.Header.Set("Content-Type", "application/json")
	pw3 := httptest.NewRecorder()
	r.ServeHTTP(pw3, postReq3)
	require.Equal(t, http.StatusUnauthorized, pw3.Code, pw3.Body.String())
}

func TestGetComments_InvalidVideoID(t *testing.T) {
	_, r, _ := newTestAPI(t)

	tests := []string{
		"/api/v1/videos/0/comments",
		"/api/v1/videos/abc/comments",
	}
	for _, path := range tests {
		t.Run(path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, path, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
		})
	}
}

// ---------------------------------------------------------------------------
// Health endpoint edge cases
// ---------------------------------------------------------------------------

func TestHealthEdgeCases(t *testing.T) {
	gin.SetMode(gin.TestMode)
	jm, err := jwttoken.NewManager("test-secret-key-for-health-only-32chars")
	require.NoError(t, err)

	// Minimal API: no DB, no Redis, just Log + Hub + JWT.
	api := &API{
		Dependencies: &Dependencies{
			Log: zap.NewNop(),
			Hub: ws.NewHub(),
			JWT: jm,
		},
	}

	// GET /api/v1/health
	r := gin.New()
	r.GET("/api/v1/health", api.Health)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	code, _ := body["code"].(float64)
	require.Equal(t, float64(0), code)
	data, _ := body["data"].(map[string]interface{})
	status, _ := data["status"].(string)
	require.Equal(t, "ok", status)
}

// ---------------------------------------------------------------------------
// Search-related handlers (without Elasticsearch)
// ---------------------------------------------------------------------------

func TestSearchAll_WithoutES(t *testing.T) {
	_, r, _ := newTestAPI(t)

	tests := []struct {
		name    string
		query   string
		want    int
		checkFn func(t *testing.T, body map[string]interface{})
	}{
		{
			name:  "empty keyword",
			query: "keyword=",
			want:  400,
		},
		{
			name:  "missing keyword",
			query: "",
			want:  400,
		},
		{
			name:  "normal search (ES unavailable)",
			query: "keyword=golang",
			want:  200,
			checkFn: func(t *testing.T, body map[string]interface{}) {
				code, _ := body["code"].(float64)
				require.Equal(t, float64(0), code, body)
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			path := "/api/v1/search"
			if tc.query != "" {
				path = path + "?" + tc.query
			}
			req := httptest.NewRequest(http.MethodGet, path, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			require.Equal(t, tc.want, w.Code, w.Body.String())
			if tc.checkFn != nil {
				var body map[string]interface{}
				require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
				tc.checkFn(t, body)
			}
		})
	}
}

func TestSearchSuggest(t *testing.T) {
	_, r, _ := newTestAPI(t)

	// Without search hot service, returns empty suggestions.
	req := httptest.NewRequest(http.MethodGet, "/api/v1/search/suggest?term=go", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	code, _ := resp["code"].(float64)
	require.Equal(t, float64(0), code)

	// Missing term still returns OK with empty tag.
	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/search/suggest", nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	require.Equal(t, http.StatusOK, w2.Code, w2.Body.String())
}





