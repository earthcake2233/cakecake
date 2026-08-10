package handler

import (
	"bytes"
	"cakecake/internal/aigateway"
	"cakecake/internal/config"
	"cakecake/internal/model/admin"
	"cakecake/internal/model/article"
	"cakecake/internal/model/dynamic"
	"cakecake/internal/model/video"
	"cakecake/internal/service/storage"
	"encoding/json"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAdminEndpoints(t *testing.T) {
	api, r, _ := newTestAPI(t)
	api.RuntimeCfg = config.NewRuntimeConfig(api.DB, nil)
	api.StorageSvc = storage.NewStorageService(api.Cfg, nil, api.Log)
	oldBackend := storage.OSSBackendOverride
	fakeOSS := &fakeOSSBackend{}
	storage.OSSBackendOverride = fakeOSS
	defer func() { storage.OSSBackendOverride = oldBackend }()
	hash, err := bcrypt.GenerateFromPassword([]byte("adminpass"), bcrypt.MinCost)
	require.NoError(t, err)
	adm := admin.Admin{Username: "rootadmin", PasswordHash: string(hash), DisplayName: "Root", Status: admin.StatusActive}
	require.NoError(t, api.DB.Create(&adm).Error)

	_, uidA := covRegister(t, r, "covma", "password12")
	vid := covSeedVideo(t, api, uidA, "admin video", video.StatusPendingReview)
	vid2 := covSeedVideo(t, api, uidA, "admin video 2", video.StatusPendingReview)
	seedArt := article.Article{UserID: uidA, Title: "admin art", BodyMD: "# x", Status: article.StatusPendingReview}
	require.NoError(t, api.DB.Create(&seedArt).Error)
	seedArt2 := article.Article{UserID: uidA, Title: "admin art 2", BodyMD: "# y", Status: article.StatusPendingReview}
	require.NoError(t, api.DB.Create(&seedArt2).Error)
	dyn := dynamic.UserDynamic{UserID: uidA, Title: "admin dyn", Content: "c"}
	require.NoError(t, api.DB.Create(&dyn).Error)

	lw := covReq(t, r, "POST", "/api/v1/admin/auth/login", "", map[string]any{"username": "rootadmin", "password": "adminpass"})
	covOK(t, lw, http.StatusOK)
	var lOut struct {
		Data struct {
			AccessToken string `json:"access_token"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(lw.Body.Bytes(), &lOut))
	at := lOut.Data.AccessToken
	covOK(t, covReq(t, r, "POST", "/api/v1/admin/auth/refresh", "", nil), http.StatusBadRequest)

	covOK(t, covReq(t, r, "GET", "/api/v1/admin/me", at, nil), http.StatusOK)
	covOK(t, covReq(t, r, "GET", "/api/v1/admin/home-banners", at, nil), http.StatusOK)
	bw := covReq(t, r, "POST", "/api/v1/admin/home-banners", at, map[string]any{"title": "b1", "image_url": "https://cdn.example.com/b.jpg", "link_url": "https://example.com"})
	covOK(t, bw, http.StatusCreated)
	var bOut struct {
		Data struct {
			ID uint64 `json:"id"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(bw.Body.Bytes(), &bOut))
	// Banner image upload (OSS-backed)
	ub := &bytes.Buffer{}
	umw := multipart.NewWriter(ub)
	ufw, err := umw.CreateFormFile("image", "banner.jpg")
	require.NoError(t, err)
	_, err = ufw.Write([]byte("jpeg"))
	require.NoError(t, err)
	require.NoError(t, umw.Close())
	ureq := httptest.NewRequest("POST", "/api/v1/admin/home-banners/upload-image", ub)
	ureq.Header.Set("Content-Type", umw.FormDataContentType())
	ureq.Header.Set("Authorization", "Bearer "+at)
	uw := httptest.NewRecorder()
	r.ServeHTTP(uw, ureq)
	covOK(t, uw, http.StatusOK)
	// Banner image upload by ID
	ub2 := &bytes.Buffer{}
	umw2 := multipart.NewWriter(ub2)
	ufw2, err := umw2.CreateFormFile("image", "b2.jpg")
	require.NoError(t, err)
	_, err = ufw2.Write([]byte("jpeg"))
	require.NoError(t, err)
	require.NoError(t, umw2.Close())
	ureq2 := httptest.NewRequest("POST", "/api/v1/admin/home-banners/"+u64s(bOut.Data.ID)+"/image", ub2)
	ureq2.Header.Set("Content-Type", umw2.FormDataContentType())
	ureq2.Header.Set("Authorization", "Bearer "+at)
	uw2 := httptest.NewRecorder()
	r.ServeHTTP(uw2, ureq2)
	covOK(t, uw2, http.StatusOK)
	covOK(t, covReq(t, r, "PUT", "/api/v1/admin/home-banners/"+u64s(bOut.Data.ID), at, map[string]any{"title": "b2"}), http.StatusOK)
	covOK(t, covReq(t, r, "DELETE", "/api/v1/admin/home-banners/"+u64s(bOut.Data.ID), at, nil), http.StatusOK)

	covOK(t, covReq(t, r, "GET", "/api/v1/admin/videos", at, nil), http.StatusOK)
	covOK(t, covReq(t, r, "GET", "/api/v1/admin/videos/"+u64s(vid), at, nil), http.StatusOK)
	covOK(t, covReq(t, r, "POST", "/api/v1/admin/videos/"+u64s(vid)+"/approve", at, map[string]any{}), http.StatusOK)
	covOK(t, covReq(t, r, "DELETE", "/api/v1/admin/videos/"+u64s(vid), at, nil), http.StatusOK)
	covOK(t, covReq(t, r, "GET", "/api/v1/admin/videos/"+u64s(vid2), at, nil), http.StatusOK)
	covOK(t, covReq(t, r, "POST", "/api/v1/admin/videos/"+u64s(vid2)+"/reject", at, map[string]any{"reason": "nope"}), http.StatusOK)
	covOK(t, covReq(t, r, "DELETE", "/api/v1/admin/videos/"+u64s(vid2), at, nil), http.StatusOK)

	covOK(t, covReq(t, r, "GET", "/api/v1/admin/articles", at, nil), http.StatusOK)
	covOK(t, covReq(t, r, "GET", "/api/v1/admin/articles/"+u64s(seedArt.ID), at, nil), http.StatusOK)
	covOK(t, covReq(t, r, "POST", "/api/v1/admin/articles/"+u64s(seedArt.ID)+"/approve", at, map[string]any{}), http.StatusOK)
	covOK(t, covReq(t, r, "DELETE", "/api/v1/admin/articles/"+u64s(seedArt.ID), at, nil), http.StatusOK)
	covOK(t, covReq(t, r, "GET", "/api/v1/admin/articles/"+u64s(seedArt2.ID), at, nil), http.StatusOK)
	covOK(t, covReq(t, r, "POST", "/api/v1/admin/articles/"+u64s(seedArt2.ID)+"/reject", at, map[string]any{"reason": "no"}), http.StatusOK)
	covOK(t, covReq(t, r, "DELETE", "/api/v1/admin/articles/"+u64s(seedArt2.ID), at, nil), http.StatusOK)

	covOK(t, covReq(t, r, "GET", "/api/v1/admin/dynamics", at, nil), http.StatusOK)
	covOK(t, covReq(t, r, "GET", "/api/v1/admin/dynamics/"+u64s(dyn.ID), at, nil), http.StatusOK)
	covOK(t, covReq(t, r, "DELETE", "/api/v1/admin/dynamics/"+u64s(dyn.ID), at, nil), http.StatusOK)

	covOK(t, covReq(t, r, "GET", "/api/v1/admin/hot-search/ops", at, nil), http.StatusOK)
	opw := covReq(t, r, "POST", "/api/v1/admin/hot-search/ops", at, map[string]any{"keyword": "kw", "op_type": "pin"})
	covOK(t, opw, http.StatusCreated)
	var opOut struct {
		Data struct {
			ID uint64 `json:"id"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(opw.Body.Bytes(), &opOut))
	covOK(t, covReq(t, r, "GET", "/api/v1/admin/hot-search/dashboard", at, nil), http.StatusOK)
	covOK(t, covReq(t, r, "GET", "/api/v1/admin/hot-search/preview", at, nil), http.StatusOK)
	covOK(t, covReq(t, r, "POST", "/api/v1/admin/hot-search/quick-op", at, map[string]any{"keyword": "q", "op_type": "block"}), http.StatusCreated)
	covOK(t, covReq(t, r, "POST", "/api/v1/admin/hot-search/reorder", at, map[string]any{"items": []map[string]any{{"keyword": "kw", "op_id": opOut.Data.ID, "source": "op"}}}), http.StatusOK)
	covOK(t, covReq(t, r, "POST", "/api/v1/admin/hot-search/display-order/reset", at, map[string]any{}), http.StatusOK)
	covOK(t, covReq(t, r, "PUT", "/api/v1/admin/hot-search/ops/"+u64s(opOut.Data.ID), at, map[string]any{"keyword": "kw2"}), http.StatusOK)
	covOK(t, covReq(t, r, "DELETE", "/api/v1/admin/hot-search/ops/"+u64s(opOut.Data.ID), at, nil), http.StatusOK)
	covOK(t, covReq(t, r, "POST", "/api/v1/admin/hot-search/redis/remove", at, map[string]any{"keyword": "kw"}), http.StatusOK)
	covOK(t, covReq(t, r, "POST", "/api/v1/admin/hot-search/redis/boost", at, map[string]any{"keyword": "kw"}), http.StatusOK)

	covOK(t, covReq(t, r, "GET", "/api/v1/admin/system-configs", at, nil), http.StatusOK)
	covOK(t, covReq(t, r, "PUT", "/api/v1/admin/system-configs", at, map[string]any{"configs": map[string]any{"rate_limit_enabled": "true"}}), http.StatusOK)
	covOK(t, covReq(t, r, "PUT", "/api/v1/admin/system-configs", at, map[string]any{}), http.StatusBadRequest)

	apw := covReq(t, r, "POST", "/api/v1/admin/agent-profiles", at, map[string]any{"slug": "assistant", "display_name": "Assistant", "sign": "hi", "system_prompt": "Please help users with their questions.", "welcome_messages": []string{"Hello!"}})
	covOK(t, apw, http.StatusOK)
	var apOut struct {
		Data struct {
			ID uint64 `json:"id"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(apw.Body.Bytes(), &apOut))
	covOK(t, covReq(t, r, "GET", "/api/v1/admin/agent-profiles", at, nil), http.StatusOK)
	covOK(t, covReq(t, r, "GET", "/api/v1/admin/agent-settings", at, nil), http.StatusOK)
	covOK(t, covReq(t, r, "PUT", "/api/v1/admin/agent-settings", at, map[string]any{"system_prompt": "Please be nice to users.", "display_name": "Assistant", "welcome_message": "Hi!"}), http.StatusOK)
	// Global-only update must not touch (or require) profile fields.
	covOK(t, covReq(t, r, "PUT", "/api/v1/admin/agent-settings", at, map[string]any{"global_system_prompt": "Global prompt for all roles."}), http.StatusOK)
	gs := covReq(t, r, "GET", "/api/v1/admin/agent-settings", at, nil)
	covOK(t, gs, http.StatusOK)
	var gsOut struct {
		Data struct {
			GlobalSystemPrompt string `json:"global_system_prompt"`
			SystemPrompt       string `json:"system_prompt"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(gs.Body.Bytes(), &gsOut))
	require.Equal(t, "Global prompt for all roles.", gsOut.Data.GlobalSystemPrompt)
	require.Equal(t, "Please be nice to users.", gsOut.Data.SystemPrompt)
	// Trigger one LLM request observation so the (otherwise zero-sample)
	// counter family is emitted by the default registry.
	aigateway.RecordLLMRequest("ok")
	mw := covReq(t, r, "GET", "/metrics", "", nil)
	covOK(t, mw, http.StatusOK)
	require.Contains(t, mw.Body.String(), "cakecake_llm_requests_total")
	apw2 := covReq(t, r, "POST", "/api/v1/admin/agent-profiles", at, map[string]any{"slug": "assistant2", "display_name": "Assistant 2", "sign": "hi", "system_prompt": "Please help users with their questions.", "welcome_messages": []string{"Hello!"}})
	covOK(t, apw2, http.StatusOK)
	covOK(t, covReq(t, r, "PUT", "/api/v1/admin/agent-profiles/"+u64s(apOut.Data.ID), at, map[string]any{"display_name": "Renamed", "system_prompt": "Please be helpful and concise.", "welcome_messages": []string{"Hey!"}}), http.StatusOK)
	covOK(t, covReq(t, r, "DELETE", "/api/v1/admin/agent-profiles/"+u64s(apOut.Data.ID), at, nil), http.StatusOK)
}
