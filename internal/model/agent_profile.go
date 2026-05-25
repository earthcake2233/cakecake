package model

import (
	"encoding/json"
	"strings"
	"time"
)

// AgentProfile is an ops-configurable AI persona in the message center.
type AgentProfile struct {
	ID                   uint64 `gorm:"primaryKey"`
	Slug                 string `gorm:"size:32;uniqueIndex;not null"`
	BotUserID            uint64 `gorm:"index;not null"`
	DisplayName          string `gorm:"size:64;not null"`
	AvatarURL            string `gorm:"size:1024"`
	Sign                 string `gorm:"size:500"`
	SystemPrompt         string `gorm:"type:text;not null"`
	WelcomeMessagesJSON  string `gorm:"type:text;not null"`
	SortOrder            int    `gorm:"not null;default:0;index"`
	Enabled              bool   `gorm:"not null;default:1;index"`
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

// ParseWelcomeMessages decodes welcome_messages JSON array.
func ParseWelcomeMessages(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var list []string
	if err := json.Unmarshal([]byte(raw), &list); err != nil {
		return nil
	}
	out := make([]string, 0, len(list))
	for _, s := range list {
		s = strings.TrimSpace(s)
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}

// EncodeWelcomeMessages encodes non-empty lines to JSON array.
func EncodeWelcomeMessages(list []string) string {
	out := make([]string, 0, len(list))
	for _, s := range list {
		s = strings.TrimSpace(s)
		if s != "" {
			out = append(out, s)
		}
	}
	if len(out) == 0 {
		return "[]"
	}
	b, _ := json.Marshal(out)
	return string(b)
}
