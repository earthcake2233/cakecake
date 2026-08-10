package agent

import (
	"cakecake/internal/aigateway"
	"cakecake/internal/model/dm"
	"context"
	"strings"
	"sync"
	"time"
)

// agentGenState supports byte-level pause/resume of a running generation:
// while paused, streamed deltas are buffered instead of pushed, then flushed
// verbatim on resume (the same LLM stream keeps running).
type agentGenState struct {
	mu      sync.Mutex
	genID   uint64
	cancel  context.CancelFunc
	running bool
	pending *pendingAgentReply

	draft   []string
	paused  bool
	buffer  []string
	dropped bool
	// pauseSeq increments on every stop; a resume only clears the paused flag
	// if no new stop happened while it was replaying the backlog.
	pauseSeq uint64
	// resuming guards the backlog replay so two continues never run it
	// concurrently (which would let live deltas interleave with the replay).
	resuming bool
	// cond wakes concurrent resumeGeneration callers instead of busy-spinning
	// while another continue is replaying the backlog.
	cond *sync.Cond
}

// pendingAgentReply is a reply that finished while the user had paused the
// stream; it is kept until a resume persists it exactly once.
type pendingAgentReply struct {
	conv   *dm.DmConversation
	result *GenerateReplyResult
}

// deltaSender returns a per-generation stream callback that routes deltas to
// the human user's ChatHub connection and honors pause/drop generation state.
// Each LLM call gets its own closure (capturing its own genID), so concurrent
// users and superseded generations never cross-wire or leak late deltas.
func (g *AgentGenerationService) deltaSender(humanID uint64, genID uint64) func(string) {
	return func(delta string) {
		if delta == "" {
			return
		}
		if st := g.svc.generationState(humanID); st != nil {
			st.mu.Lock()
			if st.dropped || st.genID != genID {
				st.mu.Unlock()
				return
			}
			// Keep the full draft server-side so a re-prompt fallback never
			// depends on text echoed back from the frontend.
			st.draft = append(st.draft, delta)
			if st.paused {
				st.buffer = append(st.buffer, delta)
				st.mu.Unlock()
				return
			}
			st.mu.Unlock()
		}
		if g.svc.ChatHub != nil || g.svc.Relay != nil {
			g.svc.publishEvent(context.Background(), humanID, map[string]interface{}{
				"type": "agent_delta",
				"body": map[string]interface{}{
					"content": delta,
				},
			})
		}
	}
}

// publishEvent delivers an agent event to the user's WebSocket connection.
// With a relay wired it is published to Redis and every replica fans it out to
// its local ChatHub; without one it is pushed directly (single-process mode).
func (g *AgentGenerationService) publishEvent(ctx context.Context, uid uint64, payload interface{}) {
	if g.svc == nil || uid == 0 {
		return
	}
	if g.svc.Relay != nil {
		_ = g.svc.Relay.PublishEvent(ctx, uid, payload)
	} else if g.svc.ChatHub != nil {
		g.svc.ChatHub.PushJSON(uid, payload)
	}
	if g.svc.EventHook != nil {
		if m, ok := payload.(map[string]interface{}); ok {
			g.svc.EventHook(uid, m)
		}
	}
}

// publishControl routes a cross-instance generation control command to the
// owner (no-op without a relay).
func (g *AgentGenerationService) publishControl(ctx context.Context, uid uint64, payload interface{}) {
	if g.svc == nil || uid == 0 || g.svc.Relay == nil {
		return
	}
	_ = g.svc.Relay.PublishControl(ctx, uid, payload)
}

// draftText returns the full text streamed so far for the user's generation.
// It survives endGeneration (copied to the sticky last-draft slot) so a
// re-prompt fallback can continue from the exact server-side draft.
func (g *AgentGenerationService) draftText(uid uint64) string {
	if uid == 0 {
		return ""
	}
	if st := g.svc.generationState(uid); st != nil {
		st.mu.Lock()
		defer st.mu.Unlock()
		return strings.Join(st.draft, "")
	}
	g.svc.draftMu.Lock()
	defer g.svc.draftMu.Unlock()
	return g.svc.lastDraft[uid]
}

// currentGenID returns the generation id registered for the user, or 0.
func (g *AgentGenerationService) currentGenID(uid uint64) uint64 {
	st := g.svc.generationState(uid)
	if st == nil {
		return 0
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	return st.genID
}

func (g *AgentGenerationService) generationState(uid uint64) *agentGenState {
	if uid == 0 {
		return nil
	}
	g.svc.genMu.Lock()
	defer g.svc.genMu.Unlock()
	if g.svc.genStates == nil {
		return nil
	}
	return g.svc.genStates[uid]
}

// beginGeneration registers a new generation state before the LLM call and
// returns its generation id. The cancel func is owned by the state: it is
// invoked when the generation is superseded, ended, or finished while paused,
// and supersedes any previous generation state for the user.
func (g *AgentGenerationService) beginGeneration(uid uint64, cancel context.CancelFunc) uint64 {
	if uid == 0 {
		return 0
	}
	g.svc.genMu.Lock()
	if g.svc.genStates == nil {
		g.svc.genStates = make(map[uint64]*agentGenState)
	}
	g.svc.genSeq++
	old := g.svc.genStates[uid]
	g.svc.genStates[uid] = &agentGenState{genID: g.svc.genSeq, cancel: cancel, running: true}
	g.svc.genMu.Unlock()
	if old != nil {
		if oldCancel := old.finish(); oldCancel != nil {
			oldCancel()
		}
	}
	g.svc.clearDraft(uid)
	return g.svc.genSeq
}

// endGeneration removes the generation state only if it still belongs to the
// given generation id (a finished goroutine can never clear a newer state).
// The attached cancel is invoked so request resources are released.
func (g *AgentGenerationService) endGeneration(uid uint64, genID uint64) {
	if uid == 0 {
		return
	}
	g.svc.genMu.Lock()
	st, ok := g.svc.genStates[uid]
	if !ok || st.genID != genID {
		g.svc.genMu.Unlock()
		return
	}
	delete(g.svc.genStates, uid)
	g.svc.genMu.Unlock()
	g.svc.clearSnapshot(uid, genID)
	g.svc.saveDraft(uid, st)
	if cancel := st.finish(); cancel != nil {
		cancel()
	}
}

// finish marks the state as no longer running and releases its cancel func,
// dropping any buffered/pending data that should never be delivered.
func (st *agentGenState) finish() context.CancelFunc {
	if st == nil {
		return nil
	}
	st.mu.Lock()
	st.running = false
	st.pending = nil
	st.draft = nil
	st.buffer = nil
	if st.cond != nil {
		st.cond.Broadcast()
	}
	cancel := st.cancel
	st.cancel = nil
	st.mu.Unlock()
	return cancel
}

// ensureCond lazily creates the resume wake-up condition under st.mu.
func (st *agentGenState) ensureCond() {
	if st.cond == nil {
		st.cond = sync.NewCond(&st.mu)
	}
}

// saveDraft copies the generation's accumulated draft into the sticky
// last-draft slot so a re-prompt fallback can continue from it after the
// generation state is gone.
func (g *AgentGenerationService) saveDraft(uid uint64, st *agentGenState) {
	if st == nil {
		return
	}
	st.mu.Lock()
	draft := strings.Join(st.draft, "")
	st.mu.Unlock()
	if draft == "" {
		return
	}
	g.svc.draftMu.Lock()
	defer g.svc.draftMu.Unlock()
	if g.svc.lastDraft == nil {
		g.svc.lastDraft = make(map[uint64]string)
	}
	g.svc.lastDraft[uid] = draft
}

// clearDraft removes the sticky last-draft slot for the user (a new generation
// or a supersede invalidates any previous draft).
func (g *AgentGenerationService) clearDraft(uid uint64) {
	g.svc.draftMu.Lock()
	delete(g.svc.lastDraft, uid)
	g.svc.draftMu.Unlock()
}

// supersedeGeneration cancels and drops the user's current generation,
// discarding its buffered deltas and any paused-completed reply that has not
// been persisted yet. The dropped state stays registered so late deltas from
// the old stream are still recognized and discarded.
func (g *AgentGenerationService) supersedeGeneration(uid uint64) {
	if uid == 0 {
		return
	}
	g.svc.genMu.Lock()
	if g.svc.genStates == nil {
		g.svc.genStates = make(map[uint64]*agentGenState)
	}
	st := g.svc.genStates[uid]
	g.svc.genMu.Unlock()
	if st == nil {
		return
	}
	st.mu.Lock()
	st.dropped = true
	st.running = false
	st.pending = nil
	st.draft = nil
	st.buffer = nil
	genID := st.genID
	if st.cond != nil {
		st.cond.Broadcast()
	}
	cancel := st.cancel
	st.cancel = nil
	st.mu.Unlock()
	g.svc.clearDraft(uid)
	if cancel != nil {
		cancel()
	}
	g.svc.clearSnapshot(uid, genID)
}

// dropCurrentGeneration marks the user's current generation as dropped (its
// buffered/live deltas are discarded). The state stays registered so late
// deltas from the old stream are still recognized and dropped.
func (g *AgentGenerationService) dropCurrentGeneration(uid uint64) {
	if uid == 0 {
		return
	}
	g.svc.genMu.Lock()
	if g.svc.genStates == nil {
		g.svc.genStates = make(map[uint64]*agentGenState)
	}
	st := g.svc.genStates[uid]
	if st == nil {
		st = &agentGenState{}
		g.svc.genStates[uid] = st
	}
	g.svc.genMu.Unlock()
	st.mu.Lock()
	st.dropped = true
	if st.cond != nil {
		st.cond.Broadcast()
	}
	st.mu.Unlock()
}

// hasRunningGeneration reports whether a generation goroutine is still active
// for the user (used by resume to decide whether buffered deltas are enough or
// a completed reply needs to be persisted).
func (g *AgentGenerationService) hasRunningGeneration(uid uint64) bool {
	if uid == 0 {
		return false
	}
	st := g.svc.generationState(uid)
	if st == nil {
		return false
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	return st.running
}

// storePendingReply records a reply that completed while the user had paused
// the stream. The generation is marked finished (its cancel is released) but
// its state stays registered so a resume can flush buffered deltas first.
func (g *AgentGenerationService) storePendingReply(uid uint64, genID uint64, conv *dm.DmConversation, result *GenerateReplyResult) {
	if uid == 0 || genID == 0 || conv == nil || result == nil {
		return
	}
	g.svc.genMu.Lock()
	st := g.svc.genStates[uid]
	g.svc.genMu.Unlock()
	if st == nil || st.genID != genID {
		return
	}
	st.mu.Lock()
	st.pending = &pendingAgentReply{conv: conv, result: result}
	st.running = false
	curGenID := st.genID
	cancel := st.cancel
	st.cancel = nil
	st.mu.Unlock()
	g.svc.snapshotPending(uid, curGenID, conv.ID, result)
	if cancel != nil {
		cancel()
	}
}

// takePendingReply consumes the user's paused-completed reply, returning the
// conversation, result and generation id. The second call returns false.
func (g *AgentGenerationService) takePendingReply(uid uint64) (*dm.DmConversation, *GenerateReplyResult, uint64, bool) {
	if uid == 0 {
		return nil, nil, 0, false
	}
	g.svc.genMu.Lock()
	st := g.svc.genStates[uid]
	g.svc.genMu.Unlock()
	if st == nil {
		return nil, nil, 0, false
	}
	st.mu.Lock()
	if st.pending == nil {
		st.mu.Unlock()
		return nil, nil, 0, false
	}
	p := st.pending
	st.pending = nil
	genID := st.genID
	st.mu.Unlock()
	return p.conv, p.result, genID, true
}

// PauseGeneration pauses the user's generation. A generation owned locally is
// paused in place; when it runs on another replica, the control command is
// routed to the owner via Redis.
func (g *AgentGenerationService) PauseGeneration(uid uint64) {
	if g.svc == nil || uid == 0 {
		return
	}
	if g.svc.hasRunningGeneration(uid) {
		g.svc.pauseGeneration(uid)
		return
	}
	g.svc.publishControl(context.Background(), uid, map[string]interface{}{"type": "pause", "from": g.svc.InstanceID})
}

// pauseGeneration stops pushing streamed deltas; they are buffered so a later
// resumeGeneration can flush them verbatim (byte-level continuation).
func (g *AgentGenerationService) pauseGeneration(uid uint64) {
	if uid == 0 {
		return
	}
	aigateway.IncAgentControl("pause")
	g.svc.genMu.Lock()
	if g.svc.genStates == nil {
		g.svc.genStates = make(map[uint64]*agentGenState)
	}
	st := g.svc.genStates[uid]
	if st == nil {
		st = &agentGenState{}
		g.svc.genStates[uid] = st
	}
	g.svc.genMu.Unlock()
	st.mu.Lock()
	st.paused = true
	st.pauseSeq++
	genID := st.genID
	pauseSeq := st.pauseSeq
	st.mu.Unlock()
	g.svc.updateSnapshotPaused(uid, genID, true, pauseSeq)
}

// resumeGeneration un-pauses and flushes the buffered deltas in order.
func (g *AgentGenerationService) resumeGeneration(uid uint64) {
	if uid == 0 {
		return
	}
	if g.svc.ChatHub == nil && g.svc.Relay == nil {
		return
	}
	for {
		st := g.svc.generationState(uid)
		if st == nil {
			return
		}
		st.mu.Lock()
		st.ensureCond()
		if st.dropped {
			st.mu.Unlock()
			return
		}
		if st.resuming {
			// Another continue is already replaying the backlog: block on the
			// condition variable instead of busy-spinning. cond.Wait releases
			// st.mu, so the replay goroutine can finish and wake us.
			st.ensureCond()
			st.cond.Wait()
			st.mu.Unlock()
			continue
		}
		st.resuming = true
		seq := st.pauseSeq
		buf := st.buffer
		st.buffer = nil
		st.mu.Unlock()

		for i, d := range buf {
			if d == "" {
				continue
			}
			// A stop clicked during the replay must interrupt it promptly:
			// check before every pushed fragment and leave the rest buffered.
			st.mu.Lock()
			if st.dropped {
				// A newer generation superseded us mid-replay: discard the
				// remaining deltas instead of leaking them to the UI.
				st.resuming = false
				st.cond.Broadcast()
				st.mu.Unlock()
				return
			}
			if repaused := st.pauseSeq != seq; repaused {
				st.buffer = append(append([]string{}, buf[i:]...), st.buffer...)
				st.resuming = false
				st.cond.Broadcast()
				st.mu.Unlock()
				return
			}
			st.mu.Unlock()
			g.svc.publishEvent(context.Background(), uid, map[string]interface{}{
				"type": "agent_delta",
				"body": map[string]interface{}{"content": d},
			})
			// Pace the backlog flush so the UI keeps a typewriter feel instead
			// of dumping the whole paused buffer at once.
			if len(buf) > 1 {
				time.Sleep(12 * time.Millisecond)
			}
		}

		st.mu.Lock()
		if st.dropped {
			st.resuming = false
			st.cond.Broadcast()
			st.mu.Unlock()
			return
		}
		repaused := st.pauseSeq != seq
		more := len(st.buffer) > 0
		st.resuming = false
		st.cond.Broadcast()
		if repaused {
			// A new stop arrived during the replay: keep paused so the
			// remaining deltas stay buffered for the next continue.
			st.mu.Unlock()
			return
		}
		if more {
			// Deltas arrived during the replay (we were still paused):
			// drain them in the next pass.
			st.mu.Unlock()
			continue
		}
		st.paused = false
		st.mu.Unlock()
		g.svc.updateSnapshotPaused(uid, st.genID, false, seq)
		return
	}
}

// isGenerationPaused reports whether the user's generation is paused.
func (g *AgentGenerationService) isGenerationPaused(uid uint64) bool {
	st := g.svc.generationState(uid)
	if st == nil {
		return false
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	return st.paused
}

// clearGenerationState removes the user's pause/buffer state.
func (g *AgentGenerationService) clearGenerationState(uid uint64) {
	if uid == 0 {
		return
	}
	g.svc.genMu.Lock()
	defer g.svc.genMu.Unlock()
	delete(g.svc.genStates, uid)
	g.svc.clearDraft(uid)
}
