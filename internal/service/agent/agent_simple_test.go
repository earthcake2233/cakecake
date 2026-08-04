package agent

import (
	"cakecake/internal/data"
	"cakecake/internal/model/agent"
	"cakecake/internal/model/user"
	"cakecake/internal/service/servicetest"
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func newAgentServiceForTest(t *testing.T) (*AgentService, *gorm.DB) {
	t.Helper()
	db := servicetest.NewDB(t)
	return &AgentService{Store: NewAgentStore(db), Log: servicetest.ZapNop()}, db
}

func TestAgentService_SimpleHelpers(t *testing.T) {
	s, _ := newAgentServiceForTest(t)
	require.Equal(t, 12, s.MaxProfiles())
	s.ReloadProfiles()

	slug, err := s.NormalizeSlug(" MyBot_1 ")
	require.NoError(t, err)
	require.Equal(t, "mybot_1", slug)
	_, err = s.NormalizeSlug("bad slug!")
	require.Error(t, err)

	raw, _ := json.Marshal([]string{"a", " b "})
	list, err := s.UnmarshalWelcomeList(raw, nil)
	require.NoError(t, err)
	require.Equal(t, []string{"a", "b"}, list)
	fallback, err := s.UnmarshalWelcomeList(nil, []string{"fb"})
	require.NoError(t, err)
	require.Equal(t, []string{"fb"}, fallback)
	_, err = s.UnmarshalWelcomeList(json.RawMessage(`{"x":1}`), nil)
	require.Error(t, err)
}

func TestAgentService_ProfileCRUD(t *testing.T) {
	s, _ := newAgentServiceForTest(t)
	ctx := context.Background()

	list, err := s.ListAgentProfiles(ctx)
	require.NoError(t, err)
	require.Empty(t, list)

	p := &agent.AgentProfile{
		Slug: "assistant", BotUserID: 100, DisplayName: "助手",
		SystemPrompt: "prompt", WelcomeMessagesJSON: `["hi"]`, Enabled: true,
	}
	require.NoError(t, s.CreateAgentProfile(ctx, p))
	require.NotZero(t, p.ID)

	got, err := s.GetAgentProfile(ctx, p.ID)
	require.NoError(t, err)
	require.Equal(t, p.Slug, got.Slug)
	_, err = s.GetAgentProfile(ctx, 999)
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)

	require.NoError(t, s.UpdateAgentProfile(ctx, p.ID, map[string]interface{}{"display_name": "新名"}))
	got, err = s.GetAgentProfile(ctx, p.ID)
	require.NoError(t, err)
	require.Equal(t, "新名", got.DisplayName)

	require.NoError(t, s.UpdateAgentAvatar(ctx, p.ID, "avatar.jpg"))
	got, err = s.GetAgentProfile(ctx, p.ID)
	require.NoError(t, err)
	require.Equal(t, "avatar.jpg", got.AvatarURL)

	// Counts and slug checks.
	n, err := s.CountActiveAgentProfiles(ctx)
	require.NoError(t, err)
	require.Equal(t, int64(1), n)
	total, err := s.ProfileCount(ctx)
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	exists, err := s.CheckAgentSlugExists(ctx, "assistant")
	require.NoError(t, err)
	require.True(t, exists)
	exists, err = s.CheckAgentSlugExists(ctx, "nope")
	require.NoError(t, err)
	require.False(t, exists)

	require.NoError(t, s.DeleteAgentProfile(ctx, p.ID))
	_, err = s.GetAgentProfile(ctx, p.ID)
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)
}

func TestAgentService_BotUserAndSync(t *testing.T) {
	s, db := newAgentServiceForTest(t)
	ctx := context.Background()

	botID, err := s.CreateAgentBotUser(ctx, "bot1", "Bot", "sign", "avatar")
	require.NoError(t, err)
	require.NotZero(t, botID)

	// Rename profile slug updates the linked bot username.
	p := &agent.AgentProfile{Slug: "bot1", BotUserID: botID, DisplayName: "Bot"}
	require.NoError(t, db.Create(p).Error)
	require.NoError(t, s.RenameAgentProfileSlug(ctx, p, "renamed"))
	require.Equal(t, "renamed", p.Slug)
	var u user.User
	require.NoError(t, db.First(&u, botID).Error)
	require.Equal(t, "ai_renamed", u.Username)

	// Sync copies display fields to the bot user.
	require.NoError(t, s.SyncAgentProfile(ctx, &agent.AgentProfile{BotUserID: botID, DisplayName: "同步名", AvatarURL: "a", Sign: "s"}))
	require.NoError(t, db.First(&u, botID).Error)
	require.Equal(t, "同步名", u.Nickname)
}

func TestAgentService_GlobalPrompt(t *testing.T) {
	s, db := newAgentServiceForTest(t)
	ctx := context.Background()

	// No settings -> default prompt.
	require.Contains(t, s.GetGlobalSystemPrompt(ctx), "AI")

	// Custom settings override the global prompt.
	require.NoError(t, data.EnsureDefaultAgentSettings(db, servicetest.ZapNop()))
	require.NoError(t, db.Model(&agent.AgentSettings{}).Where("id = ?", 1).
		Update("system_prompt", "custom prompt").Error)
	require.Equal(t, "custom prompt", s.GetGlobalSystemPrompt(ctx))
}
