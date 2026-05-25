package handler

import (
	"fmt"
	"regexp"
	"strings"

	"go.uber.org/zap"

	"minibili/internal/config"
	"minibili/internal/model"
	"minibili/internal/storage"
)

var articleCoverOSSExts = []string{"jpg", "jpeg", "png", "webp", "gif", "bmp"}
var articleMDImageURLRe = regexp.MustCompile(`!\[[^\]]*\]\(([^)]+)\)`)

func articleOSSObjectKeys(cfg *config.C, art model.Article) []string {
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
		add(cfg.OSSObjectKeyFromURL(art.CoverURL))
	}
	for _, ext := range articleCoverOSSExts {
		add(fmt.Sprintf("article-covers/%d.%s", art.ID, ext))
	}
	for _, m := range articleMDImageURLRe.FindAllStringSubmatch(art.BodyMD, -1) {
		if len(m) < 2 {
			continue
		}
		if cfg != nil {
			add(cfg.OSSObjectKeyFromURL(strings.TrimSpace(m[1])))
		}
	}
	return out
}

func purgeArticleOSSObjects(cfg *config.C, ossClient *storage.OSS, log *zap.Logger, art model.Article) {
	if ossClient == nil {
		return
	}
	keys := articleOSSObjectKeys(cfg, art)
	if len(keys) == 0 {
		return
	}
	if err := ossClient.DeleteObjects(keys); err != nil {
		if log != nil {
			log.Warn("oss delete article objects",
				zap.Uint64("article_id", art.ID),
				zap.Strings("keys", keys),
				zap.Error(err),
			)
		}
		return
	}
	if log != nil {
		log.Info("oss deleted article objects",
			zap.Uint64("article_id", art.ID),
			zap.Strings("keys", keys),
		)
	}
}
