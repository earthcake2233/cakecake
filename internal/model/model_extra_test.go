package model_test

import (
	"encoding/json"
	"testing"
	"time"

	"minibili/internal/model"

	"github.com/stretchr/testify/require"
)

func TestNotification_ReadStatus(t *testing.T) {
	n := model.Notification{RecipientID: 1, Type: "reply", IsRead: true}
	require.True(t, n.IsRead)
	n2 := model.Notification{RecipientID: 1, Type: "like", IsRead: false}
	require.False(t, n2.IsRead)
}

func TestNotification_ZeroValues(t *testing.T) {
	var n model.Notification
	require.Empty(t, n.Type)
	require.Empty(t, n.SenderNamesJSON)
	require.Equal(t, 0, n.TotalLikes)
}

func TestNotification_JSONSerialization(t *testing.T) {
	n := model.Notification{
		RecipientID: 42,
		Type:        "system",
		RelatedID:   1,
	}
	b, err := json.Marshal(n)
	require.NoError(t, err)
	require.Contains(t, string(b), "RecipientID")
	require.Contains(t, string(b), "Type")
}

func TestLikeNotifMute_StructFields(t *testing.T) {
	m := model.LikeNotifMute{
		RecipientID: 42,
		CommentID:   100,
	}
	require.Equal(t, uint64(42), m.RecipientID)
	require.Equal(t, uint64(100), m.CommentID)
}

func TestIsUserAnonymized_Anonymized(t *testing.T) {
	now := time.Now()
	u := &model.User{AnonymizedAt: &now}
	require.True(t, model.IsUserAnonymized(u))
}

func TestIsUserAnonymized_Nil(t *testing.T) {
	require.False(t, model.IsUserAnonymized(nil))
}

func TestIsUserAnonymized_NotAnonymized(t *testing.T) {
	u := &model.User{}
	require.False(t, model.IsUserAnonymized(u))
}

func TestDisplayUsername_Anonymized(t *testing.T) {
	now := time.Now()
	u := &model.User{
		Username:     "testuser",
		AnonymizedAt: &now,
	}
	require.Equal(t, "已注销用户", model.DisplayUsername(u))
}

func TestDisplayUsername_Normal(t *testing.T) {
	u := &model.User{Username: "alice"}
	require.Equal(t, "alice", model.DisplayUsername(u))
}

func TestDisplayUsername_Nil(t *testing.T) {
	require.Empty(t, model.DisplayUsername(nil))
}

func TestUserBlock_StructFields(t *testing.T) {
	b := model.UserBlock{
		BlockerID: 1,
		BlockedID: 42,
	}
	require.Equal(t, uint64(1), b.BlockerID)
	require.Equal(t, uint64(42), b.BlockedID)
}

func TestAdmin_StructFields(t *testing.T) {
	now := time.Now()
	a := model.Admin{
		ID:          1,
		Username:    "ops",
		PasswordHash: "hash",
		DisplayName: "Operator",
		Status:      "active",
		LastLoginAt: &now,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	require.Equal(t, "ops", a.Username)
	require.Equal(t, "Operator", a.DisplayName)
	require.NotNil(t, a.LastLoginAt)
}

func TestAdmin_Disabled(t *testing.T) {
	a := model.Admin{
		Username: "inactive",
		Status:   "disabled",
	}
	require.Equal(t, "disabled", a.Status)
}

func TestAdmin_ZeroValues(t *testing.T) {
	var a model.Admin
	require.Empty(t, a.Username)
	require.Empty(t, a.DisplayName)
	require.Nil(t, a.LastLoginAt)
}

func TestDmParticipant_StructFields(t *testing.T) {
	now := time.Now()
	p := model.DmParticipant{
		ConversationID: 10,
		UserID:         42,
		UnreadCount:    5,
		Pinned:         true,
		PinnedAt:       &now,
		Muted:          false,
	}
	require.Equal(t, uint64(10), p.ConversationID)
	require.Equal(t, uint32(5), p.UnreadCount)
	require.True(t, p.Pinned)
	require.False(t, p.Muted)
	require.NotNil(t, p.PinnedAt)
	require.Nil(t, p.HiddenAt)
}

func TestDmParticipant_DefaultValues(t *testing.T) {
	var p model.DmParticipant
	require.Equal(t, uint32(0), p.UnreadCount)
	require.False(t, p.Pinned)
	require.False(t, p.Muted)
}

func TestDmMessage_StructFields(t *testing.T) {
	msg := model.DmMessage{
		ConversationID: 10,
		SenderID:       42,
		Role:           "user",
		Content:        "Hello!",
	}
	require.Equal(t, uint64(10), msg.ConversationID)
	require.Equal(t, uint64(42), msg.SenderID)
	require.Equal(t, "user", msg.Role)
	require.Equal(t, "Hello!", msg.Content)
}

func TestDmMessage_EmptyContent(t *testing.T) {
	msg := model.DmMessage{
		ConversationID: 1,
		SenderID:       1,
		Role:           "assistant",
	}
	require.Empty(t, msg.Content)
}

func TestDmConversation_Struct(t *testing.T) {
	c := model.DmConversation{
		UserLow:      1,
		UserHigh:     2,
		Kind:         "human",
		LastPreview:  "Hey!",
	}
	require.Equal(t, "human", c.Kind)
	require.Equal(t, "Hey!", c.LastPreview)
}

func TestDmConversation_AgentKind(t *testing.T) {
	c := model.DmConversation{
		UserLow:        1,
		UserHigh:       2,
		Kind:           "agent",
		AgentProfileID: 5,
	}
	require.Equal(t, "agent", c.Kind)
	require.Equal(t, uint64(5), c.AgentProfileID)
}

func TestVideoFavorite_StructFields(t *testing.T) {
	vf := model.VideoFavorite{
		FolderID: 1,
		VideoID:  100,
	}
	require.Equal(t, uint64(1), vf.FolderID)
	require.Equal(t, uint64(100), vf.VideoID)
}

func TestVideoFavorite_ZeroFolder(t *testing.T) {
	vf := model.VideoFavorite{VideoID: 42}
	require.Equal(t, uint64(0), vf.FolderID)
	require.Equal(t, uint64(42), vf.VideoID)
}

func TestHomeBanner_Struct(t *testing.T) {
	now := time.Now()
	b := model.HomeBanner{
		Title:      "Welcome",
		ImageURL:   "https://example.com/banner.jpg",
		LinkType:   "video",
		LinkTarget: "100",
		SortOrder:  1,
		Enabled:    true,
		StartAt:    &now,
		EndAt:      &now,
	}
	require.Equal(t, "Welcome", b.Title)
	require.Equal(t, "video", b.LinkType)
	require.True(t, b.Enabled)
}

func TestHomeBanner_Defaults(t *testing.T) {
	var b model.HomeBanner
	// In Go, struct fields default to zero values; GORM default:"none" only applies at DB level.
	require.Empty(t, b.LinkType)
	require.Empty(t, b.Title)
	require.False(t, b.Enabled)
}

func TestHotSearchOp_Struct(t *testing.T) {
	op := model.HotSearchOp{
		OpType:       "pin",
		Keyword:      "news",
		DisplayTitle: "Breaking News",
		Badge:        "热",
		PinRank:      1,
		Enabled:      true,
	}
	require.Equal(t, "pin", op.OpType)
	require.Equal(t, "Breaking News", op.DisplayTitle)
	require.Equal(t, "热", op.Badge)
}

func TestHotSearchOp_Defaults(t *testing.T) {
	var op model.HotSearchOp
	require.Empty(t, op.OpType)
	require.Equal(t, 0, op.PinRank)
	require.False(t, op.Enabled)
}

func TestAgentSettings_Struct(t *testing.T) {
	s := model.AgentSettings{
		DisplayName:      "AI Assistant",
		SystemPrompt:     "Be helpful",
		AssistantEnabled: true,
	}
	require.Equal(t, "AI Assistant", s.DisplayName)
	require.True(t, s.AssistantEnabled)
}

func TestAgentProfile_Struct(t *testing.T) {
	p := model.AgentProfile{
		Slug:                "default",
		BotUserID:           1,
		DisplayName:         "Bot",
		SystemPrompt:        "Be nice",
		WelcomeMessagesJSON: `["Hi"]`,
		SortOrder:           0,
		Enabled:             true,
	}
	require.Equal(t, "default", p.Slug)
	require.Equal(t, "Bot", p.DisplayName)
	require.True(t, p.Enabled)
}

func TestAgentProfile_Disabled(t *testing.T) {
	p := model.AgentProfile{
		Slug:    "offline",
		Enabled: false,
	}
	require.False(t, p.Enabled)
}

func TestParseWelcomeMessages(t *testing.T) {
	tests := []struct {
		raw  string
		want []string
	}{
		{`["hello","world"]`, []string{"hello", "world"}},
		{`["hello","","world"]`, []string{"hello", "world"}},
		{"", nil},
		{`invalid json`, nil},
		{`[]`, []string{}},
		{`["  "]`, []string{}},
		{`["hello"]`, []string{"hello"}},
	}
	for _, tc := range tests {
		got := model.ParseWelcomeMessages(tc.raw)
		require.Equal(t, tc.want, got, "ParseWelcomeMessages(%q)", tc.raw)
	}
}

func TestEncodeWelcomeMessages(t *testing.T) {
	tests := []struct {
		input []string
		want  string
	}{
		{[]string{"hello", "world"}, `["hello","world"]`},
		{[]string{"hello", "", "world"}, `["hello","world"]`},
		{[]string{}, `[]`},
		{nil, `[]`},
		{[]string{""}, `[]`},
		{[]string{"你好"}, `["你好"]`},
	}
	for _, tc := range tests {
		got := model.EncodeWelcomeMessages(tc.input)
		require.Equal(t, tc.want, got, "EncodeWelcomeMessages(%v)", tc.input)
	}
}

func TestUser_DeletionFields(t *testing.T) {
	now := time.Now()
	u := model.User{
		DeletionRequestedAt: &now,
		DeletionEffectiveAt: &now,
		AnonymizedAt:        &now,
		FirstPublishedAt:    &now,
	}
	require.NotNil(t, u.DeletionRequestedAt)
	require.NotNil(t, u.DeletionEffectiveAt)
	require.NotNil(t, u.AnonymizedAt)
	require.NotNil(t, u.FirstPublishedAt)
}

func TestUser_PrivacyDefaults(t *testing.T) {
	var u model.User
	require.False(t, u.PrivacyPublicFavorites)
	require.False(t, u.PrivacyPublicRecentCoins)
	require.False(t, u.PrivacyPublicFollowing)
	require.False(t, u.PrivacyPublicFans)
	// In Go, bool zero value is false; GORM default:1 only applies at DB level.
	require.False(t, u.PrivacyPublicBirthday)
}

func TestConstants(t *testing.T) {
	require.Equal(t, "human", model.DmKindHuman)
	require.Equal(t, "agent", model.DmKindAgent)
}
