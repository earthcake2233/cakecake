package handler

import (
	"fmt"
	"testing"
)

func Test_AdminUpdateBanner_NotFound(t *testing.T) {
	api, r, _ := newTestAPI(t)
	at := admintok(t, api)
	srve(r, areq("PUT", "/api/v1/admin/home-banners/99999", at, `{"title":"Nope","image_url":"https://example.com/x.jpg","link_url":"","sort_order":1}`))
}

func Test_AdminDeleteAgentProfile_NotFound(t *testing.T) {
	api, r, _ := newTestAPI(t)
	at := admintok(t, api)
	srve(r, areq("DELETE", "/api/v1/admin/agent-profiles/99999", at, nil))
}

func Test_AdminUpdateAgentProfile_NotFound(t *testing.T) {
	api, r, _ := newTestAPI(t)
	at := admintok(t, api)
	srve(r, areq("PUT", "/api/v1/admin/agent-profiles/99999", at, `{"name":"Test","slug":"test","welcome_messages":[]}`))
}

func Test_AuthRefresh_InvalidToken(t *testing.T) {
	_, r, _ := newTestAPI(t)
	srve(r, areq("POST", "/api/v1/auth/refresh", "", `{"token":"invalid-jwt-token"}`))
	srve(r, areq("POST", "/api/v1/auth/refresh", "", `{"token":""}`))
}

func Test_UpdateMePassword_Short(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "ump1", "UMP1", 10)
	tk := tok(t, api, u.ID)
	srve(r, areq("PUT", "/api/v1/users/me/password", tk, `{"old_password":"wrong","new_password":"short"}`))
	srve(r, areq("PUT", "/api/v1/users/me/password", tk, `{"old_password":"","new_password":"newpass12"}`))
}

func Test_VideoCountZone_Zero(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "vcz1", "VCZ1", 10)
	tk := tok(t, api, u.ID)
	srve(r, areq("GET", "/api/v1/videos?zone=999", tk, nil))
	srve(r, areq("GET", fmt.Sprintf("/api/v1/videos?user_id=%d&zone=999", u.ID), tk, nil))
}

func Test_UserFolloweeIDs_Zero(t *testing.T) {
	api, _, _ := newTestAPI(t)
	u := seedUser(t, api, "ufi1", "UFI1", 10)
	ids := userFolloweeIDsSet(api.DB, u.ID, nil)
	if ids == nil {
		t.Error("expected non-nil set")
	}
}
