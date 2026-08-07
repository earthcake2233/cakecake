package agent

import (
	"testing"

	"github.com/stretchr/testify/require"

	"cakecake/internal/aigateway"
	"cakecake/internal/model/dm"
	"cakecake/internal/ws"
)

func TestGenerateTraceID(t *testing.T) {
	id1 := generateTraceID()
	id2 := generateTraceID()
	require.Len(t, id1, 8)
	require.Len(t, id2, 8)
	require.NotEqual(t, id1, id2)
}

func TestEnabledTools_NilRC(t *testing.T) {
	s := &AgentService{}
	m := s.enabledTools()
	require.NotEmpty(t, m)
	for _, name := range m {
		require.True(t, name)
	}
}

func TestSetupToolCallbacks_NilGateway(t *testing.T) {
	s := &AgentService{ChatHub: ws.NewChatHub()}
	s.setupToolCallbacks("test123", 42)
	require.Nil(t, s.Gateway)
}

func TestHumanPeerForConversation(t *testing.T) {
	// Bot has the smaller id: user_low is the bot, but the human must win.
	require.Equal(t, uint64(18), humanPeerForConversation(&dm.DmConversation{UserLow: 14, UserHigh: 18}, 14))
	// Bot has the larger id: user_high is the bot, the human is user_low.
	require.Equal(t, uint64(3), humanPeerForConversation(&dm.DmConversation{UserLow: 3, UserHigh: 99}, 99))
	// Nil conversation is safe.
	require.Zero(t, humanPeerForConversation(nil, 14))
}

func TestClearToolCallbacks_NilGateway(t *testing.T) {
	s := &AgentService{}
	s.clearToolCallbacks()
	require.Nil(t, s.Gateway)
}

func TestSetupClearToolCallbacks_WithGateway(t *testing.T) {
	g := &aigateway.Gateway{}
	ch := ws.NewChatHub()
	s := &AgentService{Gateway: g, ChatHub: ch}
	s.setupToolCallbacks("test456", 99)
	require.NotNil(t, g.OnToolCallStart)
	require.NotNil(t, g.OnToolCallEnd)
	require.NotNil(t, g.OnToolResultData)
	s.clearToolCallbacks()
	require.Nil(t, g.OnToolCallStart)
	require.Nil(t, g.OnToolCallEnd)
	require.Nil(t, g.OnToolResultData)
}
