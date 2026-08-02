package service

import (
	"cakecake/internal/model/dm"
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func newDmService(t *testing.T) *DmService {
	t.Helper()
	db := newAgentTestDB(t)
	_, rdb := newAgentTestRedis(t)
	return NewDmService(db, rdb, zapNop())
}

func TestDmPairIDs(t *testing.T) {
	low, high := DmPairIDs(5, 3)
	require.Equal(t, uint64(3), low)
	require.Equal(t, uint64(5), high)
}

func TestDmService_ConversationLifecycle(t *testing.T) {
	s := newDmService(t)
	ctx := context.Background()

	conv, part, err := s.GetOrCreateConversation(ctx, 1, 2)
	require.NoError(t, err)
	require.NotNil(t, part)
	require.Equal(t, uint64(1), conv.UserLow)
	require.Equal(t, uint64(2), conv.UserHigh)

	// Existing conversation is returned with participants.
	conv2, part2, err := s.GetOrCreateConversation(ctx, 2, 1)
	require.NoError(t, err)
	require.Equal(t, conv.ID, conv2.ID)
	require.NotNil(t, part2)

	// FindConversationByUserIDs.
	found, err := s.FindConversationByUserIDs(ctx, 1, 2)
	require.NoError(t, err)
	require.Equal(t, conv.ID, found.ID)
	_, err = s.FindConversationByUserIDs(ctx, 1, 99)
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)

	// GetConversationByID + GetParticipant.
	gotConv, err := s.GetConversationByID(ctx, conv.ID)
	require.NoError(t, err)
	require.Equal(t, conv.ID, gotConv.ID)
	gotPart, err := s.GetParticipant(ctx, conv.ID, 1)
	require.NoError(t, err)
	require.Equal(t, uint64(1), gotPart.UserID)

	// DmPeerID.
	require.Equal(t, uint64(2), DmPeerID(conv, 1))
	require.Equal(t, uint64(1), DmPeerID(conv, 2))

	// ListConversations.
	convs, parts, err := s.ListConversations(ctx, 1)
	require.NoError(t, err)
	require.Len(t, convs, 1)
	require.Len(t, parts, 1)
	convs, parts, err = s.ListConversations(ctx, 99)
	require.NoError(t, err)
	require.Empty(t, convs)
	require.Empty(t, parts)
}

func TestDmService_Messages(t *testing.T) {
	s := newDmService(t)
	ctx := context.Background()
	conv, _, err := s.GetOrCreateConversation(ctx, 1, 2)
	require.NoError(t, err)

	msg, err := s.CreateMessage(ctx, conv.ID, 1, "hello", "user")
	require.NoError(t, err)
	require.Equal(t, "hello", msg.Content)
	require.Equal(t, conv.ID, msg.ConversationID)

	// CreateMessageInTransaction with callback.
	var cbMsg *dm.DmMessage
	err = s.CreateMessageInTransaction(ctx, conv.ID, 1, "tx", "user", func(tx *gorm.DB, m *dm.DmMessage) error {
		cbMsg = m
		return nil
	})
	require.NoError(t, err)
	require.NotNil(t, cbMsg)

	// ListMessages with cursor.
	msgs, err := s.ListMessages(ctx, conv.ID, 0, 10)
	require.NoError(t, err)
	require.Len(t, msgs, 2)
	msgs, err = s.ListMessages(ctx, conv.ID, cbMsg.ID, 10)
	require.NoError(t, err)
	require.Len(t, msgs, 1)
}

func TestDmService_PostMessage(t *testing.T) {
	s := newDmService(t)
	ctx := context.Background()
	conv, _, err := s.GetOrCreateConversation(ctx, 1, 2)
	require.NoError(t, err)

	res, err := s.PostMessage(ctx, conv.ID, 1, 2, "hi", "hi", false)
	require.NoError(t, err)
	require.NotNil(t, res.Message)
	require.NotNil(t, res.SelfPart)
	require.NotNil(t, res.PeerPart)
	require.Equal(t, uint64(2), res.PeerPart.UserID)
	require.Equal(t, uint32(1), res.PeerPart.UnreadCount)

	// Agent messages don't increment peer unread / create peer participant.
	agentRes, err := s.PostMessage(ctx, conv.ID, 1, 2, "bot", "bot", true)
	require.NoError(t, err)
	require.Nil(t, agentRes.PeerPart)

	// Mark read + unread total.
	require.NoError(t, s.MarkConversationRead(ctx, conv.ID, 2))
	require.Equal(t, int64(0), s.UnreadTotal(ctx, 2))

	// Settings update.
	require.NoError(t, s.UpdateConversationSettings(ctx, conv.ID, 1, map[string]interface{}{"pinned": true}))
	part, err := s.GetParticipant(ctx, conv.ID, 1)
	require.NoError(t, err)
	require.True(t, part.Pinned)
}

func TestDmService_DeleteAndReset(t *testing.T) {
	s := newDmService(t)
	ctx := context.Background()
	conv, _, err := s.GetOrCreateConversation(ctx, 1, 2)
	require.NoError(t, err)
	_, err = s.CreateMessage(ctx, conv.ID, 1, "x", "user")
	require.NoError(t, err)

	require.NoError(t, s.ResetConversationForAgent(ctx, conv.ID))
	var n int64
	require.NoError(t, s.db.Model(&dm.DmMessage{}).Count(&n).Error)
	require.Zero(t, n)

	require.NoError(t, s.DeleteConversation(ctx, conv.ID, 1))
	_, err = s.GetParticipant(ctx, conv.ID, 1)
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)
}

func TestDmService_EnsureParticipant(t *testing.T) {
	s := newDmService(t)
	ctx := context.Background()
	conv, _, err := s.GetOrCreateConversation(ctx, 1, 2)
	require.NoError(t, err)

	s.EnsureParticipant(nil, conv.ID, 9)
	part, err := s.GetParticipant(ctx, conv.ID, 9)
	require.NoError(t, err)
	require.Equal(t, uint64(9), part.UserID)
}
