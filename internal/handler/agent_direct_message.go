package handler

import (
	"cakecake/internal/model/dm"
	"context"
	"strings"

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

// runAgentReply generates and delivers an assistant message asynchronously.
func (a *API) runAgentReply(humanID uint64, conv *dm.DmConversation, userText string) {
	if a.Agent == nil || conv == nil {
		return
	}
	go func() {
		ctx := context.Background()
		if !a.Agent.CheckQuota(ctx, humanID) {
			a.pushAgentFallback(ctx, humanID, conv, "今日 AI 对话次数已达上限，请明天再试。")
			return
		}
		result, err := a.Agent.GenerateReply(ctx, conv, userText)
		if err != nil {
			a.Log.Warn("agent generate", zap.Uint64("conv", conv.ID), zap.Error(err))
			msg := "AI 助手暂时不可用，请稍后再试。"
			if strings.Contains(err.Error(), "sensitive") {
				msg = "消息包含敏感内容，请修改后重试。"
			}
			if strings.Contains(err.Error(), "not configured") {
				msg = "AI 助手未配置（需设置 DEEPSEEK_API_KEY）。"
			}
			if strings.Contains(err.Error(), "disabled") {
				msg = "AI 助手已暂停服务，请稍后再试。"
			}
			a.pushAgentFallback(ctx, humanID, conv, msg)
			return
		}
		a.Agent.IncrQuota(ctx, humanID)
		msg, err := a.Agent.PostAssistantMessage(conv, humanID, result.Content, string(result.ToolActivities), string(result.ToolResultData))
		if err != nil {
			a.Log.Error("agent persist reply", zap.Error(err))
			return
		}
		a.dmPushAgentMessage(ctx, humanID, conv, msg)
	}()
}

func (a *API) pushAgentFallback(ctx context.Context, humanID uint64, conv *dm.DmConversation, text string) {
	msg, err := a.Agent.PostAssistantMessage(conv, humanID, text)
	if err != nil {
		a.Log.Error("agent fallback message", zap.Error(err))
		return
	}
	a.dmPushAgentMessage(ctx, humanID, conv, msg)
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

// dmTrimPreview and dmFormatMessage - dmFormatMessage needs role in output for frontend optional
