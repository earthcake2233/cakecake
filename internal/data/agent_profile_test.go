package data

import (
	"cakecake/internal/config"
	"cakecake/internal/model/agent"
	"cakecake/internal/model/dm"
	"cakecake/internal/model/user"
	"encoding/json"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func newAgentProfileDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&agent.AgentProfile{},
		&agent.AgentSettings{},
		&user.User{},
		&dm.DmConversation{},
		&dm.DmParticipant{},
		&dm.DmMessage{},
	))
	return db
}

func seedAgentProfile(t *testing.T, db *gorm.DB) agent.AgentProfile {
	t.Helper()
	p := agent.AgentProfile{
		Slug:                "assistant",
		BotUserID:           100,
		DisplayName:         "助手",
		SystemPrompt:        "prompt",
		WelcomeMessagesJSON: agent.EncodeWelcomeMessages([]string{"hi"}),
		Enabled:             true,
	}
	require.NoError(t, db.Create(&p).Error)
	return p
}

func TestEnsureAgentProfiles_CreatesDefault(t *testing.T) {
	db := newAgentProfileDB(t)
	cfg := &config.C{AgentBotUsername: "bot-user"}
	require.NoError(t, EnsureAgentProfiles(db, cfg, zap.NewNop()))

	var list []agent.AgentProfile
	require.NoError(t, db.Find(&list).Error)
	require.Len(t, list, 1)
	require.Equal(t, "default", list[0].Slug)
	var u user.User
	require.NoError(t, db.Where("username = ?", "bot-user").First(&u).Error)
	require.NotZero(t, u.ID)
}

func TestEnsureAgentProfiles_Existing(t *testing.T) {
	db := newAgentProfileDB(t)
	p := seedAgentProfile(t, db)
	require.NoError(t, EnsureAgentProfiles(db, nil, nil))
	var n int64
	require.NoError(t, db.Model(&agent.AgentProfile{}).Count(&n).Error)
	require.Equal(t, int64(1), n)
	require.NoError(t, db.First(&agent.AgentProfile{}, p.ID).Error)
}

func TestEnsureAgentProfiles_NilDB(t *testing.T) {
	require.NoError(t, EnsureAgentProfiles(nil, nil, nil))
}

func TestFindOrCreateLegacyBotUser(t *testing.T) {
	db := newAgentProfileDB(t)
	id, err := findOrCreateLegacyBotUser(db, nil, "助手", "sign", "avatar", nil)
	require.NoError(t, err)
	require.NotZero(t, id)
	id2, err := findOrCreateLegacyBotUser(db, nil, "助手", "sign", "avatar", nil)
	require.NoError(t, err)
	require.Equal(t, id, id2)
}

func TestProfileQueries(t *testing.T) {
	db := newAgentProfileDB(t)
	p := seedAgentProfile(t, db)
	disabled := agent.AgentProfile{Slug: "off", BotUserID: 101, DisplayName: "off", Enabled: false}
	require.NoError(t, db.Create(&disabled).Error)
	require.NoError(t, db.Model(&disabled).Update("enabled", false).Error)

	list, err := ListAgentProfiles(db)
	require.NoError(t, err)
	require.Len(t, list, 2)
	enabled, err := ListEnabledAgentProfiles(db)
	require.NoError(t, err)
	require.Len(t, enabled, 1)
	got, err := GetAgentProfile(db, p.ID)
	require.NoError(t, err)
	require.Equal(t, p.Slug, got.Slug)
	_, err = GetAgentProfile(db, 999)
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)
	byBot, err := GetAgentProfileByBotUserID(db, 100)
	require.NoError(t, err)
	require.Equal(t, p.ID, byBot.ID)

	n, err := ProfileCount(db)
	require.NoError(t, err)
	require.Equal(t, int64(2), n)
	require.Equal(t, maxAgentProfiles, MaxAgentProfilesLimit())
}

func TestPickWelcomeMessage(t *testing.T) {
	require.Equal(t, defaultAgentWelcome, PickWelcomeMessage(nil))
	require.Equal(t, defaultAgentWelcome, PickWelcomeMessage(&agent.AgentProfile{WelcomeMessagesJSON: "[]"}))
	require.Equal(t, "hi", PickWelcomeMessage(&agent.AgentProfile{WelcomeMessagesJSON: `["hi"]`}))
	// Multi-item: any of them.
	got := PickWelcomeMessage(&agent.AgentProfile{WelcomeMessagesJSON: `["a","b","c"]`})
	require.Contains(t, []string{"a", "b", "c"}, got)
}

func TestEnsureAgentConversationForProfile(t *testing.T) {
	db := newAgentProfileDB(t)
	p := seedAgentProfile(t, db)

	// Invalid inputs.
	_, _, err := EnsureAgentConversationForProfile(db, 0, &p)
	require.Error(t, err)
	_, _, err = EnsureAgentConversationForProfile(nil, 1, &p)
	require.Error(t, err)
	_, _, err = EnsureAgentConversationForProfile(db, 1, nil)
	require.Error(t, err)

	// Disabled profile -> nil.
	off := agent.AgentProfile{Slug: "off", BotUserID: 101, DisplayName: "off", Enabled: false}
	require.NoError(t, db.Create(&off).Error)
	require.NoError(t, db.Model(&off).Update("enabled", false).Error)
	conv, created, err := EnsureAgentConversationForProfile(db, 1, &off)
	require.NoError(t, err)
	require.Nil(t, conv)
	require.False(t, created)

	// Create new.
	conv, created, err = EnsureAgentConversationForProfile(db, 1, &p)
	require.NoError(t, err)
	require.True(t, created)
	require.Equal(t, dm.DmKindAgent, conv.Kind)
	var msgCount int64
	require.NoError(t, db.Model(&dm.DmMessage{}).Count(&msgCount).Error)
	require.Equal(t, int64(1), msgCount)

	// Reuse existing.
	conv2, created, err := EnsureAgentConversationForProfile(db, 1, &p)
	require.NoError(t, err)
	require.False(t, created)
	require.Equal(t, conv.ID, conv2.ID)

	// EnsureAll.
	require.NoError(t, EnsureAllAgentConversationsForUser(db, 2))
}

func TestAgentBotUserAndSlug(t *testing.T) {
	db := newAgentProfileDB(t)
	id, err := CreateAgentBotUser(db, "my-bot", "Bot", "s", "a")
	require.NoError(t, err)
	require.NotZero(t, id)
	require.Equal(t, "ai_my-bot", AgentBotUsername("my-bot"))

	slug, err := NormalizeAgentSlug("  MyBot_1 ")
	require.NoError(t, err)
	require.Equal(t, "mybot_1", slug)
	_, err = NormalizeAgentSlug("1bad")
	require.Error(t, err)
	_, err = NormalizeAgentSlug("")
	require.Error(t, err)
}

func TestRenameAgentProfileSlug(t *testing.T) {
	db := newAgentProfileDB(t)
	require.NoError(t, db.Create(&user.User{ID: 100, Username: "ai_assistant", PasswordHash: "x"}).Error)
	p := seedAgentProfile(t, db)

	// Same slug -> no-op.
	require.NoError(t, RenameAgentProfileSlug(db, &p, "assistant"))
	// Rename.
	require.NoError(t, RenameAgentProfileSlug(db, &p, "new_name"))
	require.Equal(t, "new_name", p.Slug)
	var u user.User
	require.NoError(t, db.First(&u, 100).Error)
	require.Equal(t, "ai_new_name", u.Username)
	// Invalid slug.
	require.Error(t, RenameAgentProfileSlug(db, &p, "Bad Slug!"))
	// Taken slug.
	require.NoError(t, db.Create(&agent.AgentProfile{Slug: "other", BotUserID: 101}).Error)
	require.Error(t, RenameAgentProfileSlug(db, &p, "other"))
}

func TestSyncAgentProfile(t *testing.T) {
	db := newAgentProfileDB(t)
	require.NoError(t, db.Create(&user.User{ID: 100, Username: "u", PasswordHash: "x"}).Error)
	p := agent.AgentProfile{BotUserID: 100, DisplayName: "新名", AvatarURL: "a", Sign: "s"}
	require.NoError(t, SyncAgentProfile(db, &p))
	require.NoError(t, SyncAgentProfile(db, nil))
	require.NoError(t, syncAgentProfileToUser(nil, &p, 100))
	require.NoError(t, syncAgentProfileToUser(db, &p, 0))
	var u user.User
	require.NoError(t, db.First(&u, 100).Error)
	require.Equal(t, "新名", u.Nickname)
}

func TestWelcomeListMarshal(t *testing.T) {
	out, err := MarshalWelcomeList([]string{"a", "b"})
	require.NoError(t, err)
	require.Equal(t, `["a","b"]`, out)
	_, err = MarshalWelcomeList(nil)
	require.Error(t, err)
	_, err = MarshalWelcomeList([]string{"", "x"})
	require.Error(t, err)

	list, err := UnmarshalWelcomeList(nil, []string{"fb"})
	require.NoError(t, err)
	require.Equal(t, []string{"fb"}, list)
	raw, _ := json.Marshal([]string{" a ", "b"})
	list, err = UnmarshalWelcomeList(raw, nil)
	require.NoError(t, err)
	require.Equal(t, []string{"a", "b"}, list)
	_, err = UnmarshalWelcomeList(json.RawMessage(`{"x":1}`), nil)
	require.Error(t, err)
	_, err = UnmarshalWelcomeList(json.RawMessage(`[""]`), nil)
	require.Error(t, err)
}

func TestBackfillDmAgentProfileIDs(t *testing.T) {
	db := newAgentProfileDB(t)
	p := seedAgentProfile(t, db)
	require.NoError(t, db.Create(&dm.DmConversation{
		UserLow: 1, UserHigh: p.BotUserID, Kind: dm.DmKindAgent,
	}).Error)
	require.NoError(t, backfillDmAgentProfileIDs(db, nil))
	var conv dm.DmConversation
	require.NoError(t, db.First(&conv).Error)
	require.Equal(t, p.ID, conv.AgentProfileID)
}

func TestEnsureDmParticipants(t *testing.T) {
	db := newAgentProfileDB(t)
	require.NoError(t, db.Create(&dm.DmConversation{UserLow: 1, UserHigh: 2}).Error)
	ensureDmParticipants(db, 1, 1, 2)
	ensureDmParticipants(db, 1, 1, 2) // idempotent
	var n int64
	require.NoError(t, db.Model(&dm.DmParticipant{}).Count(&n).Error)
	require.Equal(t, int64(2), n)
}

func TestAgentBotUsernameTrim(t *testing.T) {
	require.Equal(t, "ai_x", AgentBotUsername("  x  "))
	require.True(t, strings.HasPrefix(AgentBotUsername("abc"), "ai_"))
}
