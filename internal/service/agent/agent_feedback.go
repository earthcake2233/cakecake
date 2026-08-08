package agent

import (
	"fmt"

	"cakecake/internal/model/agent"
	"context"
)

// SetMessageFeedback records or toggles a user's like/dislike on a message.
func (s *AgentService) SetMessageFeedback(ctx context.Context, messageID uint64, userID uint64, feedback string) error {
	if s == nil || s.Store == nil {
		return fmt.Errorf("agent service not ready")
	}
	if feedback != "like" && feedback != "dislike" {
		return fmt.Errorf("invalid feedback value")
	}
	return s.Store.SetMessageFeedback(ctx, messageID, userID, feedback)
}

// ListAgentFeedbacks returns feedback rows for the admin console.
func (s *AgentService) ListAgentFeedbacks(ctx context.Context, limit int, offset int) ([]agent.AgentFeedback, error) {
	if s == nil || s.Store == nil {
		return nil, fmt.Errorf("agent service not ready")
	}
	return s.Store.ListAgentFeedbacks(ctx, limit, offset)
}

// ListAgentFeedbacksWithContent returns feedback rows joined with the rated
// assistant message content for the admin console.
func (s *AgentService) ListAgentFeedbacksWithContent(ctx context.Context, limit int, offset int) ([]AgentFeedbackRow, error) {
	if s == nil || s.Store == nil {
		return nil, fmt.Errorf("agent service not ready")
	}
	return s.Store.ListAgentFeedbacksWithContent(ctx, limit, offset)
}
