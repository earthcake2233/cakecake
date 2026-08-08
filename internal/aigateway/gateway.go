package aigateway

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"cakecake/internal/logger"
)

const defaultSystemPrompt = `你是 cakecake 站内 AI 助手。帮助用户了解本站功能。
回答风格要求：
- 说人话，自然口语化，像朋友聊天一样
- 不！要！用！任！何！emoji！表情符号
- 不要用夸张语气、不要营销号腔
- 简洁直接，普通用户看得懂
- 当你在回复中引用了工具结果（搜索/详情/榜单）时，在回复最后单独输出一行展示清单，格式：【展示】工具名#ID,工具名#ID（例如【展示】search_videos#23）。ID 必须是工具结果中的 id，只列你明确推荐展示的结果；这行不会显示给用户，不要解释它
- 不要编造不存在的功能
- 不确定时诚实说不知道`

const maxToolIterations = 5

// Progress callbacks for front-end tool call display.
type ToolCallStartCB func(traceID, spanID, parentSpanID, toolName string, argsJSON json.RawMessage)

// ToolCallEndCB is invoked when a tool call finishes.
type ToolCallEndCB func(traceID, spanID, toolName string, durationMs int64, resultSummary string)

// ToolResultDataCB is called after a tool executes successfully and the result JSON contains an "items" field.
// items is the raw JSON array of structured results for the front-end.
type ToolResultDataCB func(traceID, spanID, toolName string, items json.RawMessage)

// Gateway orchestrates LLM calls and short-term memory in Redis.
type Gateway struct {
	LLM           *Client
	Redis         *redis.Client
	MaxHistory    int
	HistoryTTL    time.Duration
	SystemPrompt  string
	HistoryPrefix string

	// ToolExecutor is optional; set externally before CompleteUserTurnWithTools.
	ToolExec interface {
		Execute(ctx context.Context, toolName string, args json.RawMessage) (string, error)
	}
	OnToolCallStart  ToolCallStartCB
	OnToolCallEnd    ToolCallEndCB
	OnToolResultData ToolResultDataCB
}

// historyEntry with optional tool fields for full message persistence.
type historyEntry struct {
	Role       string     `json:"role"`
	Content    string     `json:"content"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
}

func chatMsgToEntry(m ChatMessage) historyEntry {
	return historyEntry(m)
}

func entryToChatMsg(e historyEntry) ChatMessage {
	return ChatMessage(e)
}

func (g *Gateway) historyKey(conversationID uint64) string {
	p := g.HistoryPrefix
	if p == "" {
		p = "mb:agent:hist:"
	}
	return fmt.Sprintf("%s%d", p, conversationID)
}

// PersistHistory stores full message history for a conversation (used by
// callers that assemble history outside the standard turn methods).
func (g *Gateway) PersistHistory(ctx context.Context, conversationID uint64, msgs []ChatMessage) {
	g.persistHistory(ctx, conversationID, msgs)
}

// BuildMessages loads ALL history (including tool messages) and appends user turn.
func (g *Gateway) BuildMessages(ctx context.Context, conversationID uint64, userText string) ([]ChatMessage, error) {
	msgs := []ChatMessage{{Role: "system", Content: g.systemPrompt()}}
	if g.Redis != nil && conversationID > 0 {
		raw, err := g.Redis.Get(ctx, g.historyKey(conversationID)).Bytes()
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

// CompleteUserTurn is the simple text-only version (no tools).
func (g *Gateway) CompleteUserTurn(ctx context.Context, conversationID uint64, userText string) (string, error) {
	if g == nil || g.LLM == nil {
		return "", fmt.Errorf("agent gateway not configured")
	}
	msgs, err := g.BuildMessages(ctx, conversationID, userText)
	if err != nil {
		return "", err
	}
	reply, err := g.LLM.Complete(ctx, msgs)
	if err != nil {
		return "", err
	}
	reply = strings.TrimSpace(reply)
	if reply == "" {
		return "", fmt.Errorf("empty model reply")
	}
	allMsgs := append(msgs, ChatMessage{Role: "assistant", Content: reply})
	g.persistHistory(ctx, conversationID, allMsgs)
	return reply, nil
}

// CompleteUserTurnStream is the streaming variant of CompleteUserTurn.
func (g *Gateway) CompleteUserTurnStream(ctx context.Context, conversationID uint64, userText string, onDelta func(string)) (string, error) {
	if g == nil || g.LLM == nil {
		return "", fmt.Errorf("agent gateway not configured")
	}
	if onDelta == nil {
		onDelta = func(string) {}
	}
	msgs, err := g.BuildMessages(ctx, conversationID, userText)
	if err != nil {
		return "", err
	}
	replyMsg, err := g.LLM.CompleteStream(ctx, msgs, onDelta)
	if err != nil {
		return "", err
	}
	reply := strings.TrimSpace(replyMsg.Content)
	if reply == "" {
		return "", fmt.Errorf("empty model reply")
	}
	allMsgs := append(msgs, replyMsg)
	g.persistHistory(ctx, conversationID, allMsgs)
	return reply, nil
}

// ContinueTurnStream resumes a user-stopped reply. The partial is sent to the
// model as its own previous assistant message (not as user text), so it
// continues naturally instead of echoing the seam — the same technique chat
// UIs use for stop/continue. History persists the stitched full reply.
func (g *Gateway) ContinueTurnStream(ctx context.Context, conversationID uint64, partial string, instruction string, onDelta func(string)) (ChatMessage, error) {
	if g == nil || g.LLM == nil {
		return ChatMessage{}, fmt.Errorf("agent gateway not configured")
	}
	if onDelta == nil {
		onDelta = func(string) {}
	}
	msgs, err := g.BuildMessages(ctx, conversationID, "")
	if err != nil {
		return ChatMessage{}, err
	}
	// Drop the trailing empty user message BuildMessages appended.
	if len(msgs) > 0 && msgs[len(msgs)-1].Role == "user" &&
		strings.TrimSpace(msgs[len(msgs)-1].Content) == "" {
		msgs = msgs[:len(msgs)-1]
	}
	r := []rune(partial)
	if len(r) > 3000 {
		partial = string(r[:3000])
	}
	partial = strings.TrimSpace(partial)
	msgs = append(msgs,
		ChatMessage{Role: "assistant", Content: partial},
		ChatMessage{Role: "user", Content: instruction},
	)
	replyMsg, err := g.LLM.CompleteStream(ctx, msgs, onDelta)
	if err != nil {
		return ChatMessage{}, err
	}
	if strings.TrimSpace(replyMsg.Content) == "" {
		return ChatMessage{}, fmt.Errorf("empty model reply")
	}
	// Persist prior turns + the stitched full reply, dropping the transient
	// partial/instruction messages from the stored history.
	fullReply := partial
	if c := strings.TrimSpace(replyMsg.Content); c != "" {
		fullReply += "\n" + c
	}
	hist := append([]ChatMessage{}, msgs[:len(msgs)-2]...)
	hist = append(hist, ChatMessage{Role: "assistant", Content: fullReply})
	g.persistHistory(ctx, conversationID, hist)
	return replyMsg, nil
}

// CompleteUserTurnWithTools runs the multi-turn tool calling loop.
// tools: the enabled tool definitions.
// traceID: a unique identifier for this turn (for observability + front-end).
func (g *Gateway) CompleteUserTurnWithTools(
	ctx context.Context,
	conversationID uint64,
	userText string,
	tools []ToolDef,
	traceID string,
) (string, error) {
	if g == nil || g.LLM == nil {
		return "", fmt.Errorf("agent gateway not configured")
	}
	msgs, err := g.BuildMessages(ctx, conversationID, userText)
	if err != nil {
		return "", err
	}

	for iter := 0; iter < maxToolIterations; iter++ {
		msg, err := g.LLM.CompleteWithTools(ctx, msgs, tools)
		if err != nil {
			return "", err
		}
		msgs = append(msgs, msg)

		if msg.FinishReason() == "stop" || (len(msg.ToolCalls) == 0 && msg.Content != "") {
			reply := strings.TrimSpace(msg.Content)
			if reply == "" {
				return "", fmt.Errorf("empty model reply")
			}
			g.persistHistory(ctx, conversationID, msgs)
			return reply, nil
		}

		if len(msg.ToolCalls) == 0 {
			// Shouldn't happen, but safeguard
			continue
		}

		// Execute tool calls
		if g.ToolExec == nil {
			return "", fmt.Errorf("tool executor not configured")
		}
		toolMsgs := g.executeToolCalls(ctx, msg.ToolCalls, traceID, iter)
		msgs = append(msgs, toolMsgs...)
	}

	// Max iterations reached: return last assistant content or fallback
	return "抱歉，操作超时，请稍后重试或简化问题。", nil
}

// CompleteUserTurnWithToolsStream is the streaming variant of
// CompleteUserTurnWithTools (text deltas are forwarded via OnTextDelta).
func (g *Gateway) CompleteUserTurnWithToolsStream(
	ctx context.Context,
	conversationID uint64,
	userText string,
	tools []ToolDef,
	traceID string,
	onDelta func(string),
) (string, error) {
	if g == nil || g.LLM == nil {
		return "", fmt.Errorf("agent gateway not configured")
	}
	if onDelta == nil {
		onDelta = func(string) {}
	}
	msgs, err := g.BuildMessages(ctx, conversationID, userText)
	if err != nil {
		return "", err
	}

	for iter := 0; iter < maxToolIterations; iter++ {
		msg, err := g.LLM.CompleteWithToolsStream(ctx, msgs, tools, onDelta)
		if err != nil {
			return "", err
		}
		msgs = append(msgs, msg)

		if msg.FinishReason() == "stop" || (len(msg.ToolCalls) == 0 && msg.Content != "") {
			reply := strings.TrimSpace(msg.Content)
			if reply == "" {
				return "", fmt.Errorf("empty model reply")
			}
			g.persistHistory(ctx, conversationID, msgs)
			return reply, nil
		}

		if len(msg.ToolCalls) == 0 {
			continue
		}
		if g.ToolExec == nil {
			return "", fmt.Errorf("tool executor not configured")
		}
		toolMsgs := g.executeToolCalls(ctx, msg.ToolCalls, traceID, iter)
		msgs = append(msgs, toolMsgs...)
	}

	return "抱歉，操作超时，请稍后重试或简化问题。", nil
}

func (g *Gateway) executeToolCalls(ctx context.Context, calls []ToolCall, traceID string, round int) []ChatMessage {
	type result struct {
		msg ChatMessage
	}
	ch := make(chan result, len(calls))

	for i, call := range calls {
		go func(idx int, tc ToolCall) {
			spanID := fmt.Sprintf("%s-r%d-t%d", traceID, round, idx)
			parentSpanID := traceID

			if g.OnToolCallStart != nil {
				var raw json.RawMessage
				if err := json.Unmarshal([]byte(tc.Function.Arguments), &raw); err != nil && logger.L != nil {
					logger.L.Warn("aigateway: parse tool call args failed", zap.String("tool", tc.Function.Name), zap.Error(err))
				}
				g.OnToolCallStart(traceID, spanID, parentSpanID, tc.Function.Name, raw)
			}

			start := time.Now()
			var res string
			if g.ToolExec != nil {
				r, err := g.ToolExec.Execute(ctx, tc.Function.Name, json.RawMessage(tc.Function.Arguments))
				if err != nil {
					res = fmt.Sprintf(`{"error": "%s"}`, err.Error())
				} else {
					res = r
				}
			} else {
				res = `{"error": "tool executor not available"}`
			}
			duration := time.Since(start).Milliseconds()

			if g.OnToolCallEnd != nil {
				summary := res
				if len(summary) > 80 {
					summary = summary[:80] + "..."
				}
				g.OnToolCallEnd(traceID, spanID, tc.Function.Name, duration, summary)
			}

			if g.OnToolResultData != nil && res != "" {
				var parsed map[string]json.RawMessage
				if json.Unmarshal([]byte(res), &parsed) == nil {
					if items, ok := parsed["items"]; ok && len(items) > 0 && items[0] == '[' {
						g.OnToolResultData(traceID, spanID, tc.Function.Name, items)
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

// persistHistory stores the full message sequence to Redis.
func (g *Gateway) persistHistory(ctx context.Context, conversationID uint64, msgs []ChatMessage) {
	if g.Redis == nil || conversationID == 0 {
		return
	}
	hist := make([]historyEntry, 0, len(msgs))
	for _, m := range msgs {
		if m.Role == "system" {
			continue
		}
		hist = append(hist, chatMsgToEntry(m))
	}
	max := g.MaxHistory
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
		ttl := g.historyTTL()
		_ = g.Redis.Set(ctx, g.historyKey(conversationID), b, ttl).Err()
	}
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

// FinishReason returns "stop" or "tool_calls" based on the message content.
func (m ChatMessage) FinishReason() string {
	if len(m.ToolCalls) > 0 {
		return "tool_calls"
	}
	return "stop"
}
