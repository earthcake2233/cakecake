package worker

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Transcode pipeline observability for the dead-letter lifecycle.
var (
	transcodeDeadLettersTotal = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "cakecake",
		Subsystem: "transcode",
		Name:      "dead_letters_total",
		Help:      "Total transcode jobs dead-lettered (retries exhausted or republish failed).",
	})
)

func incrTranscodeDeadLetters() {
	transcodeDeadLettersTotal.Inc()
}
