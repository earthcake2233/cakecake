package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.uber.org/zap"

	"cakecake/internal/config"
	"cakecake/internal/queue"
)

// Transcode pipeline observability: dead-letter lifecycle, retry scheduling,
// per-result job counters, job duration and RabbitMQ queue depth.
var (
	transcodeDeadLettersTotal = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "cakecake",
		Subsystem: "transcode",
		Name:      "dead_letters_total",
		Help:      "Total transcode jobs dead-lettered (retries exhausted or republish failed).",
	})

	transcodeRetriesScheduledTotal = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "cakecake",
		Subsystem: "transcode",
		Name:      "retries_scheduled_total",
		Help:      "Total transcode retries scheduled into TTL delay queues.",
	})

	transcodeJobsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "cakecake",
		Subsystem: "transcode",
		Name:      "jobs_total",
		Help:      "Total transcode jobs by result (success|permanent_failure).",
	}, []string{"result"})

	transcodeDurationSeconds = promauto.NewHistogram(prometheus.HistogramOpts{
		Namespace: "cakecake",
		Subsystem: "transcode",
		Name:      "duration_seconds",
		Help:      "End-to-end transcode job duration (download -> ffmpeg -> OSS -> DB).",
		Buckets:   prometheus.ExponentialBuckets(1, 2, 12), // 1s .. ~2048s
	})

	transcodeQueueDepth = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "cakecake",
		Subsystem: "transcode",
		Name:      "queue_depth",
		Help:      "Current message backlog per RabbitMQ transcode queue (management API).",
	}, []string{"queue"})
)

func incrTranscodeDeadLetters() {
	transcodeDeadLettersTotal.Inc()
}

func incrTranscodeRetriesScheduled() {
	transcodeRetriesScheduledTotal.Inc()
}

func incrTranscodeSuccess(d time.Duration) {
	transcodeJobsTotal.WithLabelValues("success").Inc()
	transcodeDurationSeconds.Observe(d.Seconds())
}

func incrTranscodePermanentFailure() {
	transcodeJobsTotal.WithLabelValues("permanent_failure").Inc()
}

// queueDepthCollectInterval is how often the management API is polled.
const queueDepthCollectInterval = 15 * time.Second

// transcodeQueuesForDepth are the queues whose backlog is exposed as a gauge.
var transcodeQueuesForDepth = []string{
	queue.TranscodeQueue,
	queue.TranscodeRetryQueue30s,
	queue.TranscodeRetryQueue60s,
	queue.TranscodeRetryQueue90s,
	queue.TranscodeDeadQueue,
}

// StartQueueDepthCollector polls the RabbitMQ management API and exposes the
// message count of every transcode queue. It is a no-op when RABBITMQ_MGMT_URL
// is empty. Credentials are taken from RABBITMQ_URL (amqp://user:pass@...).
func StartQueueDepthCollector(ctx context.Context, cfg *config.C, lg *zap.Logger) {
	if cfg == nil || cfg.RabbitMQMgmtURL == "" {
		return
	}
	user, pass := amqpCredentials(cfg.RabbitMQURL)
	fetch := func() {
		depths, err := fetchQueueDepths(cfg.RabbitMQMgmtURL, user, pass, transcodeQueuesForDepth)
		if err != nil {
			lg.Warn("fetch rabbitmq queue depth", zap.Error(err))
			return
		}
		for q, n := range depths {
			transcodeQueueDepth.WithLabelValues(q).Set(float64(n))
		}
	}
	fetch()
	t := time.NewTicker(queueDepthCollectInterval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			fetch()
		}
	}
}

func fetchQueueDepths(mgmtURL, user, pass string, queues []string) (map[string]int64, error) {
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

func amqpCredentials(raw string) (string, string) {
	u, err := url.Parse(raw)
	if err != nil || u.User == nil {
		return "", ""
	}
	pass, _ := u.User.Password()
	return u.User.Username(), pass
}
