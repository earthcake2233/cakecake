//go:build integration

package handler

import (
	"cakecake/internal/model/article"
	"cakecake/internal/model/comment"
	"cakecake/internal/model/danmaku"
	"cakecake/internal/model/dynamic"
	"cakecake/internal/model/user"
	"cakecake/internal/model/video"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCreatorDanmaku_Endpoints(t *testing.T) {
	api, r, token := setupHandlerIntegrationDB(t)
	require.NoError(t, api.DB.Create(&user.User{ID: 1, Username: "u1", PasswordHash: "x", CoinBalanceTenths: 230}).Error)
	require.NoError(t, api.DB.Create(&user.User{ID: 2, Username: "u2", PasswordHash: "x", CoinBalanceTenths: 230}).Error)
	require.NoError(t, api.DB.Create(&video.Video{ID: 10, UserID: 1, Title: "v", Status: video.StatusPublished}).Error)
	d := danmaku.Danmaku{VideoID: 10, UserID: 2, Content: "hello", Type: "top", VideoTime: 5.0}
	require.NoError(t, api.DB.Create(&d).Error)

	// List creator danmakus.
	w := doReq(r, "GET", "/api/v1/users/me/creator/danmakus?limit=10&q=hello&type=top&video_id=10", token, "", nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	// Bad video_id.
	w = doReq(r, "GET", "/api/v1/users/me/creator/danmakus?video_id=abc", token, "", nil)
	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())

	// Toggle like.
	w = doJSON(r, "POST", fmt.Sprintf("/api/v1/danmakus/%d/like", d.ID), token, nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	// Missing danmaku.
	w = doJSON(r, "POST", "/api/v1/danmakus/999/like", token, nil)
	require.Equal(t, http.StatusNotFound, w.Code, w.Body.String())

	// Delete danmaku (video owner).
	w = doReq(r, "DELETE", fmt.Sprintf("/api/v1/danmakus/%d", d.ID), token, "", nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
}

func TestCreatorComments_ArticleAndDynamic(t *testing.T) {
	api, r, token := setupHandlerIntegrationDB(t)
	require.NoError(t, api.DB.Create(&user.User{ID: 1, Username: "u1", PasswordHash: "x", CoinBalanceTenths: 230}).Error)
	require.NoError(t, api.DB.Create(&user.User{ID: 2, Username: "u2", PasswordHash: "x", CoinBalanceTenths: 230}).Error)
	require.NoError(t, api.DB.Create(&article.Article{ID: 20, UserID: 1, Title: "a", BodyMD: "b", Status: article.StatusPublished}).Error)
	require.NoError(t, api.DB.Create(&dynamic.UserDynamic{ID: 30, UserID: 1, Title: "d"}).Error)
	require.NoError(t, api.DB.Create(&comment.ArticleComment{ArticleID: 20, UserID: 2, Content: "ac", Approved: true}).Error)
	require.NoError(t, api.DB.Create(&comment.DynamicComment{DynamicID: 30, UserID: 2, Content: "dc", Approved: true}).Error)

	w := doReq(r, "GET", "/api/v1/users/me/creator/comments?media=article&page=1&page_size=10", token, "", nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.Contains(t, w.Body.String(), "ac")

	w = doReq(r, "GET", "/api/v1/users/me/creator/comments?media=dynamic&page=1&page_size=10", token, "", nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.Contains(t, w.Body.String(), "dc")

	// Pending filters + keyword + sort.
	w = doReq(r, "GET", "/api/v1/users/me/creator/comments?media=article&pending=1&sort=likes&q=ac", token, "", nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	// Bad video_id on the video branch.
	w = doReq(r, "GET", "/api/v1/users/me/creator/comments?video_id=abc", token, "", nil)
	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
}
