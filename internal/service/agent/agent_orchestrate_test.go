package agent

import (
	"cakecake/internal/config"
	"cakecake/internal/model/agent"
	"cakecake/internal/model/dm"
	dmsvc "cakecake/internal/service/dm"
	"cakecake/internal/service/servicetest"
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// recordingPusher captures what the service decided to push (the transport
// adapter is mocked so orchestration tests stay HTTP-free).
type recordingPusher struct {
	mu       sync.Mutex
	messages []*dm.DmMessage
	events   []map[string]interface{}
}

func (p *recordingPusher) FormatAgentMessage(_ context.Context, _ uint64, _ *dm.DmConversation, msg *dm.DmMessage) ([]map[string]interface{}, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.messages = append(p.messages, msg)
	return []map[string]interface{}{{"type": "dm_message", "message": map[string]interface{}{"id": msg.ID}}}, nil
}

func (p *recordingPusher) messageCount() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.messages)
}

func (p *recordingPusher) lastMessage() *dm.DmMessage {
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.messages) == 0 {
		return nil
	}
	return p.messages[len(p.messages)-1]
}

func (p *recordingPusher) continueModes() []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	var out []string
	for _, e := range p.events {
		if e["type"] == "agent_continue_mode" {
			if m, ok := e["mode"].(string); ok {
				out = append(out, m)
			}
		}
	}
	return out
}

func newOrchestrateService(t *testing.T) (*AgentService, *gorm.DB, *recordingPusher) {
	t.Helper()
	db := servicetest.NewDB(t)
	_, rdb := servicetest.NewRedis(t)
	log := zap.NewNop()
	dmSvc := dmsvc.NewDmService(db, rdb, log)
	pusher := &recordingPusher{}
	s := &AgentService{
		Store: NewAgentStore(db),
		Redis: rdb,
		Log:   log,
		Cfg: &config.C{
			AgentEnabled:        true,
			AgentDailyQuota:     80,
			AgentRequestTimeout: 5 * time.Second,
		},
		Dm:     dmSvc,
		Pusher: pusher,
	}
	s.EventHook = func(_ uint64, payload map[string]interface{}) {
		pusher.mu.Lock()
		pusher.events = append(pusher.events, payload)
		pusher.mu.Unlock()
	}
	return s, db, pusher
}

func seedOrchestrateConv(t *testing.T, db *gorm.DB, humanID, botID uint64) (*dm.DmConversation, *agent.AgentProfile) {
	t.Helper()
	p := &agent.AgentProfile{
		Slug:         "orch-assistant",
		BotUserID:    botID,
		DisplayName:  "O",
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

func seedOrchestrateMsg(t *testing.T, db *gorm.DB, convID, senderID uint64, role, content string) *dm.DmMessage {
	t.Helper()
	msg := &dm.DmMessage{ConversationID: convID, SenderID: senderID, Role: role, Content: content}
	require.NoError(t, db.Create(msg).Error)
	return msg
}

func countOrchestrateAssistant(t *testing.T, db *gorm.DB, convID uint64) int {
	t.Helper()
	var n int64
	require.NoError(t, db.Model(&dm.DmMessage{}).
		Where("conversation_id = ? AND role = ?", convID, "assistant").
		Count(&n).Error)
	return int(n)
}

func waitFor(t *testing.T, timeout time.Duration, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("condition not met within", timeout)
}

func TestService_LatestUserTurnHasAssistantReply(t *testing.T) {
	s, db, _ := newOrchestrateService(t)
	conv, _ := seedOrchestrateConv(t, db, 18, 14)

	require.False(t, s.latestUserTurnHasAssistantReply(18, conv.ID))
	seedOrchestrateMsg(t, db, conv.ID, 18, "user", "hi")
	require.False(t, s.latestUserTurnHasAssistantReply(18, conv.ID))
	seedOrchestrateMsg(t, db, conv.ID, 14, "assistant", "hello")
	require.True(t, s.latestUserTurnHasAssistantReply(18, conv.ID))
	seedOrchestrateMsg(t, db, conv.ID, 18, "user", "again")
	require.False(t, s.latestUserTurnHasAssistantReply(18, conv.ID))
}

func TestService_ResumeReply_NoOpWhenReplyAlreadyPersisted(t *testing.T) {
	s, db, _ := newOrchestrateService(t)
	conv, _ := seedOrchestrateConv(t, db, 18, 14)
	seedOrchestrateMsg(t, db, conv.ID, 18, "user", "hi")
	seedOrchestrateMsg(t, db, conv.ID, 14, "assistant", "already done")

	s.ResumeReply(18, conv.ID)
	time.Sleep(80 * time.Millisecond)
	require.Equal(t, 1, countOrchestrateAssistant(t, db, conv.ID))
}

func TestService_ResumeReply_PendingPathPersistsOnce(t *testing.T) {
	s, db, pusher := newOrchestrateService(t)
	conv, _ := seedOrchestrateConv(t, db, 18, 14)
	seedOrchestrateMsg(t, db, conv.ID, 18, "user", "hi")

	genID := s.beginGeneration(18, nil)
	s.PauseGeneration(18)
	s.storePendingReply(18, genID, conv, &GenerateReplyResult{Content: "待落库回复"})

	s.ResumeReply(18, conv.ID)
	waitFor(t, 5*time.Second, func() bool { return countOrchestrateAssistant(t, db, conv.ID) == 1 })
	require.Equal(t, 1, countOrchestrateAssistant(t, db, conv.ID))
	require.Equal(t, 1, pusher.messageCount())
	require.Equal(t, "待落库回复", pusher.lastMessage().Content)
	require.Equal(t, []string{"buffer"}, pusher.continueModes())
}

func TestService_RegenerateReply_NotConfiguredFallback(t *testing.T) {
	s, db, pusher := newOrchestrateService(t)
	conv, _ := seedOrchestrateConv(t, db, 18, 14)
	seedOrchestrateMsg(t, db, conv.ID, 18, "user", "hi")

	s.RegenerateReply(18, conv.ID)
	waitFor(t, 5*time.Second, func() bool { return pusher.messageCount() == 1 })
	require.Equal(t, 1, countOrchestrateAssistant(t, db, conv.ID))
	require.Contains(t, pusher.lastMessage().Content, "未配置")
}

func TestService_RegenerateReply_QuotaExceeded(t *testing.T) {
	s, db, pusher := newOrchestrateService(t)
	s.Cfg.AgentDailyQuota = 1
	s.IncrQuota(context.Background(), 18)

	conv, _ := seedOrchestrateConv(t, db, 18, 14)
	seedOrchestrateMsg(t, db, conv.ID, 18, "user", "hi")
	s.RegenerateReply(18, conv.ID)

	waitFor(t, 5*time.Second, func() bool { return pusher.messageCount() == 1 })
	require.Equal(t, 1, countOrchestrateAssistant(t, db, conv.ID))
	require.Contains(t, pusher.lastMessage().Content, "今日 AI 对话次数已达上限")
}

func TestService_ContinueReply_NotConfiguredNoMessage(t *testing.T) {
	s, db, _ := newOrchestrateService(t)
	conv, _ := seedOrchestrateConv(t, db, 18, 14)
	seedOrchestrateMsg(t, db, conv.ID, 18, "user", "hi")

	s.continueReply(18, conv.ID, "partial text")
	time.Sleep(200 * time.Millisecond)
	require.Equal(t, 0, countOrchestrateAssistant(t, db, conv.ID))
}

func TestService_AttachSuggestions_NoGateway(t *testing.T) {
	s, db, pusher := newOrchestrateService(t)
	conv, _ := seedOrchestrateConv(t, db, 18, 14)
	msg := seedOrchestrateMsg(t, db, conv.ID, 14, "assistant", "reply")
	s.attachSuggestions(18, msg.ID, "reply")
	s.attachSuggestions(18, 0, "reply")
	require.Equal(t, 0, len(pusher.events))
}

func TestService_RunLock(t *testing.T) {
	s, _, _ := newOrchestrateService(t)
	mu1 := s.runLock(18)
	mu2 := s.runLock(18)
	mu3 := s.runLock(19)
	require.Same(t, mu1, mu2)
	require.NotSame(t, mu1, mu3)
}

func TestLastUserText_NewestFirst(t *testing.T) {
	msgs := []dm.DmMessage{
		{ID: 5, Role: "assistant", Content: "reply2"},
		{ID: 4, Role: "user", Content: "go如何连接mysql"},
		{ID: 3, Role: "assistant", Content: "reply1"},
		{ID: 2, Role: "user", Content: "你好"},
	}
	require.Equal(t, "go如何连接mysql", lastUserText(msgs))
	require.Equal(t, "", lastUserText([]dm.DmMessage{{ID: 1, Role: "assistant", Content: "x"}}))
}

func TestService_RegenerateWelcome_NoUserMessage(t *testing.T) {
	s, db, pusher := newOrchestrateService(t)
	conv, p := seedOrchestrateConv(t, db, 18, 14)
	p.WelcomeMessagesJSON = `["欢迎语A","欢迎语B"]`
	require.NoError(t, db.Save(p).Error)
	seedOrchestrateMsg(t, db, conv.ID, 14, "assistant", "欢迎语A")

	s.RegenerateReply(18, conv.ID)
	waitFor(t, 5*time.Second, func() bool { return pusher.messageCount() == 1 })
	require.Equal(t, 2, countOrchestrateAssistant(t, db, conv.ID))
	require.Equal(t, "欢迎语B", pusher.lastMessage().Content)
}

func TestPickDifferentWelcome(t *testing.T) {
	require.Equal(t, "B", pickDifferentWelcome([]string{"A", "B"}, "A"))
	require.Equal(t, "A", pickDifferentWelcome([]string{"A"}, "A"))
	require.Equal(t, "", pickDifferentWelcome(nil, "A"))
}
