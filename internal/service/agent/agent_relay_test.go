package agent

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"cakecake/internal/service/servicetest"
	"cakecake/internal/ws"
)

func startRelay(t *testing.T, svc *AgentService, hub *ws.ChatHub) {
	t.Helper()
	relay := NewAgentRelay(svc.Redis, hub, zap.NewNop(), svc)
	svc.Relay = relay
	ctx, cancel := context.WithCancel(context.Background())
	go relay.RunSubscriber(ctx)
	t.Cleanup(cancel)
}

func TestAgentSnapshot_LifecycleAndGuards(t *testing.T) {
	_, rdb := servicetest.NewRedis(t)
	s := &AgentService{Redis: rdb, InstanceID: "inst-1"}

	s.snapshotRunning(7, 1, 42)
	snap := s.readSnapshot(7)
	require.NotNil(t, snap)
	require.Equal(t, "inst-1", snap.Owner)
	require.True(t, snap.Running)
	require.Equal(t, uint64(42), snap.ConvID)

	// A stale generation id must not clear the newer snapshot.
	s.clearSnapshot(7, 2)
	require.NotNil(t, s.readSnapshot(7))
	s.clearSnapshot(7, 1)
	require.Nil(t, s.readSnapshot(7))

	s.snapshotRunning(7, 3, 42)
	s.snapshotPending(7, 3, 42, &GenerateReplyResult{Content: "待落库"})
	snap = s.readSnapshot(7)
	require.NotNil(t, snap)
	require.False(t, snap.Running)
	require.NotNil(t, snap.Pending)
	require.Equal(t, "待落库", snap.Pending.Content)
}

func TestHandleControl_IgnoresSelfPublishedSupersede(t *testing.T) {
	_, rdb := servicetest.NewRedis(t)
	svcA := &AgentService{Redis: rdb, InstanceID: "inst-a", Log: zap.NewNop()}
	genID := svcA.beginGeneration(7, nil)
	svcA.snapshotRunning(7, genID, 1)

	// A supersede this instance published itself (echoed by the relay) must
	// not cancel its own fresh generation.
	svcA.handleControl(7, map[string]interface{}{"type": "supersede", "from": "inst-a"})
	require.True(t, svcA.hasRunningGeneration(7))

	// A supersede from another replica does cancel it.
	svcA.handleControl(7, map[string]interface{}{"type": "supersede", "from": "inst-b"})
	require.False(t, svcA.hasRunningGeneration(7))
}

// TestMultiInstance_PauseResumeAndEventFanout is the core multi-replica
// evidence: instance A owns the generation, instance B holds the user's WS and
// issues pause/resume; Redis Pub/Sub routes the controls to A and fans events
// out to B's ChatHub.
func TestMultiInstance_PauseResumeAndEventFanout(t *testing.T) {
	_, rdb := servicetest.NewRedis(t)
	const uid = uint64(77)
	const convID = uint64(42)

	// Instance A: generation owner (no local WS needed).
	svcA := &AgentService{Redis: rdb, InstanceID: "inst-a", Log: zap.NewNop()}
	startRelay(t, svcA, nil)

	// Instance B: the replica holding the user's WebSocket connection.
	hubB, connB := newStateTestHub(t, uid)
	svcB := &AgentService{Redis: rdb, InstanceID: "inst-b", ChatHub: hubB, Log: zap.NewNop()}
	startRelay(t, svcB, hubB)
	time.Sleep(100 * time.Millisecond) // let both subscribers land

	genID := svcA.beginGeneration(uid, nil)
	require.NotZero(t, genID)
	svcA.snapshotRunning(uid, genID, convID)

	// A delta produced on A must reach B's local hub.
	svcA.deltaSender(uid, genID)("你好")
	d, ok := readDelta(t, connB)
	require.True(t, ok)
	require.Equal(t, "你好", d)

	// Pause issued on B must reach A (the owner).
	svcB.PauseGeneration(uid)
	require.Eventually(t, func() bool { return svcA.isGenerationPaused(uid) }, 3*time.Second, 10*time.Millisecond)

	// Deltas produced while paused are buffered on A.
	svcA.deltaSender(uid, genID)("世界")

	// Resume issued on B must reach A, flush the buffer, and fan it to B.
	svcB.ResumeReply(uid, convID)
	require.Eventually(t, func() bool { return !svcA.isGenerationPaused(uid) }, 3*time.Second, 10*time.Millisecond)
	replayed, ok := readDelta(t, connB)
	require.True(t, ok)
	require.Equal(t, "世界", replayed)

	svcA.endGeneration(uid, genID)
	require.Nil(t, svcA.readSnapshot(uid))
}
