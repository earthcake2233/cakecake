package data

import (
	"cakecake/internal/model/agent"
	"cakecake/internal/model/user"
	"strings"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

const defaultAgentDisplayName = "cakecake AI"
const defaultAgentSign = "站内 AI 助手"
const defaultAgentWelcome = "你好，我是 cakecake AI 助手。可以问我站内功能、投稿与观看相关问题～"

const defaultAgentSystemPrompt = `你是 cakecake 站内 AI 助手。帮助用户了解本站功能。
回答风格要求：
- 说人话，口语化，像朋友聊天一样自然
- 一律使用 Markdown 排版：**加粗** 强调关键词、### 小节标题、列表列要点、表格做对比
- 代码或命令一律用三个反引号包成代码块并标注语言（go/bash/sql），禁止裸贴代码
- 每次回答都必须用 Markdown 排版输出，不要给纯文本段落
- 回答结尾用一两句话自然收尾（总结要点或邀请追问），不要戛然而止
- 站内搜索工具可以用于任何问题（站内可能有相关教程），但只有当结果与用户问题相关时才展示
- 工具结果与问题无关或为空时，明确告诉用户站内没有相关内容，不要展示无关结果
- 当你在回复中引用了工具结果（搜索/详情/榜单）时，在回复最后单独输出一行展示清单，格式：【展示】工具名#ID,工具名#ID（例如【展示】search_videos#23）。ID 必须是工具结果中的 id，只列你明确推荐展示的结果；这行不会显示给用户，不要解释它
- 不要用表情符号（emoji），除非用户明确要求
- 可以带有角色的个性色彩和语气
- 不要用夸张营销腔
- 不要编造不存在的功能
- 不确定时诚实说不知道`

// EnsureDefaultAgentSettings creates the singleton settings row when missing.
func EnsureDefaultAgentSettings(db *gorm.DB, lg *zap.Logger) error {
	if db == nil {
		return nil
	}
	var st agent.AgentSettings
	err := db.First(&st, agent.AgentSettingsRowID).Error
	if err == nil {
		// Always sync global prompt from code constant on startup
		st.SystemPrompt = defaultAgentSystemPrompt
		if err := db.Model(&st).Update("system_prompt", st.SystemPrompt).Error; err != nil && lg != nil {
			lg.Warn("sync agent_settings system_prompt failed", zap.Error(err))
		}
		if lg != nil {
			lg.Info("synced agent_settings system_prompt from code constant")
		}
		return nil
	}
	if err != gorm.ErrRecordNotFound {
		return err
	}
	st = agent.AgentSettings{
		ID:               agent.AgentSettingsRowID,
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
func GetAgentSettings(db *gorm.DB) (*agent.AgentSettings, error) {
	if db == nil {
		return nil, gorm.ErrRecordNotFound
	}
	var st agent.AgentSettings
	if err := db.First(&st, agent.AgentSettingsRowID).Error; err != nil {
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
func SyncAgentBotProfile(db *gorm.DB, botUserID uint64, st *agent.AgentSettings) error {
	if db == nil || botUserID == 0 || st == nil {
		return nil
	}
	name := strings.TrimSpace(st.DisplayName)
	if name == "" {
		name = defaultAgentDisplayName
	}
	return db.Model(&user.User{}).Where("id = ?", botUserID).Updates(map[string]interface{}{
		"nickname":   name,
		"avatar_url": strings.TrimSpace(st.AvatarURL),
		"sign":       strings.TrimSpace(st.Sign),
	}).Error
}

// GetGlobalSystemPrompt returns the agent_settings system_prompt (global layer).
// This is the general/default prompt that applies to ALL agent profiles.
func GetGlobalSystemPrompt(db *gorm.DB) string {
	st, err := GetAgentSettings(db)
	if err != nil || st == nil {
		return defaultAgentSystemPrompt
	}
	if v := strings.TrimSpace(st.SystemPrompt); v != "" {
		return v
	}
	return defaultAgentSystemPrompt
}
