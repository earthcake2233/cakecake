package aigateway

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
	"unicode/utf8"
)

// HistoryStore owns short-term conversation history persistence for the gateway.
type HistoryStore struct {
	gw *Gateway
}

func (h *HistoryStore) historyTTL() time.Duration {
	if h.gw != nil && h.gw.HistoryTTL > 0 {
		return h.gw.HistoryTTL
	}
	return 30 * 24 * time.Hour
}

// PersistHistory stores full message history for a conversation (used by
// callers that assemble history outside the standard turn methods).
func (h *HistoryStore) PersistHistory(ctx context.Context, conversationID uint64, msgs []ChatMessage) {
	h.gw.persistHistory(ctx, conversationID, msgs)
}

// ClearHistory removes short-term LLM memory for a conversation.
func (h *HistoryStore) ClearHistory(ctx context.Context, conversationID uint64) {
	if h.gw == nil || h.gw.Redis == nil || conversationID == 0 {
		return
	}
	_ = h.gw.Redis.Del(ctx, h.gw.historyKey(conversationID)).Err()
}

func (h *HistoryStore) persistHistory(ctx context.Context, conversationID uint64, msgs []ChatMessage) {
	if h.gw.Redis == nil || conversationID == 0 {
		return
	}
	hist := make([]historyEntry, 0, len(msgs))
	for _, m := range msgs {
		if m.Role == "system" {
			continue
		}
		hist = append(hist, chatMsgToEntry(m))
	}
	max := h.gw.MaxHistory
	if max <= 0 {
		max = 20
	}
	// Estimate: each "turn" = user msg + assistant + tool calls + tool results
	// Keep roughly max*8 entries to accommodate tool messages
	cap := max * 8
	if len(hist) > cap {
		hist = hist[len(hist)-cap:]
	}
	if b, e := json.Marshal(hist); e == nil {
		ttl := h.gw.historyTTL()
		_ = h.gw.Redis.Set(ctx, h.gw.historyKey(conversationID), b, ttl).Err()
	}
}

// ClearHistory removes short-term LLM memory for a conversation.

func (h *HistoryStore) historyKey(conversationID uint64) string {
	p := h.gw.HistoryPrefix
	if p == "" {
		p = "mb:agent:hist:"
	}
	return fmt.Sprintf("%s%d", p, conversationID)
}

// BuildMessages loads ALL history (including tool messages) and appends user turn.
func (h *HistoryStore) BuildMessages(ctx context.Context, conversationID uint64, userText string) ([]ChatMessage, error) {
	msgs := []ChatMessage{{Role: "system", Content: h.gw.systemPrompt()}}
	if h.gw.Redis != nil && conversationID > 0 {
		raw, err := h.gw.Redis.Get(ctx, h.gw.historyKey(conversationID)).Bytes()
		if err == nil && len(raw) > 0 {
			var hist []historyEntry
			if json.Unmarshal(raw, &hist) == nil {
				for _, h := range hist {
					if h.Role == "user" || h.Role == "assistant" || h.Role == "tool" {
						msgs = append(msgs, entryToChatMsg(h))
					}
				}
			}
		}
	}
	msgs = append(msgs, ChatMessage{Role: "user", Content: userText})
	return trimMessagesToTokenBudget(msgs, h.gw.MaxTokens), nil
}

// estimatedTokens is a cheap heuristic (≈ 4 runes/token) used only to bound
// the prompt size; it is not an exact tokenizer count.
func estimatedTokens(s string) int {
	return (utf8.RuneCountInString(s) + 3) / 4
}

// trimMessagesToTokenBudget keeps the system message plus the newest messages
// that fit within maxTokens. The most recent user message is always kept, so a
// single oversized message is still delivered. maxTokens <= 0 disables trimming.
func trimMessagesToTokenBudget(msgs []ChatMessage, maxTokens int) []ChatMessage {
	if maxTokens <= 0 || len(msgs) <= 1 {
		return msgs
	}
	keep := make([]ChatMessage, 0, len(msgs)-1)
	total := 0
	for i := len(msgs) - 1; i >= 1; i-- {
		cost := estimatedTokens(msgs[i].Content)
		for _, tc := range msgs[i].ToolCalls {
			cost += estimatedTokens(tc.Function.Arguments)
		}
		if len(keep) > 0 && total+cost > maxTokens {
			break
		}
		total += cost
		keep = append(keep, msgs[i])
	}
	out := make([]ChatMessage, 0, len(keep)+1)
	out = append(out, msgs[0]) // system
	for i := len(keep) - 1; i >= 0; i-- {
		out = append(out, keep[i])
	}
	return out
}
