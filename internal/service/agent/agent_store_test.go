package agent

import (
	"cakecake/internal/model/agent"
	"cakecake/internal/model/dm"
	"cakecake/internal/service/servicetest"
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAgentStore_UpdateMessageSuggestions(t *testing.T) {
	db := servicetest.NewDB(t)
	store := NewAgentStore(db)

	msg := dm.DmMessage{ConversationID: 1, SenderID: 14, Role: "assistant", Content: "hi"}
	require.NoError(t, db.Create(&msg).Error)

	require.NoError(t, store.UpdateMessageSuggestions(context.Background(), msg.ID,
		[]string{"追问一", "追问二"}))

	var got dm.DmMessage
	require.NoError(t, db.First(&got, msg.ID).Error)
	require.Contains(t, got.Suggestions, "追问一")
	require.Contains(t, got.Suggestions, "追问二")

	// Empty list is a no-op.
	require.NoError(t, store.UpdateMessageSuggestions(context.Background(), msg.ID, nil))
	require.Error(t, store.UpdateMessageSuggestions(context.Background(), 0, []string{"x"}))
}

func TestAgentStore_SetAndListFeedback(t *testing.T) {
	db := servicetest.NewDB(t)
	store := NewAgentStore(db)

	msg := dm.DmMessage{ConversationID: 1, SenderID: 14, Role: "assistant", Content: "回复内容"}
	require.NoError(t, db.Create(&msg).Error)

	ctx := context.Background()
	require.NoError(t, store.SetMessageFeedback(ctx, msg.ID, 7, "like"))
	rows, err := store.ListAgentFeedbacks(ctx, 50, 0)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, "like", rows[0].Feedback)

	// Toggle off.
	require.NoError(t, store.SetMessageFeedback(ctx, msg.ID, 7, "like"))
	rows, err = store.ListAgentFeedbacks(ctx, 50, 0)
	require.NoError(t, err)
	require.Len(t, rows, 0)

	// Switch to dislike.
	require.NoError(t, store.SetMessageFeedback(ctx, msg.ID, 7, "dislike"))
	withContent, err := store.ListAgentFeedbacksWithContent(ctx, 50, 0)
	require.NoError(t, err)
	require.Len(t, withContent, 1)
	require.Equal(t, "dislike", withContent[0].Feedback)
	require.Equal(t, "回复内容", withContent[0].MessageContent)

	require.Error(t, store.SetMessageFeedback(ctx, 0, 7, "like"))
}

func TestAgentFeedbackModel_TableName(t *testing.T) {
	require.Equal(t, "agent_feedbacks", (agent.AgentFeedback{}).TableName())
}
