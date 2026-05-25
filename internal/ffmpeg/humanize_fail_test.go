package ffmpeg

import (
	"strings"
	"testing"
)

func TestHumanizeFailReason_Empty(t *testing.T) {
	if HumanizeFailReason("") != "" || HumanizeFailReason("  \n") != "" {
		t.Fatal("expected empty for blank input")
	}
}

func TestHumanizeFailReason_FriendlyChinese(t *testing.T) {
	msg := "OSS 未配置"
	if got := HumanizeFailReason(msg); got != msg {
		t.Fatalf("got %q want %q", got, msg)
	}
}

func TestHumanizeFailReason_FFmpegBanner(t *testing.T) {
	raw := "ffmpeg version 8.1.1-full_build-www.gyan.dev Copyright (c) 2000-2026\n" +
		"built with gcc 15.2.0\n" +
		"configuration: --enable-gpl --enable-version3 --enable-static\n"
	got := HumanizeFailReason(raw)
	if got == "" || strings.Contains(got, "--enable-gpl") || strings.Contains(got, "ffmpeg version") {
		t.Fatalf("expected friendly message, got %q", got)
	}
	if got != defaultTranscodeFail {
		t.Fatalf("expected default transcode message for pure banner, got %q", got)
	}
}

func TestHumanizeFailReason_InvalidData(t *testing.T) {
	raw := "ffmpeg version 6.0\n[in#0 @ ...] Error opening input: Invalid data found when processing input\n"
	got := HumanizeFailReason(raw)
	if !strings.Contains(got, "无法识别") {
		t.Fatalf("expected invalid-data hint, got %q", got)
	}
}
