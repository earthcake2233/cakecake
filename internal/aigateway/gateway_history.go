package aigateway

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
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
	return msgs, nil
}
