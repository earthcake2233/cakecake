package aigateway

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestClient_CompleteStream_TextDeltas(t *testing.T) {
	llm := mockLLMClient(t, func(r *http.Request) (int, string) {
		body := "data: {\"choices\":[{\"delta\":{\"content\":\"你\"}}]}\n\n" +
			"data: {\"choices\":[{\"delta\":{\"content\":\"好\"}}]}\n\n" +
			"data: {\"choices\":[{\"delta\":{},\"finish_reason\":\"stop\"}]}\n\n" +
			"data: [DONE]\n\n"
		return http.StatusOK, body
	})

	var got []string
	msg, err := llm.CompleteStream(
		context.Background(),
		[]ChatMessage{{Role: "user", Content: "hi"}},
		func(d string) { got = append(got, d) },
	)
	require.NoError(t, err)
	require.Equal(t, "你好", msg.Content)
	require.Equal(t, []string{"你", "好"}, got)
	require.Equal(t, "stop", msg.FinishReason())
}

func TestClient_CompleteStream_ToolCallFragments(t *testing.T) {
	llm := mockLLMClient(t, func(r *http.Request) (int, string) {
		body := "data: {\"choices\":[{\"delta\":{\"tool_calls\":[{\"index\":0,\"id\":\"call_1\",\"function\":{\"name\":\"get_video_detail\",\"arguments\":\"{\\\"video_id\\\":\"}}]}}]}\n\n" +
			"data: {\"choices\":[{\"delta\":{\"tool_calls\":[{\"index\":0,\"function\":{\"arguments\":\"10}\"}}]}}]}\n\n" +
			"data: {\"choices\":[{\"delta\":{},\"finish_reason\":\"tool_calls\"}]}\n\n" +
			"data: [DONE]\n\n"
		return http.StatusOK, body
	})

	msg, err := llm.CompleteWithToolsStream(
		context.Background(),
		[]ChatMessage{{Role: "user", Content: "hi"}},
		[]ToolDef{},
		nil,
	)
	require.NoError(t, err)
	require.Len(t, msg.ToolCalls, 1)
	require.Equal(t, "call_1", msg.ToolCalls[0].ID)
	require.Equal(t, "get_video_detail", msg.ToolCalls[0].Function.Name)
	require.Equal(t, `{"video_id":10}`, msg.ToolCalls[0].Function.Arguments)
	require.Equal(t, "tool_calls", msg.FinishReason())
}
