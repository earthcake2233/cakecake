package handler

import (
	"testing"

	"minibili/internal/config"
	"minibili/internal/model"
)

func TestVideoOSSObjectKeys(t *testing.T) {
	cfg := &config.C{
		OSSPublicURLPrefix: "https://b.oss.aliyuncs.com",
	}
	v := model.Video{
		ID:       7,
		VideoURL: "https://b.oss.aliyuncs.com/videos/7.mp4",
		CoverURL: "https://b.oss.aliyuncs.com/covers/7.png",
	}
	keys := videoOSSObjectKeys(cfg, v)
	seen := map[string]bool{}
	for _, k := range keys {
		seen[k] = true
	}
	for _, want := range []string{
		"videos/7.mp4",
		"covers/7.png",
		"covers/7.jpg",
		"covers/7.jpeg",
		"covers/7.webp",
	} {
		if !seen[want] {
			t.Fatalf("missing key %q in %v", want, keys)
		}
	}
}
