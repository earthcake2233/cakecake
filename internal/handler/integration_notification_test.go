//go:build integration

package handler

import (
	"cakecake/internal/model/comment"
	"cakecake/internal/model/notification"
	"cakecake/internal/model/user"
	"cakecake/internal/model/video"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNotification_Endpoints(t *testing.T) {
	api, r, token := setupHandlerIntegrationDB(t)
	require.NoError(t, api.DB.Create(&user.User{ID: 1, Username: "u1", PasswordHash: "x", CoinBalanceTenths: 230}).Error)
	require.NoError(t, api.DB.Create(&notification.Notification{RecipientID: 1, Type: "reply", RelatedID: 6}).Error)
	require.NoError(t, api.DB.Create(&notification.Notification{
		RecipientID: 1, Type: "like_aggregation", RelatedID: 6, PayloadJSON: "like_comment:6",
	}).Error)
	require.NoError(t, api.DB.Create(&video.Video{ID: 10, UserID: 2, Title: "v", Status: video.StatusPublished}).Error)
	require.NoError(t, api.DB.Create(&comment.Comment{ID: 6, VideoID: 10, UserID: 2, Content: "c"}).Error)

	// Unread summary.
	w := doReq(r, "GET", "/api/v1/notifications/unread-summary", token, "", nil)
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())

	// List.
	w = doReq(r, "GET", "/api/v1/notifications?category=reply&page=1&page_size=10", token, "", nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var lr struct {
		Data struct {
			Items []struct {
				ID uint64 `json:"id"`
			} `json:"items"`
		} `json:"data"`
	}
	decodeBody(t, w, &lr)
	require.NotEmpty(t, lr.Data.Items)
	replyID := lr.Data.Items[0].ID

	// Mark single read.
	w = doJSON(r, "PATCH", fmt.Sprintf("/api/v1/notifications/%d/read", replyID), token, nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	// Mark batch read.
	w = doJSON(r, "PATCH", "/api/v1/notifications/read-batch", token, []uint64{replyID})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	// Bad batch body.
	w = doJSON(r, "PATCH", "/api/v1/notifications/read-batch", token, "not-an-array")
	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())

	// Mark category read.
	w = doJSON(r, "PATCH", "/api/v1/notifications/read-by-category?category=like", token, nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	// Like likers.
	w = doReq(r, "GET", "/api/v1/notifications/2/like-likers", token, "", nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	// Mute likes.
	w = doJSON(r, "POST", "/api/v1/notifications/2/mute-likes", token, nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	// Toggle comment like via notification.
	w = doJSON(r, "POST", "/api/v1/notifications/2/comment-like", token, nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	// Post reply via notification.
	w = doJSON(r, "POST", "/api/v1/notifications/1/comment-reply", token, map[string]interface{}{"content": "thanks"})
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())

	// Delete notification.
	w = doReq(r, "DELETE", fmt.Sprintf("/api/v1/notifications/%d", replyID), token, "", nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
}
