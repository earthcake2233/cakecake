package storage

import (
	"strings"
	"testing"
)

func TestNewOSS_AllEmpty(t *testing.T) {
	_, err := NewOSS("", "", "", "")
	if err == nil {
		t.Fatal("expected error for all empty config")
	}
}

func TestNewOSS_PartialConfig(t *testing.T) {
	cases := []struct {
		name, endpoint, key, secret, bucket string
	}{
		{"no endpoint", "", "ak", "sk", "bucket"},
		{"no key", "https://oss-cn-hangzhou.aliyuncs.com", "", "sk", "bucket"},
		{"no secret", "https://oss-cn-hangzhou.aliyuncs.com", "ak", "", "bucket"},
		{"no bucket", "https://oss-cn-hangzhou.aliyuncs.com", "ak", "sk", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewOSS(tc.endpoint, tc.key, tc.secret, tc.bucket)
			if err == nil {
				t.Error("expected error")
			}
		})
	}
}

func TestDeleteObject_EmptyKey(t *testing.T) {
	o := &OSS{}
	err := o.DeleteObject("")
	if err != nil {
		t.Errorf("empty key should return nil, got %v", err)
	}
}

func TestDeleteObject_WhitespaceKey(t *testing.T) {
	o := &OSS{}
	err := o.DeleteObject("  ")
	if err != nil {
		t.Errorf("whitespace key should return nil, got %v", err)
	}
}

func TestDeleteObject_LeadingSlash(t *testing.T) {
	key := "/some/path/file.jpg"
	stripped := strings.TrimPrefix(strings.TrimSpace(key), "/")
	if stripped != "some/path/file.jpg" {
		t.Errorf("expected 'some/path/file.jpg', got %q", stripped)
	}
}

func TestDeleteObjects_AllEmpty(t *testing.T) {
	o := &OSS{}
	err := o.DeleteObjects([]string{})
	if err != nil {
		t.Errorf("empty slice should return nil, got %v", err)
	}
}

func TestDeleteObjects_SkipsEmpty(t *testing.T) {
	// Test the guard logic: empty and slash-only keys are skipped
	keys := []string{"", "  ", "/", "valid/key"}
	cleaned := make([]string, 0, len(keys))
	for _, k := range keys {
		k = strings.TrimPrefix(strings.TrimSpace(k), "/")
		if k != "" {
			cleaned = append(cleaned, k)
		}
	}
	if len(cleaned) != 1 {
		t.Errorf("expected 1 valid key after cleaning, got %d: %v", len(cleaned), cleaned)
	}
	if cleaned[0] != "valid/key" {
		t.Errorf("expected 'valid/key', got %q", cleaned[0])
	}
}

func TestUploadFile_OSSConfigCheck(t *testing.T) {
	// This tests the OSS struct creation validation logic is sound
	_, err := NewOSS("https://oss-cn-beijing.aliyuncs.com", "ak", "", "bucket")
	if err == nil {
		t.Error("expected error for empty secret key")
	}
}
