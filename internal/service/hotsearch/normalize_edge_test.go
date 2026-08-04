package hotsearch

import "testing"

// ---------- IsAgentConversation ----------

// ---------- normalizeSearchKeyword more edge cases ----------

func TestNormalizeSearchKeywordEdge(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"TAB\tHERE", "tabhere"},
		{"new\nline", "newline"},
		{"  mixed\tSPACES\nHERE  ", "mixedspaceshere"},
		{"a", "a"},
		{"Hello123", "hello123"},
		{"  ", ""},
	}
	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got := normalizeSearchKeyword(tc.input)
			if got != tc.want {
				t.Errorf("normalizeSearchKeyword(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}
