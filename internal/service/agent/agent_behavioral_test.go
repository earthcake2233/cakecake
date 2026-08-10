package agent

import (
	"cakecake/internal/aigateway"
	"cakecake/internal/config"
	"cakecake/internal/model/agent"
	"cakecake/internal/model/dm"
	"cakecake/internal/service/servicetest"
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type recordingToolExec struct {
	calls []string
}

func (r *recordingToolExec) Execute(_ context.Context, toolName string, args json.RawMessage) (string, error) {
	r.calls = append(r.calls, toolName+" "+string(args))
	return `{"items":[{"id":1,"title":"Go 入门指南"}]}`, nil
}

func behavioralAgentService(t *testing.T) (*AgentService, *dm.DmConversation, *gorm.DB) {
	t.Helper()
	db := servicetest.NewDB(t)
	store := NewAgentStore(db)
	p := &agent.AgentProfile{
		Slug: "behavioral", BotUserID: 14, DisplayName: "G",
		SystemPrompt: "You are helpful.", Enabled: true,
	}
	require.NoError(t, db.Create(p).Error)
	conv := &dm.DmConversation{
		UserLow: 14, UserHigh: 18, Kind: dm.DmKindAgent, AgentProfileID: p.ID,
	}
	require.NoError(t, db.Create(conv).Error)
	require.NoError(t, db.Create(&dm.DmParticipant{ConversationID: conv.ID, UserID: 14}).Error)
	require.NoError(t, db.Create(&dm.DmParticipant{ConversationID: conv.ID, UserID: 18}).Error)

	s := &AgentService{
		Store: store,
		Cfg: &config.C{
			DeepSeekAPIKey:      "sk-test",
			AgentEnabled:        true,
			AgentRequestTimeout: 10 * time.Second,
		},
		Log: zap.NewNop(),
	}
	return s, conv, db
}

// TestBehavioral_AgentToolCallPersists verifies the full AI tool-calling chain:
// the LLM requests a tool, the executor runs it, the reply carries tool
// activities/results, and the assistant message is persisted with those fields.
func TestBehavioral_AgentToolCallPersists(t *testing.T) {
	s, conv, db := behavioralAgentService(t)
	hub, _ := newStateTestHub(t, 18)
	s.ChatHub = hub

	exec := &recordingToolExec{}
	s.ToolExec = exec

	callCount := 0
	llm := mockLLM(t, func(r *http.Request) (int, string) {
		callCount++
		if callCount == 1 {
			return http.StatusOK,
				"data: {\"choices\":[{\"delta\":{\"tool_calls\":[{\"index\":0,\"id\":\"c1\",\"function\":{\"name\":\"search_videos\",\"arguments\":\"{}\"}}]}}]}\n\n" +
					"data: {\"choices\":[{\"delta\":{},\"finish_reason\":\"tool_calls\"}]}\n\n" +
					"data: [DONE]\n\n"
		}
		return http.StatusOK,
			"data: {\"choices\":[{\"delta\":{\"content\":\"推荐《Go 入门指南》\"}}]}\n\n" +
				"data: {\"choices\":[{\"delta\":{},\"finish_reason\":\"stop\"}]}\n\n" +
				"data: [DONE]\n\n"
	})
	s.Gateway = &aigateway.Gateway{
		LLM: llm, MaxHistory: 10, HistoryTTL: time.Hour, HistoryPrefix: "mb:agent:hist:",
	}

	result, err := s.GenerateReply(context.Background(), conv, "search")
	require.NoError(t, err)
	require.Equal(t, "推荐《Go 入门指南》", result.Content)
	require.NotEmpty(t, result.ToolActivities, "tool activities must be captured")
	require.NotEmpty(t, result.ToolResultData, "tool result data must be captured")
	require.Len(t, exec.calls, 1)
	require.Contains(t, exec.calls[0], "search_videos")

	// Persist the finished reply and verify DB side effects.
	msg := (&AgentGenerationService{svc: s}).persistAndPushReply(context.Background(), 18, conv, result)
	require.NotNil(t, msg)
	require.Equal(t, "assistant", msg.Role)
	require.Equal(t, "推荐《Go 入门指南》", msg.Content)
	require.NotEmpty(t, msg.ToolActivities)
	require.NotEmpty(t, msg.ToolResultData)

	var rows int64
	require.NoError(t, db.Model(&dm.DmMessage{}).Where("conversation_id = ?", conv.ID).Count(&rows).Error)
	require.Equal(t, int64(1), rows)

	var part dm.DmParticipant
	require.NoError(t, db.Where("conversation_id = ? AND user_id = ?", conv.ID, 18).First(&part).Error)
	require.Equal(t, uint32(1), part.UnreadCount)
}
