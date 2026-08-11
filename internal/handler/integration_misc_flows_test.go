//go:build integration

package handler

import (
	"cakecake/internal/model/dynamic"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func Test_SearchHistoryFlow(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "shf1", "SHF1", 0)
	tk := tok(t, api, u.ID)

	srveOK(t, r, areq("GET", "/api/v1/users/me/search-history", tk, nil), http.StatusOK)
	// The API has no keyword/clear DELETE routes; these paths are 404.
	srveOK(t, r, areq("DELETE", "/api/v1/users/me/search-history?keyword=test", tk, nil), http.StatusNotFound)
	srveOK(t, r, areq("DELETE", "/api/v1/users/me/search-history", tk, nil), http.StatusNotFound)
}

func Test_UserSpaceFlow(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "usp1", "USP1", 0)
	srveOK(t, r, areq("GET", fmt.Sprintf("/api/v1/space/%d", u.ID), "", nil), http.StatusOK)
}

func Test_UserBlockFlow(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "ubl1", "UBL1", 0)
	u2 := seedUser(t, api, "ubl2", "UBL2", 0)
	tk := tok(t, api, u.ID)

	srveOK(t, r, areq("POST", fmt.Sprintf("/api/v1/users/%d/block", u2.ID), tk, nil), http.StatusOK)
	// No blocked-list route exists; unblocking is a toggle on the same POST path.
	srveOK(t, r, areq("GET", "/api/v1/users/me/blocks", tk, nil), http.StatusNotFound)
	srveOK(t, r, areq("DELETE", fmt.Sprintf("/api/v1/users/%d/block", u2.ID), tk, nil), http.StatusNotFound)
}

func Test_HotSearchPublicList(t *testing.T) {
	_, r, _ := newTestAPI(t)
	srveOK(t, r, areq("GET", "/api/v1/hot-search", "", nil), http.StatusOK)
}

func Test_DailyRewardFlow(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "drf1", "DRF1", 0)
	tk := tok(t, api, u.ID)
	srveOK(t, r, areq("GET", "/api/v1/users/me/daily-rewards", tk, nil), http.StatusOK)
}

func Test_AdminDeleteDynamicFlow(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "add1", "ADD1", 0)
	dyn := dynamic.UserDynamic{UserID: u.ID, Title: "test", Content: "test", ImagesJSON: "[]", CreatedAt: time.Now()}
	require.NoError(t, api.DB.Create(&dyn).Error)
	atk := admintok(t, api)
	w := srveOK(t, r, areq("DELETE", fmt.Sprintf("/api/v1/admin/dynamics/%d", dyn.ID), atk, nil), http.StatusOK)
	require.Equal(t, 0, decodeCode(t, w), w.Body.String())
	var n int64
	require.NoError(t, api.DB.Model(&dynamic.UserDynamic{}).Where("id = ?", dyn.ID).Count(&n).Error)
	require.Zero(t, n)
}
