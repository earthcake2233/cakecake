//go:build integration

package handler

import (
	"cakecake/internal/model/article"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAdminArticle_Endpoints(t *testing.T) {
	api, r, jm := newTestAPI(t)
	access, _, _, err := jm.IssueAdminPair(1)
	require.NoError(t, err)
	require.NoError(t, api.DB.Create(&article.Article{
		ID: 10, UserID: 1, Title: "review me", BodyMD: "b", Status: article.StatusPendingReview,
	}).Error)
	require.NoError(t, api.DB.Create(&article.Article{
		ID: 11, UserID: 1, Title: "published", BodyMD: "b", Status: article.StatusPublished,
	}).Error)

	// List with filter.
	w := doReq(r, "GET", "/api/v1/admin/articles?status=pending_review&q=review", access, "", nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	w = doReq(r, "GET", "/api/v1/admin/articles", access, "", nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	// Get one.
	w = doReq(r, "GET", "/api/v1/admin/articles/10", access, "", nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	// Reject pending_review.
	w = doJSON(r, "POST", "/api/v1/admin/articles/10/reject", access, map[string]interface{}{"reason": "no"})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var a article.Article
	require.NoError(t, api.DB.First(&a, 10).Error)
	require.Equal(t, article.StatusRejected, a.Status)

	// Approve requires pending_review: published article -> bad request.
	w = doJSON(r, "POST", "/api/v1/admin/articles/11/approve", access, nil)
	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())

	// Approve a pending_review article.
	require.NoError(t, api.DB.Create(&article.Article{
		ID: 12, UserID: 1, Title: "approve me", BodyMD: "b", Status: article.StatusPendingReview,
	}).Error)
	w = doJSON(r, "POST", "/api/v1/admin/articles/12/approve", access, nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	a = article.Article{}
	require.NoError(t, api.DB.First(&a, 12).Error)
	require.Equal(t, article.StatusPublished, a.Status)

	// Delete via POST and DELETE.
	w = doJSON(r, "POST", "/api/v1/admin/articles/12/delete", access, nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.Error(t, api.DB.First(&article.Article{}, 12).Error)
	w = doReq(r, "DELETE", "/api/v1/admin/articles/11", access, "", nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
}
