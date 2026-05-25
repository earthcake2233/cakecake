package handler

import (
	"fmt"
	"strings"

	"go.uber.org/zap"

	"minibili/internal/config"
	"minibili/internal/model"
	"minibili/internal/storage"
)

var bannerImageOSSExts = []string{"jpg", "jpeg", "png", "webp", "gif", "bmp"}

func bannerOSSObjectKeys(cfg *config.C, b model.HomeBanner) []string {
	seen := make(map[string]struct{})
	out := make([]string, 0, 4)
	add := func(key string) {
		key = strings.TrimPrefix(strings.TrimSpace(key), "/")
		if key == "" {
			return
		}
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		out = append(out, key)
	}
	if cfg != nil {
		add(cfg.OSSObjectKeyFromURL(b.ImageURL))
	}
	for _, ext := range bannerImageOSSExts {
		add(fmt.Sprintf("home-banners/%d.%s", b.ID, ext))
	}
	return out
}

func purgeBannerOSSObjects(cfg *config.C, ossClient *storage.OSS, log *zap.Logger, b model.HomeBanner) {
	if ossClient == nil {
		return
	}
	keys := bannerOSSObjectKeys(cfg, b)
	if len(keys) == 0 {
		return
	}
	if err := ossClient.DeleteObjects(keys); err != nil {
		if log != nil {
			log.Warn("oss delete banner objects",
				zap.Uint64("banner_id", b.ID),
				zap.Strings("keys", keys),
				zap.Error(err),
			)
		}
		return
	}
	if log != nil {
		log.Info("oss deleted banner objects",
			zap.Uint64("banner_id", b.ID),
			zap.Strings("keys", keys),
		)
	}
}

func purgeBannerImageURL(cfg *config.C, ossClient *storage.OSS, log *zap.Logger, imageURL string) {
	if cfg == nil || ossClient == nil {
		return
	}
	key := cfg.OSSObjectKeyFromURL(imageURL)
	if key == "" {
		return
	}
	if err := ossClient.DeleteObject(key); err != nil && log != nil {
		log.Warn("oss delete banner image", zap.String("key", key), zap.Error(err))
	}
}
