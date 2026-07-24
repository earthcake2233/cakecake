package handler

import (
	"net/http"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// -- AdminRejectVideo: status not pending_review -> 400

func TestAdminRejectVideo_NotPending(t *testing.T) {
	gormDB, mock := newMockGORM(t)
	api := newMockAPISimple(t, gormDB)
	c, w := newMockGinCtx(t, "POST", "/api/v1/admin/videos/1/reject", nil)
	c.Set("admin_id", uint64(1))
	c.Params = gin.Params{{Key: "id", Value: "1"}}

	mock.ExpectQuery("SELECT .+ FROM `videos` WHERE").
		WillReturnRows(sqlmock.NewRows([]string{"id","user_id","title","status"}).
			AddRow(1, 10, "TV", "published"))

	api.AdminRejectVideo(c)
	require.Equal(t, http.StatusBadRequest, w.Code)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAdminRejectVideo_NotFound(t *testing.T) {
	gormDB, mock := newMockGORM(t)
	api := newMockAPISimple(t, gormDB)
	c, w := newMockGinCtx(t, "POST", "/api/v1/admin/videos/999/reject", nil)
	c.Set("admin_id", uint64(1))
	c.Params = gin.Params{{Key: "id", Value: "999"}}

	mock.ExpectQuery("SELECT .+ FROM `videos` WHERE").
		WillReturnRows(sqlmock.NewRows([]string{"id"}))

	api.AdminRejectVideo(c)
	require.Equal(t, http.StatusNotFound, w.Code)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAdminRejectVideo_BadParam(t *testing.T) {
	gormDB, mock := newMockGORM(t)
	api := newMockAPISimple(t, gormDB)
	c, w := newMockGinCtx(t, "POST", "/api/v1/admin/videos/abc/reject", nil)
	c.Set("admin_id", uint64(1))
	c.Params = gin.Params{{Key: "id", Value: "abc"}}

	api.AdminRejectVideo(c)
	require.Equal(t, http.StatusBadRequest, w.Code)
	require.NoError(t, mock.ExpectationsWereMet())
}

// -- AdminRejectArticle: status not pending_review -> 400

func TestAdminRejectArticle_NotPending(t *testing.T) {
	gormDB, mock := newMockGORM(t)
	api := newMockAPISimple(t, gormDB)
	c, w := newMockGinCtx(t, "POST", "/api/v1/admin/articles/1/reject", nil)
	c.Set("admin_id", uint64(1))
	c.Params = gin.Params{{Key: "id", Value: "1"}}

	mock.ExpectQuery("SELECT .+ FROM `articles` WHERE").
		WillReturnRows(sqlmock.NewRows([]string{"id","user_id","title","status"}).
			AddRow(1, 10, "TA", "published"))

	api.AdminRejectArticle(c)
	require.Equal(t, http.StatusBadRequest, w.Code)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAdminRejectArticle_NotFound(t *testing.T) {
	gormDB, mock := newMockGORM(t)
	api := newMockAPISimple(t, gormDB)
	c, w := newMockGinCtx(t, "POST", "/api/v1/admin/articles/999/reject", nil)
	c.Set("admin_id", uint64(1))
	c.Params = gin.Params{{Key: "id", Value: "999"}}

	mock.ExpectQuery("SELECT .+ FROM `articles` WHERE").
		WillReturnRows(sqlmock.NewRows([]string{"id"}))

	api.AdminRejectArticle(c)
	require.Equal(t, http.StatusNotFound, w.Code)
	require.NoError(t, mock.ExpectationsWereMet())
}

// -- AdminDeleteVideo: status not published/rejected -> 400

func TestAdminDeleteVideo_NotPublished(t *testing.T) {
	gormDB, mock := newMockGORM(t)
	api := newMockAPISimple(t, gormDB)
	c, w := newMockGinCtx(t, "DELETE", "/api/v1/admin/videos/1", nil)
	c.Set("admin_id", uint64(1))
	c.Params = gin.Params{{Key: "id", Value: "1"}}

	mock.ExpectQuery("SELECT .+ FROM `videos` WHERE").
		WillReturnRows(sqlmock.NewRows([]string{"id","user_id","title","status"}).
			AddRow(1, 10, "TV", "pending_review"))

	api.AdminDeleteVideo(c)
	require.Equal(t, http.StatusBadRequest, w.Code)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAdminDeleteVideo_NotFound(t *testing.T) {
	gormDB, mock := newMockGORM(t)
	api := newMockAPISimple(t, gormDB)
	c, w := newMockGinCtx(t, "DELETE", "/api/v1/admin/videos/999", nil)
	c.Set("admin_id", uint64(1))
	c.Params = gin.Params{{Key: "id", Value: "999"}}

	mock.ExpectQuery("SELECT .+ FROM `videos` WHERE").
		WillReturnRows(sqlmock.NewRows([]string{"id"}))

	api.AdminDeleteVideo(c)
	require.Equal(t, http.StatusNotFound, w.Code)
	require.NoError(t, mock.ExpectationsWereMet())
}

// -- AdminDeleteArticle: status not published/rejected -> 400

func TestAdminDeleteArticle_NotPublished(t *testing.T) {
	gormDB, mock := newMockGORM(t)
	api := newMockAPISimple(t, gormDB)
	c, w := newMockGinCtx(t, "DELETE", "/api/v1/admin/articles/1", nil)
	c.Set("admin_id", uint64(1))
	c.Params = gin.Params{{Key: "id", Value: "1"}}

	mock.ExpectQuery("SELECT .+ FROM `articles` WHERE").
		WillReturnRows(sqlmock.NewRows([]string{"id","user_id","title","status"}).
			AddRow(1, 10, "TA", "pending_review"))

	api.AdminDeleteArticle(c)
	require.Equal(t, http.StatusBadRequest, w.Code)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAdminDeleteArticle_NotFound(t *testing.T) {
	gormDB, mock := newMockGORM(t)
	api := newMockAPISimple(t, gormDB)
	c, w := newMockGinCtx(t, "DELETE", "/api/v1/admin/articles/999", nil)
	c.Set("admin_id", uint64(1))
	c.Params = gin.Params{{Key: "id", Value: "999"}}

	mock.ExpectQuery("SELECT .+ FROM `articles` WHERE").
		WillReturnRows(sqlmock.NewRows([]string{"id"}))

	api.AdminDeleteArticle(c)
	require.Equal(t, http.StatusNotFound, w.Code)
	require.NoError(t, mock.ExpectationsWereMet())
}

// -- AdminApproveVideo: status not pending_review -> 400 (no ES needed)

func TestAdminApproveVideo_NotPending(t *testing.T) {
	gormDB, mock := newMockGORM(t)
	api := newMockAPISimple(t, gormDB)
	c, w := newMockGinCtx(t, "POST", "/api/v1/admin/videos/1/approve", nil)
	c.Set("admin_id", uint64(1))
	c.Params = gin.Params{{Key: "id", Value: "1"}}

	mock.ExpectQuery("SELECT .+ FROM `videos` WHERE").
		WillReturnRows(sqlmock.NewRows([]string{"id","user_id","title","status","video_url"}).
			AddRow(1, 10, "TV", "published", "https://ex.com/v.mp4"))

	api.AdminApproveVideo(c)
	require.Equal(t, http.StatusBadRequest, w.Code)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAdminApproveVideo_EmptyVideoURL(t *testing.T) {
	gormDB, mock := newMockGORM(t)
	api := newMockAPISimple(t, gormDB)
	c, w := newMockGinCtx(t, "POST", "/api/v1/admin/videos/1/approve", nil)
	c.Set("admin_id", uint64(1))
	c.Params = gin.Params{{Key: "id", Value: "1"}}

	mock.ExpectQuery("SELECT .+ FROM `videos` WHERE").
		WillReturnRows(sqlmock.NewRows([]string{"id","user_id","title","status","video_url"}).
			AddRow(1, 10, "TV", "pending_review", ""))

	api.AdminApproveVideo(c)
	require.Equal(t, http.StatusBadRequest, w.Code)
	require.NoError(t, mock.ExpectationsWereMet())
}

// -- AdminApproveArticle: status not pending_review -> 400 (no ES needed)

func TestAdminApproveArticle_NotPending(t *testing.T) {
	gormDB, mock := newMockGORM(t)
	api := newMockAPISimple(t, gormDB)
	c, w := newMockGinCtx(t, "POST", "/api/v1/admin/articles/1/approve", nil)
	c.Set("admin_id", uint64(1))
	c.Params = gin.Params{{Key: "id", Value: "1"}}

	mock.ExpectQuery("SELECT .+ FROM `articles` WHERE").
		WillReturnRows(sqlmock.NewRows([]string{"id","user_id","title","status"}).
			AddRow(1, 10, "TA", "published"))

	api.AdminApproveArticle(c)
	require.Equal(t, http.StatusBadRequest, w.Code)
	require.NoError(t, mock.ExpectationsWereMet())
}
