package worker

import (
	"context"
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

	orphanObjectsDeletedTotal = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "cakecake",
		Subsystem: "upload",
		Name:      "orphan_objects_deleted_total",
		Help:      "Total unlinked uploads/drafts objects deleted by the in-app orphan cleanup task.",
	})
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

func incrOrphanObjectsDeleted() {
	orphanObjectsDeletedTotal.Inc()
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
	user, pass := queue.AMQPCredentials(cfg.RabbitMQURL)
	fetch := func() {
		depths, err := queue.FetchQueueDepths(cfg.RabbitMQMgmtURL, user, pass, transcodeQueuesForDepth)
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
