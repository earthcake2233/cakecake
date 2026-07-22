package useravatar

import (
	"testing"
	"time"

	"minibili/internal/model"
)

func TestPublicURL_NilUser(t *testing.T) {
	if PublicURL(nil) != "" {
		t.Fatal("expected empty for nil user")
	}
}

func TestPublicURL_AnonymizedUser(t *testing.T) {
	now := time.Now()
	u := &model.User{
		AvatarURL:    "https://example.com/avatar.jpg",
		AnonymizedAt: &now,
	}
	if PublicURL(u) != "" {
		t.Fatal("expected empty for anonymized user")
	}
}

func TestPublicURL_EmptyURL(t *testing.T) {
	u := &model.User{
		AvatarURL: "",
		UpdatedAt: time.Unix(1000, 0),
	}
	if PublicURL(u) != "" {
		t.Fatal("expected empty for empty avatar URL")
	}
}

func TestPublicURL_WhitespaceURL(t *testing.T) {
	u := &model.User{
		AvatarURL: "   ",
	}
	if PublicURL(u) != "" {
		t.Fatal("expected empty for whitespace avatar URL")
	}
}

func TestPublicURL_NoUpdatedAt(t *testing.T) {
	u := &model.User{
		AvatarURL: "https://example.com/avatar.jpg",
	}
	got := PublicURL(u)
	want := "https://example.com/avatar.jpg"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestPublicURL_WithExistingQueryParams(t *testing.T) {
	ts := time.Unix(1700000000, 0)
	u := &model.User{
		AvatarURL: "https://example.com/avatar.jpg?x=1",
		UpdatedAt: ts,
	}
	got := PublicURL(u)
	want := "https://example.com/avatar.jpg?x=1&v=1700000000"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestPublicURL_WithMultipleExistingParams(t *testing.T) {
	ts := time.Unix(1700000000, 0)
	u := &model.User{
		AvatarURL: "https://example.com/avatar.jpg?a=1&b=2",
		UpdatedAt: ts,
	}
	got := PublicURL(u)
	want := "https://example.com/avatar.jpg?a=1&b=2&v=1700000000"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestPublicURL_CacheBustValueChanges(t *testing.T) {
	ts1 := time.Unix(1000, 0)
	ts2 := time.Unix(2000, 0)
	u := &model.User{
		AvatarURL: "https://example.com/avatar.jpg",
	}
	u.UpdatedAt = ts1
	v1 := PublicURL(u)
	u.UpdatedAt = ts2
	v2 := PublicURL(u)
	if v1 == v2 {
		t.Fatal("expected different URLs for different update times")
	}
	if v1 != "https://example.com/avatar.jpg?v=1000" {
		t.Fatalf("v1 = %q, want https://example.com/avatar.jpg?v=1000", v1)
	}
	if v2 != "https://example.com/avatar.jpg?v=2000" {
		t.Fatalf("v2 = %q, want https://example.com/avatar.jpg?v=2000", v2)
	}
}

func TestPublicURL_AllFieldsUser(t *testing.T) {
	ts := time.Unix(987654321, 0)
	u := &model.User{
		ID:        42,
		Username:  "testuser",
		Nickname:  "Test",
		AvatarURL: "https://cdn.example.com/avatars/42.jpg",
		UpdatedAt: ts,
	}
	got := PublicURL(u)
	want := "https://cdn.example.com/avatars/42.jpg?v=987654321"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}
