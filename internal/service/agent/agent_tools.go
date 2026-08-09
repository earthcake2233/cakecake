package agent

import (
	"cakecake/internal/aigateway"
	"cakecake/internal/aigateway/toolkit"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"go.uber.org/zap"
	"sort"
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

// toolActivityCollector captures tool calls/results so they can be persisted.
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
	if len(coll.results) > 0 {
		if sel, cleaned, ok := extractDisplaySelections(reply); ok {
			// The model explicitly decided which tool results to display; the
			// marker line is stripped from the visible reply.
			result.Content = cleaned
			if b := buildSelectedResultData(coll, sel); len(b) > 0 {
				result.ToolResultData = b
			}
			return result, nil
		}
		result.ToolResultData = buildReferencedResultData(reply, coll)
	}
	return result, nil
}

// buildReferencedResultData is the fallback used when the model did not emit a
// display marker: keep only result items whose title/content is cited in the
// reply, deduplicated across repeated tool calls.
func buildReferencedResultData(reply string, coll *toolActivityCollector) json.RawMessage {
	rm := make(map[string]json.RawMessage, len(coll.results))
	seen := make(map[string]bool)
	for k, v := range coll.results {
		if filtered := filterReferencedItems(reply, v, seen); filtered != nil {
			rm[k] = filtered
		}
	}
	if len(rm) == 0 {
		return nil
	}
	b, err := json.Marshal(rm)
	if err != nil {
		return nil
	}
	return b
}

// extractDisplaySelections parses the model-declared display list
// (【展示】search_videos#23,get_video_detail#24) and returns it keyed by tool
// name, plus the reply with the marker line removed. ok is false when the
// model did not emit a usable list (callers fall back to title matching).
func extractDisplaySelections(reply string) (map[string][]uint64, string, bool) {
	m := displayMarkerRe.FindStringSubmatch(reply)
	if m == nil {
		return nil, reply, false
	}
	sel := make(map[string][]uint64)
	payload := strings.NewReplacer(" ", "", "\t", "").Replace(m[1])
	for _, tok := range strings.FieldsFunc(payload, func(r rune) bool {
		return r == ',' || r == '，' || r == '、' || r == ';' || r == '；'
	}) {
		parts := strings.Split(strings.TrimSpace(tok), "#")
		if len(parts) != 2 {
			continue
		}
		tool := strings.TrimSpace(parts[0])
		if !validToolName(tool) {
			continue
		}
		id, err := strconv.ParseUint(strings.TrimSpace(parts[1]), 10, 64)
		if err != nil || id == 0 {
			continue
		}
		sel[tool] = append(sel[tool], id)
	}
	if len(sel) == 0 {
		return nil, reply, false
	}
	cleaned := strings.TrimSpace(displayMarkerLineRe.ReplaceAllString(reply, ""))
	return sel, cleaned, true
}

func validToolName(s string) bool {
	switch s {
	case "search_videos", "get_video_detail", "get_trending",
		"get_video_comments", "get_video_danmaku":
		return true
	}
	return false
}

// buildSelectedResultData assembles tool_result_data from the model-declared
// display list. Spans are visited in tool execution order and each requested
// id is matched against the items of the matching tool, deduplicated by item
// identity.
func buildSelectedResultData(coll *toolActivityCollector, sel map[string][]uint64) json.RawMessage {
	if coll == nil || len(sel) == 0 {
		return nil
	}
	type spanTool struct{ spanID, tool string }
	var order []spanTool
	toolBySpan := make(map[string]string)
	for _, act := range coll.acts {
		spanID, _ := act["span_id"].(string)
		tool, _ := act["tool_name"].(string)
		if spanID == "" || tool == "" {
			continue
		}
		if _, ok := toolBySpan[spanID]; !ok {
			order = append(order, spanTool{spanID: spanID, tool: tool})
		}
		toolBySpan[spanID] = tool
	}
	parsed := make(map[string][]map[string]interface{})
	for spanID, raw := range coll.results {
		var arr []map[string]interface{}
		if json.Unmarshal(raw, &arr) == nil {
			parsed[spanID] = arr
		}
	}
	out := make(map[string][]map[string]interface{})
	seen := make(map[string]bool)
	tools := make([]string, 0, len(sel))
	for tool := range sel {
		tools = append(tools, tool)
	}
	sort.Strings(tools)
	for _, tool := range tools {
		for _, id := range sel[tool] {
			for _, st := range order {
				if st.tool != tool {
					continue
				}
				found := false
				for _, it := range parsed[st.spanID] {
					if !itemIDMatches(it, id) {
						continue
					}
					key := itemKey(it)
					if key == "" || seen[key] {
						found = true
						break
					}
					seen[key] = true
					out[st.spanID] = append(out[st.spanID], it)
					found = true
					break
				}
				if found {
					break
				}
			}
		}
	}
	if len(out) == 0 {
		return nil
	}
	rm := make(map[string]json.RawMessage, len(out))
	for spanID, items := range out {
		if b, err := json.Marshal(items); err == nil {
			rm[spanID] = b
		}
	}
	b, err := json.Marshal(rm)
	if err != nil {
		return nil
	}
	return b
}

// itemIDMatches reports whether the item carries the requested id. Tool
// results use either "id" (search/detail) or "video_id" (trending); trending
// items also expose "rank", which the model may cite instead.
func itemIDMatches(it map[string]interface{}, id uint64) bool {
	for _, key := range []string{"id", "video_id", "rank"} {
		if raw, ok := it[key]; ok {
			if f, ok := raw.(float64); ok && f > 0 && uint64(f) == id {
				return true
			}
		}
	}
	return false
}

// filterReferencedItems keeps only tool result items that the final reply
// actually mentions (by title/content fragment). Unreferenced results (e.g. a
// search that found nothing relevant) are dropped so the UI never shows cards
// the assistant did not cite. Items already kept from an earlier tool call
// (e.g. the model searched twice with identical results) are deduplicated.
func filterReferencedItems(reply string, items json.RawMessage, seen map[string]bool) json.RawMessage {
	var arr []map[string]interface{}
	if err := json.Unmarshal(items, &arr); err != nil {
		return items
	}
	kept := make([]map[string]interface{}, 0, len(arr))
	for _, it := range arr {
		key := itemKey(it)
		if key == "" || seen[key] {
			continue
		}
		if itemReferenced(reply, it) {
			seen[key] = true
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

// itemKey returns a stable identity for deduplication (numeric id when
// available, otherwise title/content text).
func itemKey(it map[string]interface{}) string {
	for _, key := range []string{"id", "video_id"} {
		if id, ok := it[key]; ok {
			if f, ok := id.(float64); ok && f > 0 {
				return "id:" + strconv.FormatUint(uint64(f), 10)
			}
		}
	}
	for _, k := range []string{"title", "content"} {
		raw, ok := it[k]
		if !ok {
			continue
		}
		if t := strings.TrimSpace(fmt.Sprintf("%v", raw)); t != "" {
			return "text:" + t
		}
	}
	return ""
}

func itemReferenced(reply string, it map[string]interface{}) bool {
	for _, k := range []string{"title", "content"} {
		raw, ok := it[k]
		if !ok {
			continue
		}
		text := strings.TrimSpace(fmt.Sprintf("%v", raw))
		if text == "" {
			continue
		}
		r := []rune(text)
		prefix := ""
		if len(r) > 12 {
			prefix = string(r[:12])
		}
		for _, sentence := range splitSentences(reply) {
			if !strings.Contains(sentence, text) &&
				!strings.Contains(compactText(sentence), compactText(text)) &&
				(prefix == "" || !strings.Contains(sentence, prefix)) {
				continue
			}
			// The video is cited in this sentence: keep it unless the same
			// sentence dismisses it (e.g. "和编程无关").
			if !mdItemDismissRe.MatchString(sentence) {
				return true
			}
		}
	}
	return false
}

// compactText removes all whitespace so title citations tolerate spacing
// differences.
func compactText(s string) string {
	return strings.Join(strings.Fields(s), "")
}

// splitSentences splits a reply into sentences so dismissal words only affect
// the video cited in the same sentence, never the whole result set.
func splitSentences(reply string) []string {
	return strings.FieldsFunc(reply, func(r rune) bool {
		// Note: '？' is intentionally NOT a separator because video titles
		// often contain it (e.g. "《溯》，你是否还知道？"), which would split
		// the citation away from its dismissive tail.
		return r == '。' || r == '！' || r == '；' || r == '\n'
	})
}
