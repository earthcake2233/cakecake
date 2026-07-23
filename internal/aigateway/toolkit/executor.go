package toolkit

import (
	"context"
	"encoding/json"
	"fmt"

	"minibili/internal/aigateway"
)

// Executor runs a tool call and returns a result string.
type Executor interface {
	Execute(ctx context.Context, toolName string, args json.RawMessage) (string, error)
}

// ExecuteToolCalls executes all tool calls in parallel and returns tool-role messages.
// Each result is wrapped as a ChatMessage with role "tool".
func ExecuteToolCalls(ctx context.Context, exec Executor, calls []aigateway.ToolCall) []aigateway.ChatMessage {
	if exec == nil || len(calls) == 0 {
		return nil
	}
	type result struct {
		callID string
		msg    aigateway.ChatMessage
	}
	ch := make(chan result, len(calls))

	for _, call := range calls {
		go func(tc aigateway.ToolCall) {
			var res string
			if tc.Function.Name == "" || tc.Function.Arguments == "" {
				res = `{"error": "invalid tool call: missing name or arguments"}`
			} else {
				raw := json.RawMessage(tc.Function.Arguments)
				r, err := exec.Execute(ctx, tc.Function.Name, raw)
				if err != nil {
					res = fmt.Sprintf(`{"error": "%s", "tool": "%s"}`, err.Error(), tc.Function.Name)
				} else {
					res = r
				}
			}
			ch <- result{
				callID: tc.ID,
				msg: aigateway.ChatMessage{
					Role:       "tool",
					ToolCallID: tc.ID,
					Content:    res,
				},
			}
		}(call)
	}

	msgs := make([]aigateway.ChatMessage, 0, len(calls))
	for i := 0; i < len(calls); i++ {
		r := <-ch
		msgs = append(msgs, r.msg)
		_ = r.callID // available for ordering if needed
	}
	return msgs
}