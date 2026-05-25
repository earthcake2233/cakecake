package searchhist

import (
	"strings"
	"unicode"
)

// Norm returns a canonical key for deduplication (lowercase, no spaces).
func Norm(keyword string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(strings.TrimSpace(keyword)) {
		if !unicode.IsSpace(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}
