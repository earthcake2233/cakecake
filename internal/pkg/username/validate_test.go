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
