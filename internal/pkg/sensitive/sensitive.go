package sensitive

import (
	"bufio"
	"os"
	"strings"
	"sync"
	"unicode"

	"go.uber.org/zap"
)

// Filter loads sensitive substrings from a file (Rule R-BIZ-5).
// Empty or missing file => reject all content (returns ErrBlocked when words empty after load attempt).
type Filter struct {
	mu     sync.RWMutex
	words  []string
	loaded bool
	path   string
	log    *zap.Logger
}

var blockedMarker = struct{}{}

// ErrBlocked indicates content must be rejected (hit sensitive word or filter disabled).
type ErrBlocked struct{}

func (ErrBlocked) Error() string { return "sensitive content blocked" }

// NewFilter creates a filter; call Reload before use.
func NewFilter(path string, lg *zap.Logger) *Filter {
	return &Filter{path: path, log: lg}
}

// Reload reads the word list from disk.
func (f *Filter) Reload() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	data, err := os.ReadFile(f.path)
	if err != nil {
		f.words = nil
		f.loaded = true
		f.log.Error("sensitive word file missing or unreadable; all danmaku will be rejected",
			zap.String("path", f.path), zap.Error(err))
		return err
	}
	var words []string
	sc := bufio.NewScanner(strings.NewReader(string(data)))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		words = append(words, strings.ToLower(line))
	}
	f.words = words
	f.loaded = true
	if len(words) == 0 {
		f.log.Error("sensitive word list is empty after load; all danmaku will be rejected",
			zap.String("path", f.path))
	}
	return nil
}

func normalize(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		if unicode.IsSpace(r) {
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// Check returns nil if content is allowed, ErrBlocked if it must be rejected.
func (f *Filter) Check(content string) error {
	f.mu.RLock()
	defer f.mu.RUnlock()
	if !f.loaded || len(f.words) == 0 {
		return ErrBlocked{}
	}
	n := normalize(content)
	for _, w := range f.words {
		if w == "" {
			continue
		}
		if strings.Contains(n, normalize(w)) {
			return ErrBlocked{}
		}
	}
	return nil
}
