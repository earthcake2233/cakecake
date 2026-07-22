package data

import (
	"testing"
	"time"
)

func TestDanmakuCooldownKey(t *testing.T) {
	key := DanmakuCooldownKey(1, 100)
	want := "danmaku:cooldown:1:100"
	if key != want {
		t.Errorf("DanmakuCooldownKey = %q, want %q", key, want)
	}
}

func TestRefreshInvalidKey(t *testing.T) {
	key := RefreshInvalidKey("tok_abc123")
	want := "refresh_token:invalid:tok_abc123"
	if key != want {
		t.Errorf("RefreshInvalidKey = %q, want %q", key, want)
	}
}

func TestAdminRefreshInvalidKey(t *testing.T) {
	key := AdminRefreshInvalidKey("adm_tok_456")
	want := "refresh_token:admin:invalid:adm_tok_456"
	if key != want {
		t.Errorf("AdminRefreshInvalidKey = %q, want %q", key, want)
	}
}

func TestVideoPlayDeltaKey(t *testing.T) {
	key := VideoPlayDeltaKey(42)
	want := "videodelta:42"
	if key != want {
		t.Errorf("VideoPlayDeltaKey = %q, want %q", key, want)
	}
}

func TestRefreshInvalidTTL(t *testing.T) {
	if RefreshInvalidTTL != 30*24*time.Hour {
		t.Errorf("RefreshInvalidTTL = %v, want %v", RefreshInvalidTTL, 30*24*time.Hour)
	}
}

func TestRedisConstants(t *testing.T) {
	if PrefixDanmakuCooldown != "danmaku:cooldown:" {
		t.Errorf("PrefixDanmakuCooldown = %q", PrefixDanmakuCooldown)
	}
	if PrefixRefreshInvalid != "refresh_token:invalid:" {
		t.Errorf("PrefixRefreshInvalid = %q", PrefixRefreshInvalid)
	}
	if PrefixAdminRefreshInvalid != "refresh_token:admin:invalid:" {
		t.Errorf("PrefixAdminRefreshInvalid = %q", PrefixAdminRefreshInvalid)
	}
	if PrefixVideoPlayDelta != "videodelta:" {
		t.Errorf("PrefixVideoPlayDelta = %q", PrefixVideoPlayDelta)
	}
	if SetPlayDirty != "playcount:dirty" {
		t.Errorf("SetPlayDirty = %q", SetPlayDirty)
	}
	if ChannelDanmakuFanout != "minibili:danmaku:fanout" {
		t.Errorf("ChannelDanmakuFanout = %q", ChannelDanmakuFanout)
	}
}

func TestDanmakuCooldownKeyFormat(t *testing.T) {
	got := DanmakuCooldownKey(12345, 67890)
	if len(got) == 0 {
		t.Error("expected non-empty key")
	}
}

func TestVideoPlayDeltaKeyEdge(t *testing.T) {
	got := VideoPlayDeltaKey(0)
	want := "videodelta:0"
	if got != want {
		t.Errorf("VideoPlayDeltaKey(0) = %q, want %q", got, want)
	}
}
