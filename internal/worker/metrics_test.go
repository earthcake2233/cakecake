package worker

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"cakecake/internal/queue"
)

func TestFetchQueueDepths(t *testing.T) {
	var hits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		require.Contains(t, r.URL.RequestURI(), "/api/queues/%2F/")
		require.Contains(t, r.URL.RequestURI(), queue.TranscodeQueue)
		user, pass, ok := r.BasicAuth()
		require.True(t, ok)
		require.Equal(t, "guest", user)
		require.Equal(t, "guest", pass)
		_, _ = w.Write([]byte(`{"messages":7}`))
	}))
	defer srv.Close()

	depths, err := fetchQueueDepths(srv.URL, "guest", "guest", []string{queue.TranscodeQueue})
	require.NoError(t, err)
	require.Equal(t, int64(7), depths[queue.TranscodeQueue])
	require.Equal(t, 1, hits)
}

func TestFetchQueueDepths_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "nope", http.StatusUnauthorized)
	}))
	defer srv.Close()

	_, err := fetchQueueDepths(srv.URL, "guest", "guest", []string{queue.TranscodeQueue})
	require.Error(t, err)
	require.Contains(t, err.Error(), "401")
}

func TestAMQPCredentials(t *testing.T) {
	user, pass := amqpCredentials("amqp://guest:guest@127.0.0.1:5672/")
	require.Equal(t, "guest", user)
	require.Equal(t, "guest", pass)

	user, pass = amqpCredentials("amqp://127.0.0.1:5672/")
	require.Empty(t, user)
	require.Empty(t, pass)

	user, pass = amqpCredentials("not a url")
	require.Empty(t, user)
	require.Empty(t, pass)
}

func TestQueueDepthCollectorDisabledWithoutMgmtURL(t *testing.T) {
	// StartQueueDepthCollector must return immediately when unconfigured;
	// it should not block the caller.
	done := make(chan struct{})
	go func() {
		StartQueueDepthCollector(context.Background(), nil, nil)
		close(done)
	}()
	<-done
}
