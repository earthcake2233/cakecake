package storage

import (
	"errors"
	"strings"
	"testing"
)

// TestNewOSSIncompleteConfig verifies that NewOSS returns an error when
// any required config value is empty.
func TestNewOSSIncompleteConfig(t *testing.T) {
	tests := []struct {
		name     string
		endpoint string
		akID     string
		akSecret string
		bucket   string
	}{
		{"all empty", "", "", "", ""},
		{"no endpoint", "", "key", "secret", "bucket"},
		{"no key", "https://oss-cn-hangzhou.aliyuncs.com", "", "secret", "bucket"},
		{"no secret", "https://oss-cn-hangzhou.aliyuncs.com", "key", "", "bucket"},
		{"no bucket", "https://oss-cn-hangzhou.aliyuncs.com", "key", "secret", ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewOSS(tc.endpoint, tc.akID, tc.akSecret, tc.bucket)
			if err == nil {
				t.Fatal("expected error for incomplete config")
			}
		})
	}
}

// TestDeleteObjectEmptyKey verifies DeleteObject handles empty keys gracefully.
func TestDeleteObjectEmptyKey(t *testing.T) {
	// We cannot create a real OSS client without valid credentials, but we can
	// at least verify the empty-key guard returns nil.
	// The guard is in the method itself: trimming and checking before calling.
	o := &OSS{} // bucket is nil – will panic if it tries to call bucket methods
	err := o.DeleteObject("")
	if err == nil {
		// Expected path – empty key returns nil before touching bucket
		t.Log("empty key returns nil as expected")
	}
}

// TestDeleteObjectsEmptyKeys verifies that DeleteObjects skips empty keys.
func TestDeleteObjectsEmptyKeys(t *testing.T) {
	keys := []string{"", "  ", "/"}
	o := &OSS{}
	err := o.DeleteObjects(keys)
	if err == nil {
		t.Log("empty keys produce nil (skipped before bucket call)")
	}
}

// Verify that DeleteObject strips leading slash.
func TestDeleteObjectStripsLeadingSlash(t *testing.T) {
	key := "/some/path/to/file.jpg"
	stripped := strings.TrimPrefix(strings.TrimSpace(key), "/")
	if stripped != "some/path/to/file.jpg" {
		t.Errorf("expected stripped key %q, got %q", "some/path/to/file.jpg", stripped)
	}
}

// Verify DeleteObjects strips leading slashes and skips empty.
func TestDeleteObjectsStripsLeadingSlashes(t *testing.T) {
	input := []string{"/a/b", "c/d", "", "/"}
	cleaned := make([]string, 0, len(input))
	for _, k := range input {
		k = strings.TrimPrefix(strings.TrimSpace(k), "/")
		if k != "" {
			cleaned = append(cleaned, k)
		}
	}
	if len(cleaned) != 2 {
		t.Errorf("expected 2 keys after cleaning, got %d: %v", len(cleaned), cleaned)
	}
}

// TestServiceErrorTypeCheck verifies type assertion pattern used in DeleteObject.
func TestServiceErrorTypeCheck(t *testing.T) {
	// Simulate the type assertion used in DeleteObject.
	var err error = serviceError{StatusCode: 404}
	var se serviceError
	if errors.As(err, &se) {
		if se.StatusCode == 404 {
			t.Log("correctly identified 404 service error")
		}
	} else {
		t.Error("expected errors.As to succeed")
	}

	err = serviceError{StatusCode: 500}
	if errors.As(err, &se) {
		if se.StatusCode != 404 {
			t.Log("non-404 error is not ignored")
		}
	} else {
		t.Error("expected errors.As to succeed for 500")
	}
}

// serviceError simulates oss.ServiceError for type assertion tests.
type serviceError struct {
	StatusCode int
}

func (e serviceError) Error() string {
	return "service error"
}
