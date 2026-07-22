package bvid

import (
	"math"
	"testing"
)

func TestEncode_Zero(t *testing.T) {
	if got := Encode(0); got != "" {
		t.Fatalf("Encode(0) = %q, want empty string", got)
	}
}

func TestEncode_One(t *testing.T) {
	if got := Encode(1); got != "BV1" {
		t.Fatalf("Encode(1) = %q, want \"BV1\"", got)
	}
}

func TestEncode_Small(t *testing.T) {
	if got := Encode(12345); got != "BV12345" {
		t.Fatalf("Encode(12345) = %q, want \"BV12345\"", got)
	}
}

func TestEncode_Large(t *testing.T) {
	if got := Encode(math.MaxUint64); got != "BV18446744073709551615" {
		t.Fatalf("Encode(MaxUint64) = %q, want \"BV18446744073709551615\"", got)
	}
}
