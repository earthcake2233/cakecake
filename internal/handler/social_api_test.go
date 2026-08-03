package handler

import (
	"cakecake/internal/model/video"
	"encoding/json"
	"github.com/stretchr/testify/require"
	"net/http"
	"testing"
)

func TestSocialFlows(t *testing.T) {
	api, r, _ := newTestAPI(t)
	tokenA, uidA := covRegister(t, r, "covfa", "password12")
	tokenB, _ := covRegister(t, r, "covfb", "password12")
	vid := covSeedVideo(t, api, uidA, "history video", video.StatusPublished)

	// Follow + groups + block
	covOK(t, covReq(t, r, "POST", "/api/v1/users/"+u64s(uidA)+"/follow", tokenB, map[string]any{}), http.StatusOK)
	gw := covReq(t, r, "POST", "/api/v1/users/me/follow-groups", tokenB, map[string]any{"name": "group1"})
	covOK(t, gw, http.StatusOK)
	var gOut struct {
		Data struct {
			ID uint64 `json:"id"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(gw.Body.Bytes(), &gOut))
	covOK(t, covReq(t, r, "GET", "/api/v1/users/me/follow-groups", tokenB, nil), http.StatusOK)
	covOK(t, covReq(t, r, "PUT", "/api/v1/users/me/follow-groups/"+u64s(gOut.Data.ID), tokenB, map[string]any{"name": "renamed"}), http.StatusOK)
	covOK(t, covReq(t, r, "POST", "/api/v1/users/me/follow-groups/"+u64s(gOut.Data.ID)+"/members", tokenB, map[string]any{"followee_id": uidA}), http.StatusOK)
	covOK(t, covReq(t, r, "GET", "/api/v1/users/me/following/"+u64s(uidA)+"/groups", tokenB, nil), http.StatusOK)
	covOK(t, covReq(t, r, "DELETE", "/api/v1/users/me/follow-groups/"+u64s(gOut.Data.ID)+"/members/"+u64s(uidA), tokenB, nil), http.StatusOK)
	covOK(t, covReq(t, r, "DELETE", "/api/v1/users/me/follow-groups/"+u64s(gOut.Data.ID), tokenB, nil), http.StatusOK)
	covOK(t, covReq(t, r, "PUT", "/api/v1/users/me/space-privacy", tokenA, map[string]any{"public_following": true, "public_fans": true, "public_favorites": true}), http.StatusOK)
	covOK(t, covReq(t, r, "GET", "/api/v1/space/"+u64s(uidA)+"/following", tokenB, nil), http.StatusOK)
	covOK(t, covReq(t, r, "GET", "/api/v1/space/"+u64s(uidA)+"/followers", tokenB, nil), http.StatusOK)

	// View history
	covOK(t, covReq(t, r, "POST", "/api/v1/videos/"+u64s(vid)+"/view-history", tokenB, map[string]any{"progress_seconds": 10, "duration_seconds": 100, "device": "web"}), http.StatusOK)
	covOK(t, covReq(t, r, "GET", "/api/v1/users/me/view-history", tokenB, nil), http.StatusOK)
	covOK(t, covReq(t, r, "GET", "/api/v1/users/me/view-history/settings", tokenB, nil), http.StatusOK)
	covOK(t, covReq(t, r, "PUT", "/api/v1/users/me/view-history/settings", tokenB, map[string]any{"paused": true}), http.StatusOK)
	covOK(t, covReq(t, r, "DELETE", "/api/v1/users/me/view-history/"+u64s(vid), tokenB, nil), http.StatusOK)
	covOK(t, covReq(t, r, "POST", "/api/v1/videos/"+u64s(vid)+"/view-history", tokenB, map[string]any{"progress_seconds": 5, "duration_seconds": 100, "device": "web"}), http.StatusOK)
	covOK(t, covReq(t, r, "DELETE", "/api/v1/users/me/view-history", tokenB, nil), http.StatusOK)

	// DM
	dmw := covReq(t, r, "POST", "/api/v1/dm/conversations", tokenA, map[string]any{"peer_id": 2})
	covOK(t, dmw, http.StatusOK)
	var dmOut struct {
		Data struct {
			ID uint64 `json:"id"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(dmw.Body.Bytes(), &dmOut))
	convID := dmOut.Data.ID
	covOK(t, covReq(t, r, "GET", "/api/v1/dm/conversations", tokenA, nil), http.StatusOK)
	covOK(t, covReq(t, r, "POST", "/api/v1/dm/conversations/"+u64s(convID)+"/messages", tokenA, map[string]any{"content": "hello"}), http.StatusOK)
	covOK(t, covReq(t, r, "GET", "/api/v1/dm/conversations/"+u64s(convID)+"/messages", tokenB, nil), http.StatusOK)
	covOK(t, covReq(t, r, "PATCH", "/api/v1/dm/conversations/"+u64s(convID)+"/settings", tokenA, map[string]any{"muted": true}), http.StatusOK)
	covOK(t, covReq(t, r, "POST", "/api/v1/dm/conversations/"+u64s(convID)+"/reset", tokenA, map[string]any{}), http.StatusOK)
	covOK(t, covReq(t, r, "DELETE", "/api/v1/dm/conversations/"+u64s(convID), tokenA, nil), http.StatusOK)
	covOK(t, covReq(t, r, "POST", "/api/v1/users/"+u64s(uidA)+"/block", tokenB, map[string]any{}), http.StatusOK)

	// Search + suggest + history + hot search
	covOK(t, covReq(t, r, "GET", "/api/v1/search?keyword=video", tokenB, nil), http.StatusOK)
	covOK(t, covReq(t, r, "GET", "/api/v1/search?keyword=", "", nil), http.StatusBadRequest)
	covOK(t, covReq(t, r, "GET", "/api/v1/search/suggest?term=vid", tokenB, nil), http.StatusOK)
	covOK(t, covReq(t, r, "GET", "/api/v1/users/me/search-history", tokenB, nil), http.StatusOK)
	covOK(t, covReq(t, r, "POST", "/api/v1/users/me/search-history", tokenB, map[string]any{"keyword": "video"}), http.StatusOK)
	covOK(t, covReq(t, r, "PUT", "/api/v1/users/me/search-history", tokenB, map[string]any{"keywords": []string{"a", "b"}}), http.StatusOK)
	covOK(t, covReq(t, r, "GET", "/api/v1/hot-search", "", nil), http.StatusOK)
	covOK(t, covReq(t, r, "GET", "/api/v1/stats/home", "", nil), http.StatusOK)
	covOK(t, covReq(t, r, "GET", "/api/v1/home-banners", "", nil), http.StatusOK)
}
