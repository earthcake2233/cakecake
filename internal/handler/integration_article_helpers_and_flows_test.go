//go:build integration

package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"cakecake/internal/config"
	"github.com/stretchr/testify/require"
)

func TestParseArticleTagsJSON(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want []string
	}{
		{"empty", "", nil},
		{"single", `["tag1"]`, []string{"tag1"}},
		{"multiple", `["tag1","tag2","tag3"]`, []string{"tag1", "tag2", "tag3"}},
		{"invalid json", `not json`, nil},
		{"nil", "null", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseArticleTagsJSON(tt.raw)
			if tt.want == nil {
				require.Nil(t, got)
			} else {
				require.Equal(t, tt.want, got)
			}
		})
	}
}

func TestValidateArticleContent_Basic(t *testing.T) {
	require.True(t, validateArticleContent("Title", "# Content", false))
	require.True(t, validateArticleContent("", "# Content", false))
	require.True(t, validateArticleContent("Title", "", false))
	require.True(t, validateArticleContent("Title", "# Content", true))
	require.False(t, validateArticleContent("", "", true))
}

func TestManuscriptArticleStatusToDB(t *testing.T) {
	require.Equal(t, "draft", manuscriptArticleStatusToDB("draft"))
	require.Equal(t, "", manuscriptArticleStatusToDB(""))
	require.Equal(t, "", manuscriptArticleStatusToDB("pending"))
	require.Equal(t, "published", manuscriptArticleStatusToDB("published"))
}

func TestOrderClauseForMyArticles(t *testing.T) {
	require.Contains(t, orderClauseForMyArticles("time"), "created_at")
	require.Contains(t, orderClauseForMyArticles("view"), "view_count")
	require.Contains(t, orderClauseForMyArticles(""), "created_at")
}

func TestArticleStatusAfterSubmit(t *testing.T) {
	api := &API{Dependencies: &Dependencies{Cfg: &config.C{}}}
	require.Equal(t, "draft", api.articleStatusAfterSubmit(false))
	// With publish=true and ArticleReviewRequired=false (default), returns "published"
	require.Equal(t, "published", api.articleStatusAfterSubmit(true))
}

func TestMergeUniqueDisplayNames(t *testing.T) {
	require.Empty(t, mergeUniqueDisplayNames(nil))
	require.Empty(t, mergeUniqueDisplayNames([]string{}))
	require.Equal(t, []string{"a"}, mergeUniqueDisplayNames([]string{"a"}))
	require.Equal(t, []string{"a", "b"}, mergeUniqueDisplayNames([]string{"a", "b"}))
	require.Equal(t, []string{"a", "b"}, mergeUniqueDisplayNames([]string{"a", "b", "a"}))
}

func TestIsReplyInboxType(t *testing.T) {
	require.True(t, isReplyInboxType("reply_received"))
	require.True(t, isReplyInboxType("article_reply_received"))
	require.False(t, isReplyInboxType(""))
	require.False(t, isReplyInboxType("like"))
}

func TestNotifUint64(t *testing.T) {
	require.Equal(t, uint64(42), notifUint64(float64(42)))
	require.Equal(t, uint64(0), notifUint64("not a number"))
	require.Equal(t, uint64(0), notifUint64(nil))
}

func TestDmTrimPreview(t *testing.T) {
	require.Equal(t, "", dmTrimPreview(""))
	require.Equal(t, "hello", dmTrimPreview("hello"))
	result := dmTrimPreview("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaabbbbb")
	require.Contains(t, result, "\u2026")
	require.Equal(t, 81, len([]rune(result)))
}

func TestDmPinnedAtAfter(t *testing.T) {
	now := time.Now()
	before := now.Add(-time.Hour)
	after := now.Add(time.Hour)
	require.True(t, dmPinnedAtAfter(&after, &before))
	require.False(t, dmPinnedAtAfter(&before, &after))
	require.False(t, dmPinnedAtAfter(nil, &after))
	require.True(t, dmPinnedAtAfter(&after, nil))
	require.False(t, dmPinnedAtAfter(nil, nil))
}

func TestParseFolderIsPublicForm(t *testing.T) {
	require.True(t, parseFolderIsPublicForm("true"))
	require.True(t, parseFolderIsPublicForm("1"))
	require.True(t, parseFolderIsPublicForm("True"))
	require.False(t, parseFolderIsPublicForm("false"))
	require.True(t, parseFolderIsPublicForm(""))
	require.False(t, parseFolderIsPublicForm("0"))
}

func TestParseFolderIDQuery(t *testing.T) {
	// This needs a gin context, test via HTTP
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "pfiq1", "PFIQ1", 10)
	tk := tok(t, api, u.ID)
	srveOK(t, r, areq("GET", "/api/v1/users/me/favorites?folder_id=0", tk, nil), http.StatusBadRequest)
	srveOK(t, r, areq("GET", "/api/v1/users/me/favorites?folder_id=abc", tk, nil), http.StatusBadRequest)
}

func TestParseVideoFolderParams_NoParam(t *testing.T) {
	// Test via HTTP with invalid params
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "pvf1", "PVF1", 10)
	tk := tok(t, api, u.ID)
	// Missing video ID
	srveOK(t, r, areq("DELETE", "/api/v1/videos/0/favorite-folders/0", tk, nil), http.StatusBadRequest)
}

func Test_UserDynamicPutAndPlayback(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "udp1", "UDP1", 10)
	tk := tok(t, api, u.ID)

	// Post a dynamic
	w := doMultipart(r, "POST", "/api/v1/users/me/dynamics", tk, map[string]string{
		"title": "EditMe", "content": "To be edited",
	})
	covOK(t, w, http.StatusOK)
	var dr struct {
		Code int `json:"code"`
		Data struct {
			ID uint64 `json:"id"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &dr))
	require.Equal(t, 0, dr.Code, w.Body.String())
	if dr.Code == 0 && dr.Data.ID > 0 {
		did := dr.Data.ID
		// Update dynamic
		uw := doMultipart(r, "PUT", fmt.Sprintf("/api/v1/users/me/dynamics/%d", did), tk, map[string]string{
			"title": "Edited", "content": "Edited content",
		})
		covOK(t, uw, http.StatusOK)
		// Patch playback
		pw := covReq(t, r, "PATCH", fmt.Sprintf("/api/v1/users/me/dynamics/%d/playback", did), tk, map[string]any{"comments_closed": true})
		covOK(t, pw, http.StatusOK)
	}

	// List dynamics via space
	srveOK(t, r, areq("GET", fmt.Sprintf("/api/v1/space/%d/dynamics", u.ID), tk, nil), http.StatusOK)
}

func Test_ArticleViewCount(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "avc1", "AVC1", 10)
	tk := tok(t, api, u.ID)
	art := seedArticle(t, api, u.ID, "View Count Test")

	// Post article view (increment count)
	srveOK(t, r, areq("POST", fmt.Sprintf("/api/v1/articles/%d/view", art.ID), tk, nil), http.StatusOK)

	// Get article detail
	srveOK(t, r, areq("GET", fmt.Sprintf("/api/v1/articles/%d", art.ID), "", nil), http.StatusOK)

	// List user published articles
	srveOK(t, r, areq("GET", fmt.Sprintf("/api/v1/space/%d/article-favorites", u.ID), "", nil), http.StatusOK)
}

func Test_ArticleCommentSubActions(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "acs1", "ACS1", 10)
	tk := tok(t, api, u.ID)
	art := seedArticle(t, api, u.ID, "AC Sub Article")

	// Post article comment with specific body
	w := srve(r, areq("POST", fmt.Sprintf("/api/v1/articles/%d/comments", art.ID), tk,
		fmt.Sprintf(`{"content":"sub comment","article_id":%d}`, art.ID)))
	var cr struct {
		Code int `json:"code"`
		Data struct {
			ID uint64 `json:"id"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &cr))
	if cr.Code == 0 && cr.Data.ID > 0 {
		cid := cr.Data.ID
		// Like
		srveOK(t, r, areq("POST", fmt.Sprintf("/api/v1/article-comments/%d/like", cid), tk, nil), http.StatusOK)
		// Dislike
		srveOK(t, r, areq("POST", fmt.Sprintf("/api/v1/article-comments/%d/dislike", cid), tk, nil), http.StatusOK)
		// List article comments
		srveOK(t, r, areq("GET", fmt.Sprintf("/api/v1/articles/%d/comments", art.ID), "", nil), http.StatusOK)
	}
}
func Test_DmPostAndList(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "dpal1", "DPAL1", 10)
	u2 := seedUser(t, api, "dpal2", "DPAL2", 10)
	tk := tok(t, api, u.ID)

	// Create a conversation (should succeed)
	w := srve(r, areq("POST", "/api/v1/dm/conversations", tk, fmt.Sprintf(`{"peer_id":%d}`, u2.ID)))
	var dcr struct {
		Code int `json:"code"`
		Data struct {
			ID uint64 `json:"id"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &dcr))
	if dcr.Code == 0 && dcr.Data.ID > 0 {
		cid := dcr.Data.ID
		// List conversations
		srveOK(t, r, areq("GET", "/api/v1/dm/conversations", tk, nil), http.StatusOK)
		// List messages
		srveOK(t, r, areq("GET", fmt.Sprintf("/api/v1/dm/conversations/%d/messages", cid), tk, nil), http.StatusOK)
		// Post a message
		srveOK(t, r, areq("POST", fmt.Sprintf("/api/v1/dm/conversations/%d/messages", cid), tk, `{"content":"Hello from test"}`), http.StatusOK)
	}
}
func Test_SpaceRecentCoinsAndFavorites(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "src1", "SRC1", 100)
	u2 := seedUser(t, api, "src2", "SRC2", 100)
	v := seedVideoWithAPI(t, api, u2.ID, "Coin Video")

	// Post a coin from u to u2's video
	tk := tok(t, api, u.ID)
	srveOK(t, r, areq("POST", fmt.Sprintf("/api/v1/videos/%d/coin", v.ID), tk, `{"amount":1}`), http.StatusOK)

	// List recent coins
	srveOK(t, r, areq("GET", fmt.Sprintf("/api/v1/space/%d/recent-coins", u.ID), "", nil), http.StatusOK)
}
