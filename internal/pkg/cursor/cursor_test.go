package cursor

import (
	"testing"
	"time"
)

func TestEncodeDecode_RoundTrip(t *testing.T) {
	now := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)
	c := VideoListC{
		PlayCount:    1000,
		CreatedAt:    now,
		DanmakuCount: 500,
		ID:           42,
	}

	encoded := Encode(c)
	if encoded == "" {
		t.Fatal("Encode returned empty string")
	}

	decoded, err := Decode(encoded)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}
	if decoded == nil {
		t.Fatal("Decode returned nil")
	}

	if decoded.PlayCount != c.PlayCount {
		t.Fatalf("PlayCount = %d, want %d", decoded.PlayCount, c.PlayCount)
	}
	if !decoded.CreatedAt.Equal(c.CreatedAt) {
		t.Fatalf("CreatedAt = %v, want %v", decoded.CreatedAt, c.CreatedAt)
	}
	if decoded.DanmakuCount != c.DanmakuCount {
		t.Fatalf("DanmakuCount = %d, want %d", decoded.DanmakuCount, c.DanmakuCount)
	}
	if decoded.ID != c.ID {
		t.Fatalf("ID = %d, want %d", decoded.ID, c.ID)
	}
}

func TestDecode_Empty(t *testing.T) {
	got, err := Decode("")
	if err != nil {
		t.Fatalf("Decode(\"\") err = %v, want nil", err)
	}
	if got != nil {
		t.Fatalf("Decode(\"\") = %v, want nil", got)
	}
}

func TestDecode_InvalidBase64(t *testing.T) {
	_, err := Decode("!!!invalid-base64!!!")
	if err == nil {
		t.Fatal("Decode with invalid base64 should return error")
	}
}

func TestDecode_InvalidJSON(t *testing.T) {
	_, err := Decode("aW52YWxpZCBqc29u") // decodes to "invalid json"
	if err == nil {
		t.Fatal("Decode with invalid JSON should return error")
	}
}
