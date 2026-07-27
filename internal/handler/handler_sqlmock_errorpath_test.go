package handler

import (
	"net/http"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)


// SEARCH HISTORY
func TestGetMySearchHistory_Unauthorized(t *testing.T) {
	gormDB, _ := newMockGORM(t)
	api := newMockAPISimple(t, gormDB)
	c, w := newMockGinCtx(t, http.MethodGet, "/api/v1/users/me/search-history", nil)
	api.GetMySearchHistory(c)
	require.Equal(t, http.StatusUnauthorized, w.Code)
}


func TestPutMySearchHistory_Unauthorized(t *testing.T) {
	gormDB, _ := newMockGORM(t)
	api := newMockAPISimple(t, gormDB)
	c, w := newMockGinCtx(t, http.MethodPut, "/api/v1/users/me/search-history", nil)
	api.PutMySearchHistory(c)
	require.Equal(t, http.StatusUnauthorized, w.Code)
}


// SEARCH HISTORY (continued)
func TestPutMySearchHistory_BadJSON(t *testing.T) {
	gormDB, _ := newMockGORM(t)
	api := newMockAPISimple(t, gormDB)
	c, w := newMockGinCtx(t, http.MethodPut, "/api/v1/users/me/search-history", []byte("not json"))
	c.Set("user_id", uint64(1))
	api.PutMySearchHistory(c)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPostMySearchHistory_Unauthorized(t *testing.T) {
	gormDB, _ := newMockGORM(t)
	api := newMockAPISimple(t, gormDB)
	c, w := newMockGinCtx(t, http.MethodPost, "/api/v1/users/me/search-history", nil)
	api.PostMySearchHistory(c)
	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestPostMySearchHistory_BadJSON(t *testing.T) {
	gormDB, _ := newMockGORM(t)
	api := newMockAPISimple(t, gormDB)
	c, w := newMockGinCtx(t, http.MethodPost, "/api/v1/users/me/search-history", []byte("not json"))
	c.Set("user_id", uint64(1))
	api.PostMySearchHistory(c)
	require.Equal(t, http.StatusBadRequest, w.Code)
}


// DM
func TestListDmConversations_Unauthorized(t *testing.T) {
	gormDB, _ := newMockGORM(t)
	api := newMockAPISimple(t, gormDB)
	c, w := newMockGinCtx(t, http.MethodGet, "/api/v1/dm/conversations", nil)
	api.ListDmConversations(c)
	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestCreateDmConversation_Unauthorized(t *testing.T) {
	gormDB, _ := newMockGORM(t)
	api := newMockAPISimple(t, gormDB)
	c, w := newMockGinCtx(t, http.MethodPost, "/api/v1/dm/conversations", nil)
	api.CreateDmConversation(c)
	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestCreateDmConversation_BadJSON(t *testing.T) {
	gormDB, _ := newMockGORM(t)
	api := newMockAPISimple(t, gormDB)
	c, w := newMockGinCtx(t, http.MethodPost, "/api/v1/dm/conversations", []byte("not json"))
	c.Set("user_id", uint64(1))
	api.CreateDmConversation(c)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDeleteDmConversation_Unauthorized(t *testing.T) {
	gormDB, _ := newMockGORM(t)
	api := newMockAPISimple(t, gormDB)
	c, w := newMockGinCtx(t, http.MethodDelete, "/api/v1/dm/conversations/1", nil)
	api.DeleteDmConversation(c)
	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestDeleteDmConversation_BadParam(t *testing.T) {
	gormDB, _ := newMockGORM(t)
	api := newMockAPISimple(t, gormDB)
	c, w := newMockGinCtx(t, http.MethodDelete, "/api/v1/dm/conversations/abc", nil)
	c.Set("user_id", uint64(1))
	c.Params = gin.Params{{Key: "id", Value: "abc"}}
	api.DeleteDmConversation(c)
	require.Equal(t, http.StatusBadRequest, w.Code)
}


func TestListDmMessages_Unauthorized(t *testing.T) {
	gormDB, _ := newMockGORM(t)
	api := newMockAPISimple(t, gormDB)
	c, w := newMockGinCtx(t, http.MethodGet, "/api/v1/dm/conversations/1/messages", nil)
	api.ListDmMessages(c)
	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestPostDmMessage_Unauthorized(t *testing.T) {
	gormDB, _ := newMockGORM(t)
	api := newMockAPISimple(t, gormDB)
	c, w := newMockGinCtx(t, http.MethodPost, "/api/v1/dm/conversations/1/messages", nil)
	api.PostDmMessage(c)
	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestPostDmMessage_BadJSON(t *testing.T) {
	gormDB, _ := newMockGORM(t)
	api := newMockAPISimple(t, gormDB)
	c, w := newMockGinCtx(t, http.MethodPost, "/api/v1/dm/conversations/1/messages", []byte("not json"))
	c.Set("user_id", uint64(1))
	c.Params = gin.Params{{Key: "id", Value: "1"}}
	api.PostDmMessage(c)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPatchDmConversationSettings_Unauthorized(t *testing.T) {
	gormDB, _ := newMockGORM(t)
	api := newMockAPISimple(t, gormDB)
	c, w := newMockGinCtx(t, http.MethodPatch, "/api/v1/dm/conversations/1/settings", nil)
	c.Params = gin.Params{{Key: "id", Value: "1"}}
	api.PatchDmConversationSettings(c)
	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestResetDmAgentConversation_Unauthorized(t *testing.T) {
	gormDB, _ := newMockGORM(t)
	api := newMockAPISimple(t, gormDB)
	c, w := newMockGinCtx(t, http.MethodPost, "/api/v1/dm/conversations/1/reset-agent", nil)
	c.Params = gin.Params{{Key: "id", Value: "1"}}
	api.ResetDmAgentConversation(c)
	require.Equal(t, http.StatusUnauthorized, w.Code)
}


// VIEW HISTORY
func TestPostVideoViewHistory_Unauthorized(t *testing.T) {
	gormDB, _ := newMockGORM(t)
	api := newMockAPISimple(t, gormDB)
	c, w := newMockGinCtx(t, http.MethodPost, "/api/v1/videos/1/view-history", nil)
	c.Params = gin.Params{{Key: "id", Value: "1"}}
	api.PostVideoViewHistory(c)
	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestPostVideoViewHistory_BadParam(t *testing.T) {
	gormDB, mock := newMockGORM(t)
	api := newMockAPISimple(t, gormDB)
	mock.ExpectQuery("SELECT `id`,`view_history_paused` FROM `users` WHERE").
		WillReturnRows(sqlmock.NewRows([]string{"id", "view_history_paused"}).AddRow(1, false))
	c, w := newMockGinCtx(t, http.MethodPost, "/api/v1/videos/abc/view-history", nil)
	c.Set("user_id", uint64(1))
	c.Params = gin.Params{{Key: "id", Value: "abc"}}
	api.PostVideoViewHistory(c)
	require.Equal(t, http.StatusBadRequest, w.Code)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListMyViewHistory_Unauthorized(t *testing.T) {
	gormDB, _ := newMockGORM(t)
	api := newMockAPISimple(t, gormDB)
	c, w := newMockGinCtx(t, http.MethodGet, "/api/v1/users/me/history", nil)
	api.ListMyViewHistory(c)
	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestDeleteMyViewHistoryEntry_Unauthorized(t *testing.T) {
	gormDB, _ := newMockGORM(t)
	api := newMockAPISimple(t, gormDB)
	c, w := newMockGinCtx(t, http.MethodDelete, "/api/v1/users/me/history/1", nil)
	api.DeleteMyViewHistoryEntry(c)
	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestDeleteMyViewHistoryEntry_BadParam(t *testing.T) {
	gormDB, _ := newMockGORM(t)
	api := newMockAPISimple(t, gormDB)
	c, w := newMockGinCtx(t, http.MethodDelete, "/api/v1/users/me/history/abc", nil)
	c.Set("user_id", uint64(1))
	c.Params = gin.Params{{Key: "id", Value: "abc"}}
	api.DeleteMyViewHistoryEntry(c)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestClearMyViewHistory_Unauthorized(t *testing.T) {
	gormDB, _ := newMockGORM(t)
	api := newMockAPISimple(t, gormDB)
	c, w := newMockGinCtx(t, http.MethodDelete, "/api/v1/users/me/history", nil)
	api.ClearMyViewHistory(c)
	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestPutMyViewHistorySettings_Unauthorized(t *testing.T) {
	gormDB, _ := newMockGORM(t)
	api := newMockAPISimple(t, gormDB)
	c, w := newMockGinCtx(t, http.MethodPut, "/api/v1/users/me/history/settings", nil)
	api.PutMyViewHistorySettings(c)
	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestPutMyViewHistorySettings_BadJSON(t *testing.T) {
	gormDB, _ := newMockGORM(t)
	api := newMockAPISimple(t, gormDB)
	c, w := newMockGinCtx(t, http.MethodPut, "/api/v1/users/me/history/settings", []byte("not json"))
	c.Set("user_id", uint64(1))
	api.PutMyViewHistorySettings(c)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

// ARTICLE ENGAGEMENT
func TestToggleArticleFavorite_Unauthorized(t *testing.T) {
	gormDB, _ := newMockGORM(t)
	api := newMockAPISimple(t, gormDB)
	c, w := newMockGinCtx(t, http.MethodPost, "/api/v1/articles/1/favorite", nil)
	c.Params = gin.Params{{Key: "id", Value: "1"}}
	api.ToggleArticleFavorite(c)
	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestToggleArticleFavorite_BadParam(t *testing.T) {
	gormDB, _ := newMockGORM(t)
	api := newMockAPISimple(t, gormDB)
	c, w := newMockGinCtx(t, http.MethodPost, "/api/v1/articles/abc/favorite", nil)
	c.Set("user_id", uint64(1))
	c.Params = gin.Params{{Key: "id", Value: "abc"}}
	api.ToggleArticleFavorite(c)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPostArticleCoin_Unauthorized(t *testing.T) {
	gormDB, _ := newMockGORM(t)
	api := newMockAPISimple(t, gormDB)
	c, w := newMockGinCtx(t, http.MethodPost, "/api/v1/articles/1/coin", nil)
	c.Params = gin.Params{{Key: "id", Value: "1"}}
	api.PostArticleCoin(c)
	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestPostArticleCoin_BadParam(t *testing.T) {
	gormDB, _ := newMockGORM(t)
	api := newMockAPISimple(t, gormDB)
	c, w := newMockGinCtx(t, http.MethodPost, "/api/v1/articles/abc/coin", nil)
	c.Set("user_id", uint64(1))
	c.Params = gin.Params{{Key: "id", Value: "abc"}}
	api.PostArticleCoin(c)
	require.Equal(t, http.StatusBadRequest, w.Code)
}


// USER DYNAMIC
func TestGetUserDynamic_NotFound(t *testing.T) {
	gormDB, mock := newMockGORM(t)
	api := newMockAPISimple(t, gormDB)
	mock.ExpectQuery("SELECT .+ FROM `user_dynamics` WHERE").
		WillReturnRows(sqlmock.NewRows([]string{"id"}))
	c, w := newMockGinCtx(t, http.MethodGet, "/api/v1/dynamics/1", nil)
	c.Params = gin.Params{{Key: "id", Value: "1"}}
	api.GetUserDynamic(c)
	require.Equal(t, http.StatusNotFound, w.Code)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestGetUserDynamic_BadParam(t *testing.T) {
	gormDB, _ := newMockGORM(t)
	api := newMockAPISimple(t, gormDB)
	c, w := newMockGinCtx(t, http.MethodGet, "/api/v1/dynamics/abc", nil)
	c.Set("user_id", uint64(1))
	c.Params = gin.Params{{Key: "id", Value: "abc"}}
	api.GetUserDynamic(c)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPatchUserDynamicPlayback_Unauthorized(t *testing.T) {
	gormDB, _ := newMockGORM(t)
	api := newMockAPISimple(t, gormDB)
	c, w := newMockGinCtx(t, http.MethodPatch, "/api/v1/dynamics/1/playback", nil)
	c.Params = gin.Params{{Key: "id", Value: "1"}}
	api.PatchUserDynamicPlayback(c)
	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestPatchUserDynamicPlayback_BadParam(t *testing.T) {
	gormDB, _ := newMockGORM(t)
	api := newMockAPISimple(t, gormDB)
	c, w := newMockGinCtx(t, http.MethodPatch, "/api/v1/dynamics/abc/playback", nil)
	c.Set("user_id", uint64(1))
	c.Params = gin.Params{{Key: "id", Value: "abc"}}
	api.PatchUserDynamicPlayback(c)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestToggleDynamicLike_Unauthorized(t *testing.T) {
	gormDB, _ := newMockGORM(t)
	api := newMockAPISimple(t, gormDB)
	c, w := newMockGinCtx(t, http.MethodPost, "/api/v1/dynamics/1/like", nil)
	c.Params = gin.Params{{Key: "id", Value: "1"}}
	api.ToggleDynamicLike(c)
	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestToggleDynamicLike_BadParam(t *testing.T) {
	gormDB, _ := newMockGORM(t)
	api := newMockAPISimple(t, gormDB)
	c, w := newMockGinCtx(t, http.MethodPost, "/api/v1/dynamics/abc/like", nil)
	c.Set("user_id", uint64(1))
	c.Params = gin.Params{{Key: "id", Value: "abc"}}
	api.ToggleDynamicLike(c)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDeleteMyDynamic_Unauthorized(t *testing.T) {
	gormDB, _ := newMockGORM(t)
	api := newMockAPISimple(t, gormDB)
	c, w := newMockGinCtx(t, http.MethodDelete, "/api/v1/dynamics/1", nil)
	c.Params = gin.Params{{Key: "id", Value: "1"}}
	api.DeleteMyDynamic(c)
	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestDeleteMyDynamic_BadParam(t *testing.T) {
	gormDB, _ := newMockGORM(t)
	api := newMockAPISimple(t, gormDB)
	c, w := newMockGinCtx(t, http.MethodDelete, "/api/v1/dynamics/abc", nil)
	c.Set("user_id", uint64(1))
	c.Params = gin.Params{{Key: "id", Value: "abc"}}
	api.DeleteMyDynamic(c)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

// FAVORITE FOLDER
func TestCreateFavoriteFolder_Unauthorized(t *testing.T) {
	gormDB, _ := newMockGORM(t)
	api := newMockAPISimple(t, gormDB)
	c, w := newMockGinCtx(t, http.MethodPost, "/api/v1/favorite-folders", nil)
	api.CreateFavoriteFolder(c)
	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestCreateFavoriteFolder_BadJSON(t *testing.T) {
	gormDB, _ := newMockGORM(t)
	api := newMockAPISimple(t, gormDB)
	c, w := newMockGinCtx(t, http.MethodPost, "/api/v1/favorite-folders", []byte("not json"))
	c.Set("user_id", uint64(1))
	api.CreateFavoriteFolder(c)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDeleteFavoriteFolder_Unauthorized(t *testing.T) {
	gormDB, _ := newMockGORM(t)
	api := newMockAPISimple(t, gormDB)
	c, w := newMockGinCtx(t, http.MethodDelete, "/api/v1/favorite-folders/1", nil)
	api.DeleteFavoriteFolder(c)
	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestDeleteFavoriteFolder_BadParam(t *testing.T) {
	gormDB, _ := newMockGORM(t)
	api := newMockAPISimple(t, gormDB)
	c, w := newMockGinCtx(t, http.MethodDelete, "/api/v1/favorite-folders/abc", nil)
	c.Set("user_id", uint64(1))
	c.Params = gin.Params{{Key: "id", Value: "abc"}}
	api.DeleteFavoriteFolder(c)
	require.Equal(t, http.StatusBadRequest, w.Code)
}


// VIDEO ENGAGEMENT
func TestToggleVideoFavorite_Unauthorized(t *testing.T) {
	gormDB, _ := newMockGORM(t)
	api := newMockAPISimple(t, gormDB)
	c, w := newMockGinCtx(t, http.MethodPost, "/api/v1/videos/1/favorite", nil)
	c.Params = gin.Params{{Key: "id", Value: "1"}}
	api.ToggleVideoFavorite(c)
	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestToggleVideoFavorite_BadParam(t *testing.T) {
	gormDB, _ := newMockGORM(t)
	api := newMockAPISimple(t, gormDB)
	c, w := newMockGinCtx(t, http.MethodPost, "/api/v1/videos/abc/favorite", nil)
	c.Set("user_id", uint64(1))
	c.Params = gin.Params{{Key: "id", Value: "abc"}}
	api.ToggleVideoFavorite(c)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPostVideoCoin_Unauthorized(t *testing.T) {
	gormDB, _ := newMockGORM(t)
	api := newMockAPISimple(t, gormDB)
	c, w := newMockGinCtx(t, http.MethodPost, "/api/v1/videos/1/coin", nil)
	c.Params = gin.Params{{Key: "id", Value: "1"}}
	api.PostVideoCoin(c)
	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestPostVideoCoin_BadParam(t *testing.T) {
	gormDB, _ := newMockGORM(t)
	api := newMockAPISimple(t, gormDB)
	c, w := newMockGinCtx(t, http.MethodPost, "/api/v1/videos/abc/coin", nil)
	c.Set("user_id", uint64(1))
	c.Params = gin.Params{{Key: "id", Value: "abc"}}
	api.PostVideoCoin(c)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestToggleWatchLater_Unauthorized(t *testing.T) {
	gormDB, _ := newMockGORM(t)
	api := newMockAPISimple(t, gormDB)
	c, w := newMockGinCtx(t, http.MethodPost, "/api/v1/videos/1/watch-later", nil)
	c.Params = gin.Params{{Key: "id", Value: "1"}}
	api.ToggleWatchLater(c)
	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestToggleWatchLater_BadParam(t *testing.T) {
	gormDB, _ := newMockGORM(t)
	api := newMockAPISimple(t, gormDB)
	c, w := newMockGinCtx(t, http.MethodPost, "/api/v1/videos/abc/watch-later", nil)
	c.Set("user_id", uint64(1))
	c.Params = gin.Params{{Key: "id", Value: "abc"}}
	api.ToggleWatchLater(c)
	require.Equal(t, http.StatusBadRequest, w.Code)
}


// ARTICLE COMMENT
func TestPostArticleComment_Unauthorized(t *testing.T) {
	gormDB, _ := newMockGORM(t)
	api := newMockAPISimple(t, gormDB)
	c, w := newMockGinCtx(t, http.MethodPost, "/api/v1/articles/1/comments", nil)
	c.Params = gin.Params{{Key: "id", Value: "1"}}
	api.PostArticleComment(c)
	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestPostArticleComment_BadParam(t *testing.T) {
	gormDB, _ := newMockGORM(t)
	api := newMockAPISimple(t, gormDB)
	c, w := newMockGinCtx(t, http.MethodPost, "/api/v1/articles/abc/comments", nil)
	c.Set("user_id", uint64(1))
	c.Params = gin.Params{{Key: "id", Value: "abc"}}
	api.PostArticleComment(c)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPostArticleComment_BadJSON(t *testing.T) {
	gormDB, mock := newMockGORM(t)
	api := newMockAPISimple(t, gormDB)
	mock.ExpectQuery("SELECT .+ FROM `articles` WHERE").
		WillReturnRows(sqlmock.NewRows([]string{"id", "author_id", "status", "comments_closed"}).AddRow(1, 1, "published", false))
	c, w := newMockGinCtx(t, http.MethodPost, "/api/v1/articles/1/comments", []byte("not json"))
	c.Set("user_id", uint64(1))
	c.Params = gin.Params{{Key: "id", Value: "1"}}
	api.PostArticleComment(c)
	require.Equal(t, http.StatusBadRequest, w.Code)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDeleteArticleComment_Unauthorized(t *testing.T) {
	gormDB, _ := newMockGORM(t)
	api := newMockAPISimple(t, gormDB)
	c, w := newMockGinCtx(t, http.MethodDelete, "/api/v1/articles/comments/1", nil)
	c.Params = gin.Params{{Key: "id", Value: "1"}}
	api.DeleteArticleComment(c)
	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestDeleteArticleComment_BadParam(t *testing.T) {
	gormDB, _ := newMockGORM(t)
	api := newMockAPISimple(t, gormDB)
	c, w := newMockGinCtx(t, http.MethodDelete, "/api/v1/articles/comments/abc", nil)
	c.Set("user_id", uint64(1))
	c.Params = gin.Params{{Key: "id", Value: "abc"}}
	api.DeleteArticleComment(c)
	require.Equal(t, http.StatusBadRequest, w.Code)
}
