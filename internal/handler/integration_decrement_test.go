//go:build integration

package handler

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_EngagementToggleStates(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "dc1", "DCUser", 200)
	u2 := seedUser(t, api, "dc2", "DCUser2", 100)
	v := seedVideoWithAPI(t, api, u2.ID, "DC Video")
	tk := tok(t, api, u.ID)

	w := srve(r, areq("POST", fmt.Sprintf("/api/v1/videos/%d/like", v.ID), tk, nil))
	require.Equal(t, 200, w.Code)
	w = srve(r, areq("POST", fmt.Sprintf("/api/v1/videos/%d/like", v.ID), tk, nil))
	require.Equal(t, 200, w.Code)

	w = srve(r, areq("POST", fmt.Sprintf("/api/v1/videos/%d/favorite", v.ID), tk, nil))
	require.Equal(t, 200, w.Code)
	w = srve(r, areq("POST", fmt.Sprintf("/api/v1/videos/%d/favorite", v.ID), tk, nil))
	require.Equal(t, 200, w.Code)

	w = srve(r, areq("POST", fmt.Sprintf("/api/v1/videos/%d/watch-later", v.ID), tk, nil))
	require.Equal(t, 200, w.Code)
	w = srve(r, areq("POST", fmt.Sprintf("/api/v1/videos/%d/watch-later", v.ID), tk, nil))
	require.Equal(t, 200, w.Code)
}

func Test_UserCoinAndLedger(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "cl1", "CLUser", 100)
	u2 := seedUser(t, api, "cl2", "CLUser2", 100)
	v := seedVideoWithAPI(t, api, u2.ID, "CL Video")
	tk := tok(t, api, u.ID)

	w := srve(r, areq("POST", fmt.Sprintf("/api/v1/videos/%d/coin", v.ID), tk, `{"amount":1}`))
	require.Equal(t, 200, w.Code)

	w = srve(r, areq("GET", "/api/v1/users/me/coin-ledger", tk, nil))
	require.Equal(t, 200, w.Code)
	resp := decodeJSON(t, w)
	require.Equal(t, 0, int(resp["code"].(float64)))
}

func Test_UserMeProfile(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "mp1", "MPUser", 100)
	tk := tok(t, api, u.ID)

	w := srve(r, areq("GET", "/api/v1/users/me", tk, nil))
	require.Equal(t, 200, w.Code)
	resp := decodeJSON(t, w)
	require.Equal(t, 0, int(resp["code"].(float64)))

	w = srve(r, areq("PUT", "/api/v1/users/me/profile", tk, `{"nickname":"NewNick","signature":"Hello!"}`))
	require.Equal(t, 200, w.Code)
	resp = decodeJSON(t, w)
	require.Equal(t, 0, int(resp["code"].(float64)))

	w = srve(r, areq("PUT", "/api/v1/users/me/announcement", tk, `{"announcement":"My announcement"}`))
	require.Equal(t, 200, w.Code)
	resp = decodeJSON(t, w)
	require.Equal(t, 0, int(resp["code"].(float64)))
}
