package handler

import (
	"cakecake/internal/model/admin"
	"cakecake/internal/model/agent"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"cakecake/internal/config"
	svcagent "cakecake/internal/service/agent"
	"cakecake/internal/ws"
)

func TestAdminAgentMeta_Configured(t *testing.T) {
	api := &API{Dependencies: &Dependencies{Cfg: &config.C{DeepSeekAPIKey: "sk-xxx"}, Log: zap.NewNop(), Hub: ws.NewHub(), Agent: &svcagent.AgentService{}}}
	m := api.adminAgentMeta()
	require.Equal(t, true, m.DeepseekConfigured)
	require.Greater(t, m.MaxProfiles, 0)
}

func TestAdminAgentMeta_NotConfigured(t *testing.T) {
	api := &API{Dependencies: &Dependencies{Cfg: &config.C{DeepSeekAPIKey: ""}, Log: zap.NewNop(), Hub: ws.NewHub(), Agent: &svcagent.AgentService{}}}
	m := api.adminAgentMeta()
	require.Equal(t, false, m.DeepseekConfigured)
}

func TestAdminAgentMeta_NilCfg(t *testing.T) {
	api := &API{Dependencies: &Dependencies{Log: zap.NewNop(), Hub: ws.NewHub(), Agent: &svcagent.AgentService{}}}
	m := api.adminAgentMeta()
	require.Equal(t, false, m.DeepseekConfigured)
}

func TestAdminAgentProfilePayload(t *testing.T) {
	now := time.Now()
	p := &agent.AgentProfile{
		ID: 1, Slug: "assistant", BotUserID: 100,
		DisplayName:         "AI Assistant",
		AvatarURL:           "https://ex.com/avatar.png",
		Sign:                "I am an AI",
		SystemPrompt:        "You are a helpful assistant",
		WelcomeMessagesJSON: `["Hello!","How can I help?"]`,
		SortOrder:           1, Enabled: true, UpdatedAt: now,
	}
	out := adminAgentProfilePayload(p, "")
	require.Equal(t, uint64(1), out.ID)
	require.Equal(t, "assistant", out.Slug)
	require.Equal(t, uint64(100), out.BotUserID)
	require.Equal(t, "AI Assistant", out.DisplayName)
	require.Equal(t, "https://ex.com/avatar.png", out.AvatarURL)
	require.Equal(t, "I am an AI", out.Sign)
	require.Equal(t, "You are a helpful assistant", out.SystemPrompt)
	require.ElementsMatch(t, []string{"Hello!", "How can I help?"}, out.WelcomeMessages)
	require.Equal(t, 1, out.SortOrder)
	require.Equal(t, true, out.Enabled)
}

func TestAdminAgentProfilePayload_Nil(t *testing.T) {
	out := adminAgentProfilePayload(nil, "")
	require.Equal(t, adminAgentProfileItem{}, out)
}

func TestAdminAgentProfilePayload_EmptyWelcome(t *testing.T) {
	p := &agent.AgentProfile{ID: 1, Slug: "test", WelcomeMessagesJSON: "[]", UpdatedAt: time.Now()}
	out := adminAgentProfilePayload(p, "")
	require.Equal(t, []string{}, out.WelcomeMessages)
}

func TestHotSearchDisplayTitle(t *testing.T) {
	tests := []struct {
		name string
		op   *admin.HotSearchOp
		want string
	}{
		{"nil", nil, ""},
		{"display title set", &admin.HotSearchOp{DisplayTitle: "Display Title", Keyword: "kw"}, "Display Title"},
		{"empty display title", &admin.HotSearchOp{DisplayTitle: "", Keyword: "keyword"}, "keyword"},
		{"whitespace display title", &admin.HotSearchOp{DisplayTitle: "  ", Keyword: "real-keyword"}, "real-keyword"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) { got := hotSearchDisplayTitle(tc.op); require.Equal(t, tc.want, got) })
	}
}

// SQLMock: AdminListBanners

func TestAdminListBanners_Success(t *testing.T) {
	gormDB, mock := newMockGORM(t)
	api := newMockAPISimple(t, gormDB)
	c, w := newMockGinCtx(t, "GET", "/api/v1/admin/home-banners", nil)

	mock.ExpectQuery("SELECT \\* FROM `home_banners` ORDER BY sort_order ASC").
		WillReturnRows(sqlmock.NewRows([]string{"id", "title", "image_url", "link_type", "link_target", "sort_order", "enabled"}).
			AddRow(1, "Banner1", "https://ex.com/1.jpg", "url", "https://ex.com", 1, true).
			AddRow(2, "Banner2", "https://ex.com/2.jpg", "none", "", 2, false))

	api.AdminListBanners(c)
	require.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Code int                    `json:"code"`
		Data map[string]interface{} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)
	items := resp.Data["items"].([]interface{})
	require.Len(t, items, 2)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAdminListBanners_DBError(t *testing.T) {
	gormDB, mock := newMockGORM(t)
	api := newMockAPISimple(t, gormDB)
	c, w := newMockGinCtx(t, "GET", "/api/v1/admin/home-banners", nil)
	mock.ExpectQuery("SELECT \\* FROM `home_banners`").WillReturnError(sqlmock.ErrCancelled)
	api.AdminListBanners(c)
	require.Equal(t, http.StatusInternalServerError, w.Code)
	require.NoError(t, mock.ExpectationsWereMet())
}

// SQLMock: AdminCreateBanner

func TestAdminCreateBanner_Success(t *testing.T) {
	gormDB, mock := newMockGORM(t)
	api := newMockAPISimple(t, gormDB)
	body := `{"title":"New Banner","image_url":"https://ex.com/img.jpg","link_type":"url","link_target":"https://ex.com","sort_order":1,"enabled":true}`
	c, w := newMockGinCtx(t, "POST", "/api/v1/admin/home-banners", []byte(body))
	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO `home_banners`").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()
	api.AdminCreateBanner(c)
	require.Equal(t, http.StatusCreated, w.Code)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAdminCreateBanner_BadRequest(t *testing.T) {
	gormDB, mock := newMockGORM(t)
	api := newMockAPISimple(t, gormDB)
	c, w := newMockGinCtx(t, "POST", "/api/v1/admin/home-banners", []byte(`{"title":"","image_url":"img.jpg"}`))
	api.AdminCreateBanner(c)
	require.Equal(t, http.StatusBadRequest, w.Code)
	c, w = newMockGinCtx(t, "POST", "/api/v1/admin/home-banners", []byte(`{"title":"T","image_url":""}`))
	api.AdminCreateBanner(c)
	require.Equal(t, http.StatusBadRequest, w.Code)
	c, w = newMockGinCtx(t, "POST", "/api/v1/admin/home-banners", []byte(`{invalid}`))
	api.AdminCreateBanner(c)
	require.Equal(t, http.StatusBadRequest, w.Code)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAdminCreateBanner_DBError(t *testing.T) {
	gormDB, mock := newMockGORM(t)
	api := newMockAPISimple(t, gormDB)
	c, w := newMockGinCtx(t, "POST", "/api/v1/admin/home-banners", []byte(`{"title":"NB","image_url":"https://ex.com/img.jpg"}`))
	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO `home_banners`").WillReturnError(sqlmock.ErrCancelled)
	mock.ExpectRollback()
	api.AdminCreateBanner(c)
	require.Equal(t, http.StatusInternalServerError, w.Code)
	require.NoError(t, mock.ExpectationsWereMet())
}
