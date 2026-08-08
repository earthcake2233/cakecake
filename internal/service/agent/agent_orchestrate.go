package agent

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"sync"
	"time"

	"cakecake/internal/aigateway"
	agentmodel "cakecake/internal/model/agent"
	"cakecake/internal/model/dm"
	"go.uber.org/zap"
)

// RunReply generates and delivers an assistant message asynchronously. It owns
// the whole generation lifecycle: supersede, quota, persistence, push and
// follow-up suggestions. The handler only forwards the HTTP trigger.
func (s *AgentService) RunReply(humanID uint64, conv *dm.DmConversation, userText string) {
	if s == nil || conv == nil {
		return
	}
	// Any new generation supersedes an in-flight/paused one: cancel it and
	// drop its state so its deltas are discarded and any paused-completed
	// reply waiting to be persisted is abandoned.
	s.publishControl(context.Background(), humanID, map[string]interface{}{"type": "supersede", "from": s.InstanceID})
	s.supersedeGeneration(humanID)
	// Register the generation synchronously so a WS agent_cancel arriving
	// right after the send can never miss the in-flight generation.
	ctx, cancel := context.WithCancel(context.Background())
	genID := s.beginGeneration(humanID, cancel)
	if genID == 0 {
		cancel()
		return
	}
	s.snapshotRunning(humanID, genID, conv.ID)
	runMu := s.runLock(humanID)
	go func() {
		defer cancel()
		runMu.Lock()
		defer runMu.Unlock()
		if !s.CheckQuota(ctx, humanID) {
			s.endGeneration(humanID, genID)
			s.pushFallback(ctx, humanID, conv, "今日 AI 对话次数已达上限，请明天再试。")
			return
		}
		result, err := s.GenerateReply(ctx, conv, userText)
		if err != nil {
			// Capture the stop decision before endGeneration releases the
			// cancel func: after that, ctx.Err() is always non-nil even for a
			// regular failure.
			stopped := errors.Is(err, context.Canceled) || ctx.Err() != nil
			s.endGeneration(humanID, genID)
			if stopped {
				// User stopped generation; the frontend already cleared the
				// streamed draft. Do not push a confusing fallback message.
				return
			}
			if s.Log != nil {
				s.Log.Warn("agent generate", zap.Uint64("conv", conv.ID), zap.Error(err))
			}
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
			s.pushFallback(ctx, humanID, conv, msg)
			return
		}
		if s.isGenerationPaused(humanID) {
			// Completed while paused: keep the state so a resume can flush the
			// buffer, then persist this reply exactly once.
			s.storePendingReply(humanID, genID, conv, result)
			return
		}
		s.endGeneration(humanID, genID)
		msg := s.persistAndPushReply(humanID, conv, result)
		if msg != nil && result != nil {
			go s.attachSuggestions(humanID, msg.ID, result.Content)
		}
	}()
}

// ResumeReply resumes a paused generation: buffered deltas are flushed
// verbatim; if it completed while paused, the full reply is persisted now.
// Only when the generation fully ended with no reply does it fall back to a
// re-prompt continuation from the server-side draft (never from frontend text).
func (s *AgentService) ResumeReply(uid uint64, convID uint64) {
	if s == nil || uid == 0 {
		return
	}
	// If the generation is owned by another replica, forward the control
	// command; the owner applies the exact same resume logic locally.
	if snap := s.readSnapshot(uid); snap != nil && snap.Owner != "" && snap.Owner != s.InstanceID {
		s.publishControl(context.Background(), uid, map[string]interface{}{
			"type": "resume", "conv_id": convID, "from": s.InstanceID,
		})
		return
	}
	s.resumeReplyLocal(uid, convID)
}

// resumeReplyLocal is the single-instance resume implementation (owner path).
func (s *AgentService) resumeReplyLocal(uid uint64, convID uint64) {
	if s == nil || uid == 0 {
		return
	}
	s.resumeGeneration(uid)
	// Fast path: the generation is still running, so continue only needs to
	// un-pause and flush the buffered deltas.
	if s.hasRunningGeneration(uid) {
		s.pushContinueMode(uid, "buffer")
		return
	}
	// Serialize the completed-generation path per user: a double continue (or a
	// continue racing a fresh send) must never re-prompt after the reply row
	// already exists, otherwise the DB ends up with two assistant messages.
	runMu := s.runLock(uid)
	runMu.Lock()
	defer runMu.Unlock()
	if s.hasRunningGeneration(uid) {
		s.pushContinueMode(uid, "buffer")
		return
	}
	conv, result, genID, ok := s.takePendingReply(uid)
	if ok {
		s.pushContinueMode(uid, "buffer")
		s.endGeneration(uid, genID)
		msg := s.persistAndPushReply(uid, conv, result)
		if msg != nil && result != nil {
			go s.attachSuggestions(uid, msg.ID, result.Content)
		}
		return
	}
	// The reply may have already completed (the stop arrived too late). Only
	// re-prompt when the latest user turn has no persisted assistant reply yet.
	draft := s.draftText(uid)
	if strings.TrimSpace(draft) == "" || s.latestUserTurnHasAssistantReply(uid, convID) {
		return
	}
	s.pushContinueMode(uid, "reprompt")
	s.continueReply(uid, convID, draft)
}

// handleControl applies a cross-instance control command. Only the replica
// that owns the user's running generation acts; everyone else ignores it.
func (s *AgentService) handleControl(uid uint64, ctrl map[string]interface{}) {
	if s == nil || uid == 0 {
		return
	}
	// Ignore controls this instance published itself: the initiating path
	// already applied them locally (e.g. RunReply superseded before starting
	// the new generation; applying the echoed supersede would cancel it).
	if from, _ := ctrl["from"].(string); from != "" && from == s.InstanceID {
		return
	}
	switch ctrl["type"] {
	case "pause":
		if s.hasRunningGeneration(uid) {
			s.pauseGeneration(uid)
		}
	case "resume":
		snap := s.readSnapshot(uid)
		if snap == nil || snap.Owner != s.InstanceID {
			return
		}
		convID := uint64(0)
		if v, ok := ctrl["conv_id"].(float64); ok {
			convID = uint64(v)
		}
		s.resumeReplyLocal(uid, convID)
	case "supersede":
		s.supersedeGeneration(uid)
	}
}

// RegenerateReply re-runs the assistant reply for the last user message in the
// conversation (posts a fresh assistant version).
func (s *AgentService) RegenerateReply(uid uint64, convID uint64) {
	if s == nil || s.Dm == nil || convID == 0 {
		return
	}
	ctx := context.Background()
	conv, err := s.Dm.GetConversationByID(ctx, convID)
	if err != nil || conv == nil {
		return
	}
	part, err := s.Dm.GetParticipant(ctx, convID, uid)
	if err != nil || part == nil {
		return
	}
	msgs, err := s.Dm.ListMessages(ctx, convID, 0, 20)
	if err != nil {
		return
	}
	lastUser := lastUserText(msgs)
	if lastUser == "" {
		// The conversation only contains the opening welcome (no user message
		// yet): regenerate it by posting a different welcome from the pool.
		// Returning silently here used to leave the frontend stuck in
		// "正在重新生成…" forever.
		current := ""
		for i := 0; i < len(msgs); i++ {
			if msgs[i].Role == "assistant" {
				current = strings.TrimSpace(msgs[i].Content)
				break
			}
		}
		s.regenerateWelcome(uid, conv, current)
		return
	}
	// Regenerate supersedes any in-flight generation (RunReply re-checks).
	s.publishControl(context.Background(), uid, map[string]interface{}{"type": "supersede", "from": s.InstanceID})
	s.supersedeGeneration(uid)
	s.RunReply(uid, conv, lastUser)
}

// regenerateWelcome posts a new opening welcome from the profile's welcome
// pool, preferring an entry different from the current one so the regenerated
// version is not an exact duplicate.
func (s *AgentService) regenerateWelcome(uid uint64, conv *dm.DmConversation, current string) {
	if s == nil || conv == nil {
		return
	}
	profile, err := s.profileForConversation(conv)
	if err != nil || profile == nil {
		return
	}
	pool := agentmodel.ParseWelcomeMessages(profile.WelcomeMessagesJSON)
	welcome := pickDifferentWelcome(pool, current)
	if strings.TrimSpace(welcome) == "" {
		return
	}
	msg, err := s.PostAssistantMessage(conv, uid, welcome)
	if err != nil {
		if s.Log != nil {
			s.Log.Error("agent persist regenerated welcome", zap.Error(err))
		}
		return
	}
	s.pushAgentMessage(context.Background(), uid, conv, msg)
}

// pickDifferentWelcome picks a random pool entry different from current,
// falling back to the first entry when there is no alternative.
func pickDifferentWelcome(pool []string, current string) string {
	if len(pool) == 0 {
		return ""
	}
	var alt []string
	for _, w := range pool {
		if strings.TrimSpace(w) != current {
			alt = append(alt, w)
		}
	}
	if len(alt) == 0 {
		alt = pool
	}
	if len(alt) == 1 {
		return alt[0]
	}
	n, err := rand.Int(rand.Reader, big.NewInt(int64(len(alt))))
	if err != nil {
		return alt[0]
	}
	return alt[n.Int64()]
}

// lastUserText returns the content of the newest user message in a
// newest-first message list. The list is id DESC, so the first "user" row is
// the latest question; scanning from the tail would return the OLDEST one and
// regenerate the wrong turn.
func lastUserText(msgs []dm.DmMessage) string {
	for i := 0; i < len(msgs); i++ {
		if msgs[i].Role == "user" {
			return strings.TrimSpace(msgs[i].Content)
		}
	}
	return ""
}

// continueReply re-prompts the model from the server-side draft text.
func (s *AgentService) continueReply(uid uint64, convID uint64, partial string) {
	if s == nil || s.Dm == nil || convID == 0 || strings.TrimSpace(partial) == "" {
		return
	}
	s.publishControl(context.Background(), uid, map[string]interface{}{"type": "supersede", "from": s.InstanceID})
	s.supersedeGeneration(uid)
	ctx, cancel := context.WithCancel(context.Background())
	genID := s.beginGeneration(uid, cancel)
	if genID == 0 {
		cancel()
		return
	}
	s.snapshotRunning(uid, genID, convID)
	runMu := s.runLock(uid)
	go func() {
		defer cancel()
		defer s.endGeneration(uid, genID)
		runMu.Lock()
		defer runMu.Unlock()
		conv, err := s.Dm.GetConversationByID(ctx, convID)
		if err != nil || conv == nil {
			return
		}
		if _, err := s.Dm.GetParticipant(ctx, convID, uid); err != nil {
			return
		}
		full, err := s.continueFromDraft(ctx, conv, partial, genID)
		if err != nil {
			if s.Log != nil {
				s.Log.Warn("agent continue", zap.Uint64("conv", conv.ID), zap.Error(err))
			}
			// Keep the user's stopped draft intact: no fallback message, so the
			// frontend can retry or copy the partial.
			return
		}
		msg, err := s.PostAssistantMessage(conv, uid, full)
		if err != nil {
			if s.Log != nil {
				s.Log.Error("agent persist continuation", zap.Error(err))
			}
			return
		}
		s.pushAgentMessage(ctx, uid, conv, msg)
		go s.attachSuggestions(uid, msg.ID, full)
	}()
}

// continueFromDraft re-prompts the model from the server-side draft (fallback
// only). It uses a NON-STREAMING completion so the continuation can be
// structurally stitched with the draft before anything reaches the client;
// the clean tail is then paced out as deltas. This removes the fuzzy
// text-overlap guessing that used to split code blocks at the seam.
func (s *AgentService) continueFromDraft(ctx context.Context, conv *dm.DmConversation, partial string, genID uint64) (string, error) {
	if s == nil || s.Gateway == nil || s.Gateway.LLM == nil {
		return "", fmt.Errorf("ai assistant is not configured")
	}
	profile, err := s.profileForConversation(conv)
	if err != nil {
		return "", fmt.Errorf("ai assistant profile missing")
	}
	if !profile.Enabled {
		return "", fmt.Errorf("ai assistant is disabled")
	}
	humanID := humanPeerForConversation(conv, profile.BotUserID)
	if humanID == 0 {
		return "", fmt.Errorf("agent conversation has no human participant")
	}
	restore := s.agentSystemPrompt(profile)
	defer restore()

	ctx2, cancel := context.WithTimeout(ctx, s.agentReplyTimeout())
	defer cancel()

	instruction := "请从中断处直接继续你的回答，不要重复已经写过的内容，也不要另起一段重新讲解。"
	if partialEndsInsideCodeFence(partial) {
		instruction = "你现在正处于未闭合的代码块内部：请先接着写完这段代码（不要重复已写行，不要跳出代码块写新段落），用三个反引号闭合代码块后，如有必要再用一两句话继续讲解。"
	}
	msgs, err := s.Gateway.BuildMessages(ctx2, conv.ID, "")
	if err != nil {
		return "", err
	}
	// Drop the trailing empty user message BuildMessages appended.
	if len(msgs) > 0 && msgs[len(msgs)-1].Role == "user" &&
		strings.TrimSpace(msgs[len(msgs)-1].Content) == "" {
		msgs = msgs[:len(msgs)-1]
	}
	r := []rune(partial)
	if len(r) > 3000 {
		partial = string(r[:3000])
	}
	partial = strings.TrimSpace(partial)
	msgs = append(msgs,
		aigateway.ChatMessage{Role: "assistant", Content: partial},
		aigateway.ChatMessage{Role: "user", Content: instruction},
	)
	replyText, err := s.Gateway.LLM.Complete(ctx2, msgs)
	if err != nil {
		return "", err
	}
	continuation := strings.TrimSpace(replyText)
	if continuation == "" {
		return "", fmt.Errorf("empty model reply")
	}
	full, tail := stitchContinuation(partial, continuation)
	// Persist prior turns + the stitched full reply, dropping the transient
	// partial/instruction messages from the stored history.
	hist := append([]aigateway.ChatMessage{}, msgs[:len(msgs)-2]...)
	hist = append(hist, aigateway.ChatMessage{Role: "assistant", Content: full})
	s.Gateway.PersistHistory(ctx2, conv.ID, hist)

	s.streamTail(humanID, genID, tail)
	return full, nil
}

// streamTail paces the stitched continuation out as deltas so the fallback
// still has a typewriter feel; the client never sees the unstitched seam.
func (s *AgentService) streamTail(humanID uint64, genID uint64, tail string) {
	if (s.ChatHub == nil && s.Relay == nil) || tail == "" {
		return
	}
	runes := []rune(tail)
	const chunk = 12
	for i := 0; i < len(runes); i += chunk {
		end := i + chunk
		if end > len(runes) {
			end = len(runes)
		}
		s.deltaSender(humanID, genID)(string(runes[i:end]))
		time.Sleep(12 * time.Millisecond)
	}
}

// persistAndPushReply writes the finished reply to the DB and pushes it.
func (s *AgentService) persistAndPushReply(humanID uint64, conv *dm.DmConversation, result *GenerateReplyResult) *dm.DmMessage {
	if s == nil || conv == nil || result == nil {
		return nil
	}
	ctx := context.Background()
	s.IncrQuota(ctx, humanID)
	sugJSON := ""
	if len(result.Suggestions) > 0 {
		if b, e := json.Marshal(result.Suggestions); e == nil {
			sugJSON = string(b)
		}
	}
	msg, err := s.PostAssistantMessage(conv, humanID, result.Content, string(result.ToolActivities), string(result.ToolResultData), sugJSON)
	if err != nil {
		if s.Log != nil {
			s.Log.Error("agent persist reply", zap.Error(err))
		}
		return nil
	}
	s.pushAgentMessage(ctx, humanID, conv, msg)
	return msg
}

// pushAgentMessage formats a persisted agent message through the Pusher
// adapter and delivers every returned payload (locally or cross-instance).
func (s *AgentService) pushAgentMessage(ctx context.Context, humanID uint64, conv *dm.DmConversation, msg *dm.DmMessage) {
	if s == nil || s.Pusher == nil || humanID == 0 || conv == nil || msg == nil {
		return
	}
	payloads, err := s.Pusher.FormatAgentMessage(ctx, humanID, conv, msg)
	if err != nil {
		if s.Log != nil {
			s.Log.Error("agent format message", zap.Error(err))
		}
		return
	}
	for _, payload := range payloads {
		s.publishEvent(ctx, humanID, payload)
	}
}

// attachSuggestions generates follow-up chips after the reply is already
// persisted and pushed, so the UI never waits for the second LLM round trip.
func (s *AgentService) attachSuggestions(humanID uint64, messageID uint64, reply string) {
	if s == nil || messageID == 0 || strings.TrimSpace(reply) == "" {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	sugg := s.GenerateSuggestions(ctx, reply)
	if len(sugg) == 0 {
		return
	}
	if err := s.UpdateMessageSuggestions(ctx, messageID, sugg); err != nil {
		if s.Log != nil {
			s.Log.Warn("agent attach suggestions", zap.Uint64("message_id", messageID), zap.Error(err))
		}
		return
	}
	s.publishEvent(ctx, humanID, map[string]interface{}{
		"type":        "agent_suggestions",
		"message_id":  messageID,
		"suggestions": sugg,
	})
}

// pushFallback persists and pushes an assistant fallback message.
func (s *AgentService) pushFallback(ctx context.Context, humanID uint64, conv *dm.DmConversation, text string) {
	msg, err := s.PostAssistantMessage(conv, humanID, text)
	if err != nil {
		if s.Log != nil {
			s.Log.Error("agent fallback message", zap.Error(err))
		}
		return
	}
	s.pushAgentMessage(ctx, humanID, conv, msg)
}

// pushContinueMode tells the frontend whether a continue is a seamless buffer
// replay or a re-prompt fallback.
func (s *AgentService) pushContinueMode(uid uint64, mode string) {
	if s == nil || uid == 0 {
		return
	}
	s.publishEvent(context.Background(), uid, map[string]interface{}{
		"type": "agent_continue_mode",
		"mode": mode,
	})
}

// latestUserTurnHasAssistantReply reports whether the latest user message in
// the conversation already has a persisted assistant reply.
func (s *AgentService) latestUserTurnHasAssistantReply(uid uint64, convID uint64) bool {
	if s == nil || s.Dm == nil || convID == 0 {
		return false
	}
	// ListMessages returns newest-first (id DESC).
	msgs, err := s.Dm.ListMessages(context.Background(), convID, 0, 20)
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

// runLock returns the per-user serialization lock: at most one agent
// generation goroutine runs per user at any moment.
func (s *AgentService) runLock(uid uint64) *sync.Mutex {
	s.runLocksMu.Lock()
	defer s.runLocksMu.Unlock()
	if s.runLocks == nil {
		s.runLocks = make(map[uint64]*sync.Mutex)
	}
	mu := s.runLocks[uid]
	if mu == nil {
		mu = &sync.Mutex{}
		s.runLocks[uid] = mu
	}
	return mu
}
