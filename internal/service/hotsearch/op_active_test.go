package hotsearch

import (
	"testing"
	"time"
)

func ptrTime(t time.Time) *time.Time {
	return &t
}

// TestHotSearchOpActiveEdge covers extra edge cases for hotSearchOpActive.
func TestHotSearchOpActiveEdge(t *testing.T) {
	now := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name  string
		start *time.Time
		end   *time.Time
		want  bool
	}{
		{"both set in middle", ptrTime(now.Add(-2 * time.Hour)), ptrTime(now.Add(2 * time.Hour)), true},
		{"start just passed end", ptrTime(now.Add(-3 * time.Hour)), ptrTime(now.Add(-2 * time.Hour)), false},
		{"same start and end equals now", ptrTime(now), ptrTime(now), true},
		{"start after end", ptrTime(now.Add(2 * time.Hour)), ptrTime(now.Add(-2 * time.Hour)), false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := hotSearchOpActive(now, tc.start, tc.end)
			if got != tc.want {
				t.Errorf("hotSearchOpActive(%v, %v, %v) = %v, want %v",
					now, tc.start, tc.end, got, tc.want)
			}
		})
	}
}
