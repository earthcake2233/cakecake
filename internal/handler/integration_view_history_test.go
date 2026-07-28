package handler

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_ViewHistory_Basic(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "vh1", "VHUser", 100)
	v := seedVideo(t, api, u.ID, "VH Video")
	tk := tok(t, api, u.ID)

	// Post view history
	w := srve(r, areq("POST", "/api/v1/videos/"+fmt.Sprint(v.ID)+"/view-history", tk, `{"progress_sec":30,"duration_sec":120,"device":"mobile"}`))
	require.Equal(t, 200, w.Code)
	resp := decodeJSON(t, w)
	require.Equal(t, 0, int(resp["code"].(float64)), w.Body.String())

	// List
	w = srve(r, areq("GET", "/api/v1/users/me/view-history", tk, nil))
	require.Equal(t, 200, w.Code)
	resp = decodeJSON(t, w)
	require.Equal(t, 0, int(resp["code"].(float64)), w.Body.String())

	// Settings get
	w = srve(r, areq("GET", "/api/v1/users/me/view-history/settings", tk, nil))
	require.Equal(t, 200, w.Code)
	resp = decodeJSON(t, w)
	require.Equal(t, 0, int(resp["code"].(float64)), w.Body.String())

	// Pause
	w = srve(r, areq("PUT", "/api/v1/users/me/view-history/settings", tk, `{"paused":true}`))
	require.Equal(t, 200, w.Code)
	resp = decodeJSON(t, w)
	require.Equal(t, 0, int(resp["code"].(float64)), w.Body.String())

	// While paused
	w = srve(r, areq("POST", "/api/v1/videos/"+fmt.Sprint(v.ID)+"/view-history", tk, `{"progress_sec":60,"duration_sec":120,"device":"mobile"}`))
	require.Equal(t, 200, w.Code)
	resp = decodeJSON(t, w)
	d := resp["data"].(map[string]interface{})
	require.True(t, d["paused"].(bool))

	// Unpause
	w = srve(r, areq("PUT", "/api/v1/users/me/view-history/settings", tk, `{"paused":false}`))
	require.Equal(t, 200, w.Code)

	// Delete single entry
	w = srve(r, areq("DELETE", "/api/v1/users/me/view-history/"+fmt.Sprint(v.ID), tk, nil))
	require.Equal(t, 200, w.Code)
	resp = decodeJSON(t, w)
	require.Equal(t, 0, int(resp["code"].(float64)), w.Body.String())

	// Clear all
	w = srve(r, areq("DELETE", "/api/v1/users/me/view-history", tk, nil))
	require.Equal(t, 200, w.Code)
	resp = decodeJSON(t, w)
	require.Equal(t, 0, int(resp["code"].(float64)), w.Body.String())
}

func Test_ViewHistory_Article(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "vh2", "VHUser2", 100)
	art := seedArticle(t, api, u.ID, "VH Art")
	tk := tok(t, api, u.ID)

	// Post article view
	w := srve(r, areq("POST", "/api/v1/articles/"+fmt.Sprint(art.ID)+"/view", tk, `{"progress_sec":10,"duration_sec":60}`))
	require.Equal(t, 200, w.Code)
	resp := decodeJSON(t, w)
	require.Equal(t, 0, int(resp["code"].(float64)), w.Body.String())

	// Delete article view history entry (note: route is view-history/articles/:articleId)
	w = srve(r, areq("DELETE", "/api/v1/users/me/view-history/articles/"+fmt.Sprint(art.ID), tk, nil))
	require.Equal(t, 200, w.Code)
	resp = decodeJSON(t, w)
	require.Equal(t, 0, int(resp["code"].(float64)), w.Body.String())
}
