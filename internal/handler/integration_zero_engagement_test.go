//go:build integration

package handler

import (
	"testing"

)

func Test_VideoEngagementByViewer_Zero(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "veb1", "VEB1", 10)
	tk := tok(t, api, u.ID)
	// Test that endpoints handle zero/non-existent gracefully
	srve(r, areq("GET", "/api/v1/videos/99999/fav", tk, nil))
	srve(r, areq("GET", "/api/v1/videos/99999/coin", tk, nil))
	srve(r, areq("GET", "/api/v1/videos/99999/engagement", tk, nil))
}

func Test_WatchLaterByViewer_Zero(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "wlb1", "WLB1", 10)
	tk := tok(t, api, u.ID)
	// Non-existent video
	srve(r, areq("POST", "/api/v1/videos/99999/watch-later", tk, nil))
}

func Test_DmUnreadTotal_Zero(t *testing.T) {
	api, _, _ := newTestAPI(t)
	u := seedUser(t, api, "dut1", "DUT1", 10)
	cnt := api.dmUnreadTotal(u.ID)
	if cnt != 0 {
		t.Errorf("expected 0 unread, got %d", cnt)
	}
}

func Test_CoinRecentListItems_Zero(t *testing.T) {
	api, _, _ := newTestAPI(t)
	u := seedUser(t, api, "crl1", "CRL1", 10)
	items, _, err := api.coinRecentListItems(nil, u.ID, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 0 {
		t.Errorf("expected empty, got %v", items)
	}
}

func Test_AccountDeletion_RequestAndRevoke(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "adr1", "ADR1", 10)
	tk := tok(t, api, u.ID)
	srve(r, areq("POST", "/api/v1/users/me/account-deletion/request", tk, nil))
	srve(r, areq("POST", "/api/v1/users/me/account-deletion/revoke", tk, nil))
}
