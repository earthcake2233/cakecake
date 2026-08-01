package data

import (
	"cakecake/internal/model/agent"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupProfileDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&agent.AgentProfile{}))
	return db
}

func TestProfileCount_Zero(t *testing.T) {
	db := setupProfileDB(t)
	n, err := ProfileCount(db)
	require.NoError(t, err)
	if n != 0 {
		t.Errorf("expected 0, got %d", n)
	}
}

func TestProfileCount_One(t *testing.T) {
	db := setupProfileDB(t)
	require.NoError(t, db.Create(&agent.AgentProfile{Slug: "test", DisplayName: "Test", BotUserID: 1}).Error)
	n, err := ProfileCount(db)
	require.NoError(t, err)
	if n != 1 {
		t.Errorf("expected 1, got %d", n)
	}
}

func TestListAgentProfiles_Empty(t *testing.T) {
	db := setupProfileDB(t)
	list, err := ListAgentProfiles(db)
	require.NoError(t, err)
	if len(list) != 0 {
		t.Errorf("expected 0, got %d", len(list))
	}
}

func TestGetAgentProfile_NotFound(t *testing.T) {
	db := setupProfileDB(t)
	_, err := GetAgentProfile(db, 999)
	if err == nil {
		t.Error("expected error for not found")
	}
}

func TestGetAgentProfile_Found(t *testing.T) {
	db := setupProfileDB(t)
	require.NoError(t, db.Create(&agent.AgentProfile{Slug: "test_bot", DisplayName: "Test", BotUserID: 1}).Error)
	var created agent.AgentProfile
	db.First(&created)
	p, err := GetAgentProfile(db, created.ID)
	require.NoError(t, err)
	if p == nil || p.Slug != "test_bot" {
		t.Errorf("got slug=%v", p)
	}
}

func TestGetAgentProfileByBotUserID_Found(t *testing.T) {
	db := setupProfileDB(t)
	require.NoError(t, db.Create(&agent.AgentProfile{Slug: "test_bot", DisplayName: "Test", BotUserID: 42}).Error)
	p, err := GetAgentProfileByBotUserID(db, 42)
	require.NoError(t, err)
	if p == nil || p.BotUserID != 42 {
		t.Errorf("got %v", p)
	}
}

func TestPickWelcomeMessage_NilProfile(t *testing.T) {
	got := PickWelcomeMessage(nil)
	if got == "" {
		t.Error("expected default message")
	}
}

func TestPickWelcomeMessage_Custom(t *testing.T) {
	msg, _ := MarshalWelcomeList([]string{"Custom Welcome"})
	p := &agent.AgentProfile{WelcomeMessagesJSON: msg}
	got := PickWelcomeMessage(p)
	if got != "Custom Welcome" {
		t.Errorf("got %q", got)
	}
}

func TestNormalizeAgentSlug_ValidSlug(t *testing.T) {
	got, err := NormalizeAgentSlug("valid_slug_1")
	if err != nil {
		t.Fatal(err)
	}
	if got != "valid_slug_1" {
		t.Errorf("got %q", got)
	}
}

func TestUnmarshalWelcomeList_Fallback(t *testing.T) {
	got, err := UnmarshalWelcomeList(nil, []string{"fb"})
	require.NoError(t, err)
	if len(got) != 1 || got[0] != "fb" {
		t.Errorf("got %v", got)
	}
}

func TestEnsureAgentConversationForProfile_NilDB(t *testing.T) {
	_, _, err := EnsureAgentConversationForProfile(nil, 0, nil)
	if err == nil {
		t.Error("expected error")
	}
}

func TestEnsureAgentConversationForProfile_Invalid(t *testing.T) {
	db := setupProfileDB(t)
	_, _, err := EnsureAgentConversationForProfile(db, 0, &agent.AgentProfile{BotUserID: 0})
	if err == nil {
		t.Error("expected error")
	}
}

func TestRenameAgentProfileSlug_NilParams(t *testing.T) {
	db := setupProfileDB(t)
	if err := RenameAgentProfileSlug(nil, nil, "x"); err == nil {
		t.Error("expected error")
	}
	if err := RenameAgentProfileSlug(nil, &agent.AgentProfile{}, "x"); err == nil {
		t.Error("expected error")
	}
	if err := RenameAgentProfileSlug(db, nil, "x"); err == nil {
		t.Error("expected error")
	}
}

func TestSyncAgentProfile_Nil(t *testing.T) {
	if err := SyncAgentProfile(nil, nil); err != nil {
		t.Errorf("got %v", err)
	}
}
