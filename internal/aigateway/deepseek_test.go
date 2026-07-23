package aigateway

import (
    "context"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "strings"
    "testing"
)

func TestClient_Complete_Success(t *testing.T) {
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if r.Method != http.MethodPost {
            t.Errorf("expected POST, got %s", r.Method)
        }
        w.WriteHeader(http.StatusOK)
        _, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"Hello!"}}]}`))
    }))
    defer srv.Close()

    c := &Client{APIKey: "test-key", BaseURL: srv.URL}
    reply, err := c.Complete(context.Background(), []ChatMessage{{Role: "user", Content: "hi"}})
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if reply != "Hello!" {
        t.Errorf("got %q, want %q", reply, "Hello!")
    }
}

func TestClient_Complete_NoAPIKey(t *testing.T) {
    c := &Client{}
    _, err := c.Complete(context.Background(), []ChatMessage{{Role: "user", Content: "hi"}})
    if err == nil || !strings.Contains(err.Error(), "api key not configured") {
        t.Fatalf("expected api key error, got %v", err)
    }
}

func TestClient_Complete_HTTPError(t *testing.T) {
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusBadRequest)
        _, _ = w.Write([]byte(`bad request`))
    }))
    defer srv.Close()

    c := &Client{APIKey: "test-key", BaseURL: srv.URL}
    _, err := c.Complete(context.Background(), []ChatMessage{{Role: "user", Content: "hi"}})
    if err == nil || !strings.Contains(err.Error(), "http 400") {
        t.Fatalf("expected http 400 error, got %v", err)
    }
}

func TestClient_EmptyCompletion(t *testing.T) {
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
        _, _ = w.Write([]byte(`{"choices":[]}`))
    }))
    defer srv.Close()

    c := &Client{APIKey: "test-key", BaseURL: srv.URL}
    _, err := c.Complete(context.Background(), []ChatMessage{{Role: "user", Content: "hi"}})
    if err == nil || !strings.Contains(err.Error(), "empty completion") {
        t.Fatalf("expected empty completion error, got %v", err)
    }
}

func TestClient_DefaultModel(t *testing.T) {
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        var req chatCompletionReq
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
            t.Fatalf("bad request body: %v", err)
        }
        if req.Model != "deepseek-chat" {
            t.Errorf("expected deepseek-chat, got %s", req.Model)
        }
        w.WriteHeader(http.StatusOK)
        _, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"ok"}}]}`))
    }))
    defer srv.Close()

    c := &Client{APIKey: "test-key", BaseURL: srv.URL}
    _, err := c.Complete(context.Background(), []ChatMessage{{Role: "user", Content: "hi"}})
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
}

func TestFinishReason(t *testing.T) {
    m := ChatMessage{Role: "assistant", Content: "hello"}
    if reason := m.FinishReason(); reason != "stop" {
        t.Errorf("expected stop, got %s", reason)
    }

    m2 := ChatMessage{Role: "assistant", ToolCalls: []ToolCall{{ID: "1"}}}
    if reason := m2.FinishReason(); reason != "tool_calls" {
        t.Errorf("expected tool_calls, got %s", reason)
    }
}
