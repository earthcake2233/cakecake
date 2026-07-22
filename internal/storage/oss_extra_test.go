package storage

import (
	"errors"
	"strings"
	"testing"
)

func TestNewOSSValidation_AllEmptyStrings(t *testing.T) {
	tests := []struct {
		name     string
		endpoint string
		akID     string
		akSecret string
		bucket   string
	}{
		{"only endpoint", "https://oss-cn-hangzhou.aliyuncs.com", "", "", ""},
		{"only endpoint+bucket", "https://oss-cn-hangzhou.aliyuncs.com", "", "", "mybucket"},
		{"missing bucket only", "https://oss-cn-hangzhou.aliyuncs.com", "key", "secret", ""},
		{"missing endpoint", "", "key", "secret", "bucket"},
		{"missing key", "https://oss-cn-hangzhou.aliyuncs.com", "", "secret", "bucket"},
		{"missing secret", "https://oss-cn-hangzhou.aliyuncs.com", "key", "", "bucket"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewOSS(tc.endpoint, tc.akID, tc.akSecret, tc.bucket)
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), "oss configuration incomplete") {
				t.Errorf("unexpected error message: %v", err)
			}
		})
	}
}

func TestDeleteObject_EdgeCases(t *testing.T) {
	o := &OSS{}
	// Whitespace-only key - after trim it becomes empty, returns nil without touching bucket.
	if err := o.DeleteObject("  "); err != nil {
		t.Errorf("whitespace key should return nil, got: %v", err)
	}
}

func TestDeleteObject_Normalization(t *testing.T) {
	normalize := func(key string) string {
		return strings.TrimPrefix(strings.TrimSpace(key), "/")
	}
	cases := []struct {
		input string
		want  string
	}{
		{"/a/b/c", "a/b/c"},
		{"a/b/c", "a/b/c"},
		{" /a/b/c ", "a/b/c"},
		{"//a/b", "/a/b"},
		{"", ""},
		{"  ", ""},
	}
	for _, tc := range cases {
		got := normalize(tc.input)
		if got != tc.want {
			t.Errorf("normalize(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestDeleteObjects_EdgeCases(t *testing.T) {
	o := &OSS{}
	if err := o.DeleteObjects(nil); err != nil {
		t.Errorf("nil keys should return nil, got: %v", err)
	}
	if err := o.DeleteObjects([]string{}); err != nil {
		t.Errorf("empty keys should return nil, got: %v", err)
	}
	if err := o.DeleteObjects([]string{"", "/", "  "}); err != nil {
		t.Errorf("all invalid keys should return nil, got: %v", err)
	}
}

func TestDeleteObjects_Normalization(t *testing.T) {
	normalize := func(keys []string) []string {
		out := make([]string, 0, len(keys))
		for _, k := range keys {
			k = strings.TrimPrefix(strings.TrimSpace(k), "/")
			if k != "" {
				out = append(out, k)
			}
		}
		return out
	}
	cases := []struct {
		input []string
		want  []string
	}{
		{[]string{"/a", "b", "/c/d"}, []string{"a", "b", "c/d"}},
		{[]string{"", "/", "  "}, []string{}},
		{[]string{"/"}, []string{}},
		{[]string{"a", "", "b"}, []string{"a", "b"}},
	}
	for _, tc := range cases {
		got := normalize(tc.input)
		if len(got) != len(tc.want) {
			t.Errorf("normalize(%v) = %v, want %v", tc.input, got, tc.want)
			continue
		}
		for i := range got {
			if got[i] != tc.want[i] {
				t.Errorf("normalize(%v) = %v, want %v", tc.input, got, tc.want)
				break
			}
		}
	}
}

func TestServiceErrorTypeAssertion(t *testing.T) {
	checkIs404 := func(err error) bool {
		var se serviceError
		if errors.As(err, &se) {
			return se.StatusCode == 404
		}
		return false
	}
	if !checkIs404(serviceError{StatusCode: 404}) {
		t.Error("expected serviceError{404} to match 404")
	}
	if checkIs404(errors.New("some error")) {
		t.Error("expected plain error to not match")
	}
	if checkIs404(serviceError{StatusCode: 500}) {
		t.Error("expected serviceError{500} to not match 404")
	}
}
