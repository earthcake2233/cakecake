package data

import (
	"strings"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"minibili/internal/model"
)

const defaultAgentDisplayName = "cakecake AI"
const defaultAgentSign = "站内 AI 助手"
const defaultAgentWelcome = "你好，我是 cakecake AI 助手。可以问我站内功能、投稿与观看相关问题～"

const defaultAgentSystemPrompt = `你是 cakecake 站内 AI 助手。帮助用户了解本站功能。
回答风格要求：
- 说人话，自然口语化，像朋友聊天一样
- 不！要！用！任！何！emoji！表情符号
- 不要用夸张语气、不要营销号腔
- 简洁直接，普通用户看得懂
- 不要编造不存在的功能
- 不确定时诚实说不知道`

// EnsureDefaultAgentSettings creates the singleton settings row when missing.
func EnsureDefaultAgentSettings(db *gorm.DB, lg *zap.Logger) error {
	if db == nil {
		return nil
	}
	var st model.AgentSettings
	err := db.First(&st, model.AgentSettingsRowID).Error
	if err == nil {
		return nil
	}
	if err != gorm.ErrRecordNotFound {
		return err
	}
	st = model.AgentSettings{
		ID:               model.AgentSettingsRowID,
		DisplayName:      defaultAgentDisplayName,
		Sign:             defaultAgentSign,
		SystemPrompt:     defaultAgentSystemPrompt,
		WelcomeMessage:   defaultAgentWelcome,
		AssistantEnabled: true,
	}
	if err := db.Create(&st).Error; err != nil {
		return err
	}
	if lg != nil {
		lg.Info("default agent settings created")
	}
	return nil
}

// GetAgentSettings loads the singleton settings (nil if missing).
func GetAgentSettings(db *gorm.DB) (*model.AgentSettings, error) {
	if db == nil {
		return nil, gorm.ErrRecordNotFound
	}
	var st model.AgentSettings
	if err := db.First(&st, model.AgentSettingsRowID).Error; err != nil {
		return nil, err
	}
	return &st, nil
}

// AgentWelcomeMessage returns welcome text for new agent conversations.
func AgentWelcomeMessage(db *gorm.DB) string {
	st, err := GetAgentSettings(db)
	if err != nil || strings.TrimSpace(st.WelcomeMessage) == "" {
		return defaultAgentWelcome
	}
	return strings.TrimSpace(st.WelcomeMessage)
}

// SyncAgentBotProfile copies display fields to the system AI user row.
func SyncAgentBotProfile(db *gorm.DB, botUserID uint64, st *model.AgentSettings) error {
	if db == nil || botUserID == 0 || st == nil {
		return nil
	}
	name := strings.TrimSpace(st.DisplayName)
	if name == "" {
		name = defaultAgentDisplayName
	}
	return db.Model(&model.User{}).Where("id = ?", botUserID).Updates(map[string]interface{}{
		"nickname":    name,
		"avatar_url":  strings.TrimSpace(st.AvatarURL),
		"sign":        strings.TrimSpace(st.Sign),
	}).Error
}
