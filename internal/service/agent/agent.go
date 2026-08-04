package agent

import (
	"cakecake/internal/model/agent"
	"cakecake/internal/model/dm"
	"context"
	"crypto/rand"
	"encoding/hex"
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
}

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

func (s *AgentService) EnsureForUser(humanID uint64) error {
	if s == nil || s.Store == nil || humanID == 0 {
		return nil
	}
	return s.Store.EnsureAllAgentConversationsForUser(humanID)
}

func (s *AgentService) IsAgentConversation(conv *dm.DmConversation) bool {
	return conv != nil && conv.Kind == dm.DmKindAgent
}

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
	nRunes := utf8.RuneCountInString(content)
	if nRunes < 1 {
		return nil, fmt.Errorf("empty content")
	}
	if nRunes > 500 {
		r := []rune(content)
		content = string(r[:500])
	}
	now := time.Now()
	toolActivities := ""
	toolResultData := ""
	if len(extra) >= 2 {
		toolActivities = extra[0]
		toolResultData = extra[1]
	}
	msg := dm.DmMessage{
		ConversationID: conv.ID,
		SenderID:       botID,
		Role:           "assistant",
		Content:        content,
		ToolActivities: toolActivities,
		ToolResultData: toolResultData,
		CreatedAt:      now,
	}
	preview := content
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
	rand.Read(b)
	return hex.EncodeToString(b)
}

func (s *AgentService) setupToolCallbacks(traceID string, humanID uint64) {
	if s.Gateway == nil || s.ChatHub == nil {
		return
	}
	s.Gateway.OnToolCallStart = func(tid, spanID, parentSpanID, toolName string, argsJSON json.RawMessage) {
		var args interface{}
		json.Unmarshal(argsJSON, &args)
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
	if s.ToolExec != nil && len(toolkit.DefineTools(s.enabledTools())) > 0 {
		traceID := generateTraceID()
		s.setupToolCallbacks(traceID, conv.UserLow)
		defer s.clearToolCallbacks()
		coll = installToolCollectors(s.Gateway)
		tools := toolkit.DefineTools(s.enabledTools())
		s.Gateway.ToolExec = s.ToolExec
		reply, err = s.Gateway.CompleteUserTurnWithTools(ctx, conv.ID, userText, tools, traceID)
	} else {
		reply, err = s.Gateway.CompleteUserTurn(ctx, conv.ID, userText)
	}
	if err != nil {
		return nil, err
	}
	if s.Sens != nil {
		if err := s.Sens.Check(reply); err != nil {
			return &GenerateReplyResult{Content: "抱歉，AI 助手暂时无法回复，请稍后再试。"}, nil
		}
	}
	return buildReplyResult(reply, coll)
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
	if len(coll.results) > 0 {
		rm := make(map[string]json.RawMessage, len(coll.results))
		for k, v := range coll.results {
			rm[k] = v
		}
		if b, e := json.Marshal(rm); e == nil {
			result.ToolResultData = b
		}
	}
	return result, nil
}
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
