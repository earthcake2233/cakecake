package toolkit

import (
	"context"
	"errors"
	"encoding/json"
	"testing"

	"minibili/internal/aigateway"
)

type mockExecutor struct {
	fn func(ctx context.Context, toolName string, args json.RawMessage) (string, error)
}

func (m *mockExecutor) Execute(ctx context.Context, toolName string, args json.RawMessage) (string, error) {
	return m.fn(ctx, toolName, args)
}

func TestExecuteToolCalls_NilExec(t *testing.T) {
	msgs := ExecuteToolCalls(context.Background(), nil, []aigateway.ToolCall{
		{ID: "call1", Function: aigateway.ToolCallFunction{Name: "test", Arguments: "{}"}},
	})
	if msgs != nil {
		t.Errorf("expected nil, got %v", msgs)
	}
}

func TestExecuteToolCalls_EmptyCalls(t *testing.T) {
	exec := &mockExecutor{fn: func(_ context.Context, _ string, _ json.RawMessage) (string, error) {
		return "ok", nil
	}}
	msgs := ExecuteToolCalls(context.Background(), exec, nil)
	if msgs != nil {
		t.Errorf("expected nil, got %v", msgs)
	}
}

func TestExecuteToolCalls_Success(t *testing.T) {
	exec := &mockExecutor{fn: func(_ context.Context, toolName string, _ json.RawMessage) (string, error) {
		if toolName == "get_weather" {
			return "sunny", nil
		}
		return "unknown", nil
	}}
	msgs := ExecuteToolCalls(context.Background(), exec, []aigateway.ToolCall{
		{ID: "call1", Function: aigateway.ToolCallFunction{Name: "get_weather", Arguments: "{}"}},
	})
	if len(msgs) != 1 {
		t.Fatalf("expected 1 msg, got %d", len(msgs))
	}
	if msgs[0].Role != "tool" {
		t.Errorf("expected role tool, got %s", msgs[0].Role)
	}
	if msgs[0].ToolCallID != "call1" {
		t.Errorf("expected call1, got %s", msgs[0].ToolCallID)
	}
	if msgs[0].Content != "sunny" {
		t.Errorf("expected sunny, got %s", msgs[0].Content)
	}
}

func TestExecuteToolCalls_Error(t *testing.T) {
	exec := &mockExecutor{fn: func(_ context.Context, _ string, _ json.RawMessage) (string, error) {
		return "", errors.New("syntax error")
	}}
	msgs := ExecuteToolCalls(context.Background(), exec, []aigateway.ToolCall{
		{ID: "err1", Function: aigateway.ToolCallFunction{Name: "fail", Arguments: "{}"}},
	})
	if len(msgs) != 1 {
		t.Fatalf("expected 1 msg, got %d", len(msgs))
	}
	if msgs[0].Role != "tool" {
		t.Errorf("expected role tool, got %s", msgs[0].Role)
	}
	if msgs[0].Content == "" {
		t.Error("expected error message in content")
	}
}

func TestExecuteToolCalls_EmptyName(t *testing.T) {
	msgs := ExecuteToolCalls(context.Background(), &mockExecutor{
		fn: func(_ context.Context, _ string, _ json.RawMessage) (string, error) { return "ok", nil },
	}, []aigateway.ToolCall{
		{ID: "empty1", Function: aigateway.ToolCallFunction{Name: "", Arguments: "{}"}},
	})
	if len(msgs) != 1 {
		t.Fatalf("expected 1 msg, got %d", len(msgs))
	}
	if msgs[0].Content == "ok" {
		t.Error("expected error for empty name, but got ok")
	}
}

func TestExecuteToolCalls_EmptyArgs(t *testing.T) {
	msgs := ExecuteToolCalls(context.Background(), &mockExecutor{
		fn: func(_ context.Context, _ string, _ json.RawMessage) (string, error) { return "ok", nil },
	}, []aigateway.ToolCall{
		{ID: "noargs", Function: aigateway.ToolCallFunction{Name: "test", Arguments: ""}},
	})
	if len(msgs) != 1 {
		t.Fatalf("expected 1 msg, got %d", len(msgs))
	}
	if msgs[0].Content == "ok" {
		t.Error("expected error for empty args, but got ok")
	}
}

func TestExecuteToolCalls_MultipleCalls(t *testing.T) {
	exec := &mockExecutor{fn: func(_ context.Context, toolName string, _ json.RawMessage) (string, error) {
		if toolName == "a" {
			return "result_a", nil
		}
		return "result_b", nil
	}}
	msgs := ExecuteToolCalls(context.Background(), exec, []aigateway.ToolCall{
		{ID: "c1", Function: aigateway.ToolCallFunction{Name: "a", Arguments: "{}"}},
		{ID: "c2", Function: aigateway.ToolCallFunction{Name: "b", Arguments: "{}"}},
	})
	if len(msgs) != 2 {
		t.Fatalf("expected 2 msgs, got %d", len(msgs))
	}
	// Results may come in any order due to goroutines
	seen := make(map[string]bool)
	for _, m := range msgs {
		seen[m.Content] = true
	}
	if !seen["result_a"] {
		t.Error("missing result_a")
	}
	if !seen["result_b"] {
		t.Error("missing result_b")
	}
}
