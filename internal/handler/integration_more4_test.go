package handler

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"minibili/internal/config"

	"minibili/internal/model"
)


func TestParseArticleTagsJSON(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want []string
	}{
		{"empty", "", nil},
		{"single", `["tag1"]`, []string{"tag1"}},
		{"multiple", `["tag1","tag2","tag3"]`, []string{"tag1","tag2","tag3"}},
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
	require.Equal(t, []string{"a","b"}, mergeUniqueDisplayNames([]string{"a","b"}))
	require.Equal(t, []string{"a","b"}, mergeUniqueDisplayNames([]string{"a","b","a"}))
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

func TestDmPairIDs(t *testing.T) {
	a, b := dmPairIDs(5, 10)
	require.Equal(t, uint64(5), a)
	require.Equal(t, uint64(10), b)
	a, b = dmPairIDs(10, 5)
	require.Equal(t, uint64(5), a)
	require.Equal(t, uint64(10), b)
}

func TestDmPeerID(t *testing.T) {
	conv := &model.DmConversation{UserLow: 1, UserHigh: 2}
	require.Equal(t, uint64(2), dmPeerID(conv, 1))
	require.Equal(t, uint64(1), dmPeerID(conv, 2))
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
	srve(r, areq("GET", "/api/v1/users/me/favorites?folder_id=0", tk, nil))
	srve(r, areq("GET", "/api/v1/users/me/favorites?folder_id=abc", tk, nil))
}

func TestParseVideoFolderParams_NoParam(t *testing.T) {
	// Test via HTTP with invalid params
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "pvf1", "PVF1", 10)
	tk := tok(t, api, u.ID)
	// Missing video ID
	srve(r, areq("DELETE", "/api/v1/videos/0/favorite-folders/0", tk, nil))
}

func Test_UserDynamicPutAndPlayback(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "udp1", "UDP1", 10)
	tk := tok(t, api, u.ID)
	
	// Post a dynamic
	w := srve(r, areq("POST", "/api/v1/users/me/dynamics", tk, `{"title":"EditMe","content":"To be edited"}`))
	var dr struct { Code int `json:"code"`; Data struct { ID uint64 `json:"id"`} `json:"data"` }
	json.Unmarshal(w.Body.Bytes(), &dr)
	if dr.Code == 0 && dr.Data.ID > 0 {
		did := dr.Data.ID
		// Update dynamic
		srve(r, areq("PUT", fmt.Sprintf("/api/v1/users/me/dynamics/%d", did), tk, `{"title":"Edited","content":"Edited content"}`))
		// Patch playback
		srve(r, areq("PATCH", fmt.Sprintf("/api/v1/users/me/dynamics/%d/playback", did), tk, `{"current_time":20.0}`))
	}
	
	// List dynamics via space
	srve(r, areq("GET", fmt.Sprintf("/api/v1/space/%d/dynamics", u.ID), "", nil))
}

func Test_ArticleViewCount(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "avc1", "AVC1", 10)
	tk := tok(t, api, u.ID)
	art := seedArticle(t, api, u.ID, "View Count Test")
	
	// Post article view (increment count)
	srve(r, areq("POST", fmt.Sprintf("/api/v1/articles/%d/view", art.ID), tk, nil))
	
	// Get article detail
	srve(r, areq("GET", fmt.Sprintf("/api/v1/articles/%d", art.ID), "", nil))
	
	// List user published articles
	srve(r, areq("GET", fmt.Sprintf("/api/v1/space/%d/article-favorites", u.ID), "", nil))
}

func Test_ArticleCommentSubActions(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "acs1", "ACS1", 10)
	tk := tok(t, api, u.ID)
	art := seedArticle(t, api, u.ID, "AC Sub Article")
	
	// Post article comment with specific body
	w := srve(r, areq("POST", fmt.Sprintf("/api/v1/articles/%d/comments", art.ID), tk, 
		fmt.Sprintf(`{"content":"sub comment","article_id":%d}`, art.ID)))
	var cr struct { Code int `json:"code"`; Data struct { ID uint64 `json:"id"`} `json:"data"` }
	json.Unmarshal(w.Body.Bytes(), &cr)
	if cr.Code == 0 && cr.Data.ID > 0 {
		cid := cr.Data.ID
		// Like
		srve(r, areq("POST", fmt.Sprintf("/api/v1/article-comments/%d/like", cid), tk, nil))
		// Dislike
		srve(r, areq("POST", fmt.Sprintf("/api/v1/article-comments/%d/dislike", cid), tk, nil))
		// List article comments
		srve(r, areq("GET", fmt.Sprintf("/api/v1/articles/%d/comments", art.ID), "", nil))
	}
}
func Test_DmPostAndList(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "dpal1", "DPAL1", 10)
	u2 := seedUser(t, api, "dpal2", "DPAL2", 10)
	tk := tok(t, api, u.ID)
	
	// Create a conversation (should succeed)
	w := srve(r, areq("POST", "/api/v1/dm/conversations", tk, fmt.Sprintf(`{"user_id":%d}`, u2.ID)))
	var dcr struct { Code int `json:"code"`; Data struct { ID uint64 `json:"id"`} `json:"data"` }
	json.Unmarshal(w.Body.Bytes(), &dcr)
	if dcr.Code == 0 && dcr.Data.ID > 0 {
		cid := dcr.Data.ID
		// List conversations
		srve(r, areq("GET", "/api/v1/dm/conversations", tk, nil))
		// List messages
		srve(r, areq("GET", fmt.Sprintf("/api/v1/dm/conversations/%d/messages", cid), tk, nil))
		// Post a message
		srve(r, areq("POST", fmt.Sprintf("/api/v1/dm/conversations/%d/messages", cid), tk, `{"content":"Hello from test"}`))
	}
}
func Test_SpaceRecentCoinsAndFavorites(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "src1", "SRC1", 100)
	u2 := seedUser(t, api, "src2", "SRC2", 100)
	v := seedVideo(t, api, u2.ID, "Coin Video")
	
	// Post a coin from u to u2's video
	tk := tok(t, api, u.ID)
	srve(r, areq("POST", fmt.Sprintf("/api/v1/videos/%d/coin", v.ID), tk, `{"amount":1}`))
	
	// List recent coins
	srve(r, areq("GET", fmt.Sprintf("/api/v1/space/%d/recent-coins", u.ID), "", nil))
}

