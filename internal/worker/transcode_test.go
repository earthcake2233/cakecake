package worker

import (
	"testing"
	"os"
	"path/filepath"
)

func TestTranscodeJob_Defaults(t *testing.T) {
	job := TranscodeJob{VideoID: 42, RawPath: "/tmp/video.mp4"}
	if job.VideoID != 42 {
		t.Errorf("VideoID = %d, want 42", job.VideoID)
	}
	if job.RawPath != "/tmp/video.mp4" {
		t.Errorf("RawPath = %q, want /tmp/video.mp4", job.RawPath)
	}
	if job.RetryCount != 0 {
		t.Errorf("RetryCount default should be 0, got %d", job.RetryCount)
	}
}

func TestTranscodeJob_CoverPath(t *testing.T) {
	job := TranscodeJob{VideoID: 1, RawPath: "/v/raw.mp4", CoverPath: "/v/cover.jpg"}
	if job.CoverPath != "/v/cover.jpg" {
		t.Errorf("CoverPath = %q, want /v/cover.jpg", job.CoverPath)
	}
}

func TestCleanupPaths_AllExist(t *testing.T) {
	dir := t.TempDir()
	f1 := filepath.Join(dir, "a.txt")
	f2 := filepath.Join(dir, "b.txt")
	_ = os.WriteFile(f1, []byte("a"), 0644)
	_ = os.WriteFile(f2, []byte("b"), 0644)

	cleanupPaths(f1, f2)
	if _, err := os.Stat(f1); !os.IsNotExist(err) {
		t.Error("f1 not removed")
	}
	if _, err := os.Stat(f2); !os.IsNotExist(err) {
		t.Error("f2 not removed")
	}
}

func TestCleanupPaths_PartialExist(t *testing.T) {
	dir := t.TempDir()
	f1 := filepath.Join(dir, "a.txt")
	f2 := filepath.Join(dir, "b.txt")
	_ = os.WriteFile(f1, []byte("a"), 0644)

	cleanupPaths(f1, f2)
	if _, err := os.Stat(f1); !os.IsNotExist(err) {
		t.Error("f1 not removed")
	}
}

func TestTranscodeJob_MaxRetry(t *testing.T) {
	job := TranscodeJob{RetryCount: 3}
	if job.RetryCount != 3 {
		t.Errorf("RetryCount should persist")
	}
}
