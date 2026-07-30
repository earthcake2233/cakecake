package handler

import (
	"minibili/internal/model/video"
	"fmt"
	"strings"

	"go.uber.org/zap"

	"minibili/internal/config"
	"minibili/internal/storage"
)

var videoCoverOSSExts = []string{"jpg", "jpeg", "png", "webp"}

func videoOSSObjectKeys(cfg *config.C, v video.Video) []string {
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
		add(cfg.OSSObjectKeyFromURL(v.VideoURL))
		add(cfg.OSSObjectKeyFromURL(v.CoverURL))
	}
	add(fmt.Sprintf("videos/%d.mp4", v.ID))
	for _, ext := range videoCoverOSSExts {
		add(fmt.Sprintf("covers/%d.%s", v.ID, ext))
	}
	return out
}

func purgeVideoOSSObjects(cfg *config.C, ossClient *storage.OSS, log *zap.Logger, v video.Video) {
	if ossClient == nil {
		return
	}
	keys := videoOSSObjectKeys(cfg, v)
	if len(keys) == 0 {
		return
	}
	if err := ossClient.DeleteObjects(keys); err != nil {
		if log != nil {
			log.Warn("oss delete video objects",
				zap.Uint64("video_id", v.ID),
				zap.Strings("keys", keys),
				zap.Error(err),
			)
		}
		return
	}
	if log != nil {
		log.Info("oss deleted video objects",
			zap.Uint64("video_id", v.ID),
			zap.Strings("keys", keys),
		)
	}
}
