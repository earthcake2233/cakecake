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
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func mockLLM(t *testing.T, handler func(r *http.Request) (int, string)) *aigateway.Client {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		code, body := handler(r)
		w.WriteHeader(code)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return &aigateway.Client{APIKey: "test", BaseURL: srv.URL}
}

func seedGenTest(t *testing.T) (*AgentService, *dm.DmConversation) {
	t.Helper()
	db := servicetest.NewDB(t)
	store := NewAgentStore(db)
	p := &agent.AgentProfile{
		Slug: "gen", BotUserID: 14, DisplayName: "G",
		SystemPrompt: "You are helpful.", Enabled: true,
	}
	require.NoError(t, db.Create(p).Error)
	conv := &dm.DmConversation{
		UserLow: 14, UserHigh: 18, Kind: dm.DmKindAgent, AgentProfileID: p.ID,
	}
	require.NoError(t, db.Create(conv).Error)
	s := &AgentService{
		Store: store,
		Cfg: &config.C{
			DeepSeekAPIKey:      "sk-test",
			AgentEnabled:        true,
			AgentRequestTimeout: 10 * time.Second,
		},
		Log: zap.NewNop(),
	}
	return s, conv
}

func TestAgentService_GenerateReply_StreamSuccess(t *testing.T) {
	s, conv := seedGenTest(t)
	callCount := 0
	llm := mockLLM(t, func(r *http.Request) (int, string) {
		callCount++
		if callCount == 1 {
			return http.StatusOK,
				"data: {\"choices\":[{\"delta\":{\"content\":\"你\"}}]}\n\n" +
					"data: {\"choices\":[{\"delta\":{\"content\":\"好\"}}]}\n\n" +
					"data: {\"choices\":[{\"delta\":{},\"finish_reason\":\"stop\"}]}\n\n" +
					"data: [DONE]\n\n"
		}
		return http.StatusOK, `{"choices":[{"message":{"role":"assistant","content":"[\"a\",\"b\"]"}}]}`
	})
	s.Gateway = &aigateway.Gateway{
		LLM: llm, MaxHistory: 10, HistoryTTL: time.Hour, HistoryPrefix: "mb:agent:hist:",
	}

	result, err := s.GenerateReply(context.Background(), conv, "hi")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, "你好", result.Content)
	require.Empty(t, result.ToolActivities)
}

func TestAgentService_GenerateReply_WithTools(t *testing.T) {
	s, conv := seedGenTest(t)
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
			"data: {\"choices\":[{\"delta\":{\"content\":\"完成\"}}]}\n\n" +
				"data: {\"choices\":[{\"delta\":{},\"finish_reason\":\"stop\"}]}\n\n" +
				"data: [DONE]\n\n"
	})
	s.Gateway = &aigateway.Gateway{
		LLM: llm, MaxHistory: 10, HistoryTTL: time.Hour, HistoryPrefix: "mb:agent:hist:",
	}
	s.ToolExec = &fakeAgentToolExec{}

	result, err := s.GenerateReply(context.Background(), conv, "search")
	require.NoError(t, err)
	require.Equal(t, "完成", result.Content)
	require.NotEmpty(t, result.ToolActivities)
}

func TestAgentService_GenerateSuggestions_Success(t *testing.T) {
	s, _ := seedGenTest(t)
	llm := mockLLM(t, func(r *http.Request) (int, string) {
		return http.StatusOK, `{"choices":[{"message":{"role":"assistant","content":"[\"q1\",\"q2\",\"q3\"]"}}]}`
	})
	s.Gateway = &aigateway.Gateway{LLM: llm}

	got := s.GenerateSuggestions(context.Background(), "reply text")
	require.Equal(t, []string{"q1", "q2", "q3"}, got)
}

func TestAgentService_FeedbackWrappers(t *testing.T) {
	s, conv := seedGenTest(t)
	db := servicetest.NewDB(t)
	msg := &dm.DmMessage{ConversationID: conv.ID, SenderID: 14, Role: "assistant", Content: "x"}
	require.NoError(t, db.Create(msg).Error)
	s.Store = NewAgentStore(db)

	ctx := context.Background()
	require.NoError(t, s.SetMessageFeedback(ctx, msg.ID, 18, "like"))
	require.Error(t, s.SetMessageFeedback(ctx, msg.ID, 18, "bogus"))

	rows, err := s.ListAgentFeedbacks(ctx, 50, 0)
	require.NoError(t, err)
	require.Len(t, rows, 1)

	withContent, err := s.ListAgentFeedbacksWithContent(ctx, 50, 0)
	require.NoError(t, err)
	require.Len(t, withContent, 1)
	require.Equal(t, "x", withContent[0].MessageContent)
}

func TestCurrentGenID(t *testing.T) {
	s := &AgentService{}
	require.Zero(t, s.currentGenID(1))
	s.BeginGeneration(1, 7)
	require.Equal(t, uint64(7), s.currentGenID(1))
}

type fakeAgentToolExec struct{}

func (f *fakeAgentToolExec) Execute(ctx context.Context, toolName string, args json.RawMessage) (string, error) {
	return `{"items":[{"id":1,"title":"v"}]}`, nil
}

func TestSetupToolCallbacks_PushesFrames(t *testing.T) {
	hub, conn := newStateTestHub(t, 18)
	g := &aigateway.Gateway{}
	s := &AgentService{Gateway: g, ChatHub: hub, Log: zap.NewNop()}
	s.setupToolCallbacks("trace", 18)
	defer s.clearToolCallbacks()

	require.NotNil(t, g.OnToolCallStart)
	g.OnToolCallStart("trace", "trace-t0", "trace", "search_videos", json.RawMessage(`{}`))
	g.OnToolCallEnd("trace", "trace-t0", "search_videos", 12, "ok")
	g.OnToolResultData("trace", "trace-t0", "search_videos", json.RawMessage(`{"items":[]}`))

	types := map[string]bool{}
	for i := 0; i < 3; i++ {
		_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
		_, data, err := conn.ReadMessage()
		require.NoError(t, err)
		var frame struct {
			Type string `json:"type"`
		}
		require.NoError(t, json.Unmarshal(data, &frame))
		types[frame.Type] = true
	}
	require.True(t, types["tool_call_start"])
	require.True(t, types["tool_call_end"])
	require.True(t, types["tool_result_data"])
}
