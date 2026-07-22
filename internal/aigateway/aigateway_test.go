package aigateway

import (
	"context"
	"strings"
	"testing"
)

func TestClientCompleteNoAPIKey(t *testing.T) {
	c := &Client{}
	_, err := c.Complete(context.Background(), []ChatMessage{{Role: "user", Content: "hi"}})
	if err == nil {
		t.Fatal("expected error for empty API key")
	}
	if !strings.Contains(err.Error(), "api key not configured") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		input string
		n     int
		want  string
	}{
		{"hello", 10, "hello"},
		{"hello", 5, "hello"},
		{"hello world", 5, "hello…"},
		{"", 5, ""},
		{"   spaced   ", 10, "spaced"},
	}
	for _, tc := range tests {
		got := truncate(tc.input, tc.n)
		if got != tc.want {
			t.Errorf("truncate(%q, %d) = %q, want %q", tc.input, tc.n, got, tc.want)
		}
	}
}

func TestGatewayHistoryKey(t *testing.T) {
	g := &Gateway{HistoryPrefix: "test:pfx:"}
	key := g.historyKey(42)
	want := "test:pfx:42"
	if key != want {
		t.Errorf("historyKey = %q, want %q", key, want)
	}

	g2 := &Gateway{}
	key2 := g2.historyKey(42)
	want2 := "mb:agent:hist:42"
	if key2 != want2 {
		t.Errorf("historyKey = %q, want %q", key2, want2)
	}
}

func TestGatewayHistoryTTL(t *testing.T) {
	g := &Gateway{}
	ttl := g.historyTTL()
	if ttl == 0 {
		t.Error("expected non-zero default TTL")
	}

	g2 := &Gateway{}
	ttl2 := g2.historyTTL()
	if ttl2 <= 0 {
		t.Error("expected positive TTL")
	}
}

func TestGatewaySystemPrompt(t *testing.T) {
	g := &Gateway{}
	p := g.systemPrompt()
	if p == "" {
		t.Error("expected non-empty default system prompt")
	}

	custom := "You are a custom bot."
	g2 := &Gateway{SystemPrompt: custom}
	p2 := g2.systemPrompt()
	if p2 != custom {
		t.Errorf("systemPrompt = %q, want %q", p2, custom)
	}
}

func TestGatewayClearHistoryNil(t *testing.T) {
	g := &Gateway{}
	// Should not panic
	g.ClearHistory(context.Background(), 42)
}

func TestGatewayBuildMessagesNilRedis(t *testing.T) {
	g := &Gateway{LLM: &Client{APIKey: "test"}}
	msgs, err := g.BuildMessages(context.Background(), 0, "hello")
	if err != nil {
		t.Fatalf("BuildMessages error: %v", err)
	}
	if len(msgs) < 2 {
		t.Fatalf("expected at least 2 messages (system + user), got %d", len(msgs))
	}
	if msgs[0].Role != "system" {
		t.Errorf("first message role = %q, want system", msgs[0].Role)
	}
	if msgs[len(msgs)-1].Role != "user" || msgs[len(msgs)-1].Content != "hello" {
		t.Errorf("last message should be user/hello, got %s/%s", msgs[len(msgs)-1].Role, msgs[len(msgs)-1].Content)
	}
}

func TestGatewayBuildMessagesWithHistory(t *testing.T) {
	g := &Gateway{
		LLM:    &Client{APIKey: "test"},
		Redis:  nil,
		MaxHistory: 5,
	}
	msgs, err := g.BuildMessages(context.Background(), 1, "test message")
	if err != nil {
		t.Fatalf("BuildMessages error: %v", err)
	}
	// Should have system prompt + user message
	if len(msgs) != 2 {
		t.Errorf("expected 2 messages, got %d", len(msgs))
	}
	if msgs[1].Content != "test message" {
		t.Errorf("user content = %q, want %q", msgs[1].Content, "test message")
	}
}

func TestGatewayCompleteUserTurnNilLLM(t *testing.T) {
	g := &Gateway{}
	_, err := g.CompleteUserTurn(context.Background(), 1, "hello")
	if err == nil {
		t.Fatal("expected error when LLM is nil")
	}
}

func TestGatewayCompleteUserTurnNilGateway(t *testing.T) {
	_, err := (*Gateway)(nil).CompleteUserTurn(context.Background(), 1, "hello")
	if err == nil {
		t.Fatal("expected error when gateway is nil")
	}
}
