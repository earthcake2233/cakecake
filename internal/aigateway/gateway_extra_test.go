package aigateway

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func newGatewayTestRedis(t *testing.T) (*miniredis.Miniredis, *redis.Client) {
	t.Helper()
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)
	return mr, redis.NewClient(&redis.Options{Addr: mr.Addr()})
}

func mockLLMClient(t *testing.T, handler func(r *http.Request) (int, string)) *Client {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		code, body := handler(r)
		w.WriteHeader(code)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return &Client{APIKey: "test-key", BaseURL: srv.URL}
}

func TestCompleteUserTurn_Success(t *testing.T) {
	_, rdb := newGatewayTestRedis(t)
	llm := mockLLMClient(t, func(r *http.Request) (int, string) {
		return http.StatusOK, `{"choices":[{"message":{"role":"assistant","content":"Hello!"}}]}`
	})
	g := &Gateway{LLM: llm, Redis: rdb, MaxHistory: 10, HistoryTTL: time.Hour, HistoryPrefix: "chat:"}
	reply, err := g.CompleteUserTurn(context.Background(), 42, "hi")
	require.NoError(t, err)
	require.Equal(t, "Hello!", reply)

	// History persisted (system excluded).
	raw, err := rdb.Get(context.Background(), "chat:42").Result()
	require.NoError(t, err)
	require.NotEmpty(t, raw)
}

func TestCompleteUserTurn_LLMError(t *testing.T) {
	llm := mockLLMClient(t, func(r *http.Request) (int, string) {
		return http.StatusBadRequest, `bad`
	})
	g := &Gateway{LLM: llm}
	_, err := g.CompleteUserTurn(context.Background(), 1, "hi")
	require.Error(t, err)
}

func TestCompleteUserTurn_EmptyReply(t *testing.T) {
	llm := mockLLMClient(t, func(r *http.Request) (int, string) {
		return http.StatusOK, `{"choices":[{"message":{"role":"assistant","content":"   "}}]}`
	})
	g := &Gateway{LLM: llm}
	_, err := g.CompleteUserTurn(context.Background(), 1, "hi")
	require.Error(t, err)
}

func TestCompleteUserTurnWithTools_ToolLoop(t *testing.T) {
	_, rdb := newGatewayTestRedis(t)
	callCount := 0
	llm := mockLLMClient(t, func(r *http.Request) (int, string) {
		callCount++
		if callCount == 1 {
			// First turn requests a tool call.
			return http.StatusOK, `{"choices":[{"message":{"role":"assistant","content":"","tool_calls":[{"id":"call1","type":"function","function":{"name":"get_video_detail","arguments":"{\"video_id\":10}"}}]}}]}`
		}
		return http.StatusOK, `{"choices":[{"message":{"role":"assistant","content":"done"}}]}`
	})

	exec := &fakeToolExec{}
	g := &Gateway{
		LLM: llm, Redis: rdb, MaxHistory: 10, HistoryTTL: time.Hour, HistoryPrefix: "chat:",
		ToolExec: exec,
	}
	reply, err := g.CompleteUserTurnWithTools(context.Background(), 7, "show me",
		[]ToolDef{{Type: "function", Function: ToolFuncDef{Name: "get_video_detail"}}}, "trace-1")
	require.NoError(t, err)
	require.Equal(t, "done", reply)
	require.Equal(t, 1, exec.calls)
}

func TestCompleteUserTurnWithTools_NoToolExec(t *testing.T) {
	llm := mockLLMClient(t, func(r *http.Request) (int, string) {
		return http.StatusOK, `{"choices":[{"message":{"role":"assistant","content":"","tool_calls":[{"id":"call1","type":"function","function":{"name":"x","arguments":"{}"}}]}}]}`
	})
	g := &Gateway{LLM: llm}
	_, err := g.CompleteUserTurnWithTools(context.Background(), 1, "hi", []ToolDef{}, "t")
	require.Error(t, err)
}

func TestExecuteToolCalls_CallbacksAndErrors(t *testing.T) {
	started := 0
	ended := 0
	resultData := 0
	g := &Gateway{
		ToolExec: &fakeToolExec{},
		OnToolCallStart: func(traceID, spanID, parentSpanID, toolName string, args json.RawMessage) {
			started++
		},
		OnToolCallEnd: func(traceID, spanID, toolName string, durationMs int64, summary string) {
			ended++
		},
		OnToolResultData: func(traceID, spanID, toolName string, items json.RawMessage) {
			resultData++
		},
	}
	msgs := g.executeToolCalls(context.Background(), []ToolCall{
		{ID: "c1", Function: ToolCallFunction{Name: "search_videos", Arguments: `{"keyword":"go"}`}},
		{ID: "c2", Function: ToolCallFunction{Name: "boom", Arguments: `{}`}},
	}, "trace-1", 0)
	require.Len(t, msgs, 2)
	require.Equal(t, "tool", msgs[0].Role)
	require.Equal(t, 2, started)
	require.Equal(t, 2, ended)
	require.Equal(t, 1, resultData)
}

func TestBuildMessages_WithHistory(t *testing.T) {
	_, rdb := newGatewayTestRedis(t)
	g := &Gateway{Redis: rdb, MaxHistory: 10, HistoryTTL: time.Hour, HistoryPrefix: "chat:"}
	hist := []historyEntry{
		{Role: "user", Content: "old question"},
		{Role: "assistant", Content: "old answer"},
		{Role: "tool", Content: `{"x":1}`},
	}
	b, _ := json.Marshal(hist)
	require.NoError(t, rdb.Set(context.Background(), "chat:5", b, time.Hour).Err())

	msgs, err := g.BuildMessages(context.Background(), 5, "new question")
	require.NoError(t, err)
	require.Len(t, msgs, 5) // system + 3 history + user
	require.Equal(t, "system", msgs[0].Role)
	require.Equal(t, "user", msgs[len(msgs)-1].Role)
	require.Equal(t, "new question", msgs[len(msgs)-1].Content)
}

func TestClearHistory_WithRedis(t *testing.T) {
	_, rdb := newGatewayTestRedis(t)
	ctx := context.Background()
	require.NoError(t, rdb.Set(ctx, "chat:9", "[]", time.Hour).Err())
	g := &Gateway{Redis: rdb, HistoryPrefix: "chat:"}
	g.ClearHistory(ctx, 9)
	require.Equal(t, int64(0), rdb.Exists(ctx, "chat:9").Val())
}

func TestPersistHistory_TrimsToCap(t *testing.T) {
	_, rdb := newGatewayTestRedis(t)
	g := &Gateway{Redis: rdb, MaxHistory: 1, HistoryTTL: time.Hour, HistoryPrefix: "chat:"}
	var msgs []ChatMessage
	for i := 0; i < 20; i++ {
		msgs = append(msgs, ChatMessage{Role: "user", Content: "u"}, ChatMessage{Role: "assistant", Content: "a"})
	}
	g.persistHistory(context.Background(), 3, msgs)
	var hist []historyEntry
	raw, err := rdb.Get(context.Background(), "chat:3").Result()
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal([]byte(raw), &hist))
	require.LessOrEqual(t, len(hist), 8) // max*8
}

type fakeToolExec struct {
	calls int
}

func (f *fakeToolExec) Execute(ctx context.Context, toolName string, args json.RawMessage) (string, error) {
	f.calls++
	if toolName == "boom" {
		return "", jsonError("boom")
	}
	return `{"items":[{"id":1,"title":"v"}]}`, nil
}

func jsonError(s string) error { return &jsonErr{s} }

type jsonErr struct{ s string }

func (e *jsonErr) Error() string { return e.s }
