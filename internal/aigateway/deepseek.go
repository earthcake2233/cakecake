package aigateway

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// --- Tool Calling types (OpenAI-compatible) ---

// ToolCall represents a function call requested by the model.
type ToolCall struct {
	ID       string           `json:"id"`
	Type     string           `json:"type"`
	Function ToolCallFunction `json:"function"`
}

// ToolCallFunction is the function details in a tool_call.
type ToolCallFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// ToolDef is a tool definition sent to the model (OpenAI tools format).
type ToolDef struct {
	Type     string      `json:"type"`
	Function ToolFuncDef `json:"function"`
}

// ToolFuncDef is the function definition within a tool.
type ToolFuncDef struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Parameters  interface{} `json:"parameters"`
}

// ChatMessage is an OpenAI-compatible chat message (text or tool).
type ChatMessage struct {
	Role       string     `json:"role"`
	Content    string     `json:"content"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
}

// Client calls DeepSeek chat/completions (OpenAI-compatible).
type Client struct {
	APIKey     string
	BaseURL    string
	Model      string
	HTTPClient *http.Client
}

type chatCompletionReq struct {
	Model       string        `json:"model"`
	Messages    []ChatMessage `json:"messages"`
	Tools       []ToolDef     `json:"tools,omitempty"`
	Temperature float64       `json:"temperature,omitempty"`
	Stream      bool          `json:"stream"`
}

type choice struct {
	Message      ChatMessage `json:"message"`
	FinishReason string      `json:"finish_reason"`
}

type chatCompletionResp struct {
	Choices []choice `json:"choices"`
	Error   *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// chatCompletionStreamChunk is one SSE `data:` line of a streaming response
// (OpenAI-compatible; tool call arguments arrive as fragments by index).
type chatCompletionStreamChunk struct {
	Choices []struct {
		Delta struct {
			Content   string `json:"content"`
			ToolCalls []struct {
				Index    int    `json:"index"`
				ID       string `json:"id"`
				Function struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				} `json:"function"`
			} `json:"tool_calls"`
		} `json:"delta"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
}

// Complete returns the assistant text (no tools).
func (c *Client) Complete(ctx context.Context, messages []ChatMessage) (string, error) {
	msg, err := c.completeInternal(ctx, messages, nil)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(msg.Content), nil
}

// CompleteWithTools calls the model with tools and returns the full ChatMessage.
func (c *Client) CompleteWithTools(ctx context.Context, messages []ChatMessage, tools []ToolDef) (ChatMessage, error) {
	return c.completeInternal(ctx, messages, tools)
}

func (c *Client) completeInternal(ctx context.Context, messages []ChatMessage, tools []ToolDef) (ChatMessage, error) {
	if c == nil || strings.TrimSpace(c.APIKey) == "" {
		return ChatMessage{}, fmt.Errorf("deepseek: api key not configured")
	}
	base := strings.TrimRight(c.BaseURL, "/")
	if base == "" {
		base = "https://api.deepseek.com"
	}
	model := c.Model
	if model == "" {
		model = "deepseek-chat"
	}
	body, err := json.Marshal(chatCompletionReq{
		Model:       model,
		Messages:    messages,
		Tools:       tools,
		Temperature: 0.5,
		Stream:      false,
	})
	if err != nil {
		return ChatMessage{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return ChatMessage{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.APIKey)

	hc := c.HTTPClient
	if hc == nil {
		hc = &http.Client{Timeout: 90 * time.Second}
	}
	res, err := hc.Do(req)
	if err != nil {
		return ChatMessage{}, err
	}
	defer res.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(res.Body, 2<<20))
	if err != nil {
		return ChatMessage{}, err
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return ChatMessage{}, fmt.Errorf("deepseek: http %d: %s", res.StatusCode, truncate(string(raw), 400))
	}
	var out chatCompletionResp
	if err := json.Unmarshal(raw, &out); err != nil {
		return ChatMessage{}, err
	}
	if out.Error != nil && out.Error.Message != "" {
		return ChatMessage{}, fmt.Errorf("deepseek: %s", out.Error.Message)
	}
	if len(out.Choices) == 0 {
		return ChatMessage{}, fmt.Errorf("deepseek: empty completion")
	}
	return out.Choices[0].Message, nil
}

// CompleteStream streams the assistant text deltas through onDelta and returns
// the fully accumulated message (including tool calls).
func (c *Client) CompleteStream(ctx context.Context, messages []ChatMessage, onDelta func(string)) (ChatMessage, error) {
	return c.completeInternalStream(ctx, messages, nil, onDelta)
}

// CompleteWithToolsStream is the streaming variant of CompleteWithTools.
func (c *Client) CompleteWithToolsStream(ctx context.Context, messages []ChatMessage, tools []ToolDef, onDelta func(string)) (ChatMessage, error) {
	return c.completeInternalStream(ctx, messages, tools, onDelta)
}

func (c *Client) completeInternalStream(ctx context.Context, messages []ChatMessage, tools []ToolDef, onDelta func(string)) (ChatMessage, error) {
	if c == nil || strings.TrimSpace(c.APIKey) == "" {
		return ChatMessage{}, fmt.Errorf("deepseek: api key not configured")
	}
	base := strings.TrimRight(c.BaseURL, "/")
	if base == "" {
		base = "https://api.deepseek.com"
	}
	model := c.Model
	if model == "" {
		model = "deepseek-chat"
	}
	body, err := json.Marshal(chatCompletionReq{
		Model:       model,
		Messages:    messages,
		Tools:       tools,
		Temperature: 0.5,
		Stream:      true,
	})
	if err != nil {
		return ChatMessage{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return ChatMessage{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	req.Header.Set("Accept", "text/event-stream")

	hc := c.HTTPClient
	if hc == nil {
		hc = &http.Client{Timeout: 90 * time.Second}
	}
	res, err := hc.Do(req)
	if err != nil {
		return ChatMessage{}, err
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		raw, _ := io.ReadAll(io.LimitReader(res.Body, 1<<20))
		return ChatMessage{}, fmt.Errorf("deepseek: http %d: %s", res.StatusCode, truncate(string(raw), 400))
	}

	var acc ChatMessage
	acc.Role = "assistant"
	scanner := bufio.NewScanner(res.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 2<<20)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if payload == "[DONE]" {
			break
		}
		var chunk chatCompletionStreamChunk
		if err := json.Unmarshal([]byte(payload), &chunk); err != nil {
			continue
		}
		if len(chunk.Choices) == 0 {
			continue
		}
		delta := chunk.Choices[0].Delta
		if delta.Content != "" {
			acc.Content += delta.Content
			if onDelta != nil {
				onDelta(delta.Content)
			}
		}
		for _, tc := range delta.ToolCalls {
			for len(acc.ToolCalls) <= tc.Index {
				acc.ToolCalls = append(acc.ToolCalls, ToolCall{Type: "function"})
			}
			if tc.ID != "" {
				acc.ToolCalls[tc.Index].ID = tc.ID
			}
			if tc.Function.Name != "" {
				acc.ToolCalls[tc.Index].Function.Name = tc.Function.Name
			}
			acc.ToolCalls[tc.Index].Function.Arguments += tc.Function.Arguments
		}
	}
	if err := scanner.Err(); err != nil {
		return ChatMessage{}, fmt.Errorf("deepseek stream: %w", err)
	}
	if strings.TrimSpace(acc.Content) == "" && len(acc.ToolCalls) == 0 {
		return ChatMessage{}, fmt.Errorf("deepseek: empty streaming completion")
	}
	return acc, nil
}

func truncate(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
