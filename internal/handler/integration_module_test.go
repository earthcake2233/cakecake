package handler

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_ArticleEngagement_Basic(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "ae1", "AEUser", 100)
	art := seedArticle(t, api, u.ID, "AE Article")
	tk := tok(t, api, u.ID)

	// PostArticleView
	w := srve(r, areq("POST", "/api/v1/articles/"+fmt.Sprint(art.ID)+"/view", tk, `{"progress_sec":0,"duration_sec":300}`))
	require.Equal(t, 200, w.Code)
	resp := decodeJSON(t, w)
	require.Equal(t, 0, int(resp["code"].(float64)), w.Body.String())

	// PatchArticlePlayback (expects comments_closed/comments_curated)
	w = srve(r, areq("PATCH", "/api/v1/users/me/articles/"+fmt.Sprint(art.ID)+"/playback", tk, `{"comments_closed":true}`))
	require.Equal(t, 200, w.Code)
	resp = decodeJSON(t, w)
	require.Equal(t, 0, int(resp["code"].(float64)), w.Body.String())
}

func Test_ArticleCommentEngagement(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "ac1", "ACUser", 100)
	u2 := seedUser(t, api, "ac2", "ACUser2", 100)
	art := seedArticle(t, api, u.ID, "AC Article")
	tk := tok(t, api, u.ID)
	tk2 := tok(t, api, u2.ID)

	w := srve(r, areq("POST", "/api/v1/articles/"+fmt.Sprint(art.ID)+"/comments", tk2, `{"content":"nice article"}`))
	require.Equal(t, 201, w.Code)
	resp := decodeJSON(t, w)
	require.Equal(t, 0, int(resp["code"].(float64)))
	cmtID := int(resp["data"].(map[string]interface{})["id"].(float64))

	w = srve(r, areq("POST", "/api/v1/article-comments/"+fmt.Sprint(cmtID)+"/like", tk, nil))
	require.Equal(t, 200, w.Code)
	resp = decodeJSON(t, w)
	require.Equal(t, 0, int(resp["code"].(float64)))

	w = srve(r, areq("POST", "/api/v1/article-comments/"+fmt.Sprint(cmtID)+"/dislike", tk, nil))
	require.Equal(t, 200, w.Code)
	resp = decodeJSON(t, w)
	require.Equal(t, 0, int(resp["code"].(float64)))

	w = srve(r, areq("POST", "/api/v1/article-comments/"+fmt.Sprint(cmtID)+"/pin", tk, nil))
	require.Equal(t, 200, w.Code)
	w = srve(r, areq("POST", "/api/v1/article-comments/"+fmt.Sprint(cmtID)+"/pin", tk, nil))
	require.Equal(t, 200, w.Code)

	w = srve(r, areq("POST", "/api/v1/article-comments/"+fmt.Sprint(cmtID)+"/approve", tk, nil))
	require.Equal(t, 200, w.Code)
	resp = decodeJSON(t, w)
	require.Equal(t, 0, int(resp["code"].(float64)))
}

func Test_DM_Basic(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "dm1", "DMUser", 100)
	u2 := seedUser(t, api, "dm2", "DMUser2", 100)
	tk := tok(t, api, u.ID)

	w := srve(r, areq("POST", "/api/v1/dm/conversations", tk, `{"peer_id":`+fmt.Sprint(u2.ID)+`}`))
	require.Equal(t, 200, w.Code)
	resp := decodeJSON(t, w)
	require.Equal(t, 0, int(resp["code"].(float64)))
	convID := int(resp["data"].(map[string]interface{})["id"].(float64))

	w = srve(r, areq("GET", "/api/v1/dm/conversations", tk, nil))
	require.Equal(t, 200, w.Code)
	resp = decodeJSON(t, w)
	require.Equal(t, 0, int(resp["code"].(float64)))

	w = srve(r, areq("POST", "/api/v1/dm/conversations/"+fmt.Sprint(convID)+"/messages", tk, `{"content":"hello"}`))
	require.Equal(t, 200, w.Code)
	resp = decodeJSON(t, w)
	require.Equal(t, 0, int(resp["code"].(float64)))

	w = srve(r, areq("GET", "/api/v1/dm/conversations/"+fmt.Sprint(convID)+"/messages", tk, nil))
	require.Equal(t, 200, w.Code)
	resp = decodeJSON(t, w)
	require.Equal(t, 0, int(resp["code"].(float64)))

	w = srve(r, areq("DELETE", "/api/v1/dm/conversations/"+fmt.Sprint(convID), tk, nil))
	require.Equal(t, 200, w.Code)
	resp = decodeJSON(t, w)
	require.Equal(t, 0, int(resp["code"].(float64)))
}

func Test_SearchHistory_Ops(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "sh1", "SHUser", 100)
	tk := tok(t, api, u.ID)

	w := srve(r, areq("POST", "/api/v1/users/me/search-history", tk, `{"keyword":"golang"}`))
	require.Equal(t, 200, w.Code)
	resp := decodeJSON(t, w)
	require.Equal(t, 0, int(resp["code"].(float64)))

	w = srve(r, areq("GET", "/api/v1/users/me/search-history", tk, nil))
	require.Equal(t, 200, w.Code)
	resp = decodeJSON(t, w)
	require.Equal(t, 0, int(resp["code"].(float64)))

	w = srve(r, areq("PUT", "/api/v1/users/me/search-history", tk, `{"keyword":"rust"}`))
	require.Equal(t, 200, w.Code)
	resp = decodeJSON(t, w)
	require.Equal(t, 0, int(resp["code"].(float64)))
}
