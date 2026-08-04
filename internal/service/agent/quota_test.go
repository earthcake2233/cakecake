package agent

import (
	"strings"
	"testing"
	"time"
)

func TestQuotaKey(t *testing.T) {
	s := &AgentService{Cfg: nil}
	key := s.quotaKey(123)
	if !strings.Contains(key, "mb:agent:quota:123:") {
		t.Errorf("quotaKey(123) = %q, missing expected pattern", key)
	}
	today := time.Now().Format("20060102")
	if !strings.Contains(key, today) {
		t.Errorf("quotaKey should contain today's date %s, got %q", today, key)
	}
}
