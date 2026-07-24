package data

import (
	"testing"
)

func TestNormalizeAgentSlug_Valid(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"default", "default"},
		{"my_agent", "my_agent"},
		{"agent42", "agent42"},
		{"  Default  ", "default"},
		{"ab", "ab"},
		{"abc123_def", "abc123_def"},
		{"uppercase", "uppercase"},
	}
	for _, tc := range tests {
		got, err := NormalizeAgentSlug(tc.input)
		if err != nil {
			t.Errorf("NormalizeAgentSlug(%q) unexpected error: %v", tc.input, err)
		}
		if got != tc.want {
			t.Errorf("NormalizeAgentSlug(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestNormalizeAgentSlug_Invalid(t *testing.T) {
	tests := []string{
		"",
		"   ",
		"a",
		"has spaces",
		"with-hyphen",
		"a@b",
		"123abc",
		"_underscore",
		"toolong_abcdefghijklmnopqrstuvwxyz12345",
	}
	for _, input := range tests {
		got, err := NormalizeAgentSlug(input)
		if err == nil {
			t.Errorf("NormalizeAgentSlug(%q) expected error, got %q", input, got)
		}
	}
}

func TestMaxAgentProfilesLimit(t *testing.T) {
	limit := MaxAgentProfilesLimit()
	if limit != 12 {
		t.Errorf("MaxAgentProfilesLimit() = %d, want 12", limit)
	}
	if limit != maxAgentProfiles {
		t.Errorf("MaxAgentProfilesLimit() = %d, want %d", limit, maxAgentProfiles)
	}
}

func TestAgentBotUsername(t *testing.T) {
	tests := []struct {
		slug string
		want string
	}{
		{"default", "ai_default"},
		{"my_agent", "ai_my_agent"},
		{"  spaced  ", "ai_spaced"},
		{"", "ai_"},
	}
	for _, tc := range tests {
		got := AgentBotUsername(tc.slug)
		if got != tc.want {
			t.Errorf("AgentBotUsername(%q) = %q, want %q", tc.slug, got, tc.want)
		}
	}
}

func TestMarshalWelcomeList(t *testing.T) {
	tests := []struct {
		input []string
		want  string
	}{
		{[]string{"Hello!"}, `["Hello!"]`},
		{[]string{"你好", "Welcome"}, `["你好","Welcome"]`},
	}
	for _, tc := range tests {
		got, err := MarshalWelcomeList(tc.input)
		if err != nil {
			t.Errorf("MarshalWelcomeList(%v) unexpected error: %v", tc.input, err)
		}
		if got != tc.want {
			t.Errorf("MarshalWelcomeList(%v) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestMarshalWelcomeList_Errors(t *testing.T) {
	tests := []struct {
		name  string
		input []string
	}{
		{"empty slice", []string{}},
		{"nil slice", nil},
		{"empty string in list", []string{"hello", ""}},
		{"whitespace only", []string{"hello", "   "}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := MarshalWelcomeList(tc.input)
			if err == nil {
				t.Error("expected error for invalid input")
			}
		})
	}
}

func TestUnmarshalWelcomeList(t *testing.T) {
	tests := []struct {
		name     string
		raw      string
		fallback []string
		want     []string
	}{
		{"empty raw, use fallback", "", []string{"d1", "d2"}, []string{"d1", "d2"}},
		{"valid json", `["a","b"]`, nil, []string{"a", "b"}},
		{"json with empty strings", `["a","","b"]`, nil, []string{"a", "b"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := UnmarshalWelcomeList([]byte(tc.raw), tc.fallback)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != len(tc.want) {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Fatalf("got %v, want %v", got, tc.want)
				}
			}
		})
	}
}

func TestUnmarshalWelcomeList_Errors(t *testing.T) {
	tests := []struct {
		name string
		raw  string
	}{
		{"invalid json", `not json`},
		{"not array", `"string"`},
		{"empty after filter", `["","  "]`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := UnmarshalWelcomeList([]byte(tc.raw), nil)
			if err == nil {
				t.Error("expected error")
			}
		})
	}
}
