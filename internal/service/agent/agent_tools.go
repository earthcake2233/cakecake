package agent

import (
	"cakecake/internal/aigateway"
	"cakecake/internal/aigateway/toolkit"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"go.uber.org/zap"
	"strconv"
	"strings"
	"sync"
	"time"
)

// enabledTools builds the tool enabled map from RuntimeConfig.
func (s *AgentService) enabledTools() map[string]bool {
	m := make(map[string]bool)
	for _, name := range toolkit.AllToolNames() {
		enabled := true
		if s.RC != nil {
			key := "tool_" + name + "_enabled"
			enabled = s.RC.GetBool(key, true)
		}
		m[name] = enabled
	}
	return m
}

// generateTraceID creates a short unique trace identifier.
func generateTraceID() string {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%x", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

func (s *AgentService) setupToolCallbacks(traceID string, humanID uint64) {
	if s.Gateway == nil || s.ChatHub == nil {
		return
	}
	s.Gateway.OnToolCallStart = func(tid, spanID, parentSpanID, toolName string, argsJSON json.RawMessage) {
		var args interface{}
		if err := json.Unmarshal(argsJSON, &args); err != nil && s.Log != nil {
			s.Log.Warn("agent: parse tool call args failed", zap.String("tool", toolName), zap.Error(err))
		}
		payload := map[string]interface{}{
			"trace_id":       tid,
			"span_id":        spanID,
			"parent_span_id": parentSpanID,
			"tool_name":      toolName,
			"arguments":      args,
			"started_at":     time.Now().Format(time.RFC3339),
		}
		s.ChatHub.PushJSON(humanID, map[string]interface{}{
			"type": "tool_call_start",
			"body": payload,
		})
	}
	s.Gateway.OnToolCallEnd = func(tid, spanID, toolName string, durationMs int64, resultSummary string) {
		payload := map[string]interface{}{
			"trace_id":       tid,
			"span_id":        spanID,
			"tool_name":      toolName,
			"duration_ms":    durationMs,
			"result_summary": resultSummary,
		}
		s.ChatHub.PushJSON(humanID, map[string]interface{}{
			"type": "tool_call_end",
			"body": payload,
		})
	}
	s.Gateway.OnToolResultData = func(tid, spanID, toolName string, items json.RawMessage) {
		payload := map[string]interface{}{
			"trace_id":  tid,
			"span_id":   spanID,
			"tool_name": toolName,
			"items":     items,
		}
		s.ChatHub.PushJSON(humanID, map[string]interface{}{
			"type": "tool_result_data",
			"body": payload,
		})
	}
}

func (s *AgentService) clearToolCallbacks() {
	if s.Gateway != nil {
		s.Gateway.OnToolCallStart = nil
		s.Gateway.OnToolCallEnd = nil
		s.Gateway.OnToolResultData = nil
	}
}

type toolActivityCollector struct {
	mu      sync.Mutex
	acts    []map[string]interface{}
	results map[string]json.RawMessage
}

// installToolCollectors wraps gateway callbacks to record tool activity.
func installToolCollectors(g *aigateway.Gateway) *toolActivityCollector {
	c := &toolActivityCollector{results: make(map[string]json.RawMessage)}
	if g.OnToolCallStart != nil {
		orig := g.OnToolCallStart
		g.OnToolCallStart = func(tid, spanID, parentSpanID, toolName string, argsJSON json.RawMessage) {
			orig(tid, spanID, parentSpanID, toolName, argsJSON)
			c.mu.Lock()
			c.acts = append(c.acts, map[string]interface{}{
				"trace_id":  tid,
				"span_id":   spanID,
				"tool_name": toolName,
				"status":    "running",
			})
			c.mu.Unlock()
		}
	}
	if g.OnToolCallEnd != nil {
		orig := g.OnToolCallEnd
		g.OnToolCallEnd = func(tid, spanID, toolName string, durationMs int64, resultSummary string) {
			orig(tid, spanID, toolName, durationMs, resultSummary)
			c.mu.Lock()
			for i := range c.acts {
				if c.acts[i]["span_id"] == spanID {
					c.acts[i]["status"] = "done"
					c.acts[i]["duration_ms"] = durationMs
					c.acts[i]["result_summary"] = resultSummary
					break
				}
			}
			c.mu.Unlock()
		}
	}
	if g.OnToolResultData != nil {
		orig := g.OnToolResultData
		g.OnToolResultData = func(tid, spanID, toolName string, items json.RawMessage) {
			orig(tid, spanID, toolName, items)
			c.mu.Lock()
			c.results[spanID] = items
			c.mu.Unlock()
		}
	}
	return c
}

// buildReplyResult attaches persisted tool data to the final reply.
func buildReplyResult(reply string, coll *toolActivityCollector) (*GenerateReplyResult, error) {
	result := &GenerateReplyResult{Content: reply}
	if coll == nil {
		return result, nil
	}
	if len(coll.acts) > 0 {
		if b, e := json.Marshal(coll.acts); e == nil {
			result.ToolActivities = b
		}
	}
	if len(coll.results) > 0 && !replyDismissesResults(reply) {
		rm := make(map[string]json.RawMessage, len(coll.results))
		for k, v := range coll.results {
			if filtered := filterReferencedItems(reply, v); filtered != nil {
				rm[k] = filtered
			}
		}
		if len(rm) > 0 {
			if b, e := json.Marshal(rm); e == nil {
				result.ToolResultData = b
			}
		}
	}
	return result, nil
}

// replyDismissesResults reports whether the assistant explicitly said the tool
// results were irrelevant/missing (e.g. "站内暂时没有相关教程"). In that case
// none of the collected results should be shown, even if a title happens to be
// mentioned while being dismissed.
func replyDismissesResults(reply string) bool {
	return mdDismissRe.MatchString(reply)
}

// filterReferencedItems keeps only tool result items that the final reply
// actually mentions (by title/content fragment or numeric id). Unreferenced
// results (e.g. a search that found nothing relevant) are dropped so the UI
// never shows cards the assistant did not cite.
func filterReferencedItems(reply string, items json.RawMessage) json.RawMessage {
	var arr []map[string]interface{}
	if err := json.Unmarshal(items, &arr); err != nil {
		return items
	}
	kept := make([]map[string]interface{}, 0, len(arr))
	for _, it := range arr {
		if itemReferenced(reply, it) {
			kept = append(kept, it)
		}
	}
	if len(kept) == 0 {
		return nil
	}
	b, err := json.Marshal(kept)
	if err != nil {
		return nil
	}
	return b
}

func itemReferenced(reply string, it map[string]interface{}) bool {
	if id, ok := it["id"]; ok {
		if f, ok := id.(float64); ok && f > 0 {
			if strings.Contains(reply, strconv.FormatUint(uint64(f), 10)) {
				return true
			}
		}
	}
	for _, k := range []string{"title", "content"} {
		raw, ok := it[k]
		if !ok {
			continue
		}
		text := strings.TrimSpace(fmt.Sprintf("%v", raw))
		if text == "" {
			continue
		}
		if strings.Contains(reply, text) {
			return true
		}
		r := []rune(text)
		if len(r) > 12 && strings.Contains(reply, string(r[:12])) {
			return true
		}
	}
	return false
}
