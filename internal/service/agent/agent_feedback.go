package agent

import (
	"fmt"

	"cakecake/internal/model/agent"
	"context"
)

// AgentFeedbackService owns the agent feedback domain.
type AgentFeedbackService struct {
	svc *AgentService
}

// SetMessageFeedback records or toggles a user's like/dislike on a message.
func (f *AgentFeedbackService) SetMessageFeedback(ctx context.Context, messageID uint64, userID uint64, feedback string) error {
	if f.svc == nil || f.svc.Store == nil {
		return fmt.Errorf("agent service not ready")
	}
	if feedback != "like" && feedback != "dislike" {
		return fmt.Errorf("invalid feedback value")
	}
	return f.svc.Store.SetMessageFeedback(ctx, messageID, userID, feedback)
}

// ListAgentFeedbacks returns feedback rows for the admin console.
func (f *AgentFeedbackService) ListAgentFeedbacks(ctx context.Context, limit int, offset int) ([]agent.AgentFeedback, error) {
	if f.svc == nil || f.svc.Store == nil {
		return nil, fmt.Errorf("agent service not ready")
	}
	return f.svc.Store.ListAgentFeedbacks(ctx, limit, offset)
}

// ListAgentFeedbacksWithContent returns feedback rows joined with the rated
// assistant message content for the admin console.
func (f *AgentFeedbackService) ListAgentFeedbacksWithContent(ctx context.Context, limit int, offset int) ([]AgentFeedbackRow, error) {
	if f.svc == nil || f.svc.Store == nil {
		return nil, fmt.Errorf("agent service not ready")
	}
	return f.svc.Store.ListAgentFeedbacksWithContent(ctx, limit, offset)
}
