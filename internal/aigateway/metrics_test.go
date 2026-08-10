package aigateway

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"
)

func resetLLMMetrics() {
	llmRequestsTotal.Reset()
	llmTokensTotal.Reset()
	llmFirstTokenSeconds.Reset()
	llmCostUSDTotal.Reset()
}

func TestComplete_ParsesUsageAndInvokesSink(t *testing.T) {
	resetLLMMetrics()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"ok"}}],"usage":{"prompt_tokens":10,"completion_tokens":5,"total_tokens":15}}`))
	}))
	defer srv.Close()

	var got *Usage
	ctx := WithUsageSink(context.Background(), &UsageSink{OnUsage: func(u Usage) { got = &u }})
	c := &Client{APIKey: "k", BaseURL: srv.URL}
	_, err := c.Complete(ctx, []ChatMessage{{Role: "user", Content: "hi"}})
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, int64(10), got.PromptTokens)
	require.Equal(t, int64(5), got.CompletionTokens)
	require.Equal(t, 10.0, testutil.ToFloat64(llmTokensTotal.WithLabelValues("prompt")))
	require.Equal(t, 5.0, testutil.ToFloat64(llmTokensTotal.WithLabelValues("completion")))
	require.Equal(t, 1.0, testutil.ToFloat64(llmRequestsTotal.WithLabelValues("ok")))
}

func TestComplete_RecordsErrorStatus(t *testing.T) {
	resetLLMMetrics()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":{"message":"boom"}}`))
	}))
	defer srv.Close()

	c := &Client{APIKey: "k", BaseURL: srv.URL}
	_, err := c.Complete(context.Background(), []ChatMessage{{Role: "user", Content: "hi"}})
	require.Error(t, err)
	require.Equal(t, 1.0, testutil.ToFloat64(llmRequestsTotal.WithLabelValues("error")))
}

func TestCompleteStream_FirstTokenAndUsage(t *testing.T) {
	resetLLMMetrics()
	body := "" +
		"data: {\"choices\":[{\"delta\":{\"content\":\"你\"}}]}\n\n" +
		"data: {\"choices\":[{\"delta\":{\"content\":\"好\"}}]}\n\n" +
		"data: {\"choices\":[{\"delta\":{},\"finish_reason\":\"stop\"}]}\n\n" +
		"data: {\"usage\":{\"prompt_tokens\":7,\"completion_tokens\":3,\"total_tokens\":10}}\n\n" +
		"data: [DONE]\n\n"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		require.Contains(t, string(raw), `"stream_options":{"include_usage":true}`)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()

	var got *Usage
	ctx := WithUsageSink(context.Background(), &UsageSink{OnUsage: func(u Usage) { got = &u }})
	c := &Client{APIKey: "k", BaseURL: srv.URL}
	msg, err := c.CompleteStream(ctx, []ChatMessage{{Role: "user", Content: "hi"}}, func(string) {})
	require.NoError(t, err)
	require.Equal(t, "你好", msg.Content)
	require.NotNil(t, got)
	require.Equal(t, int64(3), got.CompletionTokens)
	require.Equal(t, 7.0, testutil.ToFloat64(llmTokensTotal.WithLabelValues("prompt")))
	require.Equal(t, 1, testutil.CollectAndCount(llmFirstTokenSeconds))
}

func TestRecordToolCallMetrics(t *testing.T) {
	resetLLMMetrics()
	toolCallsTotal.Reset()
	toolCallDurationSeconds.Reset()

	RecordToolCall("search_videos", "ok", 80*time.Millisecond)
	RecordToolCall("search_videos", "error", 120*time.Millisecond)

	require.Equal(t, 1.0, testutil.ToFloat64(toolCallsTotal.WithLabelValues("search_videos", "ok")))
	require.Equal(t, 1.0, testutil.ToFloat64(toolCallsTotal.WithLabelValues("search_videos", "error")))
	require.Equal(t, 1, testutil.CollectAndCount(toolCallDurationSeconds))
}

func TestRecordUserCostLabelsPerUserAndDate(t *testing.T) {
	resetLLMMetrics()
	llmCostUSDTotal.Reset()

	RecordUserCost(42, Usage{PromptTokens: 1_000_000, CompletionTokens: 0})

	require.Equal(t, 0.2, testutil.ToFloat64(llmCostUSDTotal.WithLabelValues("42", time.Now().Format("20060102"))))
}

func TestUsageCostUSD(t *testing.T) {
	u := &Usage{PromptTokens: 1_000_000, CompletionTokens: 1_000_000}
	require.InDelta(t, 1.0, u.costUSD(), 0.0001)
}

func TestUsageSinkContext(t *testing.T) {
	require.Nil(t, usageSinkFromContext(context.Background()))
	sink := &UsageSink{OnUsage: func(Usage) {}}
	require.Same(t, sink, usageSinkFromContext(WithUsageSink(context.Background(), sink)))
}
