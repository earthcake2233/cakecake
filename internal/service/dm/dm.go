package dm

import (
	"cakecake/internal/model/dm"
	"context"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// DmService handles direct message business logic.
type DmService struct {
	store DmStore
	rdb   *redis.Client
	log   *zap.Logger
}

func NewDmService(db *gorm.DB, rdb *redis.Client, log *zap.Logger) *DmService {
	return &DmService{store: NewDmStore(db), rdb: rdb, log: log}
}

// DmStore is the direct-message storage boundary (Phase 1: *gorm.DB impl).
type DmStore interface {
	GetOrCreateConversation(ctx context.Context, uid, peerID uint64) (*dm.DmConversation, *dm.DmParticipant, error)
	FindConversationByUserIDs(ctx context.Context, uidA, uidB uint64) (*dm.DmConversation, error)
	CreateMessageInTransaction(ctx context.Context, convID, senderID uint64, content, role string, fn func(tx *gorm.DB, msg *dm.DmMessage) error) error
	GetConversationByID(ctx context.Context, convID uint64) (*dm.DmConversation, error)
	GetParticipant(ctx context.Context, convID, userID uint64) (*dm.DmParticipant, error)
	ListConversations(ctx context.Context, uid uint64) ([]dm.DmConversation, []dm.DmParticipant, error)
	ListMessages(ctx context.Context, convID uint64, beforeID uint64, limit int) ([]dm.DmMessage, error)
	CreateMessage(ctx context.Context, convID, senderID uint64, content, role string) (*dm.DmMessage, error)
	MarkConversationRead(ctx context.Context, convID, userID uint64) error
	UpdateConversationSettings(ctx context.Context, convID, userID uint64, updates map[string]interface{}) error
	DeleteConversation(ctx context.Context, convID, userID uint64) error
	ResetConversationForAgent(ctx context.Context, convID uint64) error
	PostMessage(ctx context.Context, convID, senderID, peerID uint64, content, preview string, isAgent bool) (*PostMessageResult, error)
	UnreadTotal(ctx context.Context, uid uint64) int64
	EnsureParticipant(tx *gorm.DB, convID, uid uint64)
}

// DmStoreImpl implements DmStore using *gorm.DB (Phase 1 monolith).
type DmStoreImpl struct {
	db *gorm.DB
}

func NewDmStore(db *gorm.DB) *DmStoreImpl {
	return &DmStoreImpl{db: db}
}

// DmPairIDs returns low/high sorted pair.
func DmPairIDs(a, b uint64) (low, high uint64) {
	if a < b {
		return a, b
	}
	return b, a
}

// DmPeerID returns the other participant"s ID.
func DmPeerID(conv *dm.DmConversation, self uint64) uint64 {
	if conv.UserLow == self {
		return conv.UserHigh
	}
	return conv.UserLow
}

// GetOrCreateConversation finds or creates a conversation between two users.
func (p *DmStoreImpl) GetOrCreateConversation(ctx context.Context, uid, peerID uint64) (*dm.DmConversation, *dm.DmParticipant, error) {
	low, high := DmPairIDs(uid, peerID)
	var conv dm.DmConversation
	err := p.db.WithContext(ctx).Where("user_low = ? AND user_high = ?", low, high).First(&conv).Error
	if err == nil {
		var part dm.DmParticipant
		_ = p.db.WithContext(ctx).Where("conversation_id = ? AND user_id = ?", conv.ID, uid).First(&part).Error
		return &conv, &part, nil
	}
	if err != gorm.ErrRecordNotFound {
		return nil, nil, err
	}
	conv = dm.DmConversation{UserLow: low, UserHigh: high}
	if err := p.db.WithContext(ctx).Create(&conv).Error; err != nil {
		return nil, nil, err
	}
	for _, u := range []uint64{low, high} {
		_ = p.db.WithContext(ctx).Create(&dm.DmParticipant{ConversationID: conv.ID, UserID: u}).Error
	}
	var part dm.DmParticipant
	_ = p.db.WithContext(ctx).Where("conversation_id = ? AND user_id = ?", conv.ID, uid).First(&part).Error
	return &conv, &part, nil
}

// FindConversationByUserIDs finds a conversation between two users without creating one.
func (p *DmStoreImpl) FindConversationByUserIDs(ctx context.Context, uidA, uidB uint64) (*dm.DmConversation, error) {
	low, high := DmPairIDs(uidA, uidB)
	var conv dm.DmConversation
	if err := p.db.WithContext(ctx).Where("user_low = ? AND user_high = ?", low, high).First(&conv).Error; err != nil {
		return nil, err
	}
	return &conv, nil
}

// CreateMessageInTransaction creates a message in a transaction callback.
func (p *DmStoreImpl) CreateMessageInTransaction(ctx context.Context, convID, senderID uint64, content, role string, fn func(tx *gorm.DB, msg *dm.DmMessage) error) error {
	return p.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		msg := dm.DmMessage{
			ConversationID: convID, SenderID: senderID,
			Content: content, Role: role,
		}
		if err := tx.Create(&msg).Error; err != nil {
			return err
		}
		_ = tx.Model(&dm.DmConversation{}).Where("id = ?", convID).
			Update("updated_at", time.Now()).Error
		if fn != nil {
			return fn(tx, &msg)
		}
		return nil
	})
}

// GetConversationByID returns a conversation by ID.
func (p *DmStoreImpl) GetConversationByID(ctx context.Context, convID uint64) (*dm.DmConversation, error) {
	var conv dm.DmConversation
	if err := p.db.WithContext(ctx).First(&conv, convID).Error; err != nil {
		return nil, err
	}
	return &conv, nil
}

// GetParticipant returns the participant record for a user in a conversation.
func (p *DmStoreImpl) GetParticipant(ctx context.Context, convID, userID uint64) (*dm.DmParticipant, error) {
	var part dm.DmParticipant
	if err := p.db.WithContext(ctx).Where("conversation_id = ? AND user_id = ?", convID, userID).First(&part).Error; err != nil {
		return nil, err
	}
	return &part, nil
}

// ListConversations returns all conversations for a user with participants.
func (p *DmStoreImpl) ListConversations(ctx context.Context, uid uint64) ([]dm.DmConversation, []dm.DmParticipant, error) {
	var convs []dm.DmConversation
	if err := p.db.WithContext(ctx).
		Where("user_low = ? OR user_high = ?", uid, uid).
		Order("updated_at DESC").Find(&convs).Error; err != nil {
		return nil, nil, err
	}
	if len(convs) == 0 {
		return convs, nil, nil
	}
	ids := make([]uint64, len(convs))
	for i := range convs {
		ids[i] = convs[i].ID
	}
	var parts []dm.DmParticipant
	_ = p.db.WithContext(ctx).Where("user_id = ? AND conversation_id IN ?", uid, ids).Find(&parts).Error
	return convs, parts, nil
}

// ListMessages returns messages for a conversation with cursor.
func (p *DmStoreImpl) ListMessages(ctx context.Context, convID uint64, beforeID uint64, limit int) ([]dm.DmMessage, error) {
	q := p.db.WithContext(ctx).Model(&dm.DmMessage{}).Where("conversation_id = ?", convID)
	if beforeID > 0 {
		q = q.Where("id < ?", beforeID)
	}
	var msgs []dm.DmMessage
	if err := q.Order("id DESC").Limit(limit).Find(&msgs).Error; err != nil {
		return nil, err
	}
	return msgs, nil
}

// CreateMessage creates a new message in a conversation.
func (p *DmStoreImpl) CreateMessage(ctx context.Context, convID, senderID uint64, content, role string) (*dm.DmMessage, error) {
	msg := dm.DmMessage{
		ConversationID: convID, SenderID: senderID,
		Content: content, Role: role,
	}
	if err := p.db.WithContext(ctx).Create(&msg).Error; err != nil {
		return nil, err
	}
	_ = p.db.WithContext(ctx).Model(&dm.DmConversation{}).Where("id = ?", convID).
		Update("updated_at", time.Now()).Error
	return &msg, nil
}

// MarkConversationRead marks all messages as read for a participant.
func (p *DmStoreImpl) MarkConversationRead(ctx context.Context, convID, userID uint64) error {
	return p.db.WithContext(ctx).Model(&dm.DmParticipant{}).
		Where("conversation_id = ? AND user_id = ?", convID, userID).
		Update("unread_count", 0).Error
}

// UpdateConversationSettings updates participant settings (pinned, muted).
func (p *DmStoreImpl) UpdateConversationSettings(ctx context.Context, convID, userID uint64, updates map[string]interface{}) error {
	return p.db.WithContext(ctx).Model(&dm.DmParticipant{}).
		Where("conversation_id = ? AND user_id = ?", convID, userID).
		Updates(updates).Error
}

// DeleteConversation soft-deletes a conversation for a user.
func (p *DmStoreImpl) DeleteConversation(ctx context.Context, convID, userID uint64) error {
	return p.db.WithContext(ctx).Where("conversation_id = ? AND user_id = ?", convID, userID).
		Delete(&dm.DmParticipant{}).Error
}

// ResetConversationForAgent resets the agent conversation (deletes all messages).
func (p *DmStoreImpl) ResetConversationForAgent(ctx context.Context, convID uint64) error {
	return p.db.WithContext(ctx).Where("conversation_id = ?", convID).
		Delete(&dm.DmMessage{}).Error
}

// PostMessageResult holds the result of a PostMessage call.
type PostMessageResult struct {
	Message      *dm.DmMessage
	Conversation *dm.DmConversation
	SelfPart     *dm.DmParticipant
	PeerPart     *dm.DmParticipant
}

// PostMessage creates a message, updates conversation metadata, manages participants, and increments unread count.
func (p *DmStoreImpl) PostMessage(ctx context.Context, convID, senderID, peerID uint64, content, preview string, isAgent bool) (*PostMessageResult, error) {
	now := time.Now()
	var result PostMessageResult
	err := p.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		msg := dm.DmMessage{
			ConversationID: convID,
			SenderID:       senderID,
			Role:           "user",
			Content:        content,
			CreatedAt:      now,
		}
		if err := tx.Create(&msg).Error; err != nil {
			return err
		}
		result.Message = &msg

		if err := tx.Model(&dm.DmConversation{}).Where("id = ?", convID).Updates(map[string]interface{}{
			"last_message_at": now,
			"last_preview":    preview,
		}).Error; err != nil {
			return err
		}

		p.EnsureParticipant(tx, convID, senderID)
		if !isAgent {
			p.EnsureParticipant(tx, convID, peerID)
		}

		if !isAgent {
			if err := tx.Model(&dm.DmParticipant{}).
				Where("conversation_id = ? AND user_id = ?", convID, peerID).
				Updates(map[string]interface{}{
					"unread_count": gorm.Expr("unread_count + ?", 1),
					"hidden_at":    nil,
				}).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	// Post-commit: refresh conversation
	var conv dm.DmConversation
	_ = p.db.WithContext(ctx).First(&conv, convID)
	result.Conversation = &conv

	var selfPart dm.DmParticipant
	_ = p.db.WithContext(ctx).Where("conversation_id = ? AND user_id = ?", convID, senderID).First(&selfPart)
	result.SelfPart = &selfPart

	if !isAgent {
		var peerPart dm.DmParticipant
		_ = p.db.WithContext(ctx).Where("conversation_id = ? AND user_id = ?", convID, peerID).First(&peerPart)
		result.PeerPart = &peerPart
	}
	return &result, nil
}

// UnreadTotal returns total unread count for a user across all conversations.
func (p *DmStoreImpl) UnreadTotal(ctx context.Context, uid uint64) int64 {
	var cnt int64
	_ = p.db.WithContext(ctx).Model(&dm.DmParticipant{}).
		Where("user_id = ? AND unread_count > 0", uid).Count(&cnt).Error
	return cnt
}

// EnsureParticipant creates a participant record if not exists.
func (p *DmStoreImpl) EnsureParticipant(tx *gorm.DB, convID, uid uint64) {
	if tx == nil {
		tx = p.db
	}
	var part dm.DmParticipant
	if err := tx.Where("conversation_id = ? AND user_id = ?", convID, uid).First(&part).Error; err != nil {
		_ = tx.Create(&dm.DmParticipant{ConversationID: convID, UserID: uid}).Error
	}
}

// GetOrCreateConversation finds or creates a conversation between two users.
func (s *DmService) GetOrCreateConversation(ctx context.Context, uid, peerID uint64) (*dm.DmConversation, *dm.DmParticipant, error) {
	return s.store.GetOrCreateConversation(ctx, uid, peerID)
}

// FindConversationByUserIDs finds a conversation between two users without creating one.
func (s *DmService) FindConversationByUserIDs(ctx context.Context, uidA, uidB uint64) (*dm.DmConversation, error) {
	return s.store.FindConversationByUserIDs(ctx, uidA, uidB)
}

// CreateMessageInTransaction creates a message in a transaction callback.
func (s *DmService) CreateMessageInTransaction(ctx context.Context, convID, senderID uint64, content, role string, fn func(tx *gorm.DB, msg *dm.DmMessage) error) error {
	return s.store.CreateMessageInTransaction(ctx, convID, senderID, content, role, fn)
}

// GetConversationByID returns a conversation by ID.
func (s *DmService) GetConversationByID(ctx context.Context, convID uint64) (*dm.DmConversation, error) {
	return s.store.GetConversationByID(ctx, convID)
}

// GetParticipant returns the participant record for a user in a conversation.
func (s *DmService) GetParticipant(ctx context.Context, convID, userID uint64) (*dm.DmParticipant, error) {
	return s.store.GetParticipant(ctx, convID, userID)
}

// ListConversations returns all conversations for a user with participants.
func (s *DmService) ListConversations(ctx context.Context, uid uint64) ([]dm.DmConversation, []dm.DmParticipant, error) {
	return s.store.ListConversations(ctx, uid)
}

// ListMessages returns messages for a conversation with cursor.
func (s *DmService) ListMessages(ctx context.Context, convID uint64, beforeID uint64, limit int) ([]dm.DmMessage, error) {
	return s.store.ListMessages(ctx, convID, beforeID, limit)
}

// CreateMessage creates a new message in a conversation.
func (s *DmService) CreateMessage(ctx context.Context, convID, senderID uint64, content, role string) (*dm.DmMessage, error) {
	return s.store.CreateMessage(ctx, convID, senderID, content, role)
}

// MarkConversationRead marks all messages as read for a participant.
func (s *DmService) MarkConversationRead(ctx context.Context, convID, userID uint64) error {
	return s.store.MarkConversationRead(ctx, convID, userID)
}

// UpdateConversationSettings updates participant settings (pinned, muted).
func (s *DmService) UpdateConversationSettings(ctx context.Context, convID, userID uint64, updates map[string]interface{}) error {
	return s.store.UpdateConversationSettings(ctx, convID, userID, updates)
}

// DeleteConversation soft-deletes a conversation for a user.
func (s *DmService) DeleteConversation(ctx context.Context, convID, userID uint64) error {
	return s.store.DeleteConversation(ctx, convID, userID)
}

// ResetConversationForAgent resets the agent conversation (deletes all messages).
func (s *DmService) ResetConversationForAgent(ctx context.Context, convID uint64) error {
	return s.store.ResetConversationForAgent(ctx, convID)
}

// PostMessage creates a message, updates conversation metadata, manages participants, and increments unread count.
func (s *DmService) PostMessage(ctx context.Context, convID, senderID, peerID uint64, content, preview string, isAgent bool) (*PostMessageResult, error) {
	return s.store.PostMessage(ctx, convID, senderID, peerID, content, preview, isAgent)
}

// UnreadTotal returns total unread count for a user across all conversations.
func (s *DmService) UnreadTotal(ctx context.Context, uid uint64) int64 {
	return s.store.UnreadTotal(ctx, uid)
}

// EnsureParticipant creates a participant record if not exists.
func (s *DmService) EnsureParticipant(tx *gorm.DB, convID, uid uint64) {
	s.store.EnsureParticipant(tx, convID, uid)
}
