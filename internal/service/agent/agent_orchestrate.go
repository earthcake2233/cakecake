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

// AgentGenerationService owns reply orchestration, generation state and snapshots.
type AgentGenerationService struct {
	svc *AgentService
}

// RunReply generates and delivers an assistant message asynchronously. It owns
// the whole generation lifecycle: supersede, quota, persistence, push and
// follow-up suggestions. The handler only forwards the HTTP trigger.
func (g *AgentGenerationService) RunReply(humanID uint64, conv *dm.DmConversation, userText string) {
	if g.svc == nil || conv == nil {
		return
	}
	// Any new generation supersedes an in-flight/paused one: cancel it and
	// drop its state so its deltas are discarded and any paused-completed
	// reply waiting to be persisted is abandoned.
	g.svc.publishControl(context.Background(), humanID, map[string]interface{}{"type": "supersede", "from": g.svc.InstanceID})
	g.svc.supersedeGeneration(humanID)
	// Register the generation synchronously so a WS agent_cancel arriving
	// right after the send can never miss the in-flight generation.
	ctx, cancel := context.WithCancel(context.Background())
	traceID := generateTraceID()
	ctx = withTraceID(ctx, traceID)
	ctx = aigateway.WithUsageSink(ctx, &aigateway.UsageSink{
		OnUsage: func(u aigateway.Usage) { aigateway.RecordUserCost(humanID, u) },
	})
	genID := g.svc.beginGeneration(humanID, cancel)
	if genID == 0 {
		cancel()
		return
	}
	g.svc.snapshotRunning(humanID, genID, conv.ID)
	runMu := g.svc.runLock(humanID)
	go func() {
		defer cancel()
		runMu.Lock()
		defer runMu.Unlock()
		if !g.svc.CheckQuota(ctx, humanID) {
			g.svc.endGeneration(humanID, genID)
			g.svc.pushFallback(ctx, humanID, conv, "今日 AI 对话次数已达上限，请明天再试。")
			return
		}
		result, err := g.svc.GenerateReply(ctx, conv, userText)
		if err != nil {
			// Capture the stop decision before endGeneration releases the
			// cancel func: after that, ctx.Err() is always non-nil even for a
			// regular failure.
			stopped := errors.Is(err, context.Canceled) || ctx.Err() != nil
			g.svc.endGeneration(humanID, genID)
			if stopped {
				// User stopped generation; the frontend already cleared the
				// streamed draft. Do not push a confusing fallback message.
				return
			}
			if g.svc.Log != nil {
				g.svc.Log.Warn("agent generate", zap.String("trace_id", traceID), zap.Uint64("conv", conv.ID), zap.Error(err))
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
			g.svc.pushFallback(ctx, humanID, conv, msg)
			return
		}
		if g.svc.isGenerationPaused(humanID) {
			// Completed while paused: keep the state so a resume can flush the
			// buffer, then persist this reply exactly once.
			g.svc.storePendingReply(humanID, genID, conv, result)
			return
		}
		g.svc.endGeneration(humanID, genID)
		msg := g.persistAndPushReply(ctx, humanID, conv, result)
		if msg != nil && result != nil {
			go g.svc.attachSuggestions(humanID, msg.ID, result.Content)
		}
	}()
}

// ResumeReply resumes a paused generation: buffered deltas are flushed
// verbatim; if it completed while paused, the full reply is persisted now.
// Only when the generation fully ended with no reply does it fall back to a
// re-prompt continuation from the server-side draft (never from frontend text).
func (g *AgentGenerationService) ResumeReply(uid uint64, convID uint64) {
	if g.svc == nil || uid == 0 {
		return
	}
	// If the generation is owned by another replica, forward the control
	// command; the owner applies the exact same resume logic locally.
	if snap := g.svc.readSnapshot(uid); snap != nil && snap.Owner != "" && snap.Owner != g.svc.InstanceID {
		g.svc.publishControl(context.Background(), uid, map[string]interface{}{
			"type": "resume", "conv_id": convID, "from": g.svc.InstanceID,
		})
		return
	}
	g.svc.resumeReplyLocal(uid, convID)
}

// resumeReplyLocal is the single-instance resume implementation (owner path).
func (g *AgentGenerationService) resumeReplyLocal(uid uint64, convID uint64) {
	if g.svc == nil || uid == 0 {
		return
	}
	aigateway.IncAgentControl("continue")
	g.svc.resumeGeneration(uid)
	// Fast path: the generation is still running, so continue only needs to
	// un-pause and flush the buffered deltas.
	if g.svc.hasRunningGeneration(uid) {
		g.svc.pushContinueMode(uid, "buffer")
		return
	}
	// Serialize the completed-generation path per user: a double continue (or a
	// continue racing a fresh send) must never re-prompt after the reply row
	// already exists, otherwise the DB ends up with two assistant messages.
	runMu := g.svc.runLock(uid)
	runMu.Lock()
	defer runMu.Unlock()
	if g.svc.hasRunningGeneration(uid) {
		g.svc.pushContinueMode(uid, "buffer")
		return
	}
	conv, result, genID, ok := g.svc.takePendingReply(uid)
	if ok {
		g.svc.pushContinueMode(uid, "buffer")
		g.svc.endGeneration(uid, genID)
		msg := g.persistAndPushReply(context.Background(), uid, conv, result)
		if msg != nil && result != nil {
			go g.svc.attachSuggestions(uid, msg.ID, result.Content)
		}
		return
	}
	// The reply may have already completed (the stop arrived too late). Only
	// re-prompt when the latest user turn has no persisted assistant reply yet.
	draft := g.svc.draftText(uid)
	if strings.TrimSpace(draft) == "" || g.svc.latestUserTurnHasAssistantReply(uid, convID) {
		return
	}
	g.svc.pushContinueMode(uid, "reprompt")
	g.svc.continueReply(uid, convID, draft)
}

// handleControl applies a cross-instance control command. Only the replica
// that owns the user's running generation acts; everyone else ignores it.
func (g *AgentGenerationService) handleControl(uid uint64, ctrl map[string]interface{}) {
	if g.svc == nil || uid == 0 {
		return
	}
	// Ignore controls this instance published itself: the initiating path
	// already applied them locally (e.g. RunReply superseded before starting
	// the new generation; applying the echoed supersede would cancel it).
	if from, _ := ctrl["from"].(string); from != "" && from == g.svc.InstanceID {
		return
	}
	switch ctrl["type"] {
	case "pause":
		if g.svc.hasRunningGeneration(uid) {
			g.svc.pauseGeneration(uid)
		}
	case "resume":
		snap := g.svc.readSnapshot(uid)
		if snap == nil || snap.Owner != g.svc.InstanceID {
			return
		}
		convID := uint64(0)
		if v, ok := ctrl["conv_id"].(float64); ok {
			convID = uint64(v)
		}
		g.svc.resumeReplyLocal(uid, convID)
	case "supersede":
		g.svc.supersedeGeneration(uid)
	}
}

// RegenerateReply re-runs the assistant reply for the last user message in the
// conversation (posts a fresh assistant version).
func (g *AgentGenerationService) RegenerateReply(uid uint64, convID uint64) {
	if g.svc == nil || g.svc.Dm == nil || convID == 0 {
		return
	}
	aigateway.IncAgentControl("regenerate")
	ctx := context.Background()
	conv, err := g.svc.Dm.GetConversationByID(ctx, convID)
	if err != nil || conv == nil {
		return
	}
	part, err := g.svc.Dm.GetParticipant(ctx, convID, uid)
	if err != nil || part == nil {
		return
	}
	msgs, err := g.svc.Dm.ListMessages(ctx, convID, 0, 20)
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
		g.svc.regenerateWelcome(uid, conv, current)
		return
	}
	// Regenerate supersedes any in-flight generation (RunReply re-checks).
	g.svc.publishControl(context.Background(), uid, map[string]interface{}{"type": "supersede", "from": g.svc.InstanceID})
	g.svc.supersedeGeneration(uid)
	g.svc.RunReply(uid, conv, lastUser)
}

// regenerateWelcome posts a new opening welcome from the profile's welcome
// pool, preferring an entry different from the current one so the regenerated
// version is not an exact duplicate.
func (g *AgentGenerationService) regenerateWelcome(uid uint64, conv *dm.DmConversation, current string) {
	if g.svc == nil || conv == nil {
		return
	}
	traceID := generateTraceID()
	profile, err := g.svc.profileForConversation(conv)
	if err != nil || profile == nil {
		return
	}
	pool := agentmodel.ParseWelcomeMessages(profile.WelcomeMessagesJSON)
	welcome := pickDifferentWelcome(pool, current)
	if strings.TrimSpace(welcome) == "" {
		return
	}
	msg, err := g.svc.PostAssistantMessage(conv, uid, welcome)
	if err != nil {
		if g.svc.Log != nil {
			g.svc.Log.Error("agent persist regenerated welcome",
				zap.String("trace_id", traceID), zap.Error(err))
		}
		return
	}
	g.svc.pushAgentMessage(context.Background(), uid, conv, msg)
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
func (g *AgentGenerationService) continueReply(uid uint64, convID uint64, partial string) {
	if g.svc == nil || g.svc.Dm == nil || convID == 0 || strings.TrimSpace(partial) == "" {
		return
	}
	g.svc.publishControl(context.Background(), uid, map[string]interface{}{"type": "supersede", "from": g.svc.InstanceID})
	g.svc.supersedeGeneration(uid)
	ctx, cancel := context.WithCancel(context.Background())
	traceID := generateTraceID()
	ctx = withTraceID(ctx, traceID)
	ctx = aigateway.WithUsageSink(ctx, &aigateway.UsageSink{
		OnUsage: func(u aigateway.Usage) { aigateway.RecordUserCost(uid, u) },
	})
	genID := g.svc.beginGeneration(uid, cancel)
	if genID == 0 {
		cancel()
		return
	}
	g.svc.snapshotRunning(uid, genID, convID)
	runMu := g.svc.runLock(uid)
	go func() {
		defer cancel()
		defer g.svc.endGeneration(uid, genID)
		runMu.Lock()
		defer runMu.Unlock()
		conv, err := g.svc.Dm.GetConversationByID(ctx, convID)
		if err != nil || conv == nil {
			return
		}
		if _, err := g.svc.Dm.GetParticipant(ctx, convID, uid); err != nil {
			return
		}
		full, err := g.svc.continueFromDraft(ctx, conv, partial, genID)
		if err != nil {
			if g.svc.Log != nil {
				g.svc.Log.Warn("agent continue", zap.String("trace_id", traceID), zap.Uint64("conv", conv.ID), zap.Error(err))
			}
			// Keep the user's stopped draft intact: no fallback message, so the
			// frontend can retry or copy the partial.
			return
		}
		msg, err := g.svc.PostAssistantMessage(conv, uid, full)
		if err != nil {
			if g.svc.Log != nil {
				g.svc.Log.Error("agent persist continuation", zap.String("trace_id", traceID), zap.Error(err))
			}
			return
		}
		g.svc.pushAgentMessage(ctx, uid, conv, msg)
		go g.svc.attachSuggestions(uid, msg.ID, full)
	}()
}

// continueFromDraft re-prompts the model from the server-side draft (fallback
// only). It uses a NON-STREAMING completion so the continuation can be
// structurally stitched with the draft before anything reaches the client;
// the clean tail is then paced out as deltas. This removes the fuzzy
// text-overlap guessing that used to split code blocks at the seam.
func (g *AgentGenerationService) continueFromDraft(ctx context.Context, conv *dm.DmConversation, partial string, genID uint64) (string, error) {
	if g.svc == nil || g.svc.Gateway == nil || g.svc.Gateway.LLM == nil {
		return "", fmt.Errorf("ai assistant is not configured")
	}
	profile, err := g.svc.profileForConversation(conv)
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
	restore := g.svc.agentSystemPrompt(profile)
	defer restore()

	ctx2, cancel := context.WithTimeout(ctx, g.svc.agentReplyTimeout())
	defer cancel()

	instruction := "请从中断处直接继续你的回答，不要重复已经写过的内容，也不要另起一段重新讲解。"
	if partialEndsInsideCodeFence(partial) {
		instruction = "你现在正处于未闭合的代码块内部：请先接着写完这段代码（不要重复已写行，不要跳出代码块写新段落），用三个反引号闭合代码块后，如有必要再用一两句话继续讲解。"
	}
	msgs, err := g.svc.Gateway.BuildMessages(ctx2, conv.ID, "")
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
	replyText, err := g.svc.Gateway.LLM.Complete(ctx2, msgs)
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
	g.svc.Gateway.PersistHistory(ctx2, conv.ID, hist)

	g.svc.streamTail(humanID, genID, tail)
	return full, nil
}

// streamTail paces the stitched continuation out as deltas so the fallback
// still has a typewriter feel; the client never sees the unstitched seam.
func (g *AgentGenerationService) streamTail(humanID uint64, genID uint64, tail string) {
	if (g.svc.ChatHub == nil && g.svc.Relay == nil) || tail == "" {
		return
	}
	runes := []rune(tail)
	const chunk = 12
	for i := 0; i < len(runes); i += chunk {
		end := i + chunk
		if end > len(runes) {
			end = len(runes)
		}
		g.svc.deltaSender(humanID, genID)(string(runes[i:end]))
		time.Sleep(12 * time.Millisecond)
	}
}

// persistAndPushReply writes the finished reply to the DB and pushes it.
// traceCtx carries the generation trace id for logs (DB work uses a fresh
// background context so a canceled generation never blocks persistence).
func (g *AgentGenerationService) persistAndPushReply(traceCtx context.Context, humanID uint64, conv *dm.DmConversation, result *GenerateReplyResult) *dm.DmMessage {
	if g.svc == nil || conv == nil || result == nil {
		return nil
	}
	ctx := context.Background()
	traceID := traceIDFromContext(traceCtx)
	g.svc.IncrQuota(ctx, humanID)
	sugJSON := ""
	if len(result.Suggestions) > 0 {
		if b, e := json.Marshal(result.Suggestions); e == nil {
			sugJSON = string(b)
		}
	}
	msg, err := g.svc.PostAssistantMessage(conv, humanID, result.Content, string(result.ToolActivities), string(result.ToolResultData), sugJSON)
	if err != nil {
		if g.svc.Log != nil {
			g.svc.Log.Error("agent persist reply", zap.String("trace_id", traceID), zap.Error(err))
		}
		return nil
	}
	g.svc.pushAgentMessage(ctx, humanID, conv, msg)
	return msg
}

// pushAgentMessage formats a persisted agent message through the Pusher
// adapter and delivers every returned payload (locally or cross-instance).
func (g *AgentGenerationService) pushAgentMessage(ctx context.Context, humanID uint64, conv *dm.DmConversation, msg *dm.DmMessage) {
	if g.svc == nil || g.svc.Pusher == nil || humanID == 0 || conv == nil || msg == nil {
		return
	}
	payloads, err := g.svc.Pusher.FormatAgentMessage(ctx, humanID, conv, msg)
	if err != nil {
		if g.svc.Log != nil {
			g.svc.Log.Error("agent format message", zap.String("trace_id", traceIDFromContext(ctx)), zap.Error(err))
		}
		return
	}
	for _, payload := range payloads {
		g.svc.publishEvent(ctx, humanID, payload)
	}
}

// attachSuggestions generates follow-up chips after the reply is already
// persisted and pushed, so the UI never waits for the second LLM round trip.
func (g *AgentGenerationService) attachSuggestions(humanID uint64, messageID uint64, reply string) {
	if g.svc == nil || messageID == 0 || strings.TrimSpace(reply) == "" {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	traceID := generateTraceID()
	ctx = withTraceID(ctx, traceID)
	ctx = aigateway.WithUsageSink(ctx, &aigateway.UsageSink{
		OnUsage: func(u aigateway.Usage) { aigateway.RecordUserCost(humanID, u) },
	})
	sugg := g.svc.GenerateSuggestions(ctx, reply)
	if len(sugg) == 0 {
		return
	}
	if err := g.svc.UpdateMessageSuggestions(ctx, messageID, sugg); err != nil {
		if g.svc.Log != nil {
			g.svc.Log.Warn("agent attach suggestions",
				zap.String("trace_id", traceID),
				zap.Uint64("message_id", messageID), zap.Error(err))
		}
		return
	}
	g.svc.publishEvent(ctx, humanID, map[string]interface{}{
		"type":        "agent_suggestions",
		"message_id":  messageID,
		"suggestions": sugg,
	})
}

// pushFallback persists and pushes an assistant fallback message.
func (g *AgentGenerationService) pushFallback(ctx context.Context, humanID uint64, conv *dm.DmConversation, text string) {
	msg, err := g.svc.PostAssistantMessage(conv, humanID, text)
	if err != nil {
		if g.svc.Log != nil {
			g.svc.Log.Error("agent fallback message", zap.String("trace_id", traceIDFromContext(ctx)), zap.Error(err))
		}
		return
	}
	g.svc.pushAgentMessage(ctx, humanID, conv, msg)
}

// pushContinueMode tells the frontend whether a continue is a seamless buffer
// replay or a re-prompt fallback.
func (g *AgentGenerationService) pushContinueMode(uid uint64, mode string) {
	if g.svc == nil || uid == 0 {
		return
	}
	g.svc.publishEvent(context.Background(), uid, map[string]interface{}{
		"type": "agent_continue_mode",
		"mode": mode,
	})
}

// latestUserTurnHasAssistantReply reports whether the latest user message in
// the conversation already has a persisted assistant reply.
func (g *AgentGenerationService) latestUserTurnHasAssistantReply(uid uint64, convID uint64) bool {
	if g.svc == nil || g.svc.Dm == nil || convID == 0 {
		return false
	}
	// ListMessages returns newest-first (id DESC).
	msgs, err := g.svc.Dm.ListMessages(context.Background(), convID, 0, 20)
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
func (g *AgentGenerationService) runLock(uid uint64) *sync.Mutex {
	g.svc.runLocksMu.Lock()
	defer g.svc.runLocksMu.Unlock()
	if g.svc.runLocks == nil {
		g.svc.runLocks = make(map[uint64]*sync.Mutex)
	}
	mu := g.svc.runLocks[uid]
	if mu == nil {
		mu = &sync.Mutex{}
		g.svc.runLocks[uid] = mu
	}
	return mu
}
