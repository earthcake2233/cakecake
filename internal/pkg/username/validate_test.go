package username

import (
	"strings"
	"testing"
)

func TestValid(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want bool
	}{
		{"ascii", "alice_01", true},
		{"chinese", "蛋糕用户", true},
		{"mixed", "用户A_1", true},
		{"too short han", "张三", false},
		{"single han", "张", false},
		{"space", "a b", false},
		{"symbol", "用户@", false},
		{"empty", "", false},
		{"max runes", strings.Repeat("你", MaxRunes), true},
		{"too many runes", strings.Repeat("你", MaxRunes+1), false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := Valid(tc.in); got != tc.want {
				t.Fatalf("Valid(%q) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}

func TestValid_EdgeCases(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want bool
	}{
		{"pure digits", "1234567", true},
		{"underscore only", "_______", true},
		{"leading underscore", "_alice", true},
		{"trailing underscore", "alice_", true},
		{"newline in middle", "ali\nce", false},
		{"tab in middle", "ali\tce", false},
		{"unicode letter", "éleve", true},
		{"korean chars", "한글유저", true},
		{"japanese chars", "日本語", true},
		{"russian chars", "Пользователь", true},
		{"min runes - 3", "abc", true},
		{"just under min", "ab", false},
		{"just over max", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", false}, // 33 a's
		{"exactly max", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", true}, // 32 a's
		{"mixed case letters", "AliceUserName", true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := Valid(tc.in); got != tc.want {
				t.Errorf("Valid(%q) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}

func TestValid_SpacesTrimmed(t *testing.T) {
	// Leading/trailing spaces should be trimmed
	if got := Valid("  alice  "); got != true {
		t.Errorf("Valid with spaces should trim and be valid, got %v", got)
	}
}

func TestValid_OnlySpaces(t *testing.T) {
	if got := Valid("   "); got != false {
		t.Errorf("Valid with only spaces should be false, got %v", got)
	}
}
