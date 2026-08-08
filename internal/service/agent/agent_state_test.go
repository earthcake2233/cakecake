package agent

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"cakecake/internal/model/dm"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"cakecake/internal/ws"
)

// newStateTestHub starts a WebSocket server joined to a ChatHub and returns
// the hub plus a connected client for the given user.
func newStateTestHub(t *testing.T, uid uint64) (*ws.ChatHub, *websocket.Conn) {
	t.Helper()
	hub := ws.NewChatHub()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		up := websocket.Upgrader{}
		conn, err := up.Upgrade(w, r, nil)
		require.NoError(t, err)
		hub.Join(uid, conn)
		defer hub.Leave(uid, conn)
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				break
			}
		}
	}))
	t.Cleanup(srv.Close)

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })
	time.Sleep(50 * time.Millisecond) // let the server Join land
	return hub, conn
}

func readDelta(t *testing.T, conn *websocket.Conn) (string, bool) {
	t.Helper()
	_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	_, data, err := conn.ReadMessage()
	if err != nil {
		return "", false
	}
	var frame struct {
		Type string `json:"type"`
		Body struct {
			Content string `json:"content"`
		} `json:"body"`
	}
	if err := json.Unmarshal(data, &frame); err != nil {
		t.Fatalf("bad frame: %v", err)
	}
	if frame.Type != "agent_delta" {
		t.Fatalf("unexpected frame type %q", frame.Type)
	}
	return frame.Body.Content, true
}

func TestResumeGeneration_ReplaysBufferInOrder(t *testing.T) {
	hub, conn := newStateTestHub(t, 7)
	s := &AgentService{ChatHub: hub, Log: zap.NewNop()}

	genID := s.BeginGeneration(7, nil)
	s.PauseGeneration(7)
	send := s.deltaSender(7, genID)
	send("你")
	send("好")
	send("世界")

	s.ResumeGeneration(7)
	got := []string{}
	for {
		d, ok := readDelta(t, conn)
		if !ok {
			break
		}
		got = append(got, d)
		if len(got) == 3 {
			break
		}
	}
	require.Equal(t, []string{"你", "好", "世界"}, got)
	require.False(t, s.IsGenerationPaused(7))
}

func TestResumeGeneration_StopDuringReplayKeepsPaused(t *testing.T) {
	hub, conn := newStateTestHub(t, 7)
	s := &AgentService{ChatHub: hub, Log: zap.NewNop()}
	genID := s.BeginGeneration(7, nil)
	s.PauseGeneration(7)
	send := s.deltaSender(7, genID)
	const n = 40
	for i := 0; i < n; i++ {
		send(string(rune('a' + i%26)))
	}

	done := make(chan struct{})
	go func() {
		s.ResumeGeneration(7)
		close(done)
	}()

	// Stop again right after the replay starts.
	_, ok := readDelta(t, conn)
	require.True(t, ok)
	s.PauseGeneration(7)
	<-done

	require.True(t, s.IsGenerationPaused(7))
	st := s.generationState(7)
	require.NotNil(t, st)
	st.mu.Lock()
	remaining := len(st.buffer)
	st.mu.Unlock()
	require.Greater(t, remaining, 0)

	// A second continue drains the rest.
	s.ResumeGeneration(7)
	total := 1
	for {
		if _, ok := readDelta(t, conn); !ok {
			break
		}
		total++
		if total == n {
			break
		}
	}
	require.Equal(t, n, total)
	require.False(t, s.IsGenerationPaused(7))
}

func TestResumeGeneration_SupersedeDuringReplayDiscardsRest(t *testing.T) {
	hub, conn := newStateTestHub(t, 13)
	s := &AgentService{ChatHub: hub, Log: zap.NewNop()}
	genID := s.BeginGeneration(13, nil)
	s.PauseGeneration(13)
	send := s.deltaSender(13, genID)
	const n = 40
	for i := 0; i < n; i++ {
		send(string(rune('a' + i%26)))
	}

	done := make(chan struct{})
	go func() {
		s.ResumeGeneration(13)
		close(done)
	}()

	_, ok := readDelta(t, conn)
	require.True(t, ok)
	s.SupersedeGeneration(13)
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("replay did not stop after supersede")
	}

	// Only deltas written before the supersede landed may arrive; the rest of
	// the backlog must be discarded instead of leaked to the UI.
	_ = conn.SetReadDeadline(time.Now().Add(300 * time.Millisecond))
	inFlight := 0
	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			break
		}
		inFlight++
	}
	require.Less(t, inFlight, n-1)
}

func TestDeltaSender_DroppedGenerationDiscards(t *testing.T) {
	hub, conn := newStateTestHub(t, 9)
	s := &AgentService{ChatHub: hub, Log: zap.NewNop()}
	genID := s.BeginGeneration(9, nil)
	s.DropCurrentGeneration(9)
	s.deltaSender(9, genID)("x")

	_ = conn.SetReadDeadline(time.Now().Add(120 * time.Millisecond))
	_, _, err := conn.ReadMessage()
	require.Error(t, err)
}

func TestConcurrentResumes_SingleOrderedStream(t *testing.T) {
	hub, conn := newStateTestHub(t, 11)
	s := &AgentService{ChatHub: hub, Log: zap.NewNop()}
	genID := s.BeginGeneration(11, nil)
	s.PauseGeneration(11)
	send := s.deltaSender(11, genID)
	const n = 30
	for i := 0; i < n; i++ {
		send(string(rune('A' + i)))
	}

	done := make(chan struct{}, 2)
	for i := 0; i < 2; i++ {
		go func() {
			s.ResumeGeneration(11)
			done <- struct{}{}
		}()
	}
	<-done
	<-done

	got := []string{}
	for {
		d, ok := readDelta(t, conn)
		if !ok {
			break
		}
		got = append(got, d)
		if len(got) == n {
			break
		}
	}
	require.Len(t, got, n)
	for i, d := range got {
		require.Equal(t, string(rune('A'+i)), d)
	}
	require.False(t, s.IsGenerationPaused(11))
}

func TestGenerationLifecycle(t *testing.T) {
	s := &AgentService{}
	id1 := s.BeginGeneration(5, nil)
	id2 := s.BeginGeneration(5, nil) // superseded
	require.Greater(t, id2, id1)
	s.EndGeneration(5, id1) // must NOT clear the newer state
	st := s.generationState(5)
	require.NotNil(t, st)
	require.Equal(t, id2, st.genID)

	s.EndGeneration(5, id2)
	require.Nil(t, s.generationState(5))

	s.BeginGeneration(5, nil)
	s.ClearGenerationState(5)
	require.Nil(t, s.generationState(5))
}

func TestBeginGeneration_EndCancelsAndRunningFlag(t *testing.T) {
	s := &AgentService{}
	cancelCalls := 0
	id1 := s.BeginGeneration(7, func() { cancelCalls++ })
	id2 := s.BeginGeneration(7, func() { cancelCalls++ })
	require.Greater(t, id2, id1)
	// Starting a newer generation releases the previous one's cancel.
	require.Equal(t, 1, cancelCalls)

	require.True(t, s.HasRunningGeneration(7))
	s.EndGeneration(7, id1) // stale id must not touch the newer state
	require.True(t, s.HasRunningGeneration(7))
	require.Equal(t, 1, cancelCalls)

	s.EndGeneration(7, id2)
	require.False(t, s.HasRunningGeneration(7))
	require.Equal(t, 2, cancelCalls)
}

func TestSupersedeGeneration_CancelsAndClearsPending(t *testing.T) {
	s := &AgentService{}
	cancelCalls := 0
	genID := s.BeginGeneration(7, func() { cancelCalls++ })
	s.deltaSender(7, genID)("partial")
	s.PauseGeneration(7)
	s.StorePendingReply(7, genID, &dm.DmConversation{ID: 1}, &GenerateReplyResult{Content: "x"})
	require.False(t, s.HasRunningGeneration(7))

	s.SupersedeGeneration(7)
	require.Equal(t, 1, cancelCalls)
	require.False(t, s.HasRunningGeneration(7))
	require.Empty(t, s.DraftText(7))
	_, _, _, ok := s.TakePendingReply(7)
	require.False(t, ok)
}

func TestStoreTakePendingReply_StaleGenIDIgnored(t *testing.T) {
	s := &AgentService{}
	conv := &dm.DmConversation{ID: 42}
	result := &GenerateReplyResult{Content: "done"}

	genID := s.BeginGeneration(7, nil)
	s.StorePendingReply(7, genID+1, conv, result) // stale id ignored
	_, _, _, ok := s.TakePendingReply(7)
	require.False(t, ok)

	s.StorePendingReply(7, genID, conv, result)
	gotConv, gotResult, gotGenID, ok := s.TakePendingReply(7)
	require.True(t, ok)
	require.Same(t, conv, gotConv)
	require.Same(t, result, gotResult)
	require.Equal(t, genID, gotGenID)
	_, _, _, ok = s.TakePendingReply(7) // consumed exactly once
	require.False(t, ok)
}

func TestDraftText_SurvivesEndGenerationAndClearsOnNewGeneration(t *testing.T) {
	s := &AgentService{}
	genID := s.BeginGeneration(7, nil)
	send := s.deltaSender(7, genID)
	send("你")
	send("好")
	require.Equal(t, "你好", s.DraftText(7))

	s.EndGeneration(7, genID)
	require.Equal(t, "你好", s.DraftText(7)) // sticky draft for re-prompt fallback

	s.BeginGeneration(7, nil) // a new generation invalidates the old draft
	require.Empty(t, s.DraftText(7))
}

func TestGenerateSuggestions_NilGateway(t *testing.T) {
	s := &AgentService{}
	require.Nil(t, s.GenerateSuggestions(context.Background(), "reply"))
}
