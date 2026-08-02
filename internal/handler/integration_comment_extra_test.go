//go:build integration

package handler

import (
	"cakecake/internal/model/comment"
	"cakecake/internal/model/user"
	"cakecake/internal/model/video"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestComment_Endpoints(t *testing.T) {
	api, r, token := setupHandlerIntegrationDB(t)
	require.NoError(t, api.DB.Create(&user.User{ID: 1, Username: "u1", PasswordHash: "x", CoinBalanceTenths: 230}).Error)
	require.NoError(t, api.DB.Create(&video.Video{ID: 10, UserID: 1, Title: "v", Status: video.StatusPublished}).Error)

	// Post comment.
	w := doJSON(r, "POST", "/api/v1/videos/10/comments", token, map[string]interface{}{
		"content": "nice video", "parent_id": 0,
	})
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	var cr struct {
		Code int `json:"code"`
		Data struct {
			ID uint64 `json:"id"`
		} `json:"data"`
	}
	decodeBody(t, w, &cr)
	require.NotZero(t, cr.Data.ID)
	cmID := cr.Data.ID

	// List comments.
	w = doReq(r, "GET", "/api/v1/videos/10/comments", token, "", nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	// Like + dislike.
	w = doJSON(r, "POST", fmt.Sprintf("/api/v1/comments/%d/like", cmID), token, nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	w = doJSON(r, "POST", fmt.Sprintf("/api/v1/comments/%d/dislike", cmID), token, nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	// Pin (uploader of video 10 is user 2, but pin endpoint only checks target exists).
	w = doJSON(r, "POST", fmt.Sprintf("/api/v1/comments/%d/pin", cmID), token, nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	// Approve + ignore.
	w = doJSON(r, "POST", fmt.Sprintf("/api/v1/comments/%d/approve", cmID), token, nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	w = doJSON(r, "POST", fmt.Sprintf("/api/v1/comments/%d/ignore-curated", cmID), token, nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	// Creator comment list.
	w = doReq(r, "GET", "/api/v1/users/me/creator/comments?page=1&page_size=10", token, "", nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	// Delete comment (owner).
	w = doReq(r, "DELETE", fmt.Sprintf("/api/v1/comments/%d", cmID), token, "", nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.Error(t, api.DB.First(&comment.Comment{}, cmID).Error)

	// Missing video comment post.
	w = doJSON(r, "POST", "/api/v1/videos/999/comments", token, map[string]interface{}{"content": "x"})
	require.Equal(t, http.StatusNotFound, w.Code, w.Body.String())

	// Empty content.
	w = doJSON(r, "POST", "/api/v1/videos/10/comments", token, map[string]interface{}{"content": ""})
	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
}
