package worker

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestTruncate(t *testing.T) {
	tests := []struct {
		s    string
		n    int
		want string
	}{
		{"hello", 5, "hello"},
		{"hello", 10, "hello"},
		{"hello", 3, "hel"},
		{"", 5, ""},
		{"abc", 0, ""},
		{"hello world", 5, "hello"},
		// truncate works on bytes, not runes
		{"你好世界", 6, "你好"},       // each Chinese char = 3 bytes, 6 bytes = 2 chars
		{"abcdef", 6, "abcdef"},
	}
	for _, tc := range tests {
		got := truncate(tc.s, tc.n)
		if got != tc.want {
			t.Errorf("truncate(%q, %d) = %q, want %q", tc.s, tc.n, got, tc.want)
		}
	}
}

func TestTruncate_BytesExact(t *testing.T) {
	got := truncate("abcdefgh", 5)
	if got != "abcde" {
		t.Errorf("truncate = %q, want %q", got, "abcde")
	}
}

func TestTruncate_Empty(t *testing.T) {
	if got := truncate("", 10); got != "" {
		t.Errorf("empty: got %q", got)
	}
	if got := truncate("test", 0); got != "" {
		t.Errorf("zero: got %q", got)
	}
}

func TestCleanupPaths(t *testing.T) {
	dir := t.TempDir()

	f1 := filepath.Join(dir, "test1.txt")
	f2 := filepath.Join(dir, "test2.txt")
	f3 := filepath.Join(dir, "test3.txt")

	_ = os.WriteFile(f1, []byte("a"), 0644)
	_ = os.WriteFile(f2, []byte("b"), 0644)
	_ = os.WriteFile(f3, []byte("c"), 0644)

	if _, err := os.Stat(f1); os.IsNotExist(err) {
		t.Fatal("f1 should exist before cleanup")
	}

	cleanupPaths(f1, f2, f3)

	if _, err := os.Stat(f1); !os.IsNotExist(err) {
		t.Error("f1 should be removed")
	}
	if _, err := os.Stat(f2); !os.IsNotExist(err) {
		t.Error("f2 should be removed")
	}
	if _, err := os.Stat(f3); !os.IsNotExist(err) {
		t.Error("f3 should be removed")
	}
}

func TestCleanupPaths_EmptyStrings(t *testing.T) {
	cleanupPaths("", "", "")
}

func TestCleanupPaths_NonExistent(t *testing.T) {
	cleanupPaths("/nonexistent/path/file.txt")
}

func TestCleanupPaths_Mixed(t *testing.T) {
	dir := t.TempDir()
	f2 := filepath.Join(dir, "remove.txt")
	_ = os.WriteFile(f2, []byte("data"), 0644)

	cleanupPaths(filepath.Join(dir, "nonexistent.txt"))
	cleanupPaths(f2)

	if _, err := os.Stat(f2); !os.IsNotExist(err) {
		t.Error("f2 should be removed")
	}
}

func TestTranscodeJob_JSONRoundTrip(t *testing.T) {
	job := TranscodeJob{
		VideoID:    123,
		RawPath:    "/tmp/raw.mp4",
		CoverPath:  "/tmp/cover.jpg",
		RetryCount: 2,
	}
	data, err := json.Marshal(job)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var decoded TranscodeJob
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if decoded.VideoID != 123 {
		t.Errorf("VideoID = %d, want 123", decoded.VideoID)
	}
	if decoded.RawPath != "/tmp/raw.mp4" {
		t.Errorf("RawPath = %q", decoded.RawPath)
	}
	if decoded.CoverPath != "/tmp/cover.jpg" {
		t.Errorf("CoverPath = %q", decoded.CoverPath)
	}
	if decoded.RetryCount != 2 {
		t.Errorf("RetryCount = %d, want 2", decoded.RetryCount)
	}
}

func TestTranscodeJob_ZeroValues(t *testing.T) {
	job := TranscodeJob{}
	data, _ := json.Marshal(job)
	var decoded TranscodeJob
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if decoded.VideoID != 0 {
		t.Errorf("VideoID = %d, want 0", decoded.VideoID)
	}
}

func TestTranscodeJob_OmitEmpty(t *testing.T) {
	job := TranscodeJob{VideoID: 1, RawPath: "/tmp/v.mp4"}
	data, _ := json.Marshal(job)
	var m map[string]interface{}
	_ = json.Unmarshal(data, &m)
	if _, exists := m["cover_path"]; exists {
		t.Errorf("cover_path should be omitted, got %#v", m)
	}
	if m["video_id"] != float64(1) {
		t.Errorf("video_id = %v", m["video_id"])
	}
}

func TestTranscodeJob_RetryCount(t *testing.T) {
	job := TranscodeJob{VideoID: 42, RawPath: "/tmp/x.mp4", RetryCount: 3}
	if job.RetryCount != 3 {
		t.Errorf("RetryCount = %d, want 3", job.RetryCount)
	}
}

func TestFailVideo_skipped(t *testing.T) {
	t.Skip("requires DB connection")
}

func TestStartTranscodeConsumer_skipped(t *testing.T) {
	t.Skip("requires RabbitMQ and ffmpeg")
}
