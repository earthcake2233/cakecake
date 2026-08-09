package data

import (
	"cakecake/internal/model/agent"
	"cakecake/internal/model/user"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func newAgentSettingsDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&agent.AgentSettings{}))
	return db
}

func TestEnsureDefaultAgentSettings_Creates(t *testing.T) {
	db := newAgentSettingsDB(t)
	err := EnsureDefaultAgentSettings(db, zap.NewNop())
	require.NoError(t, err)

	var st agent.AgentSettings
	err = db.First(&st, agent.AgentSettingsRowID).Error
	require.NoError(t, err)
	assert.Equal(t, defaultAgentDisplayName, st.DisplayName)
	assert.Equal(t, defaultAgentSign, st.Sign)
	assert.Equal(t, defaultAgentSystemPrompt, st.SystemPrompt)
	assert.Equal(t, defaultAgentWelcome, st.WelcomeMessage)
	assert.True(t, st.AssistantEnabled)
}

func TestEnsureDefaultAgentSettings_AlreadyExists(t *testing.T) {
	db := newAgentSettingsDB(t)
	require.NoError(t, EnsureDefaultAgentSettings(db, zap.NewNop()))
	// Admin edits the global prompt; a restart must NOT revert it.
	require.NoError(t, UpdateGlobalAgentSettings(db, "custom global prompt"))
	require.NoError(t, EnsureDefaultAgentSettings(db, zap.NewNop()))

	var count int64
	db.Model(&agent.AgentSettings{}).Count(&count)
	assert.Equal(t, int64(1), count)
	var st agent.AgentSettings
	require.NoError(t, db.First(&st, agent.AgentSettingsRowID).Error)
	assert.Equal(t, "custom global prompt", st.SystemPrompt)
}

func TestUpdateGlobalAgentSettings(t *testing.T) {
	db := newAgentSettingsDB(t)
	require.NoError(t, EnsureDefaultAgentSettings(db, zap.NewNop()))
	require.NoError(t, UpdateGlobalAgentSettings(db, "  新的全局提示词  "))

	var st agent.AgentSettings
	require.NoError(t, db.First(&st, agent.AgentSettingsRowID).Error)
	assert.Equal(t, "新的全局提示词", st.SystemPrompt)

	require.Error(t, UpdateGlobalAgentSettings(nil, "x"))
}

// Regression: MySQL reports 0 affected rows when the new value equals the
// stored one. The upsert must never misread that as "row missing" and attempt
// a duplicate Create (primary-key conflict).
func TestUpdateGlobalAgentSettings_SameValueTwice(t *testing.T) {
	db := newAgentSettingsDB(t)
	require.NoError(t, EnsureDefaultAgentSettings(db, zap.NewNop()))
	require.NoError(t, UpdateGlobalAgentSettings(db, "unchanged prompt"))
	require.NoError(t, UpdateGlobalAgentSettings(db, "unchanged prompt"))

	var count int64
	require.NoError(t, db.Model(&agent.AgentSettings{}).Count(&count).Error)
	require.Equal(t, int64(1), count)

	var st agent.AgentSettings
	require.NoError(t, db.First(&st, agent.AgentSettingsRowID).Error)
	assert.Equal(t, "unchanged prompt", st.SystemPrompt)
}

func TestEnsureDefaultAgentSettings_NilDB(t *testing.T) {
	err := EnsureDefaultAgentSettings(nil, nil)
	assert.NoError(t, err)
}

func TestGetAgentSettings_Success(t *testing.T) {
	db := newAgentSettingsDB(t)
	require.NoError(t, EnsureDefaultAgentSettings(db, zap.NewNop()))

	st, err := GetAgentSettings(db)
	require.NoError(t, err)
	require.NotNil(t, st)
	assert.Equal(t, defaultAgentDisplayName, st.DisplayName)
}

func TestGetAgentSettings_Missing(t *testing.T) {
	db := newAgentSettingsDB(t)
	st, err := GetAgentSettings(db)
	assert.Error(t, err)
	assert.Nil(t, st)
}

func TestGetAgentSettings_NilDB(t *testing.T) {
	st, err := GetAgentSettings(nil)
	assert.Error(t, err)
	assert.Nil(t, st)
}

func TestAgentWelcomeMessage_Default(t *testing.T) {
	db := newAgentSettingsDB(t)
	msg := AgentWelcomeMessage(db)
	assert.Equal(t, defaultAgentWelcome, msg)
}

func TestAgentWelcomeMessage_Custom(t *testing.T) {
	db := newAgentSettingsDB(t)
	require.NoError(t, EnsureDefaultAgentSettings(db, nil))
	db.Model(&agent.AgentSettings{}).Where("id = ?", agent.AgentSettingsRowID).
		Update("welcome_message", "Custom welcome!")

	msg := AgentWelcomeMessage(db)
	assert.Equal(t, "Custom welcome!", msg)
}

func TestAgentWelcomeMessage_EmptyCustom(t *testing.T) {
	db := newAgentSettingsDB(t)
	require.NoError(t, EnsureDefaultAgentSettings(db, nil))
	db.Model(&agent.AgentSettings{}).Where("id = ?", agent.AgentSettingsRowID).
		Update("welcome_message", "   ")

	msg := AgentWelcomeMessage(db)
	assert.Equal(t, defaultAgentWelcome, msg)
}

func TestSyncAgentBotProfile_Success(t *testing.T) {
	db := newAgentSettingsDB(t)
	require.NoError(t, db.AutoMigrate(&user.User{}))

	botUser := user.User{Nickname: "old", AvatarURL: "", Sign: ""}
	require.NoError(t, db.Create(&botUser).Error)

	st := &agent.AgentSettings{
		DisplayName: "New Bot Name",
		AvatarURL:   "https://example.com/avatar.png",
		Sign:        "I am a bot",
	}
	err := SyncAgentBotProfile(db, botUser.ID, st)
	require.NoError(t, err)

	var updated user.User
	require.NoError(t, db.First(&updated, botUser.ID).Error)
	assert.Equal(t, "New Bot Name", updated.Nickname)
	assert.Equal(t, "https://example.com/avatar.png", updated.AvatarURL)
	assert.Equal(t, "I am a bot", updated.Sign)
}

func TestSyncAgentBotProfile_NilParams(t *testing.T) {
	err := SyncAgentBotProfile(nil, 0, nil)
	assert.NoError(t, err)
}

func TestSyncAgentBotProfile_EmptyNameDefaults(t *testing.T) {
	db := newAgentSettingsDB(t)
	require.NoError(t, db.AutoMigrate(&user.User{}))

	botUser := user.User{Nickname: "old", AvatarURL: "", Sign: ""}
	require.NoError(t, db.Create(&botUser).Error)

	st := &agent.AgentSettings{DisplayName: "   "}
	err := SyncAgentBotProfile(db, botUser.ID, st)
	require.NoError(t, err)

	var updated user.User
	require.NoError(t, db.First(&updated, botUser.ID).Error)
	assert.Equal(t, defaultAgentDisplayName, updated.Nickname)
}

func TestGetGlobalSystemPrompt_Default(t *testing.T) {
	db := newAgentSettingsDB(t)
	prompt := GetGlobalSystemPrompt(db)
	assert.Equal(t, defaultAgentSystemPrompt, prompt)
	prompt2 := GetGlobalSystemPrompt(nil)
	assert.Equal(t, defaultAgentSystemPrompt, prompt2)
}

func TestGetGlobalSystemPrompt_Custom(t *testing.T) {
	db := newAgentSettingsDB(t)
	require.NoError(t, EnsureDefaultAgentSettings(db, nil))
	db.Model(&agent.AgentSettings{}).Where("id = ?", agent.AgentSettingsRowID).
		Update("system_prompt", "Custom prompt")

	prompt := GetGlobalSystemPrompt(db)
	assert.True(t, strings.HasPrefix(prompt, "Custom"))
}

func TestGetGlobalSystemPrompt_EmptyFallback(t *testing.T) {
	db := newAgentSettingsDB(t)
	require.NoError(t, EnsureDefaultAgentSettings(db, nil))
	db.Model(&agent.AgentSettings{}).Where("id = ?", agent.AgentSettingsRowID).
		Update("system_prompt", "   ")

	prompt := GetGlobalSystemPrompt(db)
	assert.Equal(t, defaultAgentSystemPrompt, prompt)
}
