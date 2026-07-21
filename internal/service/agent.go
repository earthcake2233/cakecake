package service

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"minibili/internal/aigateway"
	"minibili/internal/config"
	"minibili/internal/data"
	"minibili/internal/model"
	"minibili/internal/pkg/sensitive"
	"minibili/internal/ws"
)

// AgentService runs AI assistant replies for agent DM threads.
type AgentService struct {
	Cfg     *config.C
	DB      *gorm.DB
	Redis   *redis.Client
	Gateway *aigateway.Gateway
	Sens    *sensitive.Filter
	ChatHub *ws.ChatHub
	Log     *zap.Logger
	RC      *config.RuntimeConfig
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

func (s *AgentService) PostAssistantMessage(conv *model.DmConversation, humanID uint64, content string) (*model.DmMessage, error) {
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
	msg := model.DmMessage{
		ConversationID: conv.ID,
		SenderID:       botID,
		Role:           "assistant",
		Content:        content,
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

func (s *AgentService) GenerateReply(ctx context.Context, conv *model.DmConversation, userText string) (string, error) {
	if !s.gatewayReady() {
		return "", fmt.Errorf("ai assistant is not configured")
	}
	s.applyDynamicGatewayConfig()
	profile, err := s.profileForConversation(conv)
	if err != nil {
		return "", fmt.Errorf("ai assistant profile missing")
	}
	if !profile.Enabled {
		return "", fmt.Errorf("ai assistant is disabled")
	}
	if s.Sens != nil {
		if err := s.Sens.Check(userText); err != nil {
			return "", fmt.Errorf("message contains sensitive words")
		}
	}
	prompt := strings.TrimSpace(profile.SystemPrompt)
	if prompt == "" {
		return "", fmt.Errorf("empty system prompt")
	}
	prev := s.Gateway.SystemPrompt
	s.Gateway.SystemPrompt = prompt
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
	reply, err := s.Gateway.CompleteUserTurn(ctx, conv.ID, userText)
	if err != nil {
		return "", err
	}
	if s.Sens != nil {
		if err := s.Sens.Check(reply); err != nil {
			return "抱歉，我无法生成该内容的回复，请换个方式提问。", nil
		}
	}
	return reply, nil
}

// ResetConversation clears chat history and seeds a fresh welcome message.
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
