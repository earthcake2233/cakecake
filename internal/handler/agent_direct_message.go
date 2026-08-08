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

// PushAgentMessage is the ReplyPusher adapter: formats and pushes a persisted
// assistant message over WebSocket. The generation orchestration in the agent
// service decides WHAT to push; this adapter decides HOW.
func (a *API) PushAgentMessage(ctx context.Context, humanID uint64, conv *dm.DmConversation, msg *dm.DmMessage) {
	a.dmPushAgentMessage(ctx, humanID, conv, msg)
}

// PushEvent is the ReplyPusher adapter for generic agent events (continue
// mode, suggestions, ...).
func (a *API) PushEvent(humanID uint64, payload map[string]interface{}) {
	a.dmPushEvent(humanID, payload)
}

func (a *API) dmPushAgentMessage(ctx context.Context, humanID uint64, conv *dm.DmConversation, msg *dm.DmMessage) {
	if msg == nil || conv == nil {
		return
	}
	senderName, senderAvatar := a.dmUserBrief(ctx, msg.SenderID)
	out := a.dmFormatMessage(msg, senderName, senderAvatar)
	part, _ := a.DmSvc.GetParticipant(ctx, conv.ID, humanID)
	convPayload := a.dmFormatConversation(ctx, conv, humanID, part)
	event := dmMessageEvent{Type: "dm_message", Message: out}
	if part == nil || !part.Muted {
		a.dmPushEvent(humanID, event)
	}
	a.dmPushEvent(humanID, dmConversationEvent{Type: "dm_conversation", Conversation: convPayload})
}
