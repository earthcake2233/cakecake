//go:build integration

package handler

import (
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"minibili/internal/model"
)

func TestAdminBannerCRUD(t *testing.T) {
	_, r, jm := newTestAPI(t)
	access, _, _, err := jm.IssueAdminPair(1)
	require.NoError(t, err)

	body := `{"title":"Test Banner","link_type":"url","link_target":"https://ex.com","image_url":"https://ex.com/img.jpg","sort_order":1,"enabled":true}`
	req := httptest.NewRequest("POST", "/api/v1/admin/home-banners", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+access)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, 201, w.Code, "Create: %s", w.Body.String())
	var cr struct { Code int `json:"code"`; Data gin.H `json:"data"` }
	json.Unmarshal(w.Body.Bytes(), &cr)
	require.Equal(t, 0, cr.Code)
	id := int(cr.Data["id"].(float64))

	// LIST
	r2 := httptest.NewRequest("GET", "/api/v1/admin/home-banners", nil)
	r2.Header.Set("Authorization", "Bearer "+access)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, r2)
	require.Equal(t, 200, w2.Code)

	// UPDATE
	r3 := httptest.NewRequest("PUT", "/api/v1/admin/home-banners/"+strconv.Itoa(id), strings.NewReader(`{"title":"UPD"}`))
	r3.Header.Set("Authorization", "Bearer "+access)
	r3.Header.Set("Content-Type", "application/json")
	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, r3)
	require.Equal(t, 200, w3.Code, "Update: %s", w3.Body.String())

	// DELETE
	r4 := httptest.NewRequest("DELETE", "/api/v1/admin/home-banners/"+strconv.Itoa(id), nil)
	r4.Header.Set("Authorization", "Bearer "+access)
	w4 := httptest.NewRecorder()
	r.ServeHTTP(w4, r4)
	require.Equal(t, 200, w4.Code, "Delete: %s", w4.Body.String())
}

func TestAdminApproveVideo(t *testing.T) {
	api, r, jm := newTestAPI(t)
	access, _, _, err := jm.IssueAdminPair(1)
	require.NoError(t, err)

	user := model.User{Username: fmt.Sprintf("vu%d", time.Now().UnixNano()), PasswordHash: "hash", CoinBalanceTenths: 230}
	require.NoError(t, api.DB.Create(&user).Error)

	v := model.Video{UserID: user.ID, Title: "Admin Test", Status: "pending_review", VideoURL: "https://cdn.example.com/video.mp4", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	require.NoError(t, api.DB.Create(&v).Error)

	// Approve
	req := httptest.NewRequest("POST", "/api/v1/admin/videos/"+strconv.Itoa(int(v.ID))+"/approve", nil)
	req.Header.Set("Authorization", "Bearer "+access)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, 200, w.Code, "Approve: %s", w.Body.String())
}

func TestAdminApproveArticle(t *testing.T) {
	api, r, jm := newTestAPI(t)
	access, _, _, err := jm.IssueAdminPair(1)
	require.NoError(t, err)

	user := model.User{Username: fmt.Sprintf("au%d", time.Now().UnixNano()), PasswordHash: "hash", CoinBalanceTenths: 230}
	require.NoError(t, api.DB.Create(&user).Error)

	a := model.Article{UserID: user.ID, Title: "Admin Art", BodyMD: "# H", Status: "pending_review", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	require.NoError(t, api.DB.Create(&a).Error)

	req := httptest.NewRequest("POST", "/api/v1/admin/articles/"+strconv.Itoa(int(a.ID))+"/approve", nil)
	req.Header.Set("Authorization", "Bearer "+access)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, 200, w.Code, "Approve article: %s", w.Body.String())
}
