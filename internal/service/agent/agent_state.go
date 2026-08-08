package agent

import (
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
func (s *AgentService) deltaSender(humanID uint64, genID uint64) func(string) {
	return func(delta string) {
		if delta == "" {
			return
		}
		if st := s.generationState(humanID); st != nil {
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
		if s.ChatHub != nil || s.Relay != nil {
			s.publishEvent(context.Background(), humanID, map[string]interface{}{
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
func (s *AgentService) publishEvent(ctx context.Context, uid uint64, payload interface{}) {
	if s == nil || uid == 0 {
		return
	}
	if s.Relay != nil {
		_ = s.Relay.PublishEvent(ctx, uid, payload)
	} else if s.ChatHub != nil {
		s.ChatHub.PushJSON(uid, payload)
	}
	if s.EventHook != nil {
		if m, ok := payload.(map[string]interface{}); ok {
			s.EventHook(uid, m)
		}
	}
}

// publishControl routes a cross-instance generation control command to the
// owner (no-op without a relay).
func (s *AgentService) publishControl(ctx context.Context, uid uint64, payload interface{}) {
	if s == nil || uid == 0 || s.Relay == nil {
		return
	}
	_ = s.Relay.PublishControl(ctx, uid, payload)
}

// draftText returns the full text streamed so far for the user's generation.
// It survives endGeneration (copied to the sticky last-draft slot) so a
// re-prompt fallback can continue from the exact server-side draft.
func (s *AgentService) draftText(uid uint64) string {
	if uid == 0 {
		return ""
	}
	if st := s.generationState(uid); st != nil {
		st.mu.Lock()
		defer st.mu.Unlock()
		return strings.Join(st.draft, "")
	}
	s.draftMu.Lock()
	defer s.draftMu.Unlock()
	return s.lastDraft[uid]
}

// currentGenID returns the generation id registered for the user, or 0.
func (s *AgentService) currentGenID(uid uint64) uint64 {
	st := s.generationState(uid)
	if st == nil {
		return 0
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	return st.genID
}

func (s *AgentService) generationState(uid uint64) *agentGenState {
	if uid == 0 {
		return nil
	}
	s.genMu.Lock()
	defer s.genMu.Unlock()
	if s.genStates == nil {
		return nil
	}
	return s.genStates[uid]
}

// beginGeneration registers a new generation state before the LLM call and
// returns its generation id. The cancel func is owned by the state: it is
// invoked when the generation is superseded, ended, or finished while paused,
// and supersedes any previous generation state for the user.
func (s *AgentService) beginGeneration(uid uint64, cancel context.CancelFunc) uint64 {
	if uid == 0 {
		return 0
	}
	s.genMu.Lock()
	if s.genStates == nil {
		s.genStates = make(map[uint64]*agentGenState)
	}
	s.genSeq++
	old := s.genStates[uid]
	s.genStates[uid] = &agentGenState{genID: s.genSeq, cancel: cancel, running: true}
	s.genMu.Unlock()
	if old != nil {
		if oldCancel := old.finish(); oldCancel != nil {
			oldCancel()
		}
	}
	s.clearDraft(uid)
	return s.genSeq
}

// endGeneration removes the generation state only if it still belongs to the
// given generation id (a finished goroutine can never clear a newer state).
// The attached cancel is invoked so request resources are released.
func (s *AgentService) endGeneration(uid uint64, genID uint64) {
	if uid == 0 {
		return
	}
	s.genMu.Lock()
	st, ok := s.genStates[uid]
	if !ok || st.genID != genID {
		s.genMu.Unlock()
		return
	}
	delete(s.genStates, uid)
	s.genMu.Unlock()
	s.clearSnapshot(uid, genID)
	s.saveDraft(uid, st)
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
func (s *AgentService) saveDraft(uid uint64, st *agentGenState) {
	if st == nil {
		return
	}
	st.mu.Lock()
	draft := strings.Join(st.draft, "")
	st.mu.Unlock()
	if draft == "" {
		return
	}
	s.draftMu.Lock()
	defer s.draftMu.Unlock()
	if s.lastDraft == nil {
		s.lastDraft = make(map[uint64]string)
	}
	s.lastDraft[uid] = draft
}

// clearDraft removes the sticky last-draft slot for the user (a new generation
// or a supersede invalidates any previous draft).
func (s *AgentService) clearDraft(uid uint64) {
	s.draftMu.Lock()
	delete(s.lastDraft, uid)
	s.draftMu.Unlock()
}

// supersedeGeneration cancels and drops the user's current generation,
// discarding its buffered deltas and any paused-completed reply that has not
// been persisted yet. The dropped state stays registered so late deltas from
// the old stream are still recognized and discarded.
func (s *AgentService) supersedeGeneration(uid uint64) {
	if uid == 0 {
		return
	}
	s.genMu.Lock()
	if s.genStates == nil {
		s.genStates = make(map[uint64]*agentGenState)
	}
	st := s.genStates[uid]
	s.genMu.Unlock()
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
	s.clearDraft(uid)
	if cancel != nil {
		cancel()
	}
	s.clearSnapshot(uid, genID)
}

// dropCurrentGeneration marks the user's current generation as dropped (its
// buffered/live deltas are discarded). The state stays registered so late
// deltas from the old stream are still recognized and dropped.
func (s *AgentService) dropCurrentGeneration(uid uint64) {
	if uid == 0 {
		return
	}
	s.genMu.Lock()
	if s.genStates == nil {
		s.genStates = make(map[uint64]*agentGenState)
	}
	st := s.genStates[uid]
	if st == nil {
		st = &agentGenState{}
		s.genStates[uid] = st
	}
	s.genMu.Unlock()
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
func (s *AgentService) hasRunningGeneration(uid uint64) bool {
	if uid == 0 {
		return false
	}
	st := s.generationState(uid)
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
func (s *AgentService) storePendingReply(uid uint64, genID uint64, conv *dm.DmConversation, result *GenerateReplyResult) {
	if uid == 0 || genID == 0 || conv == nil || result == nil {
		return
	}
	s.genMu.Lock()
	st := s.genStates[uid]
	s.genMu.Unlock()
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
	s.snapshotPending(uid, curGenID, conv.ID, result)
	if cancel != nil {
		cancel()
	}
}

// takePendingReply consumes the user's paused-completed reply, returning the
// conversation, result and generation id. The second call returns false.
func (s *AgentService) takePendingReply(uid uint64) (*dm.DmConversation, *GenerateReplyResult, uint64, bool) {
	if uid == 0 {
		return nil, nil, 0, false
	}
	s.genMu.Lock()
	st := s.genStates[uid]
	s.genMu.Unlock()
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
func (s *AgentService) PauseGeneration(uid uint64) {
	if s == nil || uid == 0 {
		return
	}
	if s.hasRunningGeneration(uid) {
		s.pauseGeneration(uid)
		return
	}
	s.publishControl(context.Background(), uid, map[string]interface{}{"type": "pause", "from": s.InstanceID})
}

// pauseGeneration stops pushing streamed deltas; they are buffered so a later
// resumeGeneration can flush them verbatim (byte-level continuation).
func (s *AgentService) pauseGeneration(uid uint64) {
	if uid == 0 {
		return
	}
	s.genMu.Lock()
	if s.genStates == nil {
		s.genStates = make(map[uint64]*agentGenState)
	}
	st := s.genStates[uid]
	if st == nil {
		st = &agentGenState{}
		s.genStates[uid] = st
	}
	s.genMu.Unlock()
	st.mu.Lock()
	st.paused = true
	st.pauseSeq++
	genID := st.genID
	pauseSeq := st.pauseSeq
	st.mu.Unlock()
	s.updateSnapshotPaused(uid, genID, true, pauseSeq)
}

// resumeGeneration un-pauses and flushes the buffered deltas in order.
func (s *AgentService) resumeGeneration(uid uint64) {
	if uid == 0 {
		return
	}
	if s.ChatHub == nil && s.Relay == nil {
		return
	}
	for {
		st := s.generationState(uid)
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
			s.publishEvent(context.Background(), uid, map[string]interface{}{
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
		s.updateSnapshotPaused(uid, st.genID, false, seq)
		return
	}
}

// isGenerationPaused reports whether the user's generation is paused.
func (s *AgentService) isGenerationPaused(uid uint64) bool {
	st := s.generationState(uid)
	if st == nil {
		return false
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	return st.paused
}

// clearGenerationState removes the user's pause/buffer state.
func (s *AgentService) clearGenerationState(uid uint64) {
	if uid == 0 {
		return
	}
	s.genMu.Lock()
	defer s.genMu.Unlock()
	delete(s.genStates, uid)
	s.clearDraft(uid)
}
