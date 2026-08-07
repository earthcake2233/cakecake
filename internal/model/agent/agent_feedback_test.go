package agent

import "testing"

func TestAgentFeedbackTableName(t *testing.T) {
	if got := (AgentFeedback{}).TableName(); got != "agent_feedbacks" {
		t.Fatalf("unexpected table name: %s", got)
	}
}
