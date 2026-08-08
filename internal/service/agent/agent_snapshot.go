package agent

import (
	"context"
	"encoding/json"
	"time"

	"cakecake/internal/data"
)

const agentSnapshotTTL = 24 * time.Hour

// genSnapshot is the cross-instance view of a user's generation: enough for a
// remote replica to route pause/resume/supersede to the owner, and to recover
// a paused-completed reply when the owner is gone.
type genSnapshot struct {
	Owner    string       `json:"owner"`
	GenID    uint64       `json:"gen_id"`
	Running  bool         `json:"running"`
	Paused   bool         `json:"paused"`
	PauseSeq uint64       `json:"pause_seq"`
	Dropped  bool         `json:"dropped"`
	ConvID   uint64       `json:"conv_id"`
	Pending  *pendingSnap `json:"pending,omitempty"`
}

// pendingSnap is the serialized paused-completed reply.
type pendingSnap struct {
	ConvID         uint64 `json:"conv_id"`
	Content        string `json:"content"`
	ToolActivities string `json:"tool_activities,omitempty"`
	ToolResultData string `json:"tool_result_data,omitempty"`
	Suggestions    string `json:"suggestions,omitempty"`
}

func (s *AgentService) writeSnapshot(uid uint64, snap *genSnapshot) {
	if s == nil || s.Redis == nil || uid == 0 || snap == nil {
		return
	}
	b, err := json.Marshal(snap)
	if err != nil {
		return
	}
	_ = s.Redis.Set(context.Background(), data.AgentGenSnapshotKey(uid), b, agentSnapshotTTL).Err()
}

func (s *AgentService) readSnapshot(uid uint64) *genSnapshot {
	if s == nil || s.Redis == nil || uid == 0 {
		return nil
	}
	raw, err := s.Redis.Get(context.Background(), data.AgentGenSnapshotKey(uid)).Bytes()
	if err != nil {
		return nil
	}
	var snap genSnapshot
	if json.Unmarshal(raw, &snap) != nil {
		return nil
	}
	return &snap
}

// clearSnapshot removes the snapshot only when it still belongs to genID (a
// finished generation can never clear a newer generation's snapshot).
func (s *AgentService) clearSnapshot(uid uint64, genID uint64) {
	if s == nil || s.Redis == nil || uid == 0 || genID == 0 {
		return
	}
	snap := s.readSnapshot(uid)
	if snap == nil || snap.GenID != genID {
		return
	}
	_ = s.Redis.Del(context.Background(), data.AgentGenSnapshotKey(uid)).Err()
}

// updateSnapshotPaused mirrors a local pause/resume change into the snapshot
// (guarded by generation id).
func (s *AgentService) updateSnapshotPaused(uid uint64, genID uint64, paused bool, pauseSeq uint64) {
	if s == nil || s.Redis == nil || uid == 0 || genID == 0 {
		return
	}
	snap := s.readSnapshot(uid)
	if snap == nil || snap.GenID != genID {
		return
	}
	snap.Paused = paused
	snap.PauseSeq = pauseSeq
	s.writeSnapshot(uid, snap)
}

// snapshotRunning marks the snapshot as a running generation owned by this
// instance (called right after beginGeneration on the owner).
func (s *AgentService) snapshotRunning(uid uint64, genID uint64, convID uint64) {
	if s == nil || s.Redis == nil || uid == 0 || genID == 0 {
		return
	}
	s.writeSnapshot(uid, &genSnapshot{
		Owner:   s.InstanceID,
		GenID:   genID,
		Running: true,
		ConvID:  convID,
	})
}

// snapshotPending marks the snapshot as finished-while-paused with the reply
// serialized, so a remote replica can recover it.
func (s *AgentService) snapshotPending(uid uint64, genID uint64, convID uint64, result *GenerateReplyResult) {
	if s == nil || s.Redis == nil || uid == 0 || genID == 0 || result == nil {
		return
	}
	snap := s.readSnapshot(uid)
	if snap == nil || snap.GenID != genID {
		return
	}
	snap.Running = false
	snap.Paused = true
	snap.Pending = &pendingSnap{
		ConvID:         convID,
		Content:        result.Content,
		ToolActivities: string(result.ToolActivities),
		ToolResultData: string(result.ToolResultData),
	}
	if len(result.Suggestions) > 0 {
		if b, err := json.Marshal(result.Suggestions); err == nil {
			snap.Pending.Suggestions = string(b)
		}
	}
	s.writeSnapshot(uid, snap)
}
