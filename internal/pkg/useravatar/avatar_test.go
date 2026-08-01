package useravatar

import (
	"cakecake/internal/model/user"
	"testing"
	"time"
)

func TestPublicURLCacheBust(t *testing.T) {
	ts := time.Unix(1700000000, 0)
	u := &user.User{
		AvatarURL: "https://bucket.oss.com/avatars/1.jpg",
		UpdatedAt: ts,
	}
	got := PublicURL(u)
	want := "https://bucket.oss.com/avatars/1.jpg?v=1700000000"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestPublicURLEmpty(t *testing.T) {
	if PublicURL(&user.User{}) != "" {
		t.Fatal("expected empty")
	}
}
