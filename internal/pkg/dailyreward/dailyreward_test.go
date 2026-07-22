package dailyreward

import (
	"testing"
	"time"
	"regexp"
)

func TestTodayDate_Format(t *testing.T) {
	date := TodayDate()
	// Must be non-empty
	if date == "" {
		t.Error("TodayDate() returned empty string")
	}
	// Must match YYYY-MM-DD
	matched, err := regexp.MatchString(`^\d{4}-\d{2}-\d{2}$`, date)
	if err != nil {
		t.Fatal(err)
	}
	if !matched {
		t.Errorf("TodayDate() = %q; does not match YYYY-MM-DD", date)
	}
	// Must be parseable
	parsed, err := time.Parse("2006-01-02", date)
	if err != nil {
		t.Errorf("TodayDate() = %q; cannot parse: %v", date, err)
	}
	// Must be close to now (within 24h)
	if diff := time.Since(parsed); diff < -24*time.Hour || diff > 24*time.Hour {
		t.Errorf("TodayDate() = %q; parsed time is too far from now", date)
	}
}

// DB-dependent functions are skipped.
func TestMarkLogin_Skipped(t *testing.T) {
	t.Skip("requires DB")
}

func TestMarkWatch_Skipped(t *testing.T) {
	t.Skip("requires DB")
}

func TestGrantCoinExp_Skipped(t *testing.T) {
	t.Skip("requires DB")
}

func TestBuildSnapshot_Skipped(t *testing.T) {
	t.Skip("requires DB")
}

func TestCoinProgress_Skipped(t *testing.T) {
	t.Skip("requires DB")
}
