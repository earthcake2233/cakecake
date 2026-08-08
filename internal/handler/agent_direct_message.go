package handler

import (
	"cakecake/internal/model/dm"
	serviceagent "cakecake/internal/service/agent"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"time"

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
	// Any new generation supersedes an in-flight/paused one: the service
	// cancels it and drops its state so its deltas are discarded and any
	// paused-completed reply waiting to be persisted is abandoned.
	a.Agent.SupersedeGeneration(humanID)
	// Register the generation synchronously so a WS agent_cancel arriving
	// right after the send can never miss the in-flight generation.
	ctx, cancel := context.WithCancel(context.Background())
	genID := a.Agent.BeginGeneration(humanID, cancel)
	if genID == 0 {
		cancel()
		return
	}
	runMu := a.agentRunLock(humanID)
	go func() {
		defer cancel()
		runMu.Lock()
		defer runMu.Unlock()
		if !a.Agent.CheckQuota(ctx, humanID) {
			a.Agent.EndGeneration(humanID, genID)
			a.pushAgentFallback(ctx, humanID, conv, "今日 AI 对话次数已达上限，请明天再试。")
			return
		}
		result, err := a.Agent.GenerateReply(ctx, conv, userText)
		if err != nil {
			// Capture the stop decision before EndGeneration releases the
			// cancel func: after that, ctx.Err() is always non-nil even for a
			// regular failure.
			stopped := errors.Is(err, context.Canceled) || ctx.Err() != nil
			a.Agent.EndGeneration(humanID, genID)
			if stopped {
				// User stopped generation; the frontend already cleared the
				// streamed draft. Do not push a confusing fallback message.
				return
			}
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
		if a.Agent.IsGenerationPaused(humanID) {
			// Completed while paused: keep the state so a resume can flush the
			// buffer, then persist this reply exactly once.
			a.Agent.StorePendingReply(humanID, genID, conv, result)
			return
		}
		a.Agent.EndGeneration(humanID, genID)
		msg := a.persistAndPushAgentReply(humanID, conv, result)
		if msg != nil && result != nil {
			go a.attachAgentSuggestions(humanID, msg.ID, result.Content)
		}
	}()
}

// agentRunLock returns the per-user serialization lock: at most one agent
// generation goroutine runs per user at any moment.
func (a *API) agentRunLock(uid uint64) *sync.Mutex {
	a.agentRunMu.Lock()
	defer a.agentRunMu.Unlock()
	if a.agentRunLocks == nil {
		a.agentRunLocks = make(map[uint64]*sync.Mutex)
	}
	mu := a.agentRunLocks[uid]
	if mu == nil {
		mu = &sync.Mutex{}
		a.agentRunLocks[uid] = mu
	}
	return mu
}

// persistAndPushAgentReply writes the finished reply to the DB and pushes it.
func (a *API) persistAndPushAgentReply(humanID uint64, conv *dm.DmConversation, result *serviceagent.GenerateReplyResult) *dm.DmMessage {
	if a.Agent == nil || conv == nil || result == nil {
		return nil
	}
	ctx := context.Background()
	a.Agent.IncrQuota(ctx, humanID)
	sugJSON := ""
	if len(result.Suggestions) > 0 {
		if b, e := json.Marshal(result.Suggestions); e == nil {
			sugJSON = string(b)
		}
	}
	msg, err := a.Agent.PostAssistantMessage(conv, humanID, result.Content, string(result.ToolActivities), string(result.ToolResultData), sugJSON)
	if err != nil {
		a.Log.Error("agent persist reply", zap.Error(err))
		return nil
	}
	a.dmPushAgentMessage(ctx, humanID, conv, msg)
	return msg
}

// pauseAgentReply pauses the user's in-flight generation: the LLM stream keeps
// running in the background and deltas are buffered for byte-level resume.
func (a *API) pauseAgentReply(uid uint64) {
	if uid == 0 || a.Agent == nil {
		return
	}
	a.Agent.PauseGeneration(uid)
}

// resumeAgentReply resumes a paused generation: buffered deltas are flushed
// verbatim; if it completed while paused, the full reply is persisted now.
// Only when the generation fully ended with no reply does it fall back to a
// re-prompt continuation from the server-side draft (never from frontend text).
func (a *API) resumeAgentReply(uid uint64, convID uint64) {
	if a.Agent == nil || uid == 0 {
		return
	}
	a.Agent.ResumeGeneration(uid)
	// Fast path: the generation is still running, so continue only needs to
	// un-pause and flush the buffered deltas.
	if a.Agent.HasRunningGeneration(uid) {
		a.pushAgentContinueMode(uid, "buffer")
		return
	}
	// Serialize the completed-generation path per user: a double continue (or a
	// continue racing a fresh send) must never re-prompt after the reply row
	// already exists, otherwise the DB ends up with two assistant messages.
	runMu := a.agentRunLock(uid)
	runMu.Lock()
	defer runMu.Unlock()
	if a.Agent.HasRunningGeneration(uid) {
		a.pushAgentContinueMode(uid, "buffer")
		return
	}
	conv, result, genID, ok := a.Agent.TakePendingReply(uid)
	if ok {
		a.pushAgentContinueMode(uid, "buffer")
		a.Agent.EndGeneration(uid, genID)
		msg := a.persistAndPushAgentReply(uid, conv, result)
		if msg != nil && result != nil {
			go a.attachAgentSuggestions(uid, msg.ID, result.Content)
		}
		return
	}
	// The reply may have already completed (the stop arrived too late). Only
	// re-prompt when the latest user turn has no persisted assistant reply yet.
	draft := a.Agent.DraftText(uid)
	if strings.TrimSpace(draft) == "" || a.latestUserTurnHasAssistantReply(uid, convID) {
		return
	}
	a.pushAgentContinueMode(uid, "reprompt")
	a.continueAgentReply(uid, convID, draft)
}

// pushAgentContinueMode tells the frontend whether a continue is a seamless
// buffer replay or a re-prompt fallback, so the UI can label the stream and
// keep its draft state accordingly.
func (a *API) pushAgentContinueMode(uid uint64, mode string) {
	if uid == 0 {
		return
	}
	a.dmPushEvent(uid, map[string]interface{}{
		"type": "agent_continue_mode",
		"mode": mode,
	})
}

// attachAgentSuggestions generates follow-up chips after the reply is already
// persisted and pushed, so the UI never waits for the second LLM round trip.
func (a *API) attachAgentSuggestions(humanID uint64, messageID uint64, reply string) {
	if a.Agent == nil || messageID == 0 || strings.TrimSpace(reply) == "" {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	sugg := a.Agent.GenerateSuggestions(ctx, reply)
	if len(sugg) == 0 {
		return
	}
	if err := a.Agent.UpdateMessageSuggestions(ctx, messageID, sugg); err != nil {
		a.Log.Warn("agent attach suggestions", zap.Uint64("message_id", messageID), zap.Error(err))
		return
	}
	a.dmPushEvent(humanID, map[string]interface{}{
		"type":        "agent_suggestions",
		"message_id":  messageID,
		"suggestions": sugg,
	})
}

// latestUserTurnHasAssistantReply reports whether the latest user message in
// the conversation already has a persisted assistant reply. This is the
// reliable no-op condition for a late continue: once the reply row exists, a
// re-prompt would only create a duplicate bubble.
func (a *API) latestUserTurnHasAssistantReply(uid uint64, convID uint64) bool {
	if a.DmSvc == nil || convID == 0 {
		return false
	}
	// ListMessages returns newest-first (id DESC).
	msgs, err := a.DmSvc.ListMessages(context.Background(), convID, 0, 20)
	if err != nil {
		return false
	}
	for _, m := range msgs {
		if m.Role == "assistant" {
			return true
		}
		if m.Role == "user" {
			return false
		}
	}
	return false
}

// regenerateAgentReply re-runs the assistant reply for the last user message in
// the conversation (posts a fresh assistant message).
func (a *API) regenerateAgentReply(uid uint64, convID uint64) {
	if a.Agent == nil || a.DmSvc == nil || convID == 0 {
		return
	}
	ctx := context.Background()
	conv, err := a.DmSvc.GetConversationByID(ctx, convID)
	if err != nil || conv == nil {
		return
	}
	part, err := a.DmSvc.GetParticipant(ctx, convID, uid)
	if err != nil || part == nil {
		return
	}
	msgs, err := a.DmSvc.ListMessages(ctx, convID, 0, 20)
	if err != nil {
		return
	}
	lastUser := ""
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i].Role == "user" {
			lastUser = strings.TrimSpace(msgs[i].Content)
			break
		}
	}
	if lastUser == "" {
		return
	}
	// Regenerate supersedes any in-flight generation (runAgentReply re-checks).
	a.Agent.SupersedeGeneration(uid)
	a.runAgentReply(uid, conv, lastUser)
}

// continueAgentReply re-prompts the model from the server-side draft text.
func (a *API) continueAgentReply(uid uint64, convID uint64, partial string) {
	if a.Agent == nil || a.DmSvc == nil || convID == 0 || strings.TrimSpace(partial) == "" {
		return
	}
	a.Agent.SupersedeGeneration(uid)
	ctx, cancel := context.WithCancel(context.Background())
	genID := a.Agent.BeginGeneration(uid, cancel)
	if genID == 0 {
		cancel()
		return
	}
	runMu := a.agentRunLock(uid)
	go func() {
		defer cancel()
		defer a.Agent.EndGeneration(uid, genID)
		runMu.Lock()
		defer runMu.Unlock()
		conv, err := a.DmSvc.GetConversationByID(ctx, convID)
		if err != nil || conv == nil {
			return
		}
		part, err := a.DmSvc.GetParticipant(ctx, convID, uid)
		if err != nil || part == nil {
			return
		}
		continuation, suggestions, err := a.Agent.ContinueReplyStream(ctx, conv, partial)
		if err != nil {
			a.Log.Warn("agent continue", zap.Uint64("conv", conv.ID), zap.Error(err))
			// Keep the user's stopped draft intact: no fallback message, so the
			// frontend can retry or copy the partial.
			return
		}
		full := continuation
		sugJSON := ""
		if len(suggestions) > 0 {
			if b, e := json.Marshal(suggestions); e == nil {
				sugJSON = string(b)
			}
		}
		msg, err := a.Agent.PostAssistantMessage(conv, uid, full, "", "", sugJSON)
		if err != nil {
			a.Log.Error("agent persist continuation", zap.Error(err))
			return
		}
		a.dmPushAgentMessage(ctx, uid, conv, msg)
		go a.attachAgentSuggestions(uid, msg.ID, full)
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
