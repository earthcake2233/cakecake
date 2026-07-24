package service

import (
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
	"gorm.io/gorm"

	"minibili/internal/aigateway"
	"minibili/internal/aigateway/toolkit"
	"minibili/internal/config"
	"minibili/internal/data"
	"minibili/internal/model"
	"minibili/internal/pkg/sensitive"
	"minibili/internal/ws"
)

// AgentService runs AI assistant replies for agent DM threads.
// GenerateReplyResult holds the AI reply along with tool call metadata for persistence.
type GenerateReplyResult struct {
	Content         string          `json:"content"`
	ToolActivities  json.RawMessage `json:"tool_activities,omitempty"`
	ToolResultData  json.RawMessage `json:"tool_result_data,omitempty"`
}

type AgentService struct {
	Cfg        *config.C
	DB         *gorm.DB
	Redis      *redis.Client
	Gateway    *aigateway.Gateway
	Sens       *sensitive.Filter
	ChatHub    *ws.ChatHub
	Log        *zap.Logger
	RC         *config.RuntimeConfig
	ToolExec   toolkit.Executor
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
	if s == nil || s.DB == nil || humanID == 0 {
		return nil
	}
	return data.EnsureAllAgentConversationsForUser(s.DB, humanID)
}

func (s *AgentService) IsAgentConversation(conv *model.DmConversation) bool {
	return conv != nil && conv.Kind == model.DmKindAgent
}

func (s *AgentService) IsBotUser(uid uint64) bool {
	if s == nil || s.DB == nil || uid == 0 {
		return false
	}
	_, err := data.GetAgentProfileByBotUserID(s.DB, uid)
	return err == nil
}

func (s *AgentService) profileForConversation(conv *model.DmConversation) (*model.AgentProfile, error) {
	if s == nil || s.DB == nil || conv == nil {
		return nil, fmt.Errorf("invalid conversation")
	}
	if conv.AgentProfileID > 0 {
		return data.GetAgentProfile(s.DB, conv.AgentProfileID)
	}
	if p, err := data.GetAgentProfileByBotUserID(s.DB, conv.UserLow); err == nil {
		return p, nil
	}
	return data.GetAgentProfileByBotUserID(s.DB, conv.UserHigh)
}

func (s *AgentService) PostAssistantMessage(conv *model.DmConversation, humanID uint64, content string, extra ...string) (*model.DmMessage, error) {
	if s == nil || s.DB == nil || conv == nil {
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
	msg := model.DmMessage{
		ConversationID: conv.ID,
		SenderID:       botID,
		Role:           "assistant",
		Content:        content,
		ToolActivities: toolActivities,
		ToolResultData: toolResultData,
		CreatedAt:      now,
	}
	tx := s.DB.Begin()
	if err := tx.Create(&msg).Error; err != nil {
		tx.Rollback()
		return nil, err
	}
	preview := content
	if utf8.RuneCountInString(preview) > 80 {
		r := []rune(preview)
		preview = string(r[:80]) + "..."
	}
	if err := tx.Model(conv).Updates(map[string]interface{}{
		"last_message_at": now,
		"last_preview":    preview,
	}).Error; err != nil {
		tx.Rollback()
		return nil, err
	}
	if err := tx.Model(&model.DmParticipant{}).
		Where("conversation_id = ? AND user_id = ?", conv.ID, humanID).
		Updates(map[string]interface{}{
			"unread_count": gorm.Expr("unread_count + ?", 1),
			"hidden_at":    nil,
		}).Error; err != nil {
		tx.Rollback()
		return nil, err
	}
	if err := tx.Commit().Error; err != nil {
		return nil, err
	}
	_ = s.DB.First(conv, conv.ID)
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
			"trace_id":        tid,
			"span_id":         spanID,
			"parent_span_id":  parentSpanID,
			"tool_name":       toolName,
			"arguments":       args,
			"started_at":      time.Now().Format(time.RFC3339),
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

func (s *AgentService) GenerateReply(ctx context.Context, conv *model.DmConversation, userText string) (*GenerateReplyResult, error) {
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
	profilePrompt := strings.TrimSpace(profile.SystemPrompt)
	if profilePrompt == "" {
		return nil, fmt.Errorf("empty system prompt")
	}
	globalPrompt := data.GetGlobalSystemPrompt(s.DB)
	prev := s.Gateway.SystemPrompt
	s.Gateway.SystemPrompt = globalPrompt + "\n\n" + profilePrompt
	defer func() { s.Gateway.SystemPrompt = prev }()

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
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Decide whether to use tools
	useTools := s.ToolExec != nil && len(toolkit.DefineTools(s.enabledTools())) > 0

	var reply string
	// Collect tool data for persistence
	var toolActs []map[string]interface{}
	var toolResults = make(map[string]json.RawMessage)
	var mu sync.Mutex

	if useTools {
		traceID := generateTraceID()
		s.setupToolCallbacks(traceID, conv.UserLow)
		defer s.clearToolCallbacks()

		// Wrap callbacks to also collect data for persistence
		if s.Gateway.OnToolCallStart != nil {
			orig := s.Gateway.OnToolCallStart
			s.Gateway.OnToolCallStart = func(tid, spanID, parentSpanID, toolName string, argsJSON json.RawMessage) {
				orig(tid, spanID, parentSpanID, toolName, argsJSON)
				mu.Lock()
				toolActs = append(toolActs, map[string]interface{}{
					"trace_id":   tid,
					"span_id":    spanID,
					"tool_name":  toolName,
					"status":     "running",
				})
				mu.Unlock()
			}
		}
		if s.Gateway.OnToolCallEnd != nil {
			orig := s.Gateway.OnToolCallEnd
			s.Gateway.OnToolCallEnd = func(tid, spanID, toolName string, durationMs int64, resultSummary string) {
				orig(tid, spanID, toolName, durationMs, resultSummary)
				mu.Lock()
				for i := range toolActs {
					if toolActs[i]["span_id"] == spanID {
						toolActs[i]["status"] = "done"
						toolActs[i]["duration_ms"] = durationMs
						toolActs[i]["result_summary"] = resultSummary
						break
					}
				}
				mu.Unlock()
			}
		}
		if s.Gateway.OnToolResultData != nil {
			orig := s.Gateway.OnToolResultData
			s.Gateway.OnToolResultData = func(tid, spanID, toolName string, items json.RawMessage) {
				orig(tid, spanID, toolName, items)
				mu.Lock()
				toolResults[spanID] = items
				mu.Unlock()
			}
		}

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
			return &GenerateReplyResult{Content: "???????????????????????"}, nil
		}
	}

	result := &GenerateReplyResult{Content: reply}
	if len(toolActs) > 0 {
		if b, e := json.Marshal(toolActs); e == nil {
			result.ToolActivities = b
		}
	}
	if len(toolResults) > 0 {
		rm := make(map[string]json.RawMessage)
		for k, v := range toolResults {
			rm[k] = v
		}
		if b, e := json.Marshal(rm); e == nil {
			result.ToolResultData = b
		}
	}
	return result, nil
}
func (s *AgentService) ResetConversation(ctx context.Context, conv *model.DmConversation, humanID uint64) (*model.DmMessage, error) {
	if s == nil || s.DB == nil || conv == nil || humanID == 0 {
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
	msg := model.DmMessage{
		ConversationID: conv.ID,
		SenderID:       profile.BotUserID,
		Role:           "assistant",
		Content:        welcome,
		CreatedAt:      now,
	}
	tx := s.DB.Begin()
	if err := tx.Where("conversation_id = ?", conv.ID).Delete(&model.DmMessage{}).Error; err != nil {
		tx.Rollback()
		return nil, err
	}
	if err := tx.Create(&msg).Error; err != nil {
		tx.Rollback()
		return nil, err
	}
	if err := tx.Model(conv).Updates(map[string]interface{}{
		"last_message_at": now,
		"last_preview":    preview,
	}).Error; err != nil {
		tx.Rollback()
		return nil, err
	}
	if err := tx.Model(&model.DmParticipant{}).
		Where("conversation_id = ? AND user_id = ?", conv.ID, humanID).
		Updates(map[string]interface{}{
			"unread_count": 0,
			"hidden_at":    nil,
		}).Error; err != nil {
		tx.Rollback()
		return nil, err
	}
	if err := tx.Commit().Error; err != nil {
		return nil, err
	}
	if s.Gateway != nil {
		s.Gateway.ClearHistory(ctx, conv.ID)
	}
	_ = s.DB.First(conv, conv.ID)
	return &msg, nil
}

// ReloadProfiles is a no-op placeholder after multi-profile migration.
func (s *AgentService) ReloadProfiles() {}