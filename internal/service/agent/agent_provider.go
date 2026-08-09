package agent

import (
	"cakecake/internal/config"
	"cakecake/internal/data"
	"cakecake/internal/model/agent"
	"cakecake/internal/model/dm"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// AgentStore is the agent-domain DB storage boundary.
// Phase 1: *gorm.DB impl. Phase 2+: replaced by gRPC client / per-domain store.
type AgentStore interface {
	EnsureAllAgentConversationsForUser(humanID uint64) error
	GetAgentProfileByBotUserID(botUserID uint64) (*agent.AgentProfile, error)
	GetAgentProfile(id uint64) (*agent.AgentProfile, error)
	GetGlobalSystemPrompt() string
	PostAssistantMessageTx(msg *dm.DmMessage, conv *dm.DmConversation, humanID uint64, now time.Time, preview string) error
	ResetConversationTx(conv *dm.DmConversation, msg *dm.DmMessage, humanID uint64, now time.Time, preview string) error
	ReloadConversation(conv *dm.DmConversation) error
	ListAgentProfiles(ctx context.Context) ([]agent.AgentProfile, error)
	CreateAgentProfile(ctx context.Context, p *agent.AgentProfile) error
	UpdateAgentProfile(ctx context.Context, id uint64, updates map[string]interface{}) error
	DeleteAgentProfile(ctx context.Context, id uint64) error
	CountActiveAgentProfiles(ctx context.Context) (int64, error)
	CheckAgentSlugExists(ctx context.Context, slug string) (bool, error)
	UpdateAgentAvatar(ctx context.Context, id uint64, avatarURL string) error
	ProfileCount(ctx context.Context) (int64, error)
	CreateAgentBotUser(ctx context.Context, slug, displayName, sign, avatarURL string) (uint64, error)
	RenameAgentProfileSlug(ctx context.Context, p *agent.AgentProfile, newSlug string) error
	SyncAgentProfile(ctx context.Context, p *agent.AgentProfile) error
	EnsureAgentProfiles(cfg *config.C, log *zap.Logger) error
	SetMessageFeedback(ctx context.Context, messageID uint64, userID uint64, feedback string) error
	UpdateMessageSuggestions(ctx context.Context, messageID uint64, suggestions []string) error
	ListAgentFeedbacks(ctx context.Context, limit int, offset int) ([]agent.AgentFeedback, error)
	ListAgentFeedbacksWithContent(ctx context.Context, limit int, offset int) ([]AgentFeedbackRow, error)
}

// AgentStoreImpl implements AgentStore using *gorm.DB (Phase 1 monolith).
type AgentStoreImpl struct {
	db *gorm.DB
}

// NewAgentStore creates a gorm-backed AgentStore implementation.
func NewAgentStore(db *gorm.DB) *AgentStoreImpl {
	return &AgentStoreImpl{db: db}
}

// UpdateMessageSuggestions stores follow-up chips on an assistant message.
func (s *AgentStoreImpl) UpdateMessageSuggestions(ctx context.Context, messageID uint64, suggestions []string) error {
	if messageID == 0 {
		return fmt.Errorf("message_id is required")
	}
	if len(suggestions) == 0 {
		return nil
	}
	b, err := json.Marshal(suggestions)
	if err != nil {
		return err
	}
	return s.db.WithContext(ctx).
		Model(&dm.DmMessage{}).
		Where("id = ?", messageID).
		Update("suggestions", string(b)).Error
}

// SetMessageFeedback upserts a like/dislike; repeating the same feedback
// removes it (toggle). Different feedback replaces the previous value.
func (s *AgentStoreImpl) SetMessageFeedback(ctx context.Context, messageID uint64, userID uint64, feedback string) error {
	if messageID == 0 || userID == 0 {
		return fmt.Errorf("message_id and user_id are required")
	}
	var existing agent.AgentFeedback
	err := s.db.WithContext(ctx).
		Where("message_id = ? AND user_id = ?", messageID, userID).
		First(&existing).Error
	if err == nil {
		if existing.Feedback == feedback {
			return s.db.WithContext(ctx).Delete(&existing).Error
		}
		return s.db.WithContext(ctx).Model(&existing).Updates(map[string]interface{}{
			"feedback":   feedback,
			"updated_at": time.Now(),
		}).Error
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	return s.db.WithContext(ctx).Create(&agent.AgentFeedback{
		MessageID: messageID,
		UserID:    userID,
		Feedback:  feedback,
	}).Error
}

// ListAgentFeedbacks returns feedback rows ordered by newest first.
func (s *AgentStoreImpl) ListAgentFeedbacks(ctx context.Context, limit int, offset int) ([]agent.AgentFeedback, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	var rows []agent.AgentFeedback
	if err := s.db.WithContext(ctx).
		Order("id DESC").
		Limit(limit).
		Offset(offset).
		Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// AgentFeedbackRow joins a feedback row with the related assistant message
// content so the admin console can show what was rated.
type AgentFeedbackRow struct {
	agent.AgentFeedback
	MessageContent string `gorm:"column:message_content" json:"message_content"`
}

// ListAgentFeedbacksWithContent joins feedback rows with assistant message content.
func (s *AgentStoreImpl) ListAgentFeedbacksWithContent(ctx context.Context, limit int, offset int) ([]AgentFeedbackRow, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	var rows []AgentFeedbackRow
	if err := s.db.WithContext(ctx).
		Table("agent_feedbacks AS af").
		Select("af.*, m.content AS message_content").
		Joins("LEFT JOIN dm_messages m ON m.id = af.message_id").
		Order("af.id DESC").
		Limit(limit).
		Offset(offset).
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// EnsureAllAgentConversationsForUser ensures threads exist for each enabled profile.
func (s *AgentStoreImpl) EnsureAllAgentConversationsForUser(humanID uint64) error {
	return data.EnsureAllAgentConversationsForUser(s.db, humanID)
}

// GetAgentProfileByBotUserID loads the profile for a bot user id.
func (s *AgentStoreImpl) GetAgentProfileByBotUserID(botUserID uint64) (*agent.AgentProfile, error) {
	return data.GetAgentProfileByBotUserID(s.db, botUserID)
}

// GetAgentProfile loads a profile by id.
func (s *AgentStoreImpl) GetAgentProfile(id uint64) (*agent.AgentProfile, error) {
	return data.GetAgentProfile(s.db, id)
}

// GetGlobalSystemPrompt returns the global agent system prompt.
func (s *AgentStoreImpl) GetGlobalSystemPrompt() string {
	return data.GetGlobalSystemPrompt(s.db)
}

// PostAssistantMessageTx persists an assistant message and updates conversation state atomically.
func (s *AgentStoreImpl) PostAssistantMessageTx(msg *dm.DmMessage, conv *dm.DmConversation, humanID uint64, now time.Time, preview string) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(msg).Error; err != nil {
			return err
		}
		if err := tx.Model(conv).Updates(map[string]interface{}{
			"last_message_at": now,
			"last_preview":    preview,
		}).Error; err != nil {
			return err
		}
		return tx.Model(&dm.DmParticipant{}).
			Where("conversation_id = ? AND user_id = ?", conv.ID, humanID).
			Updates(map[string]interface{}{
				"unread_count": gorm.Expr("unread_count + ?", 1),
				"hidden_at":    nil,
			}).Error
	})
}

// ResetConversationTx clears conversation history, writes the opening message, and resets state atomically.
func (s *AgentStoreImpl) ResetConversationTx(conv *dm.DmConversation, msg *dm.DmMessage, humanID uint64, now time.Time, preview string) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("conversation_id = ?", conv.ID).Delete(&dm.DmMessage{}).Error; err != nil {
			return err
		}
		if err := tx.Create(msg).Error; err != nil {
			return err
		}
		if err := tx.Model(conv).Updates(map[string]interface{}{
			"last_message_at": now,
			"last_preview":    preview,
		}).Error; err != nil {
			return err
		}
		return tx.Model(&dm.DmParticipant{}).
			Where("conversation_id = ? AND user_id = ?", conv.ID, humanID).
			Updates(map[string]interface{}{
				"unread_count": 0,
				"hidden_at":    nil,
			}).Error
	})
}

// ReloadConversation refreshes a conversation row from the database.
func (s *AgentStoreImpl) ReloadConversation(conv *dm.DmConversation) error {
	return s.db.First(conv, conv.ID).Error
}

// ListAgentProfiles lists all agent profiles.
func (s *AgentStoreImpl) ListAgentProfiles(ctx context.Context) ([]agent.AgentProfile, error) {
	var rows []agent.AgentProfile
	if err := s.db.WithContext(ctx).Order("id ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// CreateAgentProfile inserts an agent profile.
func (s *AgentStoreImpl) CreateAgentProfile(ctx context.Context, profile *agent.AgentProfile) error {
	return s.db.WithContext(ctx).Create(profile).Error
}

// UpdateAgentProfile applies partial updates to an agent profile.
func (s *AgentStoreImpl) UpdateAgentProfile(ctx context.Context, id uint64, updates map[string]interface{}) error {
	return s.db.WithContext(ctx).Model(&agent.AgentProfile{}).Where("id = ?", id).Updates(updates).Error
}

// DeleteAgentProfile removes an agent profile by id.
func (s *AgentStoreImpl) DeleteAgentProfile(ctx context.Context, id uint64) error {
	return s.db.WithContext(ctx).Delete(&agent.AgentProfile{}, id).Error
}

// CountActiveAgentProfiles counts enabled profiles.
func (s *AgentStoreImpl) CountActiveAgentProfiles(ctx context.Context) (int64, error) {
	var cnt int64
	err := s.db.WithContext(ctx).Model(&agent.AgentProfile{}).Where("enabled = ?", true).Count(&cnt).Error
	return cnt, err
}

// CheckAgentSlugExists reports whether a profile slug is taken.
func (s *AgentStoreImpl) CheckAgentSlugExists(ctx context.Context, slug string) (bool, error) {
	var cnt int64
	err := s.db.WithContext(ctx).Model(&agent.AgentProfile{}).Where("slug = ?", slug).Count(&cnt).Error
	return cnt > 0, err
}

// UpdateAgentAvatar sets a profile's avatar URL.
func (s *AgentStoreImpl) UpdateAgentAvatar(ctx context.Context, id uint64, avatarURL string) error {
	return s.db.WithContext(ctx).Model(&agent.AgentProfile{}).Where("id = ?", id).Update("avatar_url", avatarURL).Error
}

// ProfileCount counts all agent profiles.
func (s *AgentStoreImpl) ProfileCount(ctx context.Context) (int64, error) {
	return data.ProfileCount(s.db)
}

// CreateAgentBotUser creates the bot user backing a profile.
func (s *AgentStoreImpl) CreateAgentBotUser(ctx context.Context, slug, displayName, sign, avatarURL string) (uint64, error) {
	return data.CreateAgentBotUser(s.db, slug, displayName, sign, avatarURL)
}

// RenameAgentProfileSlug renames a profile's slug.
func (s *AgentStoreImpl) RenameAgentProfileSlug(ctx context.Context, profile *agent.AgentProfile, newSlug string) error {
	return data.RenameAgentProfileSlug(s.db, profile, newSlug)
}

// SyncAgentProfile mirrors profile display fields to the bot user row.
func (s *AgentStoreImpl) SyncAgentProfile(ctx context.Context, profile *agent.AgentProfile) error {
	return data.SyncAgentProfile(s.db, profile)
}

// EnsureAgentProfiles runs the startup migration that guarantees a default profile.
func (s *AgentStoreImpl) EnsureAgentProfiles(cfg *config.C, log *zap.Logger) error {
	return data.EnsureAgentProfiles(s.db, cfg, log)
}
