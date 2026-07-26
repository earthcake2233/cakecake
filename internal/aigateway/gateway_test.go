package aigateway

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func TestGateway_CompleteUserTurn_NilGateway(t *testing.T) {
	var g *Gateway
	_, err := g.CompleteUserTurn(context.Background(), 0, "hello")
	if err == nil {
		t.Fatal("expected error for nil gateway")
	}
}

func TestGateway_CompleteUserTurn_NilLLM(t *testing.T) {
	g := &Gateway{LLM: nil, Redis: nil}
	_, err := g.CompleteUserTurn(context.Background(), 1, "hello")
	if err == nil {
		t.Fatal("expected error for nil LLM")
	}
}

func TestGateway_BuildMessages_NoRedis(t *testing.T) {
	g := &Gateway{Redis: nil}
	msgs, err := g.BuildMessages(context.Background(), 0, "hello")
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages (system+user), got %d", len(msgs))
	}
	if msgs[0].Role != "system" {
		t.Errorf("first msg role=%q", msgs[0].Role)
	}
	if msgs[1].Role != "user" || msgs[1].Content != "hello" {
		t.Errorf("second msg role=%q content=%q", msgs[1].Role, msgs[1].Content)
	}
}

func TestGateway_ClearHistory_NilGateway(t *testing.T) {
	var g *Gateway
	g.ClearHistory(context.Background(), 0)
	// should not panic
}

func TestGateway_ClearHistory_NilRedis(t *testing.T) {
	g := &Gateway{Redis: nil}
	g.ClearHistory(context.Background(), 1)
	// should not panic
}

func TestGateway_CompleteUserTurnWithTools_NilGateway(t *testing.T) {
	var g *Gateway
	_, err := g.CompleteUserTurnWithTools(context.Background(), 0, "hello", nil, "")
	if err == nil {
		t.Fatal("expected error for nil gateway")
	}
}

func TestGateway_persistHistory_NilRedis(t *testing.T) {
	g := &Gateway{Redis: nil}
	g.persistHistory(context.Background(), 1, nil)
	// should not panic
}

func TestGateway_persistHistory_WithRedis(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}
	defer mr.Close()
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	g := &Gateway{Redis: rdb, MaxHistory: 10, HistoryTTL: time.Hour, HistoryPrefix: "chat:"}
	// persist with empty messages should not error
	g.persistHistory(context.Background(), 42, nil)
	g.persistHistory(context.Background(), 42, []ChatMessage{})
}

func TestGateway_executeToolCalls_NilGateway(t *testing.T) {
	var g *Gateway
	msgs := g.executeToolCalls(context.Background(), nil, "trace-1", 0)
	if len(msgs) != 0 {
		t.Fatalf("expected 0, got %d", len(msgs))
	}
}

func TestGateway_executeToolCalls_NilToolExec(t *testing.T) {
	g := &Gateway{ToolExec: nil}
	calls := []ToolCall{{Function: ToolCallFunction{Name: "test_func", Arguments: "{}"}}}
	msgs := g.executeToolCalls(context.Background(), calls, "trace-1", 0)
	if len(msgs) != 1 {
		t.Fatalf("expected 1 error tool msg, got %d", len(msgs))
	}
	if msgs[0].Role != "tool" {
		t.Fatalf("expected tool role, got %s", msgs[0].Role)
	}
}
