package agent

import (
	"cakecake/internal/model/agent"
	"cakecake/internal/model/dm"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"
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
	// Dm is the read-only DM port used by the generation orchestration
	// (regenerate lookup, duplicate-reply guard). Wired at the composition
	// root; injected as an interface to keep the service free of HTTP concerns.
	Dm DmReader
	// Pusher delivers formatted agent events to the human user. It is the only
	// transport seam left in the orchestration: the service decides WHAT to
	// push, the adapter decides HOW (WebSocket formatting).
	Pusher ReplyPusher
	// Relay is the cross-instance Redis Pub/Sub transport for agent events and
	// generation control commands. When nil the service falls back to pushing
	// directly to the local ChatHub (single-process mode / unit tests).
	Relay *AgentRelay
	// InstanceID uniquely identifies this replica; it is stamped into the
	// generation snapshot so pause/resume controls can be routed to the owner.
	InstanceID string
	// EventHook is a test/telemetry hook invoked for every published agent
	// event after delivery. It lets unit tests observe cross-instance events
	// without a real WebSocket connection.
	EventHook func(uid uint64, payload map[string]interface{})

	genMu     sync.Mutex
	genStates map[uint64]*agentGenState
	genSeq    uint64
	draftMu   sync.Mutex
	lastDraft map[uint64]string

	runLocksMu sync.Mutex
	runLocks   map[uint64]*sync.Mutex
}

// DmReader is the DM read port required by agent orchestration.
type DmReader interface {
	GetConversationByID(ctx context.Context, convID uint64) (*dm.DmConversation, error)
	GetParticipant(ctx context.Context, convID uint64, userID uint64) (*dm.DmParticipant, error)
	ListMessages(ctx context.Context, convID uint64, beforeID uint64, limit int) ([]dm.DmMessage, error)
}

// ReplyPusher is the formatting port for persisted agent messages. Implemented
// by the HTTP/WS adapter: it formats presentation DTOs and returns the WS
// payloads; the service is responsible for delivering them (locally or via the
// cross-instance relay).
type ReplyPusher interface {
	FormatAgentMessage(ctx context.Context, humanID uint64, conv *dm.DmConversation, msg *dm.DmMessage) ([]map[string]interface{}, error)
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
		s.Cfg != nil && strings.TrimSpace(s.Cfg.DeepSeekAPIKey) != ""
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
