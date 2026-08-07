package handler

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"cakecake/internal/middleware"
)

func TestPostAgentFeedback(t *testing.T) {
	gin.SetMode(gin.TestMode)
	api, db, _ := newAgentHandlerAPI(t)
	conv, _ := seedAgentConvForHandler(t, db, 18, 14)
	msg := seedDmMessage(t, db, conv.ID, 14, "assistant", "reply")

	do := func(body string, uid interface{}) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(rec)
		c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/dm/agent/feedback", bytes.NewReader([]byte(body)))
		c.Request.Header.Set("Content-Type", "application/json")
		if uid != nil {
			c.Set(middleware.CtxUserIDKey, uid)
		}
		api.PostAgentFeedback(c)
		return rec
	}

	w := do(`{"message_id":`+uintStr(msg.ID)+`,"feedback":"like"}`, uint64(18))
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	w = do(`{"message_id":0,"feedback":"like"}`, uint64(18))
	require.Equal(t, http.StatusBadRequest, w.Code)

	w = do(`{"message_id":`+uintStr(msg.ID)+`,"feedback":"bad"}`, uint64(18))
	require.Equal(t, http.StatusBadRequest, w.Code)

	w = do(`{"message_id":`+uintStr(msg.ID)+`,"feedback":"like"}`, nil)
	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAdminListAgentFeedbacks(t *testing.T) {
	gin.SetMode(gin.TestMode)
	api, db, _ := newAgentHandlerAPI(t)
	conv, _ := seedAgentConvForHandler(t, db, 18, 14)
	msg := seedDmMessage(t, db, conv.ID, 14, "assistant", "内容")
	require.NoError(t, api.Agent.SetMessageFeedback(t.Context(), msg.ID, 18, "like"))

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/admin/agent-feedbacks?limit=50", nil)
	api.AdminListAgentFeedbacks(c)
	require.Equal(t, http.StatusOK, rec.Code)
}

func uintStr(v uint64) string {
	return strconv.FormatUint(v, 10)
}
