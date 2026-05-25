package handler

import (
	"testing"

	"minibili/internal/config"
	"minibili/internal/model"
)

func TestBannerOSSObjectKeys(t *testing.T) {
	cfg := &config.C{
		OSSBucket:          "your-bucket",
		OSSEndpoint:        "https://oss-cn-beijing.aliyuncs.com",
		OSSPublicURLPrefix: "https://your-bucket.oss-cn-beijing.aliyuncs.com",
	}
	b := model.HomeBanner{
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
