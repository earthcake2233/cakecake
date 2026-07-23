package aigateway

import (
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
		Temperature: 0.7,
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

func truncate(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}