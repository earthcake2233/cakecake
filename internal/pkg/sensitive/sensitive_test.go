package sensitive

import (
	"os"
	"testing"

	"go.uber.org/zap"
)

func TestFilter_Reload_WithValidWords(t *testing.T) {
	lg, _ := zap.NewDevelopment()
	defer lg.Sync()

	content := "badword1\n# comment\nbadword2\n\n"
	f, err := os.CreateTemp("", "sensitive-*.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())

	if _, err := f.WriteString(content); err != nil {
		t.Fatal(err)
	}
	f.Close()

	filter := NewFilter(f.Name(), lg)
	if err := filter.Reload(); err != nil {
		t.Fatalf("Reload() = %v; want nil", err)
	}

	// Matching content should be blocked
	if err := filter.Check("contains badword1 here"); err == nil {
		t.Error("Check() = nil; want ErrBlocked for matching content")
	}

	// Non-matching content should be allowed
	if err := filter.Check("clean content"); err != nil {
		t.Errorf("Check() = %v; want nil for clean content", err)
	}
}

func TestFilter_Reload_EmptyWordsBlocksAll(t *testing.T) {
	lg, _ := zap.NewDevelopment()
	defer lg.Sync()

	f, err := os.CreateTemp("", "sensitive-*.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())

	// Only comments and blank lines
	if _, err := f.WriteString("# just a comment\n\n# another comment\n"); err != nil {
		t.Fatal(err)
	}
	f.Close()

	filter := NewFilter(f.Name(), lg)
	if err := filter.Reload(); err != nil {
		t.Fatalf("Reload() = %v; want nil", err)
	}

	// Empty word list should block all content
	if err := filter.Check("anything"); err == nil {
		t.Error("Check() = nil; want ErrBlocked when words list is empty")
	}
}

func TestFilter_MissingFile_ReloadError(t *testing.T) {
	lg, _ := zap.NewDevelopment()
	defer lg.Sync()

	filter := NewFilter("/nonexistent/path/sensitive.txt", lg)
	if err := filter.Reload(); err == nil {
		t.Fatal("Reload() = nil; want error for missing file")
	}

	// Missing file should also block all on Check
	if err := filter.Check("any content"); err == nil {
		t.Error("Check() = nil; want ErrBlocked when file is missing")
	}
}

func TestFilter_Check_BeforeReload(t *testing.T) {
	lg, _ := zap.NewDevelopment()
	defer lg.Sync()

	filter := NewFilter("some/path.txt", lg)

	// Before Reload, Check should block everything (loaded is false)
	if err := filter.Check("anything"); err == nil {
		t.Error("Check() = nil; want ErrBlocked before Reload")
	}
}
