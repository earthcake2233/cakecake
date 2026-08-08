package handler

import (
	"cakecake/internal/model/dm"
	"context"

	"go.uber.org/zap"
)

func (a *API) dmIsAgentConv(conv *dm.DmConversation) bool {
	return a.Agent != nil && a.Agent.IsAgentConversation(conv)
}

func (a *API) ensureAgentConversationFor(uid uint64) {
	if a.Agent == nil || uid == 0 {
		return
	}
	if err := a.Agent.EnsureForUser(uid); err != nil {
		a.Log.Warn("ensure agent conversation", zap.Uint64("user_id", uid), zap.Error(err))
	}
}

// FormatAgentMessage is the ReplyPusher adapter: it formats a persisted
// assistant message into the WebSocket payloads (dm_message + dm_conversation)
// and returns them. The agent service is responsible for delivering the
// payloads (locally or through the cross-instance relay), so multi-replica
// deployments never need to know where the user's WS connection lives.
func (a *API) FormatAgentMessage(ctx context.Context, humanID uint64, conv *dm.DmConversation, msg *dm.DmMessage) ([]map[string]interface{}, error) {
	if msg == nil || conv == nil {
		return nil, nil
	}
	senderName, senderAvatar := a.dmUserBrief(ctx, msg.SenderID)
	out := a.dmFormatMessage(msg, senderName, senderAvatar)
	part, _ := a.DmSvc.GetParticipant(ctx, conv.ID, humanID)
	convPayload := a.dmFormatConversation(ctx, conv, humanID, part)
	payloads := []map[string]interface{}{
		{"type": "dm_message", "message": out},
		{"type": "dm_conversation", "conversation": convPayload},
	}
	if part != nil && part.Muted {
		// Muted users still get the conversation update but not the message
		// event itself (matching the previous push behavior).
		payloads = payloads[1:]
	}
	return payloads, nil
}
