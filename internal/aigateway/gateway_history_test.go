package aigateway

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func longMsg(role string, n int) ChatMessage {
	return ChatMessage{Role: role, Content: strings.Repeat("x", n)}
}

func TestTrimMessagesToTokenBudget_KeepsNewestWithinBudget(t *testing.T) {
	msgs := []ChatMessage{
		longMsg("system", 20),    // ≈5 tokens
		longMsg("user", 20),      // u1
		longMsg("assistant", 20), // a1
		longMsg("user", 20),      // u2
		longMsg("assistant", 20), // a2
	}
	// Budget 12: system(5) + a2(5) + u2(5) = 15 exceeds; a1(5) would too.
	got := trimMessagesToTokenBudget(msgs, 12)
	require.Equal(t, 3, len(got))
	require.Equal(t, "system", got[0].Role)
	require.Equal(t, "user", got[1].Role)      // newest user
	require.Equal(t, "assistant", got[2].Role) // newest assistant
}

func TestTrimMessagesToTokenBudget_AlwaysKeepsLatestUser(t *testing.T) {
	msgs := []ChatMessage{
		longMsg("system", 20),
		longMsg("user", 200), // ≈50 tokens, far over budget
	}
	got := trimMessagesToTokenBudget(msgs, 2)
	require.Equal(t, 2, len(got))
	require.Equal(t, "user", got[1].Role)
}

func TestTrimMessagesToTokenBudget_DisabledWhenZero(t *testing.T) {
	msgs := []ChatMessage{
		longMsg("system", 20),
		longMsg("user", 20),
		longMsg("assistant", 20),
		longMsg("user", 20),
	}
	got := trimMessagesToTokenBudget(msgs, 0)
	require.Equal(t, msgs, got)
}

func TestTrimMessagesToTokenBudget_CountsToolCallArgs(t *testing.T) {
	msgs := []ChatMessage{
		longMsg("system", 20),
		{
			Role: "assistant",
			ToolCalls: []ToolCall{{
				ID: "call-1",
				Function: ToolCallFunction{
					Name:      "search_videos",
					Arguments: strings.Repeat("a", 80), // ≈20 tokens
				},
			}},
		},
		longMsg("user", 20),
	}
	// Budget 11: system(5) + user(5) fit; tool-call message (≈20) would exceed.
	got := trimMessagesToTokenBudget(msgs, 11)
	require.Equal(t, 2, len(got))
	require.Equal(t, "system", got[0].Role)
	require.Equal(t, "user", got[1].Role)
}

func TestBuildMessages_AppliesTokenBudget(t *testing.T) {
	gw := &Gateway{MaxTokens: 8}
	h := &HistoryStore{gw: gw}
	msgs, err := h.BuildMessages(t.Context(), 0, strings.Repeat("y", 20))
	require.NoError(t, err)
	require.Equal(t, 2, len(msgs)) // system + latest user always kept
	require.Equal(t, "user", msgs[1].Role)
}
