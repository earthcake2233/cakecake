package aigateway

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

const defaultSystemPrompt = `你是 cakecake 站内 AI 助手「cakecake AI」。你帮助用户了解本站功能：视频观看、弹幕、投稿、私信、个人空间、收藏与历史等。
回答请简洁、友好，使用中文。不要编造不存在的接口或功能；不确定时请诚实说明并给出合理建议。`

// Gateway orchestrates LLM calls and short-term memory in Redis.
type Gateway struct {
	LLM           *Client
	Redis         *redis.Client
	MaxHistory    int
	HistoryTTL    time.Duration
	SystemPrompt  string
	HistoryPrefix string
}

type historyEntry struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

func (g *Gateway) historyKey(conversationID uint64) string {
	p := g.HistoryPrefix
	if p == "" {
		p = "mb:agent:hist:"
	}
	return fmt.Sprintf("%s%d", p, conversationID)
}

// BuildMessages loads Redis history and appends the latest user turn.
func (g *Gateway) BuildMessages(ctx context.Context, conversationID uint64, userText string) ([]ChatMessage, error) {
	msgs := []ChatMessage{{Role: "system", Content: g.systemPrompt()}}
	if g.Redis != nil && conversationID > 0 {
		raw, err := g.Redis.Get(ctx, g.historyKey(conversationID)).Bytes()
		if err == nil && len(raw) > 0 {
			var hist []historyEntry
			if json.Unmarshal(raw, &hist) == nil {
				for _, h := range hist {
					if h.Role == "user" || h.Role == "assistant" {
						msgs = append(msgs, ChatMessage{Role: h.Role, Content: h.Content})
					}
				}
			}
		}
	}
	msgs = append(msgs, ChatMessage{Role: "user", Content: userText})
	return msgs, nil
}

// CompleteUserTurn calls the model and persists history on success.
func (g *Gateway) CompleteUserTurn(ctx context.Context, conversationID uint64, userText string) (reply string, err error) {
	if g == nil || g.LLM == nil {
		return "", fmt.Errorf("agent gateway not configured")
	}
	msgs, err := g.BuildMessages(ctx, conversationID, userText)
	if err != nil {
		return "", err
	}
	reply, err = g.LLM.Complete(ctx, msgs)
	if err != nil {
		return "", err
	}
	reply = strings.TrimSpace(reply)
	if reply == "" {
		return "", fmt.Errorf("empty model reply")
	}
	if g.Redis != nil && conversationID > 0 {
		hist := make([]historyEntry, 0, len(msgs))
		for _, m := range msgs {
			if m.Role == "user" || m.Role == "assistant" {
				hist = append(hist, historyEntry{Role: m.Role, Content: m.Content})
			}
		}
		hist = append(hist, historyEntry{Role: "assistant", Content: reply})
		max := g.MaxHistory
		if max <= 0 {
			max = 20
		}
		// Each turn adds user+assistant; keep last max*2 entries.
		cap := max * 2
		if len(hist) > cap {
			hist = hist[len(hist)-cap:]
		}
		if b, e := json.Marshal(hist); e == nil {
			ttl := g.historyTTL()
			_ = g.Redis.Set(ctx, g.historyKey(conversationID), b, ttl).Err()
		}
	}
	return reply, nil
}

// ClearHistory removes short-term LLM memory for a conversation.
func (g *Gateway) ClearHistory(ctx context.Context, conversationID uint64) {
	if g == nil || g.Redis == nil || conversationID == 0 {
		return
	}
	_ = g.Redis.Del(ctx, g.historyKey(conversationID)).Err()
}

func (g *Gateway) historyTTL() time.Duration {
	if g != nil && g.HistoryTTL > 0 {
		return g.HistoryTTL
	}
	return 30 * 24 * time.Hour
}

func (g *Gateway) systemPrompt() string {
	if g != nil && strings.TrimSpace(g.SystemPrompt) != "" {
		return g.SystemPrompt
	}
	return defaultSystemPrompt
}
