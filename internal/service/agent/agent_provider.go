package agent

import (
	"cakecake/internal/config"
	"cakecake/internal/data"
	"cakecake/internal/model/agent"
	"cakecake/internal/model/dm"
	"context"
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
}

// AgentStoreImpl implements AgentStore using *gorm.DB (Phase 1 monolith).
type AgentStoreImpl struct {
	db *gorm.DB
}

func NewAgentStore(db *gorm.DB) *AgentStoreImpl {
	return &AgentStoreImpl{db: db}
}

func (p *AgentStoreImpl) EnsureAllAgentConversationsForUser(humanID uint64) error {
	return data.EnsureAllAgentConversationsForUser(p.db, humanID)
}

func (p *AgentStoreImpl) GetAgentProfileByBotUserID(botUserID uint64) (*agent.AgentProfile, error) {
	return data.GetAgentProfileByBotUserID(p.db, botUserID)
}

func (p *AgentStoreImpl) GetAgentProfile(id uint64) (*agent.AgentProfile, error) {
	return data.GetAgentProfile(p.db, id)
}

func (p *AgentStoreImpl) GetGlobalSystemPrompt() string {
	return data.GetGlobalSystemPrompt(p.db)
}

func (p *AgentStoreImpl) PostAssistantMessageTx(msg *dm.DmMessage, conv *dm.DmConversation, humanID uint64, now time.Time, preview string) error {
	return p.db.Transaction(func(tx *gorm.DB) error {
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

func (p *AgentStoreImpl) ResetConversationTx(conv *dm.DmConversation, msg *dm.DmMessage, humanID uint64, now time.Time, preview string) error {
	return p.db.Transaction(func(tx *gorm.DB) error {
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

func (p *AgentStoreImpl) ReloadConversation(conv *dm.DmConversation) error {
	return p.db.First(conv, conv.ID).Error
}

func (p *AgentStoreImpl) ListAgentProfiles(ctx context.Context) ([]agent.AgentProfile, error) {
	var rows []agent.AgentProfile
	if err := p.db.WithContext(ctx).Order("id ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func (p *AgentStoreImpl) CreateAgentProfile(ctx context.Context, profile *agent.AgentProfile) error {
	return p.db.WithContext(ctx).Create(profile).Error
}

func (p *AgentStoreImpl) UpdateAgentProfile(ctx context.Context, id uint64, updates map[string]interface{}) error {
	return p.db.WithContext(ctx).Model(&agent.AgentProfile{}).Where("id = ?", id).Updates(updates).Error
}

func (p *AgentStoreImpl) DeleteAgentProfile(ctx context.Context, id uint64) error {
	return p.db.WithContext(ctx).Delete(&agent.AgentProfile{}, id).Error
}

func (p *AgentStoreImpl) CountActiveAgentProfiles(ctx context.Context) (int64, error) {
	var cnt int64
	err := p.db.WithContext(ctx).Model(&agent.AgentProfile{}).Where("enabled = ?", true).Count(&cnt).Error
	return cnt, err
}

func (p *AgentStoreImpl) CheckAgentSlugExists(ctx context.Context, slug string) (bool, error) {
	var cnt int64
	err := p.db.WithContext(ctx).Model(&agent.AgentProfile{}).Where("slug = ?", slug).Count(&cnt).Error
	return cnt > 0, err
}

func (p *AgentStoreImpl) UpdateAgentAvatar(ctx context.Context, id uint64, avatarURL string) error {
	return p.db.WithContext(ctx).Model(&agent.AgentProfile{}).Where("id = ?", id).Update("avatar_url", avatarURL).Error
}

func (p *AgentStoreImpl) ProfileCount(ctx context.Context) (int64, error) {
	return data.ProfileCount(p.db)
}

func (p *AgentStoreImpl) CreateAgentBotUser(ctx context.Context, slug, displayName, sign, avatarURL string) (uint64, error) {
	return data.CreateAgentBotUser(p.db, slug, displayName, sign, avatarURL)
}

func (p *AgentStoreImpl) RenameAgentProfileSlug(ctx context.Context, profile *agent.AgentProfile, newSlug string) error {
	return data.RenameAgentProfileSlug(p.db, profile, newSlug)
}

func (p *AgentStoreImpl) SyncAgentProfile(ctx context.Context, profile *agent.AgentProfile) error {
	return data.SyncAgentProfile(p.db, profile)
}

func (p *AgentStoreImpl) EnsureAgentProfiles(cfg *config.C, log *zap.Logger) error {
	return data.EnsureAgentProfiles(p.db, cfg, log)
}
