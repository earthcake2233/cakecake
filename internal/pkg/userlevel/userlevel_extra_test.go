package userlevel

import (
	"testing"
)

func TestFromExperience_AllLevelBoundaries(t *testing.T) {
	cases := []struct {
		exp  uint64
		lv   int
		min  uint64
		next uint64
	}{
		{0, 1, 0, 20},
		{1, 1, 0, 20},
		{19, 1, 0, 20},
		{20, 2, 20, 150},
		{21, 2, 20, 150},
		{149, 2, 20, 150},
		{150, 3, 150, 450},
		{151, 3, 150, 450},
		{449, 3, 150, 450},
		{450, 4, 450, 1080},
		{451, 4, 450, 1080},
		{1079, 4, 450, 1080},
		{1080, 5, 1080, 2880},
		{1081, 5, 1080, 2880},
		{2879, 5, 1080, 2880},
		{2880, 6, 2880, 2880},
		{2881, 6, 2880, 2880},
		{999999, 6, 2880, 2880},
	}
	for _, tc := range cases {
		got := FromExperience(tc.exp)
		if got.CurrentLevel != tc.lv {
			t.Errorf("exp=%d: got level %d, want %d", tc.exp, got.CurrentLevel, tc.lv)
		}
		if got.CurrentMin != tc.min {
			t.Errorf("exp=%d: got min %d, want %d", tc.exp, got.CurrentMin, tc.min)
		}
		if got.NextExp != tc.next {
			t.Errorf("exp=%d: got nextExp %d, want %d", tc.exp, got.NextExp, tc.next)
		}
		if got.CurrentExp != tc.exp {
			t.Errorf("exp=%d: got currentExp %d, want %d", tc.exp, got.CurrentExp, tc.exp)
		}
	}
}

func TestFromExperience_InfoStruct(t *testing.T) {
	info := FromExperience(100)
	if info.CurrentLevel <= 0 {
		t.Errorf("expected positive level, got %d", info.CurrentLevel)
	}
	if info.CurrentExp != 100 {
		t.Errorf("expected CurrentExp=100, got %d", info.CurrentExp)
	}
	if info.CurrentMin >= info.NextExp && info.CurrentLevel < MaxLevel {
		t.Errorf("min %d should be < next %d for non-max level", info.CurrentMin, info.NextExp)
	}
}

func TestMaxLevelConstant(t *testing.T) {
	if MaxLevel != 6 {
		t.Errorf("MaxLevel = %d, want 6", MaxLevel)
	}
	if len(Thresholds) != MaxLevel {
		t.Errorf("Thresholds length %d != MaxLevel %d", len(Thresholds), MaxLevel)
	}
}

func TestThresholds_OrderAndValues(t *testing.T) {
	expected := []uint64{0, 20, 150, 450, 1080, 2880}
	if len(Thresholds) != len(expected) {
		t.Fatalf("Thresholds length = %d, want %d", len(Thresholds), len(expected))
	}
	for i := range expected {
		if Thresholds[i] != expected[i] {
			t.Errorf("Thresholds[%d] = %d, want %d", i, Thresholds[i], expected[i])
		}
	}
	for i := 1; i < len(Thresholds); i++ {
		if Thresholds[i] <= Thresholds[i-1] {
			t.Errorf("Thresholds not strictly increasing at index %d", i)
		}
	}
}

func TestBatchCurrentLevels_NilDB(t *testing.T) {
	got := BatchCurrentLevels(nil, []uint64{1, 2, 3})
	if len(got) != 0 {
		t.Errorf("expected empty map for nil db, got %v", got)
	}
}

func TestBatchCurrentLevels_EmptyUIDs(t *testing.T) {
	got := BatchCurrentLevels(nil, nil)
	if len(got) != 0 {
		t.Errorf("expected empty map for nil uids, got %v", got)
	}
}

func TestBatchCurrentLevels_AllZeroUIDs(t *testing.T) {
	got := BatchCurrentLevels(nil, []uint64{0, 0, 0})
	if len(got) != 0 {
		t.Errorf("expected empty map for all-zero uids, got %v", got)
	}
}
