package handler

import (
	"context"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"minibili/internal/model"
)

func (a *API) dmIsAgentConv(conv *model.DmConversation) bool {
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
func (a *API) runAgentReply(humanID uint64, conv *model.DmConversation, userText string) {
	if a.Agent == nil || conv == nil {
		return
	}
	go func() {
		ctx := context.Background()
		if !a.Agent.CheckQuota(ctx, humanID) {
			a.pushAgentFallback(humanID, conv, "今日 AI 对话次数已达上限，请明天再试。")
			return
		}
		reply, err := a.Agent.GenerateReply(ctx, conv, userText)
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
			a.pushAgentFallback(humanID, conv, msg)
			return
		}
		a.Agent.IncrQuota(ctx, humanID)
		msg, err := a.Agent.PostAssistantMessage(conv, humanID, reply)
		if err != nil {
			a.Log.Error("agent persist reply", zap.Error(err))
			return
		}
		a.dmPushAgentMessage(humanID, conv, msg)
	}()
}

func (a *API) pushAgentFallback(humanID uint64, conv *model.DmConversation, text string) {
	msg, err := a.Agent.PostAssistantMessage(conv, humanID, text)
	if err != nil {
		a.Log.Error("agent fallback message", zap.Error(err))
		return
	}
	a.dmPushAgentMessage(humanID, conv, msg)
}

func (a *API) dmPushAgentMessage(humanID uint64, conv *model.DmConversation, msg *model.DmMessage) {
	if msg == nil || conv == nil {
		return
	}
	senderName, senderAvatar := a.dmUserBrief(a.DB, msg.SenderID)
	out := a.dmFormatMessage(msg, senderName, senderAvatar)
	var part model.DmParticipant
	_ = a.DB.Where("conversation_id = ? AND user_id = ?", conv.ID, humanID).First(&part).Error
	convPayload := a.dmFormatConversation(conv, humanID, &part)
	event := gin.H{"type": "dm_message", "message": out}
	if !part.Muted {
		a.dmPushEvent(humanID, event)
	}
	a.dmPushEvent(humanID, gin.H{"type": "dm_conversation", "conversation": convPayload})
}

// dmTrimPreview and dmFormatMessage - dmFormatMessage needs role in output for frontend optional
