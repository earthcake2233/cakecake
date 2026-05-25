package userlevel

import "testing"

func TestFromExperience(t *testing.T) {
	cases := []struct {
		exp  uint64
		lv   int
		min  uint64
		next uint64
	}{
		{0, 1, 0, 20},
		{19, 1, 0, 20},
		{20, 2, 20, 150},
		{149, 2, 20, 150},
		{150, 3, 150, 450},
		{449, 3, 150, 450},
		{450, 4, 450, 1080},
		{1079, 4, 450, 1080},
		{1080, 5, 1080, 2880},
		{2879, 5, 1080, 2880},
		{2880, 6, 2880, 2880},
		{40747, 6, 2880, 2880},
		{5000, 6, 2880, 2880},
	}
	for _, tc := range cases {
		got := FromExperience(tc.exp)
		if got.CurrentLevel != tc.lv || got.CurrentMin != tc.min || got.NextExp != tc.next || got.CurrentExp != tc.exp {
			t.Fatalf("exp=%d: got %+v, want lv=%d min=%d next=%d", tc.exp, got, tc.lv, tc.min, tc.next)
		}
	}
}
