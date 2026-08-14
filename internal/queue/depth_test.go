package queue

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFetchQueueDepths(t *testing.T) {
	var hits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		require.Contains(t, r.URL.RequestURI(), "/api/queues/%2F/")
		require.Contains(t, r.URL.RequestURI(), TranscodeQueue)
		user, pass, ok := r.BasicAuth()
		require.True(t, ok)
		require.Equal(t, "guest", user)
		require.Equal(t, "guest", pass)
		_, _ = w.Write([]byte(`{"messages":7}`))
	}))
	defer srv.Close()

	depths, err := FetchQueueDepths(srv.URL, "guest", "guest", []string{TranscodeQueue})
	require.NoError(t, err)
	require.Equal(t, int64(7), depths[TranscodeQueue])
	require.Equal(t, 1, hits)
}

func TestFetchQueueDepths_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "nope", http.StatusUnauthorized)
	}))
	defer srv.Close()

	_, err := FetchQueueDepths(srv.URL, "guest", "guest", []string{TranscodeQueue})
	require.Error(t, err)
	require.Contains(t, err.Error(), "401")
}

func TestAMQPCredentials(t *testing.T) {
	user, pass := AMQPCredentials("amqp://guest:guest@127.0.0.1:5672/")
	require.Equal(t, "guest", user)
	require.Equal(t, "guest", pass)

	user, pass = AMQPCredentials("amqp://127.0.0.1:5672/")
	require.Empty(t, user)
	require.Empty(t, pass)

	user, pass = AMQPCredentials("not a url")
	require.Empty(t, user)
	require.Empty(t, pass)
}

func TestTranscodeBackpressure_RejectsAboveCapacity(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"messages":120}`))
	}))
	defer srv.Close()
	bp := NewTranscodeBackpressure(srv.URL, "guest", "guest", 100)

	err := bp.CheckTranscodeCapacity(context.Background())
	require.ErrorIs(t, err, ErrTranscodeQueueFull)
}

func TestTranscodeBackpressure_AllowsWithinCapacity(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"messages":10}`))
	}))
	defer srv.Close()
	bp := NewTranscodeBackpressure(srv.URL, "guest", "guest", 100)

	require.NoError(t, bp.CheckTranscodeCapacity(context.Background()))
}

func TestTranscodeBackpressure_DisabledOrMgmtDown(t *testing.T) {
	require.NoError(t, (*TranscodeBackpressure)(nil).CheckTranscodeCapacity(context.Background()))
	require.NoError(t, NewTranscodeBackpressure("", "", "", 100).CheckTranscodeCapacity(context.Background()))
	require.NoError(t, NewTranscodeBackpressure("http://127.0.0.1:1", "", "", 0).CheckTranscodeCapacity(context.Background()))

	// Management API outage must not false-positive reject uploads.
	bp := NewTranscodeBackpressure("http://127.0.0.1:1", "guest", "guest", 100)
	require.NoError(t, bp.CheckTranscodeCapacity(context.Background()))
}
