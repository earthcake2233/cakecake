package aigateway

import (
	"context"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// AI gateway observability: Prometheus metrics plus a request-scoped usage
// sink so the agent service can attribute token cost per user/day.

var (
	llmRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "cakecake", Subsystem: "llm",
		Name: "requests_total", Help: "LLM HTTP requests by status (ok/error).",
	}, []string{"status"})
	llmFirstTokenSeconds = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "cakecake", Subsystem: "llm",
		Name:    "first_token_seconds",
		Help:    "Time from request start to the first streamed content/tool delta.",
		Buckets: []float64{0.1, 0.25, 0.5, 1, 2.5, 5, 10, 30},
	}, []string{})
	llmTokensTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "cakecake", Subsystem: "llm",
		Name: "tokens_total", Help: "LLM tokens consumed by type (prompt/completion).",
	}, []string{"type"})
	llmCostUSDTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "cakecake", Subsystem: "llm",
		Name: "cost_usd_total", Help: "Estimated LLM cost in USD by user and day (YYYYMMDD).",
	}, []string{"user", "date"})
	toolCallsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "cakecake", Subsystem: "agent",
		Name: "tool_calls_total", Help: "Tool calls by tool and status (ok/error).",
	}, []string{"tool", "status"})
	toolCallDurationSeconds = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "cakecake", Subsystem: "agent",
		Name:    "tool_call_seconds",
		Help:    "Tool execution duration.",
		Buckets: prometheus.DefBuckets,
	}, []string{"tool"})
	agentControlsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "cakecake", Subsystem: "agent",
		Name: "controls_total", Help: "Generation controls by type (pause/continue/regenerate).",
	}, []string{"type"})
)

// Estimated USD cost per million tokens; tunable for the deployed model
// (defaults approximate DeepSeek v4-flash pricing).
const (
	costUSDPerMillionPrompt     = 0.2
	costUSDPerMillionCompletion = 0.8
)

// Usage is an OpenAI-compatible token usage block.
type Usage struct {
	PromptTokens     int64 `json:"prompt_tokens"`
	CompletionTokens int64 `json:"completion_tokens"`
	TotalTokens      int64 `json:"total_tokens"`
}

func (u *Usage) costUSD() float64 {
	if u == nil {
		return 0
	}
	return float64(u.PromptTokens)/1e6*costUSDPerMillionPrompt +
		float64(u.CompletionTokens)/1e6*costUSDPerMillionCompletion
}

// UsageSink is a request-scoped hook invoked when an LLM response reports its
// token usage; the agent service uses it for per-user/per-day cost accounting.
type UsageSink struct {
	OnUsage func(u Usage)
}

type usageSinkKey struct{}

// WithUsageSink attaches a per-request usage sink to the context.
func WithUsageSink(ctx context.Context, sink *UsageSink) context.Context {
	if ctx == nil || sink == nil {
		return ctx
	}
	return context.WithValue(ctx, usageSinkKey{}, sink)
}

func usageSinkFromContext(ctx context.Context) *UsageSink {
	if ctx == nil {
		return nil
	}
	if s, ok := ctx.Value(usageSinkKey{}).(*UsageSink); ok {
		return s
	}
	return nil
}

func recordLLMUsage(ctx context.Context, u *Usage) {
	if u == nil {
		return
	}
	llmTokensTotal.WithLabelValues("prompt").Add(float64(u.PromptTokens))
	llmTokensTotal.WithLabelValues("completion").Add(float64(u.CompletionTokens))
	if sink := usageSinkFromContext(ctx); sink != nil && sink.OnUsage != nil {
		sink.OnUsage(*u)
	}
}

// RecordUserCost attributes token cost to a user for the current day.
func RecordUserCost(userID uint64, u Usage) {
	if userID == 0 {
		return
	}
	day := time.Now().Format("20060102")
	llmCostUSDTotal.WithLabelValues(strconv.FormatUint(userID, 10), day).Add(u.costUSD())
}

// RecordLLMRequest records an LLM request result (ok/error).
func RecordLLMRequest(status string) {
	llmRequestsTotal.WithLabelValues(status).Inc()
}

// ObserveFirstToken records the time-to-first-token of a streamed response.
func ObserveFirstToken(seconds float64) {
	llmFirstTokenSeconds.WithLabelValues().Observe(seconds)
}

// RecordToolCall records a tool execution result and duration.
func RecordToolCall(tool, status string, duration time.Duration) {
	toolCallsTotal.WithLabelValues(tool, status).Inc()
	toolCallDurationSeconds.WithLabelValues(tool).Observe(duration.Seconds())
}

// IncAgentControl records a generation control action.
func IncAgentControl(kind string) {
	agentControlsTotal.WithLabelValues(kind).Inc()
}
