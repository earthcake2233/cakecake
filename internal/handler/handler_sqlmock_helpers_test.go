package handler

import (
	"bytes"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"cakecake/internal/service"
	"cakecake/internal/ws"
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
	commentSvc := service.NewCommentService(gormDB, nil, zap.NewNop(), nil, nil, nil, nil, nil, nil)
	viewHistorySvc := service.NewViewHistoryService(gormDB, nil, zap.NewNop())
	videoSvc := service.NewVideoService(gormDB, nil, zap.NewNop())
	hotSearchSvc := service.NewHotSearchService(gormDB, nil)
	articleSvc := service.NewArticleService(gormDB, nil, zap.NewNop(), nil)
	userSvc := service.NewUserService(gormDB, zap.NewNop())
	dynamicSvc := service.NewDynamicService(gormDB, nil, zap.NewNop())
	return &API{
		Dependencies: &Dependencies{
			DB:             gormDB,
			Log:            zap.NewNop(),
			Hub:            ws.NewHub(),
			CommentSvc:     commentSvc,
			ViewHistorySvc: viewHistorySvc,
			VideoSvc:       videoSvc,
			HotSearchSvc:   hotSearchSvc,
			ArticleSvc:     articleSvc,
			UserSvc:        userSvc,
			DynamicSvc:     dynamicSvc,
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
