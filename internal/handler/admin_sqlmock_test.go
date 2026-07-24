package handler

import (
	"net/http"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// -- AdminListVideos --

func TestAdminListVideos_Success(t *testing.T) {
	gormDB, mock := newMockGORM(t)
	api := newMockAPISimple(t, gormDB)
	c, w := newMockGinCtx(t, "GET", "/api/v1/admin/videos?page=1&page_size=10&status=pending_review", nil)

	mock.ExpectQuery("SELECT count(.+) FROM `videos` WHERE status IN").
		WillReturnRows(sqlmock.NewRows([]string{"count(*)"}).AddRow(2))

	mock.ExpectQuery("SELECT .+ FROM `videos` WHERE status IN .+ ORDER BY").
		WillReturnRows(sqlmock.NewRows([]string{"id","user_id","title","status","fail_reason","cover_url","video_url","duration_sec","zone","play_count"}).
			AddRow(1, 10, "Video1", "pending_review", "", "", "", 120, "tech", 0).
			AddRow(2, 11, "Video2", "pending_review", "", "", "", 60, "music", 0))

	mock.ExpectQuery("SELECT .+ FROM `users` WHERE id IN").
		WillReturnRows(sqlmock.NewRows([]string{"id","username","coin_balance_tenths"}).
			AddRow(10, "user10", 230).AddRow(11, "user11", 230))

	mock.ExpectQuery("SELECT count(.+) FROM `videos` WHERE status =").
		WillReturnRows(sqlmock.NewRows([]string{"count(*)"}).AddRow(2))

	api.AdminListVideos(c)
	require.Equal(t, http.StatusOK, w.Code)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAdminListVideos_DBError(t *testing.T) {
	gormDB, mock := newMockGORM(t)
	api := newMockAPISimple(t, gormDB)
	c, w := newMockGinCtx(t, "GET", "/api/v1/admin/videos", nil)

	mock.ExpectQuery("SELECT count(.+) FROM `videos`").WillReturnError(sqlmock.ErrCancelled)

	api.AdminListVideos(c)
	require.Equal(t, http.StatusInternalServerError, w.Code)
	require.NoError(t, mock.ExpectationsWereMet())
}

// -- AdminGetVideo --

func TestAdminGetVideo_Success(t *testing.T) {
	gormDB, mock := newMockGORM(t)
	api := newMockAPISimple(t, gormDB)
	c, w := newMockGinCtx(t, "GET", "/api/v1/admin/videos/1", nil)
	c.Params = gin.Params{{Key: "id", Value: "1"}}

	mock.ExpectQuery("SELECT .+ FROM `videos` WHERE").
		WillReturnRows(sqlmock.NewRows([]string{"id","user_id","title","status","fail_reason","cover_url","video_url","duration_sec","zone","play_count"}).
			AddRow(1, 10, "TestVideo", "pending_review", "", "", "", 120, "tech", 0))

	mock.ExpectQuery("SELECT .+ FROM `users` WHERE").
		WillReturnRows(sqlmock.NewRows([]string{"id","username","coin_balance_tenths"}).
			AddRow(10, "uploader", 230))

	api.AdminGetVideo(c)
	require.Equal(t, http.StatusOK, w.Code)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAdminGetVideo_NotFound(t *testing.T) {
	gormDB, mock := newMockGORM(t)
	api := newMockAPISimple(t, gormDB)
	c, w := newMockGinCtx(t, "GET", "/api/v1/admin/videos/999", nil)
	c.Params = gin.Params{{Key: "id", Value: "999"}}

	mock.ExpectQuery("SELECT .+ FROM `videos` WHERE").
		WillReturnRows(sqlmock.NewRows([]string{"id"}))

	api.AdminGetVideo(c)
	require.Equal(t, http.StatusNotFound, w.Code)
	require.NoError(t, mock.ExpectationsWereMet())
}

// -- AdminListArticles --

func TestAdminListArticles_Success(t *testing.T) {
	gormDB, mock := newMockGORM(t)
	api := newMockAPISimple(t, gormDB)
	c, w := newMockGinCtx(t, "GET", "/api/v1/admin/articles?status=pending_review", nil)

	mock.ExpectQuery("SELECT count(.+) FROM `articles` WHERE status IN").
		WillReturnRows(sqlmock.NewRows([]string{"count(*)"}).AddRow(1))

	mock.ExpectQuery("SELECT .+ FROM `articles` WHERE status IN .+ ORDER BY").
		WillReturnRows(sqlmock.NewRows([]string{"id","user_id","title","status","body_md","cover_url","view_count","comment_count"}).
			AddRow(1, 10, "Article1", "pending_review", "# Hello", "", 0, 0))

	mock.ExpectQuery("SELECT .+ FROM `users` WHERE id IN").
		WillReturnRows(sqlmock.NewRows([]string{"id","username","coin_balance_tenths"}).
			AddRow(10, "author1", 230))

	mock.ExpectQuery("SELECT count(.+) FROM `articles` WHERE status =").
		WillReturnRows(sqlmock.NewRows([]string{"count(*)"}).AddRow(1))

	api.AdminListArticles(c)
	require.Equal(t, http.StatusOK, w.Code)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAdminListArticles_DBError(t *testing.T) {
	gormDB, mock := newMockGORM(t)
	api := newMockAPISimple(t, gormDB)
	c, w := newMockGinCtx(t, "GET", "/api/v1/admin/articles", nil)

	mock.ExpectQuery("SELECT count(.+) FROM `articles`").WillReturnError(sqlmock.ErrCancelled)

	api.AdminListArticles(c)
	require.Equal(t, http.StatusInternalServerError, w.Code)
	require.NoError(t, mock.ExpectationsWereMet())
}

// -- AdminGetArticle --

func TestAdminGetArticle_Success(t *testing.T) {
	gormDB, mock := newMockGORM(t)
	api := newMockAPISimple(t, gormDB)
	c, w := newMockGinCtx(t, "GET", "/api/v1/admin/articles/1", nil)
	c.Params = gin.Params{{Key: "id", Value: "1"}}

	mock.ExpectQuery("SELECT .+ FROM `articles` WHERE").
		WillReturnRows(sqlmock.NewRows([]string{"id","user_id","title","status","body_md","cover_url","view_count","comment_count"}).
			AddRow(1, 10, "TestArticle", "pending_review", "# Hello", "", 0, 0))

	mock.ExpectQuery("SELECT .+ FROM `users` WHERE").
		WillReturnRows(sqlmock.NewRows([]string{"id","username","coin_balance_tenths"}).
			AddRow(10, "author1", 230))

	api.AdminGetArticle(c)
	require.Equal(t, http.StatusOK, w.Code)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAdminGetArticle_NotFound(t *testing.T) {
	gormDB, mock := newMockGORM(t)
	api := newMockAPISimple(t, gormDB)
	c, w := newMockGinCtx(t, "GET", "/api/v1/admin/articles/999", nil)
	c.Params = gin.Params{{Key: "id", Value: "999"}}

	mock.ExpectQuery("SELECT .+ FROM `articles` WHERE").
		WillReturnRows(sqlmock.NewRows([]string{"id"}))

	api.AdminGetArticle(c)
	require.Equal(t, http.StatusNotFound, w.Code)
	require.NoError(t, mock.ExpectationsWereMet())
}

// -- AdminListDynamics --

func TestAdminListDynamics_Success(t *testing.T) {
	gormDB, mock := newMockGORM(t)
	api := newMockAPISimple(t, gormDB)
	c, w := newMockGinCtx(t, "GET", "/api/v1/admin/dynamics", nil)

	mock.ExpectQuery("SELECT count(.+) FROM `user_dynamics`").
		WillReturnRows(sqlmock.NewRows([]string{"count(*)"}).AddRow(1))

	mock.ExpectQuery("SELECT .+ FROM `user_dynamics` ORDER BY").
		WillReturnRows(sqlmock.NewRows([]string{"id","user_id","content","status","image_count","share_count","comment_count"}).
			AddRow(1, 10, "Hello", "pending_review", 0, 0, 0))

	mock.ExpectQuery("SELECT .+ FROM `users` WHERE id IN").
		WillReturnRows(sqlmock.NewRows([]string{"id","username","coin_balance_tenths"}).
			AddRow(10, "user10", 230))

	api.AdminListDynamics(c)
	require.Equal(t, http.StatusOK, w.Code)
	require.NoError(t, mock.ExpectationsWereMet())
}

// -- AdminGetDynamic --

func TestAdminGetDynamic_Success(t *testing.T) {
	gormDB, mock := newMockGORM(t)
	api := newMockAPISimple(t, gormDB)
	c, w := newMockGinCtx(t, "GET", "/api/v1/admin/dynamics/1", nil)
	c.Params = gin.Params{{Key: "id", Value: "1"}}

	mock.ExpectQuery("SELECT .+ FROM `user_dynamics` WHERE").
		WillReturnRows(sqlmock.NewRows([]string{"id","user_id","content","status","image_count"}).
			AddRow(1, 10, "Dynamic content", "pending_review", 0))

	mock.ExpectQuery("SELECT .+ FROM `users` WHERE").
		WillReturnRows(sqlmock.NewRows([]string{"id","username","coin_balance_tenths"}).
			AddRow(10, "user10", 230))

	api.AdminGetDynamic(c)
	require.Equal(t, http.StatusOK, w.Code)
	require.NoError(t, mock.ExpectationsWereMet())
}

// -- AdminListHotSearchOps --

func TestAdminListHotSearchOps_Success(t *testing.T) {
	gormDB, mock := newMockGORM(t)
	api := newMockAPISimple(t, gormDB)
	c, w := newMockGinCtx(t, "GET", "/api/v1/admin/hot-search-ops", nil)

	mock.ExpectQuery("SELECT .+ FROM `hot_search_ops` ORDER BY pin_rank ASC, id ASC").
		WillReturnRows(sqlmock.NewRows([]string{"id","keyword","display_title","sort_order","enabled","pin_rank"}).
			AddRow(1, "keyword1", "Display1", 1, true, 0).
			AddRow(2, "keyword2", "", 2, false, 0))

	api.AdminListHotSearchOps(c)
	require.Equal(t, http.StatusOK, w.Code)
	require.NoError(t, mock.ExpectationsWereMet())
}

// -- AdminDeleteHotSearchOp --

func TestAdminDeleteHotSearchOp_Success(t *testing.T) {
	gormDB, mock := newMockGORM(t)
	api := newMockAPISimple(t, gormDB)
	c, w := newMockGinCtx(t, "DELETE", "/api/v1/admin/hot-search-ops/1", nil)
	c.Params = gin.Params{{Key: "id", Value: "1"}}

	mock.ExpectBegin()
	mock.ExpectExec("DELETE FROM `hot_search_ops`").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	api.AdminDeleteHotSearchOp(c)
	require.Equal(t, http.StatusOK, w.Code)
	require.NoError(t, mock.ExpectationsWereMet())
}
