package video

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	transcodeRequeuedTotal = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "cakecake",
		Subsystem: "transcode",
		Name:      "requeued_total",
		Help:      "Total transcode dead letters requeued via the admin API.",
	})
)

func incrTranscodeRequeued() {
	transcodeRequeuedTotal.Inc()
}
