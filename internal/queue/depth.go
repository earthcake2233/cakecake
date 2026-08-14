package queue

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// ErrTranscodeQueueFull reports that the main transcode queue is above its
// configured capacity and new enqueues are rejected (backpressure).
var ErrTranscodeQueueFull = errors.New("transcode queue full")

// FetchQueueDepths returns the current message count for each queue via the
// RabbitMQ management HTTP API (vhost "/").
func FetchQueueDepths(mgmtURL, user, pass string, queues []string) (map[string]int64, error) {
	client := &http.Client{Timeout: 5 * time.Second}
	out := make(map[string]int64, len(queues))
	for _, q := range queues {
		endpoint := fmt.Sprintf("%s/api/queues/%s/%s", strings.TrimSuffix(mgmtURL, "/"), url.PathEscape("/"), q)
		req, err := http.NewRequest(http.MethodGet, endpoint, nil)
		if err != nil {
			return nil, err
		}
		req.SetBasicAuth(user, pass)
		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		body, readErr := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if readErr != nil {
			return nil, readErr
		}
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("management api %s: HTTP %d", q, resp.StatusCode)
		}
		var d struct {
			Messages int64 `json:"messages"`
		}
		if err := json.Unmarshal(body, &d); err != nil {
			return nil, err
		}
		out[q] = d.Messages
	}
	return out, nil
}

// AMQPCredentials extracts the username/password from an amqp:// URL.
func AMQPCredentials(raw string) (string, string) {
	u, err := url.Parse(raw)
	if err != nil || u.User == nil {
		return "", ""
	}
	pass, _ := u.User.Password()
	return u.User.Username(), pass
}

// TranscodeBackpressure rejects new transcode enqueues while the main queue
// holds maxDepth or more messages. Empty management URL or maxDepth <= 0
// disables the check (nil receiver is also disabled).
type TranscodeBackpressure struct {
	MgmtURL  string
	User     string
	Pass     string
	MaxDepth int64
}

// NewTranscodeBackpressure builds a checker for the main transcode queue.
func NewTranscodeBackpressure(mgmtURL, user, pass string, maxDepth int64) *TranscodeBackpressure {
	return &TranscodeBackpressure{MgmtURL: mgmtURL, User: user, Pass: pass, MaxDepth: maxDepth}
}

// CheckTranscodeCapacity returns ErrTranscodeQueueFull when the main queue is
// above capacity. Management API failures are deliberately ignored so a
// monitoring outage never false-positively rejects uploads.
func (b *TranscodeBackpressure) CheckTranscodeCapacity(ctx context.Context) error {
	if b == nil || b.MgmtURL == "" || b.MaxDepth <= 0 {
		return nil
	}
	depths, err := FetchQueueDepths(b.MgmtURL, b.User, b.Pass, []string{TranscodeQueue})
	if err != nil {
		return nil
	}
	if depths[TranscodeQueue] >= b.MaxDepth {
		return fmt.Errorf("%w: depth %d (limit %d)", ErrTranscodeQueueFull, depths[TranscodeQueue], b.MaxDepth)
	}
	return nil
}
