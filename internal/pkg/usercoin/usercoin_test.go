package usercoin

import (
	"testing"
	"time"
)

func TestBalanceFloat(t *testing.T) {
	cases := []struct {
		tenths int64
		want   float64
	}{
		{230, 23.0},
		{0, 0},
		{5, 0.5},
		{1, 0.1},
		{10, 1.0},
		{100, 10.0},
	}
	for _, tc := range cases {
		if got := BalanceFloat(tc.tenths); got != tc.want {
			t.Errorf("BalanceFloat(%d) = %f; want %f", tc.tenths, got, tc.want)
		}
	}
}

func TestCostTenths(t *testing.T) {
	cases := []struct {
		amount int
		want   int64
	}{
		{1, 10},
		{2, 20},
		{0, 0},
		{5, 50},
	}
	for _, tc := range cases {
		if got := CostTenths(tc.amount); got != tc.want {
			t.Errorf("CostTenths(%d) = %d; want %d", tc.amount, got, tc.want)
		}
	}
}

func TestCreatorShareTenths(t *testing.T) {
	cases := []struct {
		amount int
		want   int64
	}{
		{1, 1},
		{2, 2},
		{0, 0},
		{10, 10},
	}
	for _, tc := range cases {
		if got := CreatorShareTenths(tc.amount); got != tc.want {
			t.Errorf("CreatorShareTenths(%d) = %d; want %d", tc.amount, got, tc.want)
		}
	}
}

func TestRecordLedgerAt_SkipsOnZeroDelta(t *testing.T) {
	if err := RecordLedgerAt(nil, 1, 0, "test_reason", 0, time.Now()); err != nil {
		t.Errorf("RecordLedgerAt with delta=0 should skip; got %v", err)
	}
}
