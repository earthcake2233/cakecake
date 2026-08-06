//go:build integration

package handler

import (
	"cakecake/internal/model/agent"
	"cakecake/internal/model/dm"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestResetDmAgentConversation_Welcome verifies that clearing an agent
// conversation deletes history, writes a fresh welcome message, resets unread,
// and returns {conversation, welcome_message} per the frontend contract.
func TestResetDmAgentConversation_Welcome(t *testing.T) {
	api, r, _ := newTestAPI(t)

	// Seed bot user + agent profile.
	bot := seedUser(t, api, "agent_bot_reset", "AI助手", 0)
	prof := agent.AgentProfile{
		Slug: "reset_test", BotUserID: bot.ID, DisplayName: "AI助手",
		Enabled: true, WelcomeMessagesJSON: `["你好，欢迎回来"]`,
	}
	require.NoError(t, api.DB.Create(&prof).Error)

	// Human user + agent conversation with unread and one history message.
	u := seedUser(t, api, "reset_human", "人类", 0)
	tk := tok(t, api, u.ID)
	conv := dm.DmConversation{Kind: dm.DmKindAgent, UserLow: u.ID, UserHigh: bot.ID, AgentProfileID: prof.ID}
	require.NoError(t, api.DB.Create(&conv).Error)
	require.NoError(t, api.DB.Create(&dm.DmParticipant{ConversationID: conv.ID, UserID: u.ID, UnreadCount: 3}).Error)
	require.NoError(t, api.DB.Create(&dm.DmParticipant{ConversationID: conv.ID, UserID: bot.ID}).Error)
	require.NoError(t, api.DB.Create(&dm.DmMessage{ConversationID: conv.ID, SenderID: u.ID, Role: "user", Content: "旧消息"}).Error)

	// Reset.
	w := doJSON(r, "POST", fmt.Sprintf("/api/v1/dm/conversations/%d/reset", conv.ID), tk, nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var body struct {
		Data struct {
			Conversation dmConversationDTO `json:"conversation"`
			Welcome      *dmMessageDTO     `json:"welcome_message"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.NotNil(t, body.Data.Welcome, "agent reset should return a welcome message")
	require.Equal(t, "assistant", body.Data.Welcome.Role)
	require.Equal(t, "你好，欢迎回来", body.Data.Welcome.Content)
	require.Equal(t, conv.ID, body.Data.Conversation.ID)
	require.Zero(t, body.Data.Conversation.UnreadCount)

	// History is replaced by exactly the welcome message.
	var n int64
	require.NoError(t, api.DB.Model(&dm.DmMessage{}).Where("conversation_id = ?", conv.ID).Count(&n).Error)
	require.Equal(t, int64(1), n)
	var part dm.DmParticipant
	require.NoError(t, api.DB.Where("conversation_id = ? AND user_id = ?", conv.ID, u.ID).First(&part).Error)
	require.Zero(t, part.UnreadCount)
}

// TestResetDmAgentConversation_NonAgent keeps the legacy clear-history behavior
// for human conversations and returns a null welcome_message.
func TestResetDmAgentConversation_NonAgent(t *testing.T) {
	api, r, _ := newTestAPI(t)
	u1 := seedUser(t, api, "reset_u1", "甲", 0)
	u2 := seedUser(t, api, "reset_u2", "乙", 0)
	tk := tok(t, api, u1.ID)
	conv := dm.DmConversation{Kind: dm.DmKindHuman, UserLow: u1.ID, UserHigh: u2.ID}
	require.NoError(t, api.DB.Create(&conv).Error)
	require.NoError(t, api.DB.Create(&dm.DmParticipant{ConversationID: conv.ID, UserID: u1.ID}).Error)
	require.NoError(t, api.DB.Create(&dm.DmParticipant{ConversationID: conv.ID, UserID: u2.ID}).Error)
	require.NoError(t, api.DB.Create(&dm.DmMessage{ConversationID: conv.ID, SenderID: u1.ID, Role: "user", Content: "hi"}).Error)

	w := doJSON(r, "POST", fmt.Sprintf("/api/v1/dm/conversations/%d/reset", conv.ID), tk, nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var body struct {
		Data struct {
			Welcome *dmMessageDTO `json:"welcome_message"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Nil(t, body.Data.Welcome)
	var n int64
	require.NoError(t, api.DB.Model(&dm.DmMessage{}).Where("conversation_id = ?", conv.ID).Count(&n).Error)
	require.Zero(t, n)
}
