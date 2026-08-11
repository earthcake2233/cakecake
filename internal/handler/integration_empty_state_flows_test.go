//go:build integration

package handler

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

func Test_VideoEngagementByViewer_Zero(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "veb1", "VEB1", 10)
	tk := tok(t, api, u.ID)
	// Test that endpoints handle zero/non-existent gracefully
	srveOK(t, r, areq("GET", "/api/v1/videos/99999", tk, nil), http.StatusNotFound)
	srveOK(t, r, areq("POST", "/api/v1/videos/99999/like", tk, nil), http.StatusNotFound)
	srveOK(t, r, areq("POST", "/api/v1/videos/99999/coin", tk, `{"amount":1}`), http.StatusNotFound)
}

func Test_WatchLaterByViewer_Zero(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "wlb1", "WLB1", 10)
	tk := tok(t, api, u.ID)
	// Non-existent video
	srveOK(t, r, areq("POST", "/api/v1/videos/99999/watch-later", tk, nil), http.StatusNotFound)
}

func Test_DmUnreadTotal_Zero(t *testing.T) {
	api, _, _ := newTestAPI(t)
	u := seedUser(t, api, "dut1", "DUT1", 10)
	cnt := api.dmUnreadTotal(context.Background(), u.ID)
	if cnt != 0 {
		t.Errorf("expected 0 unread, got %d", cnt)
	}
}

func Test_CoinRecentListItems_Zero(t *testing.T) {
	api, _, _ := newTestAPI(t)
	u := seedUser(t, api, "crl1", "CRL1", 10)
	items, _, err := api.coinRecentListItems(context.TODO(), u.ID, 10)
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
	hash, err := bcrypt.GenerateFromPassword([]byte("password12"), bcrypt.MinCost)
	require.NoError(t, err)
	require.NoError(t, api.DB.Model(&u).Update("password_hash", string(hash)).Error)
	tk := tok(t, api, u.ID)
	srveOK(t, r, areq("POST", "/api/v1/users/me/deletion/request", tk, `{"password":"password12"}`), http.StatusOK)
	srveOK(t, r, areq("POST", "/api/v1/users/me/deletion/revoke", tk, nil), http.StatusOK)
}
