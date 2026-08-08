package handler

import (
	"cakecake/internal/config"
	"cakecake/internal/model/agent"
	"cakecake/internal/model/dm"
	serviceagent "cakecake/internal/service/agent"
	dmsvc "cakecake/internal/service/dm"
	"cakecake/internal/service/servicetest"
	"cakecake/internal/service/user"
	"cakecake/internal/ws"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func newAgentHandlerAPI(t *testing.T) (*API, *gorm.DB, *serviceagent.AgentService) {
	t.Helper()
	db := servicetest.NewDB(t)
	mr, rdb := servicetest.NewRedis(t)
	_ = mr
	log := zap.NewNop()
	dmSvc := dmsvc.NewDmService(db, rdb, log)
	store := serviceagent.NewAgentStore(db)
	agentSvc := &serviceagent.AgentService{
		Store: store,
		Redis: rdb,
		Log:   log,
		Cfg: &config.C{
			AgentEnabled:        true,
			AgentDailyQuota:     80,
			AgentRequestTimeout: 5 * time.Second,
		},
		Dm: dmSvc,
	}
	api := &API{
		Dependencies: &Dependencies{
			DB:      db,
			Redis:   rdb,
			Log:     log,
			ChatHub: ws.NewChatHub(),
			DmSvc:   dmSvc,
			UserSvc: user.NewUserService(db, log),
			Agent:   agentSvc,
		},
	}
	agentSvc.Pusher = api
	return api, db, agentSvc
}

func seedAgentConvForHandler(t *testing.T, db *gorm.DB, humanID, botID uint64) (*dm.DmConversation, *agent.AgentProfile) {
	t.Helper()
	p := &agent.AgentProfile{
		Slug:         "handler-assistant",
		BotUserID:    botID,
		DisplayName:  "H",
		SystemPrompt: "You are a test assistant.",
		Enabled:      true,
	}
	require.NoError(t, db.Create(p).Error)
	low, high := humanID, botID
	if low > high {
		low, high = high, low
	}
	conv := &dm.DmConversation{
		UserLow:        low,
		UserHigh:       high,
		Kind:           dm.DmKindAgent,
		AgentProfileID: p.ID,
	}
	require.NoError(t, db.Create(conv).Error)
	for _, u := range []uint64{low, high} {
		require.NoError(t, db.Create(&dm.DmParticipant{ConversationID: conv.ID, UserID: u}).Error)
	}
	return conv, p
}

func seedDmMessage(t *testing.T, db *gorm.DB, convID, senderID uint64, role, content string) *dm.DmMessage {
	t.Helper()
	msg := &dm.DmMessage{ConversationID: convID, SenderID: senderID, Role: role, Content: content}
	require.NoError(t, db.Create(msg).Error)
	return msg
}

func countAssistantAfter(t *testing.T, db *gorm.DB, convID, afterID uint64) int {
	t.Helper()
	var n int64
	require.NoError(t, db.Model(&dm.DmMessage{}).
		Where("conversation_id = ? AND id > ? AND role = ?", convID, afterID, "assistant").
		Count(&n).Error)
	return int(n)
}
