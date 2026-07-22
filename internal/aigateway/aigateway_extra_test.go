package aigateway

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestTruncate_EdgeCases(t *testing.T) {
	tests := []struct {
		input string
		n     int
		want  string
	}{
		// len("")=0, n=0 → returns "" (len <= n)
		{"", 0, ""},
		// len("hello")=5, n=0 → s[:0] + "…" = "…"
		{"hello", 0, "…"},
		// len(trimmed "  x  ")=1, n=1 → returns "x" (len <= n)
		{"  x  ", 1, "x"},
		// len(trimmed "    ")=0, n=5 → returns "" (len <= n)
		{"    ", 5, ""},
		// len("abc")=3, n=3 → returns "abc"
		{"abc", 3, "abc"},
		// len("abcd")=4, n=3 → s[:3] + "…" = "abc…"
		{"abcd", 3, "abc…"},
		// len("a\nb\nc")=5, n=10 → returns whole string
		{"a\nb\nc", 10, "a\nb\nc"},
		// len("a\nb\nc")=5, n=3 → s[:3] + "…" = "a\nb…"
		{"a\nb\nc", 3, "a\nb…"},
	}
	for _, tc := range tests {
		got := truncate(tc.input, tc.n)
		if got != tc.want {
			t.Errorf("truncate(%q, %d) = %q, want %q", tc.input, tc.n, got, tc.want)
		}
	}
}

func TestHistoryTTL_Custom(t *testing.T) {
	g := &Gateway{HistoryTTL: 24 * time.Hour}
	ttl := g.historyTTL()
	if ttl != 24*time.Hour {
		t.Errorf("historyTTL = %v, want 24h", ttl)
	}
}

func TestHistoryTTL_NilGateway(t *testing.T) {
	ttl := (*Gateway)(nil).historyTTL()
	if ttl != 30*24*time.Hour {
		t.Errorf("historyTTL on nil = %v, want 30d", ttl)
	}
}

func TestSystemPrompt_NilGateway(t *testing.T) {
	p := (*Gateway)(nil).systemPrompt()
	if p != defaultSystemPrompt {
		t.Errorf("systemPrompt on nil = %q, want default", p)
	}
}

func TestSystemPrompt_WhitespaceOnly(t *testing.T) {
	g := &Gateway{SystemPrompt: "   "}
	p := g.systemPrompt()
	if p != defaultSystemPrompt {
		t.Errorf("systemPrompt with whitespace = %q, want default", p)
	}
}

func TestClearHistory_NilRedis(t *testing.T) {
	g := &Gateway{LLM: &Client{APIKey: "test"}}
	g.ClearHistory(context.Background(), 42)
}

func TestClearHistory_ZeroConversationID(t *testing.T) {
	g := &Gateway{Redis: nil}
	g.ClearHistory(context.Background(), 0)
}

func TestClearHistory_NilGateway(t *testing.T) {
	(*Gateway)(nil).ClearHistory(context.Background(), 42)
}

func TestBuildMessages_WithSystemPromptCustom(t *testing.T) {
	customPrompt := "You are a helpful bot."
	g := &Gateway{SystemPrompt: customPrompt, LLM: &Client{APIKey: "test"}}
	msgs, err := g.BuildMessages(context.Background(), 0, "hello")
	if err != nil {
		t.Fatalf("BuildMessages error: %v", err)
	}
	if len(msgs) < 2 {
		t.Fatalf("expected >= 2 messages, got %d", len(msgs))
	}
	if msgs[0].Content != customPrompt {
		t.Errorf("system prompt = %q, want %q", msgs[0].Content, customPrompt)
	}
}

func TestBuildMessages_EdgeCases(t *testing.T) {
	g := &Gateway{LLM: &Client{APIKey: "test"}}
	msgs, err := g.BuildMessages(context.Background(), 0, "")
	if err != nil {
		t.Fatalf("BuildMessages error: %v", err)
	}
	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages for empty text, got %d", len(msgs))
	}
	msgs, err = g.BuildMessages(context.Background(), 0, "你好世界")
	if err != nil {
		t.Fatalf("BuildMessages error: %v", err)
	}
	if msgs[len(msgs)-1].Content != "你好世界" {
		t.Errorf("user content = %q, want 你好世界", msgs[len(msgs)-1].Content)
	}
}

func TestHistoryKey_EmptyPrefix(t *testing.T) {
	g := &Gateway{HistoryPrefix: ""}
	key := g.historyKey(1)
	if !strings.HasPrefix(key, "mb:agent:hist:") {
		t.Errorf("historyKey with empty prefix = %q, want mb:agent:hist:1", key)
	}
}

func TestHistoryKey_EdgeIDs(t *testing.T) {
	g := &Gateway{HistoryPrefix: "pfx:"}
	cases := []uint64{0, 1, ^uint64(0)}
	for _, id := range cases {
		key := g.historyKey(id)
		if key == "" {
			t.Errorf("historyKey(%d) returned empty", id)
		}
	}
}

func TestCompleteUserTurn_NilLLMWithinGateway(t *testing.T) {
	g := &Gateway{LLM: nil}
	_, err := g.CompleteUserTurn(context.Background(), 1, "hello")
	if err == nil {
		t.Fatal("expected error when LLM is nil")
	}
}
