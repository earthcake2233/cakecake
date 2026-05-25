package handler

import (
	"strings"

	"go.uber.org/zap"

	"minibili/internal/config"
	"minibili/internal/model"
	"minibili/internal/storage"
)

func dynamicOSSObjectKeys(cfg *config.C, dyn model.UserDynamic) []string {
	if cfg == nil {
		return nil
	}
	seen := make(map[string]struct{})
	out := make([]string, 0, 8)
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
	for _, u := range parseDynamicImagesJSON(dyn.ImagesJSON) {
		add(cfg.OSSObjectKeyFromURL(u))
	}
	return out
}

func purgeDynamicImageURLs(cfg *config.C, ossClient *storage.OSS, log *zap.Logger, urls []string) {
	if ossClient == nil || cfg == nil || len(urls) == 0 {
		return
	}
	seen := make(map[string]struct{})
	keys := make([]string, 0, len(urls))
	for _, u := range urls {
		key := strings.TrimPrefix(strings.TrimSpace(cfg.OSSObjectKeyFromURL(u)), "/")
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		keys = append(keys, key)
	}
	if len(keys) == 0 {
		return
	}
	if err := ossClient.DeleteObjects(keys); err != nil {
		if log != nil {
			log.Warn("oss delete dynamic image urls", zap.Strings("keys", keys), zap.Error(err))
		}
		return
	}
}

func purgeRemovedDynamicImageURLs(cfg *config.C, ossClient *storage.OSS, log *zap.Logger, oldURLs, newURLs []string) {
	newSet := make(map[string]struct{}, len(newURLs))
	for _, u := range newURLs {
		u = strings.TrimSpace(u)
		if u != "" {
			newSet[u] = struct{}{}
		}
	}
	removed := make([]string, 0)
	for _, u := range oldURLs {
		u = strings.TrimSpace(u)
		if u != "" {
			if _, ok := newSet[u]; !ok {
				removed = append(removed, u)
			}
		}
	}
	purgeDynamicImageURLs(cfg, ossClient, log, removed)
}

func purgeDynamicOSSObjects(cfg *config.C, ossClient *storage.OSS, log *zap.Logger, dyn model.UserDynamic) {
	if ossClient == nil {
		return
	}
	keys := dynamicOSSObjectKeys(cfg, dyn)
	if len(keys) == 0 {
		return
	}
	if err := ossClient.DeleteObjects(keys); err != nil {
		if log != nil {
			log.Warn("oss delete dynamic objects",
				zap.Uint64("dynamic_id", dyn.ID),
				zap.Strings("keys", keys),
				zap.Error(err),
			)
		}
		return
	}
	if log != nil {
		log.Info("oss deleted dynamic objects",
			zap.Uint64("dynamic_id", dyn.ID),
			zap.Strings("keys", keys),
		)
	}
}
