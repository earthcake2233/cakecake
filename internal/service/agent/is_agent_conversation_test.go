package agent

import (
	"cakecake/internal/model/dm"
	"testing"
)

func TestIsAgentConversation(t *testing.T) {
	s := &AgentService{}
	tests := []struct {
		name string
		conv *dm.DmConversation
		want bool
	}{
		{"nil conv", nil, false},
		{"empty kind", &dm.DmConversation{}, false},
		{"wrong kind", &dm.DmConversation{Kind: "normal"}, false},
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
