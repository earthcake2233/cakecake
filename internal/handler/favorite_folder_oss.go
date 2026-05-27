package handler

import (
	"fmt"
	"strings"

	"go.uber.org/zap"

	"minibili/internal/config"
	"minibili/internal/model"
	"minibili/internal/storage"
)

var favoriteFolderCoverOSSExts = []string{"jpg", "jpeg", "png", "webp", "gif", "bmp"}

func favoriteFolderOSSObjectKeys(cfg *config.C, f model.FavoriteFolder) []string {
	seen := make(map[string]struct{})
	out := make([]string, 0, len(favoriteFolderCoverOSSExts)+1)
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
		add(cfg.OSSObjectKeyFromURL(f.CoverURL))
	}
	for _, ext := range favoriteFolderCoverOSSExts {
		add(fmt.Sprintf("favorite-folders/%d/%d.%s", f.UserID, f.ID, ext))
	}
	return out
}

func purgeFavoriteFolderOSSObjects(cfg *config.C, ossClient *storage.OSS, log *zap.Logger, f model.FavoriteFolder) {
	if ossClient == nil {
		return
	}
	keys := favoriteFolderOSSObjectKeys(cfg, f)
	if len(keys) == 0 {
		return
	}
	if err := ossClient.DeleteObjects(keys); err != nil {
		if log != nil {
			log.Warn("oss delete favorite folder cover",
				zap.Uint64("folder_id", f.ID),
				zap.Uint64("user_id", f.UserID),
				zap.Strings("keys", keys),
				zap.Error(err),
			)
		}
		return
	}
	if log != nil {
		log.Info("oss deleted favorite folder cover",
			zap.Uint64("folder_id", f.ID),
			zap.Uint64("user_id", f.UserID),
			zap.Strings("keys", keys),
		)
	}
}

func purgeFavoriteFolderCoverURL(cfg *config.C, ossClient *storage.OSS, log *zap.Logger, coverURL string, uid, folderID uint64) {
	if cfg == nil || ossClient == nil {
		return
	}
	purgeFavoriteFolderOSSObjects(cfg, ossClient, log, model.FavoriteFolder{
		ID:       folderID,
		UserID:   uid,
		CoverURL: coverURL,
	})
}
