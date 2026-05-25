package model

import "time"

// AgentSettings is a singleton row (ID=1) for the message-center AI assistant.
type AgentSettings struct {
	ID             uint64 `gorm:"primaryKey"`
	DisplayName    string `gorm:"size:64;not null"`
	AvatarURL      string `gorm:"size:1024"`
	Sign           string `gorm:"size:500"`
	SystemPrompt   string `gorm:"type:text;not null"`
	WelcomeMessage string `gorm:"size:500;not null"`
	// AssistantEnabled lets ops pause AI replies without removing API keys.
	AssistantEnabled bool `gorm:"not null;default:1"`
	UpdatedAt        time.Time
}

const AgentSettingsRowID uint64 = 1
