package model

import "time"

const (
	DmKindHuman = "human"
	DmKindAgent = "agent"
)

// DmConversation is a 1:1 chat thread between two users (UserLow < UserHigh).
type DmConversation struct {
	ID            uint64    `gorm:"primaryKey"`
	UserLow       uint64    `gorm:"uniqueIndex:idx_dm_pair_low_high;not null"`
	UserHigh      uint64    `gorm:"uniqueIndex:idx_dm_pair_low_high;not null"`
	Kind            string `gorm:"size:16;not null;default:human;index"`
	AgentProfileID  uint64 `gorm:"index;not null;default:0"`
	LastMessageAt   time.Time `gorm:"index"`
	LastPreview   string    `gorm:"size:500"`
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// DmParticipant tracks per-user unread state in a conversation.
type DmParticipant struct {
	ID             uint64 `gorm:"primaryKey"`
	ConversationID uint64 `gorm:"uniqueIndex:idx_dm_part_user_conv;not null"`
	UserID         uint64 `gorm:"uniqueIndex:idx_dm_part_user_conv;index;not null"`
	UnreadCount    uint32 `gorm:"not null;default:0"`
	Pinned         bool       `gorm:"not null;default:0"`
	PinnedAt       *time.Time `gorm:"index"`
	Muted          bool       `gorm:"not null;default:0"`
	HiddenAt       *time.Time `gorm:"index"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// DmMessage is a private message in a conversation.
type DmMessage struct {
	ID             uint64 `gorm:"primaryKey"`
	ConversationID uint64 `gorm:"index:idx_dm_msg_conv;not null"`
	SenderID       uint64 `gorm:"index;not null"`
	// Role is user | assistant for agent threads (empty for legacy human-human rows).
	Role           string `gorm:"size:16;not null;default:''"`
	Content        string `gorm:"size:500;not null"`
	CreatedAt      time.Time `gorm:"index"`
}
