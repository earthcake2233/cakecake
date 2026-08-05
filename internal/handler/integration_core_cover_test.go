//go:build integration

package handler

import (
	"cakecake/internal/model/dynamic"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func Test_SearchHistoryFlow(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "shf1", "SHF1", 0)
	tk := tok(t, api, u.ID)

	srve(r, areq("GET", "/api/v1/users/me/search-history", tk, nil))
	srve(r, areq("DELETE", "/api/v1/users/me/search-history?keyword=test", tk, nil))
	srve(r, areq("DELETE", "/api/v1/users/me/search-history", tk, nil))
}

func Test_UserSpaceFlow(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "usp1", "USP1", 0)
	srve(r, areq("GET", fmt.Sprintf("/api/v1/space/%d", u.ID), "", nil))
}

func Test_UserBlockFlow(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "ubl1", "UBL1", 0)
	u2 := seedUser(t, api, "ubl2", "UBL2", 0)
	tk := tok(t, api, u.ID)

	srve(r, areq("POST", fmt.Sprintf("/api/v1/users/%d/block", u2.ID), tk, nil))
	srve(r, areq("GET", "/api/v1/users/me/blocks", tk, nil))
	srve(r, areq("DELETE", fmt.Sprintf("/api/v1/users/%d/block", u2.ID), tk, nil))
}

func Test_HotSearchListFlow(t *testing.T) {
	_, r, _ := newTestAPI(t)
	srve(r, areq("GET", "/api/v1/hot-search/list", "", nil))
}

func Test_DailyRewardFlow(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "drf1", "DRF1", 0)
	tk := tok(t, api, u.ID)
	srve(r, areq("GET", "/api/v1/users/me/daily-rewards", tk, nil))
}

func Test_AdminDeleteDynamicFlow(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "add1", "ADD1", 0)
	dyn := dynamic.UserDynamic{UserID: u.ID, Title: "test", Content: "test", ImagesJSON: "[]", CreatedAt: time.Now()}
	require.NoError(t, api.DB.Create(&dyn).Error)
	atk := admintok(t, api)
	srve(r, areq("DELETE", fmt.Sprintf("/api/v1/admin/dynamics/%d", dyn.ID), atk, nil))
}
