package config

import "testing"

func TestOSSObjectKeyFromURL(t *testing.T) {
	cfg := &C{
		OSSBucket:          "your-bucket",
		OSSEndpoint:        "https://oss-cn-beijing.aliyuncs.com",
		OSSPublicURLPrefix: "https://your-bucket.oss-cn-beijing.aliyuncs.com",
	}
	cases := []struct {
		url  string
		want string
	}{
		{
			"https://your-bucket.oss-cn-beijing.aliyuncs.com/videos/42.mp4",
			"videos/42.mp4",
		},
		{
			"https://your-bucket.oss-cn-beijing.aliyuncs.com/covers/42.jpg",
			"covers/42.jpg",
		},
		{"", ""},
	}
	for _, tc := range cases {
		if got := cfg.OSSObjectKeyFromURL(tc.url); got != tc.want {
			t.Fatalf("OSSObjectKeyFromURL(%q) = %q, want %q", tc.url, got, tc.want)
		}
	}
}

func TestOSSObjectURLRoundTrip(t *testing.T) {
	cfg := &C{
		OSSBucket:          "your-bucket",
		OSSEndpoint:        "https://oss-cn-beijing.aliyuncs.com",
		OSSPublicURLPrefix: "https://your-bucket.oss-cn-beijing.aliyuncs.com",
	}
	key := "videos/99.mp4"
	u := cfg.OSSObjectURL(key)
	if got := cfg.OSSObjectKeyFromURL(u); got != key {
		t.Fatalf("round trip: got %q", got)
	}
}
