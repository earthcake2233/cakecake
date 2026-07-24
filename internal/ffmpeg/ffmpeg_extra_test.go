package ffmpeg

import (
	"os"
	"testing"
)

func TestInit_OverrideBothToSameValue(t *testing.T) {
	origFfprobe := ffprobeExe
	origFfmpeg := ffmpegExe
	defer func() {
		ffprobeExe = origFfprobe
		ffmpegExe = origFfmpeg
	}()
	Init("/opt/bin/ffmpeg", "/opt/bin/ffmpeg")
	if FFprobeExe() != "/opt/bin/ffmpeg" {
		t.Errorf("FFprobeExe = %q, want /opt/bin/ffmpeg", FFprobeExe())
	}
	if FFmpegExe() != "/opt/bin/ffmpeg" {
		t.Errorf("FFmpegExe = %q, want /opt/bin/ffmpeg", FFmpegExe())
	}
}

func TestInit_WhitespaceOnly(t *testing.T) {
	origFfprobe := ffprobeExe
	origFfmpeg := ffmpegExe
	defer func() {
		ffprobeExe = origFfprobe
		ffmpegExe = origFfmpeg
	}()
	Init("   ", "   ")
	if FFprobeExe() != "ffprobe" {
		t.Errorf("FFprobeExe with whitespace = %q, want ffprobe", FFprobeExe())
	}
	if FFmpegExe() != "ffmpeg" {
		t.Errorf("FFmpegExe with whitespace = %q, want ffmpeg", FFmpegExe())
	}
}

func TestInit_EmptyAfterOverrideKeepsOverride(t *testing.T) {
	origFfprobe := ffprobeExe
	origFfmpeg := ffmpegExe
	defer func() {
		ffprobeExe = origFfprobe
		ffmpegExe = origFfmpeg
	}()
	Init("/custom/probe", "/custom/mpeg")
	Init("", "")
	if FFprobeExe() != "/custom/probe" {
		t.Errorf("FFprobeExe after Init('','') = %q, want /custom/probe", FFprobeExe())
	}
	if FFmpegExe() != "/custom/mpeg" {
		t.Errorf("FFmpegExe after Init('','') = %q, want /custom/mpeg", FFmpegExe())
	}
}

func TestIsPermanentTranscodeFailure_EdgeCases(t *testing.T) {
	tests := []struct {
		stderr string
		want   bool
	}{
		{"ERROR: Invalid data found when processing input at offset 1234", true},
		{"ffmpeg: Unsupported codec 'vp9' in stream", true},
		{"open /path/to/file: No such file or directory", true},
		{"Permission denied accessing /etc/shadow", true},
		{"Error while decoding stream #0:0", false},
		{"Conversion failed!", false},
		{"invalid data", false},
		{"no such file or directory", false},
		{"permission denied", false},
		{"line1\nInvalid data found when processing input\nline3", true},
		{"line1\nNo such file or directory\nline3", true},
		{"Unsupported codec: h264 is not supported", true},
		{"Prefix Invalid data found when processing input suffix", true},
	}
	for _, tc := range tests {
		got := IsPermanentTranscodeFailure(tc.stderr)
		if got != tc.want {
			t.Errorf("IsPermanentTranscodeFailure(%q) = %v, want %v", tc.stderr, got, tc.want)
		}
	}
}

func TestIsPermanentTranscodeFailure_MultiplePatterns(t *testing.T) {
	got := IsPermanentTranscodeFailure("Invalid data found when processing input\nNo such file or directory")
	if !got {
		t.Error("expected true when both patterns match")
	}
}

func TestIsPermanentTranscodeFailure_ExactMatches(t *testing.T) {
	patterns := []string{
		"Invalid data found when processing input",
		"Unsupported codec",
		"No such file or directory",
		"Permission denied",
	}
	for _, p := range patterns {
		if !IsPermanentTranscodeFailure(p) {
			t.Errorf("IsPermanentTranscodeFailure(%q) should be true", p)
		}
	}
}

func TestCheckFFprobe_Skip(t *testing.T) {
	t.Skip("CheckFFprobe requires ffprobe on PATH")
	_ = CheckFFprobe()
}

func TestProbeDurationSeconds_Skip(t *testing.T) {
	t.Skip("ProbeDurationSeconds requires ffprobe on PATH")
	_, _ = ProbeDurationSeconds("/nonexistent/file.mp4")
}

func TestTranscodeToH264MP4_Skip(t *testing.T) {
	t.Skip("TranscodeToH264MP4 requires ffmpeg on PATH")
	_, _ = TranscodeToH264MP4("/nonexistent/input.mp4", "/tmp/output.mp4")
}

func TestScreenshotJPEG_Skip(t *testing.T) {
	t.Skip("ScreenshotJPEG requires ffmpeg on PATH")
	_, _ = ScreenshotJPEG("/nonexistent/input.mp4", "/tmp/screenshot.jpg", 10.5)
}

func TestVarDefaults(t *testing.T) {
	if ffprobeExe != "ffprobe" {
		t.Errorf("ffprobeExe default = %q, want ffprobe", ffprobeExe)
	}
	if ffmpegExe != "ffmpeg" {
		t.Errorf("ffmpegExe default = %q, want ffmpeg", ffmpegExe)
	}
}

func TestInit_EnvironmentVariableSimulation(t *testing.T) {
	origFfprobe := ffprobeExe
	origFfmpeg := ffmpegExe
	defer func() {
		ffprobeExe = origFfprobe
		ffmpegExe = origFfmpeg
	}()
	ffprobeFromEnv := os.Getenv("FFPROBE_PATH")
	ffmpegFromEnv := os.Getenv("FFMPEG_PATH")
	Init(ffprobeFromEnv, ffmpegFromEnv)
	_ = FFprobeExe()
	_ = FFmpegExe()
}

func TestFFprobeExe_AfterInit(t *testing.T) {
	orig := ffprobeExe
	defer func() { ffprobeExe = orig }()
	Init("/custom/probe", "")
	got := FFprobeExe()
	if got != "/custom/probe" {
		t.Errorf("FFprobeExe = %q, want /custom/probe", got)
	}
}

func TestFFmpegExe_AfterInit(t *testing.T) {
	orig := ffmpegExe
	defer func() { ffmpegExe = orig }()
	Init("", "/custom/mpeg")
	got := FFmpegExe()
	if got != "/custom/mpeg" {
		t.Errorf("FFmpegExe = %q, want /custom/mpeg", got)
	}
}
