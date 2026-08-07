package agent

import "time"

// AgentFeedback records a user's like/dislike on an AI assistant message.
// One row per (message_id, user_id); a repeat click toggles it off.
type AgentFeedback struct {
	ID        uint64    `gorm:"primaryKey" json:"id"`
	MessageID uint64    `gorm:"uniqueIndex:idx_agent_fb_msg_user;not null" json:"message_id"`
	UserID    uint64    `gorm:"uniqueIndex:idx_agent_fb_msg_user;not null" json:"user_id"`
	Feedback  string    `gorm:"size:16;not null" json:"feedback"` // like | dislike
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName returns the agent_feedbacks table name.
func (AgentFeedback) TableName() string { return "agent_feedbacks" }
