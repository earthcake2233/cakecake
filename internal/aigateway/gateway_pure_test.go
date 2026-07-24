package aigateway

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestChatMsgToEntry(t *testing.T) {
	msg := ChatMessage{
		Role:    "user",
		Content: "Hello",
		ToolCalls: []ToolCall{{
			ID:   "call1",
			Type: "function",
			Function: ToolCallFunction{
				Name:      "test",
				Arguments: "{}",
			},
		}},
		ToolCallID: "call2",
	}
	entry := chatMsgToEntry(msg)
	require.Equal(t, msg.Role, entry.Role)
	require.Equal(t, msg.Content, entry.Content)
	require.Len(t, entry.ToolCalls, 1)
	require.Equal(t, msg.ToolCallID, entry.ToolCallID)
}

func TestEntryToChatMsg(t *testing.T) {
	entry := historyEntry{
		Role:       "assistant",
		Content:    "Response",
		ToolCalls:  []ToolCall{{ID: "tc1"}},
		ToolCallID: "tc2",
	}
	msg := entryToChatMsg(entry)
	require.Equal(t, entry.Role, msg.Role)
	require.Equal(t, entry.Content, msg.Content)
	require.Equal(t, entry.ToolCalls, msg.ToolCalls)
	require.Equal(t, entry.ToolCallID, msg.ToolCallID)
}

func TestHistoryKey(t *testing.T) {
	g := &Gateway{HistoryPrefix: "custom:"}
	require.Equal(t, "custom:123", g.historyKey(123))

	g2 := &Gateway{}
	key := g2.historyKey(456)
	require.Contains(t, key, "456")
	require.Contains(t, key, "mb:agent:hist:")
}

func TestHistoryTTL(t *testing.T) {
	g := &Gateway{}
	ttl := g.historyTTL()
	require.Equal(t, 30*24*time.Hour, ttl)

	g2 := &Gateway{HistoryTTL: 999 * time.Hour}
	require.Equal(t, 999*time.Hour, g2.historyTTL())
}

func TestSystemPrompt(t *testing.T) {
	g := &Gateway{SystemPrompt: "You are a helpful assistant"}
	require.Equal(t, "You are a helpful assistant", g.systemPrompt())

	g2 := &Gateway{}
	p := g2.systemPrompt()
	require.NotEmpty(t, p)
}

func TestChatMessageFinishReason(t *testing.T) {
	m := ChatMessage{Role: "assistant", Content: "Hello"}
	require.Equal(t, "stop", m.FinishReason())

	m2 := ChatMessage{Role: "assistant", ToolCalls: []ToolCall{{ID: "c1"}}}
	require.Equal(t, "tool_calls", m2.FinishReason())

	m3 := ChatMessage{Role: "user", Content: "hi"}
	require.Equal(t, "stop", m3.FinishReason())
}
