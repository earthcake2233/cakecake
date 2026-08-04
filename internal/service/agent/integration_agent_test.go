package agent

import (
	"cakecake/internal/model/dm"
	"testing"
)

func TestAgentService_IsAgentConversationIntegration(t *testing.T) {
	s := &AgentService{}
	tests := []struct {
		name string
		conv *dm.DmConversation
		want bool
	}{
		{"nil conv", nil, false},
		{"empty kind", &dm.DmConversation{}, false},
		{"wrong kind", &dm.DmConversation{Kind: "human"}, false},
		{"agent kind", &dm.DmConversation{Kind: dm.DmKindAgent}, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := s.IsAgentConversation(tc.conv)
			if got != tc.want {
				t.Errorf("IsAgentConversation(%+v) = %v, want %v", tc.conv, got, tc.want)
			}
		})
	}
}
