package dm

import "time"

// DmKindHuman identifies a human-to-human conversation.
const (
	DmKindHuman = "human"
	DmKindAgent = "agent"
)

// DmConversation is a direct-message thread between two users (or a user and an agent).
type DmConversation struct {
	ID             uint64    `gorm:"primaryKey"`
	UserLow        uint64    `gorm:"uniqueIndex:idx_dm_pair_low_high;not null"`
	UserHigh       uint64    `gorm:"uniqueIndex:idx_dm_pair_low_high;not null"`
	Kind           string    `gorm:"size:16;not null;default:human;index"`
	AgentProfileID uint64    `gorm:"index;not null;default:0"`
	LastMessageAt  time.Time `gorm:"index"`
	LastPreview    string    `gorm:"size:500"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// DmParticipant tracks one side of a DM conversation.
type DmParticipant struct {
	ID             uint64     `gorm:"primaryKey"`
	ConversationID uint64     `gorm:"uniqueIndex:idx_dm_part_user_conv;not null"`
	UserID         uint64     `gorm:"uniqueIndex:idx_dm_part_user_conv;index;not null"`
	UnreadCount    uint32     `gorm:"not null;default:0"`
	Pinned         bool       `gorm:"not null;default:0"`
	PinnedAt       *time.Time `gorm:"index"`
	Muted          bool       `gorm:"not null;default:0"`
	HiddenAt       *time.Time `gorm:"index"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// DmMessage is a single direct-message row.
type DmMessage struct {
	ID             uint64 `gorm:"primaryKey"`
	ConversationID uint64 `gorm:"index:idx_dm_msg_conv;not null"`
	SenderID       uint64 `gorm:"index;not null"`
	// Role is user | assistant for agent threads (empty for legacy human-human rows).
	Role      string    `gorm:"size:16;not null;default:''"`
	Content   string    `gorm:"type:text;not null"`
	CreatedAt time.Time `gorm:"index"`
	// ToolActivities stores JSON array of tool call activities (name, status, duration, etc.)
	ToolActivities string `gorm:"type:text"`
	// ToolResultData stores JSON object of tool result items keyed by span_id
	ToolResultData string `gorm:"type:text"`
	// Suggestions stores JSON array of model-generated follow-up question chips.
	Suggestions string `gorm:"type:text"`
}
