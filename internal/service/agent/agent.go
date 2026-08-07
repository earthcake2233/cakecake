package agent

import (
	"cakecake/internal/model/agent"
	"cakecake/internal/model/dm"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"cakecake/internal/aigateway"
	"cakecake/internal/aigateway/toolkit"
	"cakecake/internal/config"
	"cakecake/internal/data"
	"cakecake/internal/pkg/sensitive"
	"cakecake/internal/ws"
)

var (
	mdFenceRe      = regexp.MustCompile("(?s)```.*?```")
	mdLinkRe       = regexp.MustCompile(`\[([^\]]+)\]\([^)]*\)`)
	mdLinePrefixRe = regexp.MustCompile("(?m)^[#>*+\\-]\\s*")
	mdFenceLineRe  = regexp.MustCompile("(?m)^(\\s*)(`{3,})(.*)$")
	mdStrayFenceRe = regexp.MustCompile("`{3,}[a-zA-Z0-9_+-]*")
	mdDismissRe    = regexp.MustCompile(
		"没有(相关|什么)?(的)?(教程|内容|视频|投稿)" +
			"|确实没有|没有找到|没找到|暂无|暂时没有|无关|不相关|没关系" +
			"|八竿子打不着|没搜到|没有搜到|只有(动画|音乐|视频)",
	)
)

// AgentService runs AI assistant replies for agent DM threads.
// GenerateReplyResult holds the AI reply along with tool call metadata for persistence.
type GenerateReplyResult struct {
	Content        string          `json:"content"`
	ToolActivities json.RawMessage `json:"tool_activities,omitempty"`
	ToolResultData json.RawMessage `json:"tool_result_data,omitempty"`
	Suggestions    []string        `json:"suggestions,omitempty"`
}

// AgentService runs AI assistant replies for agent DM threads.
type AgentService struct {
	Cfg      *config.C
	Store    AgentStore
	Redis    *redis.Client
	Gateway  *aigateway.Gateway
	Sens     *sensitive.Filter
	ChatHub  *ws.ChatHub
	Log      *zap.Logger
	RC       *config.RuntimeConfig
	ToolExec toolkit.Executor

	genMu     sync.Mutex
	genStates map[uint64]*agentGenState
}

// agentGenState supports byte-level pause/resume of a running generation:
// while paused, streamed deltas are buffered instead of pushed, then flushed
// verbatim on resume (the same LLM stream keeps running).
type agentGenState struct {
	mu      sync.Mutex
	paused  bool
	buffer  []string
	dropped bool
	genID   uint64
	// pauseSeq increments on every stop; a resume only clears the paused flag
	// if no new stop happened while it was replaying the backlog.
	pauseSeq uint64
	// resuming guards the backlog replay so two continues never run it
	// concurrently (which would let live deltas interleave with the replay).
	resuming bool
}

// MaxProfiles returns the maximum number of agent profiles allowed.
func (s *AgentService) MaxProfiles() int {
	return data.MaxAgentProfilesLimit()
}

// UnmarshalWelcomeList parses welcome messages from JSON, falling back when needed.
func (s *AgentService) UnmarshalWelcomeList(raw json.RawMessage, fallback []string) ([]string, error) {
	return data.UnmarshalWelcomeList(raw, fallback)
}

// NormalizeSlug normalizes an agent profile slug.
func (s *AgentService) NormalizeSlug(slug string) (string, error) {
	return data.NormalizeAgentSlug(slug)
}

func (s *AgentService) gatewayReady() bool {
	enabled := false
	if s.RC != nil {
		enabled = s.RC.GetBool("agent_enabled", s.Cfg != nil && s.Cfg.AgentEnabled)
	}
	if !enabled && s.Cfg != nil {
		enabled = s.Cfg.AgentEnabled
	}
	return s != nil && enabled && s.Gateway != nil && s.Gateway.LLM != nil &&
		strings.TrimSpace(s.Cfg.DeepSeekAPIKey) != ""
}

func (s *AgentService) quotaKey(userID uint64) string {
	day := time.Now().Format("20060102")
	return fmt.Sprintf("mb:agent:quota:%d:%s", userID, day)
}

// CheckQuota reports whether the user still has daily agent quota (true when unconfigured).
func (s *AgentService) CheckQuota(ctx context.Context, userID uint64) bool {
	if s == nil || s.Redis == nil || s.Cfg == nil {
		return true
	}
	quota := s.Cfg.AgentDailyQuota
	if s.RC != nil {
		quota = s.RC.GetInt("agent_daily_quota", quota)
	}
	if quota <= 0 {
		return true
	}
	n, err := s.Redis.Get(ctx, s.quotaKey(userID)).Int()
	if err == redis.Nil {
		return true
	}
	return err != nil || n < quota
}

// IncrQuota increments the user's daily agent quota counter.
func (s *AgentService) IncrQuota(ctx context.Context, userID uint64) {
	if s == nil || s.Redis == nil {
		return
	}
	key := s.quotaKey(userID)
	pipe := s.Redis.Pipeline()
	pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, 48*time.Hour)
	_, _ = pipe.Exec(ctx)
}

// EnsureForUser ensures agent conversations exist for a human user.
func (s *AgentService) EnsureForUser(humanID uint64) error {
	if s == nil || s.Store == nil || humanID == 0 {
		return nil
	}
	return s.Store.EnsureAllAgentConversationsForUser(humanID)
}

// IsAgentConversation reports whether a conversation belongs to an agent.
func (s *AgentService) IsAgentConversation(conv *dm.DmConversation) bool {
	return conv != nil && conv.Kind == dm.DmKindAgent
}

// IsBotUser reports whether a user id belongs to an agent bot.
func (s *AgentService) IsBotUser(uid uint64) bool {
	if s == nil || s.Store == nil || uid == 0 {
		return false
	}
	_, err := s.Store.GetAgentProfileByBotUserID(uid)
	return err == nil
}

func (s *AgentService) profileForConversation(conv *dm.DmConversation) (*agent.AgentProfile, error) {
	if s == nil || s.Store == nil || conv == nil {
		return nil, fmt.Errorf("invalid conversation")
	}
	if conv.AgentProfileID > 0 {
		return s.Store.GetAgentProfile(conv.AgentProfileID)
	}
	if p, err := s.Store.GetAgentProfileByBotUserID(conv.UserLow); err == nil {
		return p, nil
	}
	return s.Store.GetAgentProfileByBotUserID(conv.UserHigh)
}

// PostAssistantMessage persists an assistant reply and updates the conversation.
func (s *AgentService) PostAssistantMessage(conv *dm.DmConversation, humanID uint64, content string, extra ...string) (*dm.DmMessage, error) {
	if s == nil || s.Store == nil || conv == nil {
		return nil, fmt.Errorf("agent service not ready")
	}
	profile, err := s.profileForConversation(conv)
	if err != nil {
		return nil, err
	}
	botID := profile.BotUserID
	content = strings.TrimSpace(content)
	content = normalizeMarkdownFences(content)
	content = dedupeConsecutiveLines(content)
	nRunes := utf8.RuneCountInString(content)
	if nRunes < 1 {
		return nil, fmt.Errorf("empty content")
	}
	if nRunes > 8000 {
		r := []rune(content)
		content = string(r[:8000])
	}
	now := time.Now()
	toolActivities := ""
	toolResultData := ""
	suggestions := ""
	if len(extra) >= 2 {
		toolActivities = extra[0]
		toolResultData = extra[1]
	}
	if len(extra) >= 3 {
		suggestions = extra[2]
	}
	msg := dm.DmMessage{
		ConversationID: conv.ID,
		SenderID:       botID,
		Role:           "assistant",
		Content:        content,
		ToolActivities: toolActivities,
		ToolResultData: toolResultData,
		Suggestions:    suggestions,
		CreatedAt:      now,
	}
	preview := plainTextPreview(content)
	if utf8.RuneCountInString(preview) > 80 {
		r := []rune(preview)
		preview = string(r[:80]) + "..."
	}
	if err := s.Store.PostAssistantMessageTx(&msg, conv, humanID, now, preview); err != nil {
		return nil, err
	}
	_ = s.Store.ReloadConversation(conv)
	return &msg, nil
}

// SetMessageFeedback records or toggles a user's like/dislike on a message.
func (s *AgentService) SetMessageFeedback(ctx context.Context, messageID uint64, userID uint64, feedback string) error {
	if s == nil || s.Store == nil {
		return fmt.Errorf("agent service not ready")
	}
	if feedback != "like" && feedback != "dislike" {
		return fmt.Errorf("invalid feedback value")
	}
	return s.Store.SetMessageFeedback(ctx, messageID, userID, feedback)
}

// ListAgentFeedbacks returns feedback rows for the admin console.
func (s *AgentService) ListAgentFeedbacks(ctx context.Context, limit int, offset int) ([]agent.AgentFeedback, error) {
	if s == nil || s.Store == nil {
		return nil, fmt.Errorf("agent service not ready")
	}
	return s.Store.ListAgentFeedbacks(ctx, limit, offset)
}

// ListAgentFeedbacksWithContent returns feedback rows joined with the rated
// assistant message content for the admin console.
func (s *AgentService) ListAgentFeedbacksWithContent(ctx context.Context, limit int, offset int) ([]AgentFeedbackRow, error) {
	if s == nil || s.Store == nil {
		return nil, fmt.Errorf("agent service not ready")
	}
	return s.Store.ListAgentFeedbacksWithContent(ctx, limit, offset)
}

// plainTextPreview strips common Markdown syntax so the conversation-list
// preview never exposes raw formatting (bold markers, list bullets, code
// fences, links) to the user.
func plainTextPreview(content string) string {
	text := content
	text = mdFenceRe.ReplaceAllString(text, " ")
	text = mdLinkRe.ReplaceAllString(text, "$1")
	text = strings.ReplaceAll(text, "**", "")
	text = strings.ReplaceAll(text, "__", "")
	text = strings.ReplaceAll(text, "`", "")
	text = mdLinePrefixRe.ReplaceAllString(text, "")
	text = strings.Join(strings.Fields(text), " ")
	return strings.TrimSpace(text)
}

func (s *AgentService) applyDynamicGatewayConfig() {
	if s.Gateway == nil || s.RC == nil {
		return
	}
	if v := s.RC.GetInt("agent_max_history", s.Gateway.MaxHistory); v > 0 {
		s.Gateway.MaxHistory = v
	}
	if v := s.RC.Get("agent_history_ttl", ""); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			s.Gateway.HistoryTTL = d
		}
	}
}

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

// humanPeerForConversation returns the non-bot participant of an agent thread.
// Dm conversations store participants as (user_low, user_high), so the bot may
// be either side; the streaming push target must always be the human user.
func humanPeerForConversation(conv *dm.DmConversation, botUserID uint64) uint64 {
	if conv == nil {
		return 0
	}
	if conv.UserLow == botUserID {
		return conv.UserHigh
	}
	if conv.UserHigh == botUserID {
		return conv.UserLow
	}
	// Unknown bot side: prefer the higher id (agent bots are seeded early with
	// small ids, while real users get larger sequential ids).
	return conv.UserHigh
}

// deltaSender returns a per-generation stream callback that routes deltas to
// the human user's ChatHub connection and honors pause/drop generation state.
// Each LLM call gets its own closure (capturing its own genID), so concurrent
// users and superseded generations never cross-wire or leak late deltas.
func (s *AgentService) deltaSender(humanID uint64, genID uint64) func(string) {
	return func(delta string) {
		if delta == "" || s.ChatHub == nil {
			return
		}
		if st := s.generationState(humanID); st != nil {
			st.mu.Lock()
			if st.dropped || st.genID != genID {
				st.mu.Unlock()
				return
			}
			if st.paused {
				st.buffer = append(st.buffer, delta)
				st.mu.Unlock()
				return
			}
			st.mu.Unlock()
		}
		s.ChatHub.PushJSON(humanID, map[string]interface{}{
			"type": "agent_delta",
			"body": map[string]interface{}{
				"content": delta,
			},
		})
	}
}

// currentGenID returns the generation id registered for the user, or 0.
func (s *AgentService) currentGenID(uid uint64) uint64 {
	st := s.generationState(uid)
	if st == nil {
		return 0
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	return st.genID
}

func (s *AgentService) generationState(uid uint64) *agentGenState {
	if uid == 0 {
		return nil
	}
	s.genMu.Lock()
	defer s.genMu.Unlock()
	if s.genStates == nil {
		return nil
	}
	return s.genStates[uid]
}

// BeginGeneration registers a new generation state before the LLM call, so
// OnTextDelta routes this generation's deltas correctly.
func (s *AgentService) BeginGeneration(uid uint64, genID uint64) {
	if uid == 0 {
		return
	}
	s.genMu.Lock()
	defer s.genMu.Unlock()
	if s.genStates == nil {
		s.genStates = make(map[uint64]*agentGenState)
	}
	s.genStates[uid] = &agentGenState{genID: genID}
}

// EndGeneration removes the generation state only if it still belongs to the
// given generation id (a finished goroutine can never clear a newer state).
func (s *AgentService) EndGeneration(uid uint64, genID uint64) {
	if uid == 0 {
		return
	}
	s.genMu.Lock()
	defer s.genMu.Unlock()
	if st, ok := s.genStates[uid]; ok && st.genID == genID {
		delete(s.genStates, uid)
	}
}

// DropCurrentGeneration marks the user's current generation as dropped (its
// buffered/live deltas are discarded). The state stays registered so late
// deltas from the old stream are still recognized and dropped.
func (s *AgentService) DropCurrentGeneration(uid uint64) {
	if uid == 0 {
		return
	}
	s.genMu.Lock()
	if s.genStates == nil {
		s.genStates = make(map[uint64]*agentGenState)
	}
	st := s.genStates[uid]
	if st == nil {
		st = &agentGenState{}
		s.genStates[uid] = st
	}
	s.genMu.Unlock()
	st.mu.Lock()
	st.dropped = true
	st.mu.Unlock()
}

// PauseGeneration stops pushing streamed deltas; they are buffered so a later
// ResumeGeneration can flush them verbatim (byte-level continuation).
func (s *AgentService) PauseGeneration(uid uint64) {
	if uid == 0 {
		return
	}
	s.genMu.Lock()
	if s.genStates == nil {
		s.genStates = make(map[uint64]*agentGenState)
	}
	st := s.genStates[uid]
	if st == nil {
		st = &agentGenState{}
		s.genStates[uid] = st
	}
	s.genMu.Unlock()
	st.mu.Lock()
	st.paused = true
	st.pauseSeq++
	st.mu.Unlock()
}

// ResumeGeneration un-pauses and flushes the buffered deltas in order.
func (s *AgentService) ResumeGeneration(uid uint64) {
	if uid == 0 {
		return
	}
	if s.ChatHub == nil {
		return
	}
	for {
		st := s.generationState(uid)
		if st == nil {
			return
		}
		st.mu.Lock()
		if st.dropped {
			st.mu.Unlock()
			return
		}
		if st.resuming {
			// Another continue is already replaying the backlog; wait for it
			// to finish so it can drain anything this continue left buffered.
			st.mu.Unlock()
			time.Sleep(5 * time.Millisecond)
			continue
		}
		st.resuming = true
		seq := st.pauseSeq
		buf := st.buffer
		st.buffer = nil
		st.mu.Unlock()

		for i, d := range buf {
			if d == "" {
				continue
			}
			// A stop clicked during the replay must interrupt it promptly:
			// check before every pushed fragment and leave the rest buffered.
			st.mu.Lock()
			repaused := st.pauseSeq != seq
			st.mu.Unlock()
			if repaused {
				st.mu.Lock()
				st.buffer = append(append([]string{}, buf[i:]...), st.buffer...)
				st.resuming = false
				st.mu.Unlock()
				return
			}
			s.ChatHub.PushJSON(uid, map[string]interface{}{
				"type": "agent_delta",
				"body": map[string]interface{}{"content": d},
			})
			// Pace the backlog flush so the UI keeps a typewriter feel instead
			// of dumping the whole paused buffer at once.
			if len(buf) > 1 {
				time.Sleep(12 * time.Millisecond)
			}
		}

		st.mu.Lock()
		if st.dropped {
			st.resuming = false
			st.mu.Unlock()
			return
		}
		repaused := st.pauseSeq != seq
		more := len(st.buffer) > 0
		st.resuming = false
		if repaused {
			// A new stop arrived during the replay: keep paused so the
			// remaining deltas stay buffered for the next continue.
			st.mu.Unlock()
			return
		}
		if more {
			// Deltas arrived during the replay (we were still paused):
			// drain them in the next pass.
			st.mu.Unlock()
			continue
		}
		st.paused = false
		st.mu.Unlock()
		return
	}
}

// IsGenerationPaused reports whether the user's generation is paused.
func (s *AgentService) IsGenerationPaused(uid uint64) bool {
	st := s.generationState(uid)
	if st == nil {
		return false
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	return st.paused
}

// ClearGenerationState removes the user's pause/buffer state.
func (s *AgentService) ClearGenerationState(uid uint64) {
	if uid == 0 {
		return
	}
	s.genMu.Lock()
	defer s.genMu.Unlock()
	delete(s.genStates, uid)
}

func (s *AgentService) clearToolCallbacks() {
	if s.Gateway != nil {
		s.Gateway.OnToolCallStart = nil
		s.Gateway.OnToolCallEnd = nil
		s.Gateway.OnToolResultData = nil
	}
}

// GenerateReply produces an AI reply (with tool calls) for a user message.
func (s *AgentService) GenerateReply(ctx context.Context, conv *dm.DmConversation, userText string) (*GenerateReplyResult, error) {
	if !s.gatewayReady() {
		return nil, fmt.Errorf("ai assistant is not configured")
	}
	s.applyDynamicGatewayConfig()
	profile, err := s.profileForConversation(conv)
	if err != nil {
		return nil, fmt.Errorf("ai assistant profile missing")
	}
	if !profile.Enabled {
		return nil, fmt.Errorf("ai assistant is disabled")
	}
	humanID := humanPeerForConversation(conv, profile.BotUserID)
	if humanID == 0 {
		return nil, fmt.Errorf("agent conversation has no human participant")
	}
	if s.Sens != nil {
		if err := s.Sens.Check(userText); err != nil {
			return nil, fmt.Errorf("message contains sensitive words")
		}
	}
	if strings.TrimSpace(profile.SystemPrompt) == "" {
		return nil, fmt.Errorf("empty system prompt")
	}
	restore := s.agentSystemPrompt(profile)
	defer restore()

	ctx, cancel := context.WithTimeout(ctx, s.agentReplyTimeout())
	defer cancel()

	var coll *toolActivityCollector
	var reply string
	genID := s.currentGenID(humanID)
	if s.ToolExec != nil && len(toolkit.DefineTools(s.enabledTools())) > 0 {
		traceID := generateTraceID()
		s.setupToolCallbacks(traceID, humanID)
		defer s.clearToolCallbacks()
		coll = installToolCollectors(s.Gateway)
		tools := toolkit.DefineTools(s.enabledTools())
		s.Gateway.ToolExec = s.ToolExec
		reply, err = s.Gateway.CompleteUserTurnWithToolsStream(ctx, conv.ID, userText, tools, traceID, s.deltaSender(humanID, genID))
	} else {
		s.setupToolCallbacks("", humanID)
		defer s.clearToolCallbacks()
		reply, err = s.Gateway.CompleteUserTurnStream(ctx, conv.ID, userText, s.deltaSender(humanID, genID))
	}
	if err != nil {
		return nil, err
	}
	if s.Sens != nil {
		if err := s.Sens.Check(reply); err != nil {
			return &GenerateReplyResult{Content: "抱歉，AI 助手暂时无法回复，请稍后再试。"}, nil
		}
	}
	result, err := buildReplyResult(reply, coll)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// GenerateSuggestions asks the model for follow-up question chips based on a
// finished reply. Callers run it after the reply is persisted so the message
// can be shown immediately instead of waiting for a second LLM round trip.
func (s *AgentService) GenerateSuggestions(ctx context.Context, reply string) []string {
	return s.generateSuggestions(ctx, reply)
}

// UpdateMessageSuggestions persists follow-up chips on an assistant message.
func (s *AgentService) UpdateMessageSuggestions(ctx context.Context, messageID uint64, suggestions []string) error {
	if s == nil || s.Store == nil {
		return fmt.Errorf("agent service not ready")
	}
	return s.Store.UpdateMessageSuggestions(ctx, messageID, suggestions)
}

// ContinueReplyStream resumes a user-stopped reply from the partial text and
// also generates follow-up suggestions. The returned string is the stitched
// FULL reply (seam duplicates between the partial tail and the continuation
// head are removed so the model's re-emitted lines/fences never appear twice).
func (s *AgentService) ContinueReplyStream(ctx context.Context, conv *dm.DmConversation, partial string) (string, []string, error) {
	if !s.gatewayReady() {
		return "", nil, fmt.Errorf("ai assistant is not configured")
	}
	profile, err := s.profileForConversation(conv)
	if err != nil {
		return "", nil, fmt.Errorf("ai assistant profile missing")
	}
	if !profile.Enabled {
		return "", nil, fmt.Errorf("ai assistant is disabled")
	}
	humanID := humanPeerForConversation(conv, profile.BotUserID)
	if humanID == 0 {
		return "", nil, fmt.Errorf("agent conversation has no human participant")
	}
	restore := s.agentSystemPrompt(profile)
	defer restore()

	ctx, cancel := context.WithTimeout(ctx, s.agentReplyTimeout())
	defer cancel()

	s.setupToolCallbacks("", humanID)
	defer s.clearToolCallbacks()
	genID := s.currentGenID(humanID)

	instruction := "请从中断处直接继续你的回答，不要重复已经写过的内容，也不要另起一段重新讲解。"
	if partialEndsInsideCodeFence(partial) {
		instruction = "你现在正处于未闭合的代码块内部：请先接着写完这段代码（不要重复已写行，不要跳出代码块写新段落），用三个反引号闭合代码块后，如有必要再用一两句话继续讲解。"
	}
	replyMsg, err := s.Gateway.ContinueTurnStream(ctx, conv.ID, partial, instruction, s.deltaSender(humanID, genID))
	if err != nil {
		return "", nil, err
	}
	continuationText := strings.TrimSpace(replyMsg.Content)
	full := mergeContinuation(strings.TrimSpace(partial), continuationText)
	// Suggestions are attached asynchronously by the handler after persistence,
	// so continue never blocks the final message on a second LLM round trip.
	return full, nil, nil
}

// mergeContinuation stitches the stopped partial and the model's continuation,
// removing the longest suffix of partial that the model re-emitted as the
// prefix of continuation (common when asked to continue from a code fence or a
// partial line).
func mergeContinuation(partial string, continuation string) string {
	p := strings.TrimSpace(partial)
	c := strings.TrimSpace(continuation)
	if p == "" {
		return c
	}
	if c == "" {
		return p
	}
	np := strings.Join(strings.Fields(p), " ")
	nc := strings.Join(strings.Fields(c), " ")
	nr := []rune(np)
	cr := []rune(nc)
	maxOverlap := len(nr)
	if l := len(cr); l < maxOverlap {
		maxOverlap = l
	}
	best := 0
	for n := maxOverlap; n >= 1; n-- {
		if strings.HasSuffix(string(nr), string(cr[:n])) {
			best = n
			break
		}
	}
	if best == 0 {
		c = dropSeamDuplicateLines(p, c)
		return p + "\n" + c
	}
	return p + string([]rune(c)[normOverlapCut(c, best):])
}

// dropSeamDuplicateLines merges the seam when the continuation re-emits the
// partial's trailing line: either verbatim (drop the duplicate) or by
// restarting the whole line from its beginning (drop the partial's incomplete
// tail line and keep the continuation's full line).
func dropSeamDuplicateLines(p string, c string) string {
	pl := strings.Split(p, "\n")
	cl := strings.Split(c, "\n")
	for len(cl) > 0 && len(pl) > 0 {
		first := strings.TrimSpace(cl[0])
		last := strings.TrimSpace(pl[len(pl)-1])
		if first == "" {
			cl = cl[1:]
			continue
		}
		if first == last {
			cl = cl[1:]
			pl = pl[:len(pl)-1]
			continue
		}
		if last != "" && utf8.RuneCountInString(last) >= 4 && strings.HasPrefix(first, last) {
			// The model restarted the whole line from its beginning: keep the
			// continuation's complete line and drop the partial's half line.
			pl = pl[:len(pl)-1]
			return strings.Join(append(pl, cl...), "\n")
		}
		break
	}
	return strings.Join(cl, "\n")
}

// normOverlapCut maps a whitespace-normalized overlap length back to a rune
// offset in the original string.
func normOverlapCut(s string, target int) int {
	runes := []rune(s)
	acc := 0
	prevSpace := false
	for i, r := range runes {
		sp := unicode.IsSpace(r)
		if sp {
			if !prevSpace {
				acc++
			}
			prevSpace = true
		} else {
			acc++
			prevSpace = false
		}
		if acc >= target {
			return i + 1
		}
	}
	return len(runes)
}

// dedupeConsecutiveLines removes exact consecutive duplicate lines (the model
// sometimes re-emits a block when continuing).
func dedupeConsecutiveLines(text string) string {
	lines := strings.Split(text, "\n")
	out := make([]string, 0, len(lines))
	for i, ln := range lines {
		if i > 0 && ln == lines[i-1] {
			continue
		}
		out = append(out, ln)
	}
	return strings.Join(out, "\n")
}

// generateSuggestions asks the model for 3 short follow-up questions based on
// the reply, so the UI can render contextual suggestion chips. Fail-soft:
// returns nil on any error or unparseable output.
func (s *AgentService) generateSuggestions(ctx context.Context, reply string) []string {
	if s.Gateway == nil || s.Gateway.LLM == nil || strings.TrimSpace(reply) == "" {
		return nil
	}
	r := []rune(reply)
	if len(r) > 1500 {
		reply = string(r[:1500])
	}
	msgs := []aigateway.ChatMessage{
		{Role: "system", Content: "你是对话助手。基于下面这段 AI 回答，生成用户最可能继续追问的 3 个短问题。只输出 JSON 数组，例如 [\"问题一\",\"问题二\",\"问题三\"]，不要输出其他内容。"},
		{Role: "user", Content: reply},
	}
	ctx2, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	out, err := s.Gateway.LLM.Complete(ctx2, msgs)
	if err != nil || strings.TrimSpace(out) == "" {
		return nil
	}
	return parseSuggestionsJSON(out)
}

func parseSuggestionsJSON(raw string) []string {
	start := strings.Index(raw, "[")
	end := strings.LastIndex(raw, "]")
	if start < 0 || end <= start {
		return nil
	}
	var arr []string
	if err := json.Unmarshal([]byte(raw[start:end+1]), &arr); err != nil {
		return nil
	}
	out := make([]string, 0, len(arr))
	for _, v := range arr {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		out = append(out, v)
		if len(out) >= 3 {
			break
		}
	}
	return out
}

// partialEndsInsideCodeFence reports whether the stopped reply ends inside an
// unclosed fenced code block (odd number of fence lines so far).
func partialEndsInsideCodeFence(partial string) bool {
	fences := 0
	for _, ln := range strings.Split(partial, "\n") {
		t := strings.TrimSpace(ln)
		if strings.HasPrefix(t, "```") {
			fences++
		}
	}
	return fences%2 == 1
}

// normalizeMarkdownFences balances fenced code blocks: every fence line is
// normalized to exactly three backticks, and an unclosed fence is closed at
// the end so the rendered reply never breaks the chat layout.
func normalizeMarkdownFences(text string) string {
	if !strings.Contains(text, "`") {
		return text
	}
	lines := strings.Split(text, "\n")
	open := false
	for i, ln := range lines {
		if !mdFenceLineRe.MatchString(ln) {
			// Stray fence markers mid-line (e.g. the model re-emitted a fence
			// at a continuation seam) are not valid markdown; strip them.
			if strings.Contains(ln, "```") {
				lines[i] = mdStrayFenceRe.ReplaceAllString(ln, "")
			}
			continue
		}
		m := mdFenceLineRe.FindStringSubmatch(ln)
		if open && strings.TrimSpace(m[3]) != "" {
			// A language-tagged fence while already inside a code block is a
			// continuation-seam artifact (the model re-emitted the opener).
			// Drop the stray line; only a bare ``` legitimately closes.
			lines[i] = ""
			continue
		}
		lines[i] = m[1] + "```" + m[3]
		open = !open
	}
	if open {
		lines = append(lines, "```")
	}
	return strings.Join(lines, "\n")
}

// agentReplyTimeout resolves the effective LLM request timeout (runtime config first).
func (s *AgentService) agentReplyTimeout() time.Duration {
	timeout := 90 * time.Second
	if s.Cfg != nil && s.Cfg.AgentRequestTimeout > 0 {
		timeout = s.Cfg.AgentRequestTimeout
	}
	if s.RC != nil {
		if v := s.RC.Get("agent_request_timeout", ""); v != "" {
			if d, err := time.ParseDuration(v); err == nil && d > 0 {
				timeout = d
			}
		}
	}
	return timeout
}

// agentSystemPrompt swaps in the effective system prompt, returning a restore func.
func (s *AgentService) agentSystemPrompt(profile *agent.AgentProfile) func() {
	globalPrompt := s.Store.GetGlobalSystemPrompt()
	prev := s.Gateway.SystemPrompt
	s.Gateway.SystemPrompt = globalPrompt + "\n\n" + strings.TrimSpace(profile.SystemPrompt)
	return func() { s.Gateway.SystemPrompt = prev }
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

// ResetConversation clears an agent conversation and posts a fresh opening message.
func (s *AgentService) ResetConversation(ctx context.Context, conv *dm.DmConversation, humanID uint64) (*dm.DmMessage, error) {
	if s == nil || s.Store == nil || conv == nil || humanID == 0 {
		return nil, fmt.Errorf("agent service not ready")
	}
	profile, err := s.profileForConversation(conv)
	if err != nil {
		return nil, err
	}
	welcome := data.PickWelcomeMessage(profile)
	now := time.Now()
	preview := welcome
	if utf8.RuneCountInString(preview) > 80 {
		r := []rune(preview)
		preview = string(r[:80]) + "..."
	}
	msg := dm.DmMessage{
		ConversationID: conv.ID,
		SenderID:       profile.BotUserID,
		Role:           "assistant",
		Content:        welcome,
		CreatedAt:      now,
	}
	if err := s.Store.ResetConversationTx(conv, &msg, humanID, now, preview); err != nil {
		return nil, err
	}
	if s.Gateway != nil {
		s.Gateway.ClearHistory(ctx, conv.ID)
	}
	_ = s.Store.ReloadConversation(conv)
	return &msg, nil
}

// ReloadProfiles is a no-op placeholder after multi-profile migration.
func (s *AgentService) ReloadProfiles() {}

// ListAgentProfiles returns all agent profiles.
func (s *AgentService) ListAgentProfiles(ctx context.Context) ([]agent.AgentProfile, error) {
	return s.Store.ListAgentProfiles(ctx)
}

// GetAgentProfile returns an agent profile by ID.
func (s *AgentService) GetAgentProfile(ctx context.Context, id uint64) (*agent.AgentProfile, error) {
	return s.Store.GetAgentProfile(id)
}

// CreateAgentProfile creates a new agent profile.
func (s *AgentService) CreateAgentProfile(ctx context.Context, p *agent.AgentProfile) error {
	return s.Store.CreateAgentProfile(ctx, p)
}

// UpdateAgentProfile updates an agent profile.
func (s *AgentService) UpdateAgentProfile(ctx context.Context, id uint64, updates map[string]interface{}) error {
	return s.Store.UpdateAgentProfile(ctx, id, updates)
}

// DeleteAgentProfile deletes an agent profile by ID.
func (s *AgentService) DeleteAgentProfile(ctx context.Context, id uint64) error {
	return s.Store.DeleteAgentProfile(ctx, id)
}

// CountActiveAgentProfiles returns the count of enabled agent profiles.
func (s *AgentService) CountActiveAgentProfiles(ctx context.Context) (int64, error) {
	return s.Store.CountActiveAgentProfiles(ctx)
}

// CheckAgentSlugExists checks if a slug is already taken.
func (s *AgentService) CheckAgentSlugExists(ctx context.Context, slug string) (bool, error) {
	return s.Store.CheckAgentSlugExists(ctx, slug)
}

// UpdateAgentAvatar updates the avatar_url of an agent profile.
func (s *AgentService) UpdateAgentAvatar(ctx context.Context, id uint64, avatarURL string) error {
	return s.Store.UpdateAgentAvatar(ctx, id, avatarURL)
}

// GetGlobalSystemPrompt returns the global system prompt from agent settings.
func (s *AgentService) GetGlobalSystemPrompt(ctx context.Context) string {
	return s.Store.GetGlobalSystemPrompt()
}

// ProfileCount returns total number of agent profiles.
func (s *AgentService) ProfileCount(ctx context.Context) (int64, error) {
	return s.Store.ProfileCount(ctx)
}

// CreateAgentBotUser creates a non-login system user for a new profile.
func (s *AgentService) CreateAgentBotUser(ctx context.Context, slug, displayName, sign, avatarURL string) (uint64, error) {
	return s.Store.CreateAgentBotUser(ctx, slug, displayName, sign, avatarURL)
}

// RenameAgentProfileSlug updates a profile slug and the linked bot user username.
func (s *AgentService) RenameAgentProfileSlug(ctx context.Context, p *agent.AgentProfile, newSlug string) error {
	return s.Store.RenameAgentProfileSlug(ctx, p, newSlug)
}

// SyncAgentProfile copies profile display fields onto the bot user row.
func (s *AgentService) SyncAgentProfile(ctx context.Context, p *agent.AgentProfile) error {
	return s.Store.SyncAgentProfile(ctx, p)
}

// EnsureAgentProfiles migrates legacy settings and guarantees at least one profile.
func (s *AgentService) EnsureAgentProfiles(ctx context.Context) error {
	return s.Store.EnsureAgentProfiles(s.Cfg, s.Log)
}

// AgentBotUsername returns the internal username for a profile slug.
func AgentBotUsername(slug string) string {
	return data.AgentBotUsername(slug)
}
