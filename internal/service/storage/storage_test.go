package storage

import (
	"testing"

	"cakecake/internal/config"
	"cakecake/internal/model/admin"
	"cakecake/internal/model/video"
)

func TestVideoOSSObjectKeys(t *testing.T) {
	cfg := &config.C{
		OSSPublicURLPrefix: "https://b.oss.aliyuncs.com",
	}
	v := video.Video{
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

func TestVideoOSSObjectKeysExtended(t *testing.T) {
	cfg := &config.C{
		OSSPublicURLPrefix: "https://bucket.oss.aliyuncs.com",
	}
	v := video.Video{
		ID:       7,
		VideoURL: "https://bucket.oss.aliyuncs.com/videos/7.mp4",
		CoverURL: "https://bucket.oss.aliyuncs.com/covers/7.png",
	}
	keys := videoOSSObjectKeys(cfg, v)
	if len(keys) < 3 {
		t.Fatalf("expected at least 3 keys, got %v", keys)
	}
}

func TestBannerOSSObjectKeys(t *testing.T) {
	cfg := &config.C{
		OSSBucket:          "your-bucket",
		OSSEndpoint:        "https://oss-cn-beijing.aliyuncs.com",
		OSSPublicURLPrefix: "https://your-bucket.oss-cn-beijing.aliyuncs.com",
	}
	b := admin.HomeBanner{
		ID:       3,
		ImageURL: "https://your-bucket.oss-cn-beijing.aliyuncs.com/home-banners/abc.jpg",
	}
	keys := bannerOSSObjectKeys(cfg, b)
	want := map[string]struct{}{
		"home-banners/abc.jpg": {},
		"home-banners/3.jpg":   {},
		"home-banners/3.jpeg":  {},
		"home-banners/3.png":   {},
		"home-banners/3.webp":  {},
		"home-banners/3.gif":   {},
		"home-banners/3.bmp":   {},
	}
	if len(keys) != len(want) {
		t.Fatalf("len(keys)=%d want %d: %v", len(keys), len(want), keys)
	}
	for _, k := range keys {
		if _, ok := want[k]; !ok {
			t.Fatalf("unexpected key %q in %v", k, keys)
		}
	}
}

func TestBannerOSSObjectKeysExtended(t *testing.T) {
	cfg := &config.C{
		OSSBucket:          "bucket",
		OSSEndpoint:        "https://oss-cn-beijing.aliyuncs.com",
		OSSPublicURLPrefix: "https://bucket.oss-cn-beijing.aliyuncs.com",
	}
	b := admin.HomeBanner{
		ID:       3,
		ImageURL: "https://bucket.oss-cn-beijing.aliyuncs.com/home-banners/abc.jpg",
	}
	keys := bannerOSSObjectKeys(cfg, b)
	if len(keys) < 2 {
		t.Fatalf("expected at least 2 keys, got %v", keys)
	}
}
