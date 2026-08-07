package aigateway

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestGateway_CompleteUserTurnStream_Deltas(t *testing.T) {
	_, rdb := newGatewayTestRedis(t)
	llm := mockLLMClient(t, func(r *http.Request) (int, string) {
		return http.StatusOK,
			"data: {\"choices\":[{\"delta\":{\"content\":\"你\"}}]}\n\n" +
				"data: {\"choices\":[{\"delta\":{\"content\":\"好\"}}]}\n\n" +
				"data: {\"choices\":[{\"delta\":{},\"finish_reason\":\"stop\"}]}\n\n" +
				"data: [DONE]\n\n"
	})
	g := &Gateway{LLM: llm, Redis: rdb, MaxHistory: 10, HistoryTTL: time.Hour, HistoryPrefix: "chat:"}

	var got []string
	reply, err := g.CompleteUserTurnStream(context.Background(), 42, "hi", func(d string) {
		got = append(got, d)
	})
	require.NoError(t, err)
	require.Equal(t, "你好", reply)
	require.Equal(t, []string{"你", "好"}, got)

	raw, err := rdb.Get(context.Background(), "chat:42").Result()
	require.NoError(t, err)
	require.Contains(t, raw, "你好")
}

func TestGateway_CompleteUserTurnStream_NilLLM(t *testing.T) {
	g := &Gateway{}
	_, err := g.CompleteUserTurnStream(context.Background(), 1, "hi", nil)
	require.Error(t, err)
}

func TestGateway_CompleteUserTurnStream_EmptyReply(t *testing.T) {
	llm := mockLLMClient(t, func(r *http.Request) (int, string) {
		return http.StatusOK,
			"data: {\"choices\":[{\"delta\":{},\"finish_reason\":\"stop\"}]}\n\n" +
				"data: [DONE]\n\n"
	})
	g := &Gateway{LLM: llm}
	_, err := g.CompleteUserTurnStream(context.Background(), 1, "hi", nil)
	require.Error(t, err)
}

func TestGateway_ContinueTurnStream(t *testing.T) {
	_, rdb := newGatewayTestRedis(t)
	llm := mockLLMClient(t, func(r *http.Request) (int, string) {
		return http.StatusOK,
			"data: {\"choices\":[{\"delta\":{\"content\":\"继续\"}}]}\n\n" +
				"data: {\"choices\":[{\"delta\":{},\"finish_reason\":\"stop\"}]}\n\n" +
				"data: [DONE]\n\n"
	})
	g := &Gateway{LLM: llm, Redis: rdb, MaxHistory: 10, HistoryTTL: time.Hour, HistoryPrefix: "chat:"}

	var got []string
	msg, err := g.ContinueTurnStream(context.Background(), 7, "前半部分", "请继续", func(d string) {
		got = append(got, d)
	})
	require.NoError(t, err)
	require.Equal(t, "继续", msg.Content)
	require.Equal(t, []string{"继续"}, got)

	raw, err := rdb.Get(context.Background(), "chat:7").Result()
	require.NoError(t, err)
	require.Contains(t, raw, "前半部分")
	require.Contains(t, raw, "继续")
}

func TestGateway_ContinueTurnStream_EmptyContinuation(t *testing.T) {
	llm := mockLLMClient(t, func(r *http.Request) (int, string) {
		return http.StatusOK,
			"data: {\"choices\":[{\"delta\":{},\"finish_reason\":\"stop\"}]}\n\n" +
				"data: [DONE]\n\n"
	})
	g := &Gateway{LLM: llm}
	_, err := g.ContinueTurnStream(context.Background(), 7, "abc", "继续", nil)
	require.Error(t, err)
}

func TestGateway_CompleteUserTurnWithToolsStream_ToolThenText(t *testing.T) {
	_, rdb := newGatewayTestRedis(t)
	callCount := 0
	llm := mockLLMClient(t, func(r *http.Request) (int, string) {
		callCount++
		if callCount == 1 {
			return http.StatusOK,
				"data: {\"choices\":[{\"delta\":{\"tool_calls\":[{\"index\":0,\"id\":\"call_1\",\"function\":{\"name\":\"get_video_detail\",\"arguments\":\"{\\\"video_id\\\":\"}}]}}]}\n\n" +
					"data: {\"choices\":[{\"delta\":{\"tool_calls\":[{\"index\":0,\"function\":{\"arguments\":\"10}\"}}]}}]}\n\n" +
					"data: {\"choices\":[{\"delta\":{},\"finish_reason\":\"tool_calls\"}]}\n\n" +
					"data: [DONE]\n\n"
		}
		return http.StatusOK,
			"data: {\"choices\":[{\"delta\":{\"content\":\"完成\"}}]}\n\n" +
				"data: {\"choices\":[{\"delta\":{},\"finish_reason\":\"stop\"}]}\n\n" +
				"data: [DONE]\n\n"
	})

	exec := &fakeToolExec{}
	g := &Gateway{
		LLM: llm, Redis: rdb, MaxHistory: 10, HistoryTTL: time.Hour, HistoryPrefix: "chat:",
		ToolExec: exec,
	}
	var starts, ends int
	g.OnToolCallStart = func(tid, spanID, parentSpanID, toolName string, argsJSON json.RawMessage) {
		starts++
	}
	g.OnToolCallEnd = func(tid, spanID, toolName string, durationMs int64, resultSummary string) {
		ends++
	}

	var got []string
	reply, err := g.CompleteUserTurnWithToolsStream(
		context.Background(), 7, "show me",
		[]ToolDef{{Type: "function", Function: ToolFuncDef{Name: "get_video_detail"}}},
		"trace-1",
		func(d string) { got = append(got, d) },
	)
	require.NoError(t, err)
	require.Equal(t, "完成", reply)
	require.Equal(t, []string{"完成"}, got)
	require.Equal(t, 2, callCount)
	require.Equal(t, 1, exec.calls)
	require.Equal(t, 1, starts)
	require.Equal(t, 1, ends)

	raw, err := rdb.Get(context.Background(), "chat:7").Result()
	require.NoError(t, err)
	require.Contains(t, raw, "完成")
}

func TestGateway_CompleteUserTurnWithToolsStream_NoToolExec(t *testing.T) {
	llm := mockLLMClient(t, func(r *http.Request) (int, string) {
		return http.StatusOK,
			"data: {\"choices\":[{\"delta\":{\"tool_calls\":[{\"index\":0,\"id\":\"c1\",\"function\":{\"name\":\"get_video_detail\",\"arguments\":\"{}\"}}]}}]}\n\n" +
				"data: {\"choices\":[{\"delta\":{},\"finish_reason\":\"tool_calls\"}]}\n\n" +
				"data: [DONE]\n\n"
	})
	g := &Gateway{LLM: llm}
	_, err := g.CompleteUserTurnWithToolsStream(
		context.Background(), 1, "x",
		[]ToolDef{{Type: "function", Function: ToolFuncDef{Name: "get_video_detail"}}},
		"trace",
		nil,
	)
	require.Error(t, err)
}
