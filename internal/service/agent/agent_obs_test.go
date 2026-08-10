package agent

import (
	"context"
	"testing"

	"cakecake/internal/aigateway"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
)

func metricCounterValue(t *testing.T, familyName, labelName, labelValue string) float64 {
	t.Helper()
	mfs, err := prometheus.DefaultGatherer.Gather()
	require.NoError(t, err)
	for _, mf := range mfs {
		if mf.GetName() != familyName {
			continue
		}
		for _, m := range mf.GetMetric() {
			for _, lp := range m.GetLabel() {
				if lp.GetName() == labelName && lp.GetValue() == labelValue {
					return m.GetCounter().GetValue()
				}
			}
		}
	}
	return 0
}

func TestTraceIDContext(t *testing.T) {
	ctx := withTraceID(context.Background(), "abc123")
	require.Equal(t, "abc123", traceIDFromContext(ctx))
	require.Empty(t, traceIDFromContext(context.Background()))
}

func TestPauseControlMetricRecorded(t *testing.T) {
	before := metricCounterValue(t, "cakecake_agent_controls_total", "type", "pause")
	g := &AgentGenerationService{svc: &AgentService{}}
	g.pauseGeneration(1)
	after := metricCounterValue(t, "cakecake_agent_controls_total", "type", "pause")
	require.Greater(t, after, before)
}

func TestContinueAndRegenerateControlMetricsRecorded(t *testing.T) {
	continueBefore := metricCounterValue(t, "cakecake_agent_controls_total", "type", "continue")
	regenerateBefore := metricCounterValue(t, "cakecake_agent_controls_total", "type", "regenerate")

	aigateway.IncAgentControl("continue")
	aigateway.IncAgentControl("regenerate")

	require.Greater(t, metricCounterValue(t, "cakecake_agent_controls_total", "type", "continue"), continueBefore)
	require.Greater(t, metricCounterValue(t, "cakecake_agent_controls_total", "type", "regenerate"), regenerateBefore)
}
