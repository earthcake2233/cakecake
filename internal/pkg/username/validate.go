package username

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	MinRunes = 3
	MaxRunes = 32
)

// Valid reports whether s is a legal display username: 3–32 runes of letters
// (including CJK), digits, or underscore.
func Valid(s string) bool {
	s = strings.TrimSpace(s)
	n := utf8.RuneCountInString(s)
	if n < MinRunes || n > MaxRunes {
		return false
	}
	for _, r := range s {
		if !(unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_') {
			return false
		}
	}
	return true
}
