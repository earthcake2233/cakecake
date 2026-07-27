package handler

import (
	"net/http"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// -- AdminCreateHotSearchOp --

func TestAdminCreateHotSearchOp_Success(t *testing.T) {
	gormDB, mock := newMockGORM(t)
	api := newMockAPISimple(t, gormDB)
	body := []byte(`{"op_type":"pin","keyword":"test","display_title":"Test"}`)
	c, w := newMockGinCtx(t, "POST", "/api/v1/admin/hot-search-ops", body)
	c.Set("admin_id", uint64(1))

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO `hot_search_ops`").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	api.AdminCreateHotSearchOp(c)
	require.Equal(t, http.StatusCreated, w.Code)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAdminCreateHotSearchOp_BadRequest(t *testing.T) {
	gormDB, mock := newMockGORM(t)
	api := newMockAPISimple(t, gormDB)
	c, w := newMockGinCtx(t, "POST", "/api/v1/admin/hot-search-ops", []byte(`{"keyword":""}`))

	api.AdminCreateHotSearchOp(c)
	require.Equal(t, http.StatusBadRequest, w.Code)
	require.NoError(t, mock.ExpectationsWereMet())
}

// -- AdminUpdateHotSearchOp --

func TestAdminUpdateHotSearchOp_Success(t *testing.T) {
	gormDB, mock := newMockGORM(t)
	api := newMockAPISimple(t, gormDB)
	body := []byte(`{"display_title":"Updated Title"}`)
	c, w := newMockGinCtx(t, "PUT", "/api/v1/admin/hot-search-ops/1", body)
	c.Set("admin_id", uint64(1))
	c.Params = gin.Params{{Key: "id", Value: "1"}}

	mock.ExpectQuery("SELECT .+ FROM `hot_search_ops` WHERE").
		WillReturnRows(sqlmock.NewRows([]string{"id","keyword","op_type","display_title","sort_order","enabled","pin_rank"}).
			AddRow(1, "test", "pin", "Old", 1, true, 0))

	mock.ExpectBegin()
	mock.ExpectExec("UPDATE `hot_search_ops` SET").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	api.AdminUpdateHotSearchOp(c)
	require.Equal(t, http.StatusOK, w.Code)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAdminUpdateHotSearchOp_NotFound(t *testing.T) {
	gormDB, mock := newMockGORM(t)
	api := newMockAPISimple(t, gormDB)
	c, w := newMockGinCtx(t, "PUT", "/api/v1/admin/hot-search-ops/999", []byte(`{"keyword":"test"}`))
	c.Set("admin_id", uint64(1))
	c.Params = gin.Params{{Key: "id", Value: "999"}}

	mock.ExpectQuery("SELECT .+ FROM `hot_search_ops` WHERE").
		WillReturnRows(sqlmock.NewRows([]string{"id"}))

	api.AdminUpdateHotSearchOp(c)
	require.Equal(t, http.StatusNotFound, w.Code)
	require.NoError(t, mock.ExpectationsWereMet())
}

// -- AdminPreviewHotSearch --

func TestAdminPreviewHotSearch_Success(t *testing.T) {
	gormDB, mock := newMockGORM(t)
	api := newMockAPISimple(t, gormDB)
	c, w := newMockGinCtx(t, "GET", "/api/v1/admin/hot-search-ops/preview", nil)
	c.Set("admin_id", uint64(1))

	mock.ExpectQuery("SELECT .+ FROM `hot_search_ops`").
		WillReturnRows(sqlmock.NewRows([]string{"id","keyword","display_title","enabled","pin_rank"}).
			AddRow(1, "news", "Hot News", true, 1).
			AddRow(2, "sports", "", true, 2))

	api.AdminPreviewHotSearch(c)
	require.Equal(t, http.StatusOK, w.Code)
	require.NoError(t, mock.ExpectationsWereMet())
}
