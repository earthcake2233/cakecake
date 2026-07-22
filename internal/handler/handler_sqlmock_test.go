package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"minibili/internal/ws"
)

func newMockGORM(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	gormDB, err := gorm.Open(mysql.New(mysql.Config{Conn: db, SkipInitializeWithVersion: true}), &gorm.Config{})
	require.NoError(t, err)
	return gormDB, mock
}

func newMockAPISimple(t *testing.T, gormDB *gorm.DB) *API {
	t.Helper()
	return &API{
		Dependencies: &Dependencies{
			DB:  gormDB,
			Log: zap.NewNop(),
			Hub: ws.NewHub(),
		},
	}
}

func newMockGinCtx(t *testing.T, method, url string, body []byte) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(method, url, bytes.NewReader(body))
	if body != nil {
		c.Request.Header.Set("Content-Type", "application/json")
	}
	return c, w
}

func TestDeleteComment_Mock_Success(t *testing.T) {
	gormDB, mock := newMockGORM(t)
	api := newMockAPISimple(t, gormDB)
	c, w := newMockGinCtx(t, http.MethodDelete, "/api/v1/comments/1", nil)
	c.Set("user_id", uint64(10))
	c.Params = gin.Params{{Key: "id", Value: "1"}}

	mock.ExpectQuery("SELECT .+ FROM `comments`").
		WithArgs(int64(1), 1).
		WillReturnRows(sqlmock.NewRows([]string{"id","video_id","user_id","content","approved","level","parent_id"}).
			AddRow(1, 100, 5, "test", true, 1, 0))

	mock.ExpectQuery("SELECT .+ FROM `videos`").
		WithArgs(int64(100), 1).
		WillReturnRows(sqlmock.NewRows([]string{"id","user_id","title","status"}).
			AddRow(100, 10, "Test Video", "published"))

	mock.ExpectQuery("SELECT `id` FROM `comments` WHERE parent_id").
		WithArgs(int64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}))

	mock.ExpectQuery("SELECT count").
		WithArgs(sqlmock.AnyArg(), true).
		WillReturnRows(sqlmock.NewRows([]string{"count(*)"}).AddRow(1))

	mock.ExpectBegin()
	mock.ExpectExec("DELETE FROM `comments`").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("DELETE.*notifications").
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("GREATEST").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	api.DeleteComment(c)
	require.Equal(t, http.StatusOK, w.Code)
	var resp struct { Code int `json:"code"` }
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDeleteComment_Mock_Forbidden(t *testing.T) {
	gormDB, mock := newMockGORM(t)
	api := newMockAPISimple(t, gormDB)
	c, w := newMockGinCtx(t, http.MethodDelete, "/api/v1/comments/1", nil)
	c.Set("user_id", uint64(99))
	c.Params = gin.Params{{Key: "id", Value: "1"}}

	mock.ExpectQuery("SELECT .+ FROM `comments`").
		WithArgs(int64(1), 1).
		WillReturnRows(sqlmock.NewRows([]string{"id","video_id","user_id"}).AddRow(1, 100, 5))

	mock.ExpectQuery("SELECT .+ FROM `videos`").
		WithArgs(int64(100), 1).
		WillReturnRows(sqlmock.NewRows([]string{"id","user_id"}).AddRow(100, 10))

	api.DeleteComment(c)
	require.Equal(t, http.StatusForbidden, w.Code)
	require.NoError(t, mock.ExpectationsWereMet())
}
