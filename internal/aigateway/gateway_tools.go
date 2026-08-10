package aigateway

import (
	"cakecake/internal/logger"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"go.uber.org/zap"
)

// ToolRunner owns tool-call execution and its progress callbacks.
type ToolRunner struct {
	gw *Gateway
}

func (t *ToolRunner) executeToolCalls(ctx context.Context, calls []ToolCall, traceID string, round int) []ChatMessage {
	type result struct {
		msg ChatMessage
	}
	ch := make(chan result, len(calls))

	for i, call := range calls {
		go func(idx int, tc ToolCall) {
			spanID := fmt.Sprintf("%s-r%d-t%d", traceID, round, idx)
			parentSpanID := traceID

			if t.gw.OnToolCallStart != nil {
				var raw json.RawMessage
				if err := json.Unmarshal([]byte(tc.Function.Arguments), &raw); err != nil && logger.L != nil {
					logger.L.Warn("aigateway: parse tool call args failed",
						zap.String("trace_id", traceID),
						zap.String("tool", tc.Function.Name), zap.Error(err))
				}
				t.gw.OnToolCallStart(traceID, spanID, parentSpanID, tc.Function.Name, raw)
			}

			start := time.Now()
			var res string
			status := "ok"
			if t.gw.ToolExec != nil {
				r, err := t.gw.ToolExec.Execute(ctx, tc.Function.Name, json.RawMessage(tc.Function.Arguments))
				if err != nil {
					status = "error"
					res = fmt.Sprintf(`{"error": "%s"}`, err.Error())
				} else {
					res = r
				}
			} else {
				status = "error"
				res = `{"error": "tool executor not available"}`
			}
			elapsed := time.Since(start)
			RecordToolCall(tc.Function.Name, status, elapsed)
			duration := elapsed.Milliseconds()

			if t.gw.OnToolCallEnd != nil {
				summary := res
				if len(summary) > 80 {
					summary = summary[:80] + "..."
				}
				t.gw.OnToolCallEnd(traceID, spanID, tc.Function.Name, duration, summary)
			}

			if t.gw.OnToolResultData != nil && res != "" {
				var parsed map[string]json.RawMessage
				if json.Unmarshal([]byte(res), &parsed) == nil {
					if items, ok := parsed["items"]; ok && len(items) > 0 && items[0] == '[' {
						t.gw.OnToolResultData(traceID, spanID, tc.Function.Name, items)
					}
				}
			}

			ch <- result{
				msg: ChatMessage{
					Role:       "tool",
					ToolCallID: tc.ID,
					Content:    res,
				},
			}
		}(i, call)
	}

	msgs := make([]ChatMessage, 0, len(calls))
	for i := 0; i < len(calls); i++ {
		r := <-ch
		msgs = append(msgs, r.msg)
	}
	return msgs
}
