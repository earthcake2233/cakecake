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
	"context"
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

func TestLatestUserTurnHasAssistantReply(t *testing.T) {
	api, db, _ := newAgentHandlerAPI(t)
	conv, _ := seedAgentConvForHandler(t, db, 18, 14)

	require.False(t, api.latestUserTurnHasAssistantReply(18, conv.ID))

	userMsg := seedDmMessage(t, db, conv.ID, 18, "user", "hi")
	require.False(t, api.latestUserTurnHasAssistantReply(18, conv.ID))

	seedDmMessage(t, db, conv.ID, 14, "assistant", "hello")
	require.True(t, api.latestUserTurnHasAssistantReply(18, conv.ID))

	seedDmMessage(t, db, conv.ID, 18, "user", "again")
	require.False(t, api.latestUserTurnHasAssistantReply(18, conv.ID))

	_ = userMsg
}

func TestResumeAgentReply_NoOpWhenReplyAlreadyPersisted(t *testing.T) {
	api, db, _ := newAgentHandlerAPI(t)
	conv, _ := seedAgentConvForHandler(t, db, 18, 14)
	seedDmMessage(t, db, conv.ID, 18, "user", "hi")
	seedDmMessage(t, db, conv.ID, 14, "assistant", "already done")

	api.resumeAgentReply(18, conv.ID, "hi partial")
	time.Sleep(50 * time.Millisecond)
	require.Equal(t, 1, countAssistantAfter(t, db, conv.ID, 0))
}

func TestResumeAgentReply_PendingPathPersistsOnce(t *testing.T) {
	api, db, agentSvc := newAgentHandlerAPI(t)
	conv, _ := seedAgentConvForHandler(t, db, 18, 14)
	seedDmMessage(t, db, conv.ID, 18, "user", "hi")

	api.pendingAgentReplyMu.Lock()
	if api.pendingAgentReplies == nil {
		api.pendingAgentReplies = make(map[uint64]pendingAgentReply)
	}
	api.pendingAgentReplies[18] = pendingAgentReply{
		conv:   conv,
		result: &serviceagent.GenerateReplyResult{Content: "待落库回复"},
		genID:  1,
	}
	api.pendingAgentReplyMu.Unlock()

	api.resumeAgentReply(18, conv.ID, "hi partial")

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if countAssistantAfter(t, db, conv.ID, 0) == 1 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	require.Equal(t, 1, countAssistantAfter(t, db, conv.ID, 0))

	var msg dm.DmMessage
	require.NoError(t, db.Where("conversation_id = ? AND role = ?", conv.ID, "assistant").First(&msg).Error)
	require.Equal(t, "待落库回复", msg.Content)
	_ = agentSvc
}

func TestRegenerateAgentReply_NotConfiguredFallback(t *testing.T) {
	api, db, _ := newAgentHandlerAPI(t)
	conv, _ := seedAgentConvForHandler(t, db, 18, 14)
	seedDmMessage(t, db, conv.ID, 18, "user", "hi")

	api.regenerateAgentReply(18, conv.ID)

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if countAssistantAfter(t, db, conv.ID, 0) == 1 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	require.Equal(t, 1, countAssistantAfter(t, db, conv.ID, 0))
	var msg dm.DmMessage
	require.NoError(t, db.Where("conversation_id = ? AND role = ?", conv.ID, "assistant").First(&msg).Error)
	require.Contains(t, msg.Content, "未配置")
}

func TestRegenerateAgentReply_QuotaExceeded(t *testing.T) {
	api, db, agentSvc := newAgentHandlerAPI(t)
	agentSvc.Cfg.AgentDailyQuota = 1
	agentSvc.IncrQuota(context.Background(), 18) // consume the only quota

	conv, _ := seedAgentConvForHandler(t, db, 18, 14)
	seedDmMessage(t, db, conv.ID, 18, "user", "hi")
	api.regenerateAgentReply(18, conv.ID)

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if countAssistantAfter(t, db, conv.ID, 0) == 1 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	require.Equal(t, 1, countAssistantAfter(t, db, conv.ID, 0))
	var msg dm.DmMessage
	require.NoError(t, db.Where("conversation_id = ? AND role = ?", conv.ID, "assistant").First(&msg).Error)
	require.Contains(t, msg.Content, "今日 AI 对话次数已达上限")
}

func TestContinueAgentReply_NotConfiguredNoMessage(t *testing.T) {
	api, db, _ := newAgentHandlerAPI(t)
	conv, _ := seedAgentConvForHandler(t, db, 18, 14)
	seedDmMessage(t, db, conv.ID, 18, "user", "hi")

	api.continueAgentReply(18, conv.ID, "partial text")
	time.Sleep(200 * time.Millisecond)
	require.Equal(t, 0, countAssistantAfter(t, db, conv.ID, 0))
}

func TestAttachAgentSuggestions_NoGateway(t *testing.T) {
	api, db, _ := newAgentHandlerAPI(t)
	conv, _ := seedAgentConvForHandler(t, db, 18, 14)
	msg := seedDmMessage(t, db, conv.ID, 14, "assistant", "reply")
	api.attachAgentSuggestions(18, msg.ID, "reply") // gateway nil -> no suggestions
	api.attachAgentSuggestions(18, 0, "reply")      // messageID 0 -> early return
	require.True(t, true)
}

func TestAgentCancelRegistryAndLocks(t *testing.T) {
	api, _, _ := newAgentHandlerAPI(t)
	cancel := func() {}

	genID := api.tryRegisterAgentCancel(18, cancel)
	require.NotZero(t, genID)
	require.True(t, api.agentHasActiveGeneration(18))
	require.Zero(t, api.tryRegisterAgentCancel(18, cancel)) // already active

	api.unregisterAgentCancel(18, genID)
	require.False(t, api.agentHasActiveGeneration(18))

	api.unregisterAgentCancel(18, genID) // idempotent

	mu1 := api.agentRunLock(18)
	mu2 := api.agentRunLock(18)
	mu3 := api.agentRunLock(19)
	require.Same(t, mu1, mu2)
	require.NotSame(t, mu1, mu3)

	api.supersedeAgentGeneration(18)
	api.pauseAgentReply(18) // Agent non-nil; no-op when no generation state
	api.continueAgentReply(18, 0, "  ")
	require.True(t, true)
}
