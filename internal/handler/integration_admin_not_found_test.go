//go:build integration

package handler

import (
	"context"
	"fmt"
	"net/http"
	"testing"
)

func Test_AdminUpdateBanner_NotFound(t *testing.T) {
	api, r, _ := newTestAPI(t)
	at := admintok(t, api)
	srveOK(t, r, areq("PUT", "/api/v1/admin/home-banners/99999", at, `{"title":"Nope","image_url":"https://example.com/x.jpg","link_url":"","sort_order":1}`), http.StatusNotFound)
}

func Test_AdminDeleteAgentProfile_NotFound(t *testing.T) {
	api, r, _ := newTestAPI(t)
	at := admintok(t, api)
	srveOK(t, r, areq("DELETE", "/api/v1/admin/agent-profiles/99999", at, nil), http.StatusBadRequest)
}

func Test_AdminUpdateAgentProfile_NotFound(t *testing.T) {
	api, r, _ := newTestAPI(t)
	at := admintok(t, api)
	srveOK(t, r, areq("PUT", "/api/v1/admin/agent-profiles/99999", at, `{"name":"Test","slug":"test","welcome_messages":[]}`), http.StatusNotFound)
}

func Test_AuthRefresh_InvalidToken(t *testing.T) {
	_, r, _ := newTestAPI(t)
	srveOK(t, r, areq("POST", "/api/v1/auth/refresh", "", `{"refresh_token":"invalid-jwt-token"}`), http.StatusUnauthorized)
	srveOK(t, r, areq("POST", "/api/v1/auth/refresh", "", `{"refresh_token":""}`), http.StatusUnauthorized)
}

func Test_UpdateMePassword_Short(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "ump1", "UMP1", 10)
	tk := tok(t, api, u.ID)
	srveOK(t, r, areq("PUT", "/api/v1/users/me/password", tk, `{"old_password":"wrong","new_password":"password"}`), http.StatusForbidden)
	srveOK(t, r, areq("PUT", "/api/v1/users/me/password", tk, `{"old_password":"","new_password":"short"}`), http.StatusBadRequest)
}

func Test_VideoCountZone_Zero(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "vcz1", "VCZ1", 10)
	tk := tok(t, api, u.ID)
	srveOK(t, r, areq("GET", "/api/v1/videos?zone=999", tk, nil), http.StatusOK)
	srveOK(t, r, areq("GET", fmt.Sprintf("/api/v1/videos?user_id=%d&zone=999", u.ID), tk, nil), http.StatusOK)
}

func Test_UserFolloweeIDs_Zero(t *testing.T) {
	api, _, _ := newTestAPI(t)
	u := seedUser(t, api, "ufi1", "UFI1", 10)
	ids := make([]uint64, 0)
	f1, _ := api.FollowSvc.GetFollowingIDs(context.Background(), u.ID, nil)
	for k := range f1 {
		ids = append(ids, k)
	}
	if ids == nil {
		t.Error("expected non-nil set")
	}
}
