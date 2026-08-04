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

	"cakecake/internal/service/article"
	"cakecake/internal/service/banner"
	"cakecake/internal/service/comment"
	"cakecake/internal/service/dynamic"
	"cakecake/internal/service/hotsearch"
	"cakecake/internal/service/user"
	"cakecake/internal/service/video"
	"cakecake/internal/service/viewhistory"
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
	commentSvc := comment.NewCommentService(gormDB, nil, zap.NewNop(), nil, nil, nil, nil, nil, nil)
	viewHistorySvc := viewhistory.NewViewHistoryService(gormDB, nil, zap.NewNop())
	videoSvc := video.NewVideoService(gormDB, nil, zap.NewNop(), nil, nil)
	hotSearchSvc := hotsearch.NewHotSearchService(gormDB, nil)
	articleSvc := article.NewArticleService(gormDB, nil, zap.NewNop(), nil)
	userSvc := user.NewUserService(gormDB, zap.NewNop())
	dynamicSvc := dynamic.NewDynamicService(gormDB, nil, zap.NewNop())
	return &API{
		Dependencies: &Dependencies{
			DB:             gormDB,
			Log:            zap.NewNop(),
			Hub:            ws.NewHub(),
			CommentSvc:     commentSvc,
			ViewHistorySvc: viewHistorySvc,
			VideoSvc:       videoSvc,
			BannerSvc:      banner.NewBannerService(gormDB),
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
