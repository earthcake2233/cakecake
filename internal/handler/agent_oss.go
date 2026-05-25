package handler

import (
	"strings"

	"go.uber.org/zap"

	"minibili/internal/config"
	"minibili/internal/storage"
)

// purgeAgentAvatarOSS removes a previously stored agent avatar object when replaced.
func purgeAgentAvatarOSS(cfg *config.C, ossClient *storage.OSS, log *zap.Logger, avatarURL string) {
	if cfg == nil || ossClient == nil {
		return
	}
	key := strings.TrimPrefix(strings.TrimSpace(cfg.OSSObjectKeyFromURL(avatarURL)), "/")
	if key == "" || !strings.HasPrefix(key, "agent/") {
		return
	}
	if err := ossClient.DeleteObject(key); err != nil && log != nil {
		log.Warn("oss delete agent avatar", zap.String("key", key), zap.Error(err))
		return
	}
	if log != nil {
		log.Info("oss deleted agent avatar", zap.String("key", key))
	}
}

func agentAvatarURLChanged(oldURL, newURL string) bool {
	return strings.TrimSpace(oldURL) != strings.TrimSpace(newURL)
}
