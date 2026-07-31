//go:build integration

package handler

import (
	"encoding/json"
	"fmt"
	"testing"
)

func Test_NotificationReadNonExistent(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "nrn1", "NRN1", 10)
	tk := tok(t, api, u.ID)
	srve(r, areq("PATCH", "/api/v1/notifications/99999/read", tk, nil))
	srve(r, areq("DELETE", "/api/v1/notifications/99999", tk, nil))
	srve(r, areq("POST", "/api/v1/notifications/99999/mute-likes", tk, nil))
	srve(r, areq("POST", "/api/v1/notifications/99999/comment-like", tk, nil))
	srve(r, areq("POST", "/api/v1/notifications/99999/comment-reply", tk, `{"content":"Reply"}`))
	srve(r, areq("PATCH", "/api/v1/notifications/read-batch", tk, `{"ids":[99999,99998]}`))
	srve(r, areq("PATCH", "/api/v1/notifications/read-by-category", tk, `{"type":"nonexistent"}`))
	srve(r, areq("GET", "/api/v1/notifications/99999/like-likers", tk, nil))
}

func Test_CommentApproveByNonOwner(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "can1", "CAN1", 10)
	u2 := seedUser(t, api, "can2", "CAN2", 10)
	u3 := seedUser(t, api, "can3", "CAN3", 10)
	tk := tok(t, api, u2.ID)
	v := seedVideoWithAPI(t, api, u3.ID, "CAN Video")
	w := srve(r, areq("POST", "/api/v1/videos/"+fmt.Sprint(v.ID)+"/comments", tok(t, api, u.ID), `{"content":"Can I be approved?"}`))
	var cr struct {
		Code int `json:"code"`
		Data struct {
			ID uint64 `json:"id"`
		}
	}
	json.Unmarshal(w.Body.Bytes(), &cr)
	if cr.Code == 0 && cr.Data.ID > 0 {
		srve(r, areq("POST", fmt.Sprintf("/api/v1/comments/%d/approve", cr.Data.ID), tk, nil))
	}
}

func Test_DynamicCommentEdgeCases(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "dce1", "DCE1", 10)
	tk := tok(t, api, u.ID)
	srve(r, areq("POST", "/api/v1/user-dynamics/99999/comments/99999/approve", tk, nil))
	srve(r, areq("POST", "/api/v1/user-dynamics/99999/comments/99999/like", tk, nil))
	srve(r, areq("POST", "/api/v1/user-dynamics/99999/comments/99999/dislike", tk, nil))
	srve(r, areq("GET", "/api/v1/user-dynamics/99999/comments", "", nil))
}

func Test_ArticleEngagementFullFlow(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "aef2", "AEF2a", 100)
	u2 := seedUser(t, api, "aef3", "AEF3", 100)
	tk := tok(t, api, u.ID)
	art := seedArticle(t, api, u2.ID, "AEF2 Article")
	srve(r, areq("POST", fmt.Sprintf("/api/v1/articles/%d/view", art.ID), tk, nil))
	srve(r, areq("POST", fmt.Sprintf("/api/v1/articles/%d/favorite", art.ID), tk, nil))
	srve(r, areq("POST", fmt.Sprintf("/api/v1/articles/%d/coin", art.ID), tk, `{"amount":1}`))
	srve(r, areq("GET", "/api/v1/users/me/article-favorites", tk, nil))
	srve(r, areq("GET", fmt.Sprintf("/api/v1/space/%d/article-favorites", u2.ID), "", nil))
}

func Test_CreatorCommentsWithFilters(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "ccf1", "CCF1", 10)
	u2 := seedUser(t, api, "ccf2", "CCF2", 10)
	tk := tok(t, api, u.ID)
	v := seedVideoWithAPI(t, api, u.ID, "CCF Video")
	srve(r, areq("POST", "/api/v1/videos/"+fmt.Sprint(v.ID)+"/comments", tok(t, api, u2.ID), `{"content":"Filter test comment"}`))
	srve(r, areq("GET", "/api/v1/users/me/creator/comments?page=1&page_size=10", tk, nil))
	srve(r, areq("GET", "/api/v1/users/me/creator/comments?page=1&page_size=10&status=pending", tk, nil))
	srve(r, areq("GET", "/api/v1/users/me/creator/danmakus?page=1&page_size=10", tk, nil))
}

func Test_AccountDeletionNonExistent(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "ade1", "ADE1", 10)
	tk := tok(t, api, u.ID)
	srve(r, areq("GET", "/api/v1/users/me/delete-account", tk, nil))
	srve(r, areq("POST", "/api/v1/users/me/cancel-deletion", tk, nil))
	srve(r, areq("POST", "/api/v1/users/me/delete-account", tk, nil))
	srve(r, areq("POST", "/api/v1/users/me/delete-account", tk, nil))
	srve(r, areq("GET", "/api/v1/users/me/delete-account", tk, nil))
	srve(r, areq("POST", "/api/v1/users/me/cancel-deletion", tk, nil))
}

func Test_SearchHistoryClear(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "shc1", "SHC1", 10)
	tk := tok(t, api, u.ID)
	srve(r, areq("PUT", "/api/v1/users/me/search-history", tk, `{"content":"search term"}`))
	srve(r, areq("POST", "/api/v1/users/me/search-history", tk, `{"content":"another term"}`))
	srve(r, areq("GET", "/api/v1/users/me/search-history", tk, nil))
	srve(r, areq("DELETE", "/api/v1/users/me/search-history", tk, nil))
}

func Test_DeleteNonExistentEntities(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u := seedUser(t, api, "dne1", "DNE1", 10)
	tk := tok(t, api, u.ID)
	srve(r, areq("DELETE", "/api/v1/comments/99999", tk, nil))
	srve(r, areq("DELETE", "/api/v1/article-comments/99999", tk, nil))
	srve(r, areq("DELETE", "/api/v1/danmakus/99999", tk, nil))
	srve(r, areq("DELETE", "/api/v1/videos/99999", tk, nil))
}
