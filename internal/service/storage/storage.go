package storage

import (
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strings"

	"go.uber.org/zap"

	"cakecake/internal/config"
	"cakecake/internal/model/admin"
	"cakecake/internal/model/article"
	"cakecake/internal/model/dynamic"
	"cakecake/internal/model/video"
	"cakecake/internal/storage"
)

// StorageService owns all object-storage (OSS) access on behalf of handlers.
// It is the single transport-facing boundary for file uploads and orphan
// object cleanup, keeping the handler layer free of storage infrastructure.
type StorageService struct {
	cfg *config.C
	oss *storage.OSS
	log *zap.Logger
}

// StorageBackend is the minimal object-storage surface StorageService needs.
type StorageBackend interface {
	UploadFile(objectKey, localPath string) error
	UploadReader(objectKey string, r io.Reader) error
	DeleteObject(objectKey string) error
	DeleteObjects(objectKeys []string) error
}

// OSSBackendOverride lets tests substitute a fake object-storage backend
// without exercising the real OSS SDK.
var OSSBackendOverride StorageBackend

func (s *StorageService) backend() StorageBackend {
	if OSSBackendOverride != nil {
		return OSSBackendOverride
	}
	if s == nil {
		return nil
	}
	return s.oss
}

func NewStorageService(cfg *config.C, oss *storage.OSS, log *zap.Logger) *StorageService {
	return &StorageService{cfg: cfg, oss: oss, log: log}
}

// Enabled reports whether OSS is configured.
func (s *StorageService) Enabled() bool {
	return s.backend() != nil
}

// UploadFile stores a local file at the given object key.
func (s *StorageService) UploadFile(objectKey, localPath string) error {
	if s.backend() == nil {
		return fmt.Errorf("oss not configured")
	}
	return s.backend().UploadFile(objectKey, localPath)
}

// UploadReader stores a reader's content at the given object key.
func (s *StorageService) UploadReader(objectKey string, r io.Reader) error {
	if s.backend() == nil {
		return fmt.Errorf("oss not configured")
	}
	return s.backend().UploadReader(objectKey, r)
}

// PurgeAgentAvatar removes a replaced agent avatar object.
func (s *StorageService) PurgeAgentAvatar(avatarURL string) {
	if s == nil || s.cfg == nil || s.backend() == nil {
		return
	}
	key := strings.TrimPrefix(strings.TrimSpace(s.cfg.OSSObjectKeyFromURL(avatarURL)), "/")
	if key == "" || !strings.HasPrefix(key, "agent/") {
		return
	}
	if err := s.backend().DeleteObject(key); err != nil {
		if s.log != nil {
			s.log.Warn("oss delete agent avatar", zap.String("key", key), zap.Error(err))
		}
		return
	}
	if s.log != nil {
		s.log.Info("oss deleted agent avatar", zap.String("key", key))
	}
}

func parseDynamicImagesJSON(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "[]" {
		return nil
	}
	var urls []string
	if err := json.Unmarshal([]byte(raw), &urls); err != nil {
		return nil
	}
	out := make([]string, 0, len(urls))
	for _, u := range urls {
		u = strings.TrimSpace(u)
		if u != "" {
			out = append(out, u)
		}
	}
	return out
}

func dynamicOSSObjectKeys(cfg *config.C, dyn dynamic.UserDynamic) []string {
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

func (s *StorageService) purgeDynamicImageURLs(urls []string) {
	if s == nil || s.cfg == nil || s.backend() == nil || len(urls) == 0 {
		return
	}
	seen := make(map[string]struct{})
	keys := make([]string, 0, len(urls))
	for _, u := range urls {
		key := strings.TrimPrefix(strings.TrimSpace(s.cfg.OSSObjectKeyFromURL(u)), "/")
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
	if err := s.backend().DeleteObjects(keys); err != nil {
		if s.log != nil {
			s.log.Warn("oss delete dynamic image urls", zap.Strings("keys", keys), zap.Error(err))
		}
		return
	}
}

// PurgeRemovedDynamicImageURLs deletes dynamic images no longer present in the new URL set.
func (s *StorageService) PurgeRemovedDynamicImageURLs(oldURLs, newURLs []string) {
	if s == nil {
		return
	}
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
	s.purgeDynamicImageURLs(removed)
}

// PurgeDynamic deletes all OSS objects referenced by a dynamic post.
func (s *StorageService) PurgeDynamic(dyn dynamic.UserDynamic) {
	if s == nil || s.backend() == nil {
		return
	}
	keys := dynamicOSSObjectKeys(s.cfg, dyn)
	if len(keys) == 0 {
		return
	}
	if err := s.backend().DeleteObjects(keys); err != nil {
		if s.log != nil {
			s.log.Warn("oss delete dynamic objects",
				zap.Uint64("dynamic_id", dyn.ID),
				zap.Strings("keys", keys),
				zap.Error(err),
			)
		}
		return
	}
	if s.log != nil {
		s.log.Info("oss deleted dynamic objects",
			zap.Uint64("dynamic_id", dyn.ID),
			zap.Strings("keys", keys),
		)
	}
}

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

// PurgeVideo deletes all OSS objects referenced by a video.
func (s *StorageService) PurgeVideo(v video.Video) {
	if s == nil || s.backend() == nil {
		return
	}
	keys := videoOSSObjectKeys(s.cfg, v)
	if len(keys) == 0 {
		return
	}
	if err := s.backend().DeleteObjects(keys); err != nil {
		if s.log != nil {
			s.log.Error("oss delete video objects",
				zap.Uint64("video_id", v.ID),
				zap.Strings("keys", keys),
				zap.Error(err),
			)
		}
		return
	}
	if s.log != nil {
		s.log.Info("oss deleted video objects",
			zap.Uint64("video_id", v.ID),
			zap.Strings("keys", keys),
		)
	}
}

var bannerImageOSSExts = []string{"jpg", "jpeg", "png", "webp", "gif", "bmp"}

func bannerOSSObjectKeys(cfg *config.C, b admin.HomeBanner) []string {
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

// PurgeBanner deletes all OSS objects referenced by a home banner.
func (s *StorageService) PurgeBanner(b admin.HomeBanner) {
	if s == nil || s.backend() == nil {
		return
	}
	keys := bannerOSSObjectKeys(s.cfg, b)
	if len(keys) == 0 {
		return
	}
	if err := s.backend().DeleteObjects(keys); err != nil {
		if s.log != nil {
			s.log.Warn("oss delete banner objects",
				zap.Uint64("banner_id", b.ID),
				zap.Strings("keys", keys),
				zap.Error(err),
			)
		}
		return
	}
	if s.log != nil {
		s.log.Info("oss deleted banner objects",
			zap.Uint64("banner_id", b.ID),
			zap.Strings("keys", keys),
		)
	}
}

// PurgeBannerImageURL deletes a single banner image object.
func (s *StorageService) PurgeBannerImageURL(imageURL string) {
	if s == nil || s.cfg == nil || s.backend() == nil {
		return
	}
	key := s.cfg.OSSObjectKeyFromURL(imageURL)
	if key == "" {
		return
	}
	if err := s.backend().DeleteObject(key); err != nil && s.log != nil {
		s.log.Warn("oss delete banner image", zap.String("key", key), zap.Error(err))
	}
}

var articleCoverOSSExts = []string{"jpg", "jpeg", "png", "webp", "gif", "bmp"}
var articleMDImageURLRe = regexp.MustCompile(`!\[[^\]]*\]\(([^)]+)\)`)

func articleOSSObjectKeys(cfg *config.C, art article.Article) []string {
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

// PurgeArticle deletes all OSS objects referenced by an article (cover + markdown images).
func (s *StorageService) PurgeArticle(art article.Article) {
	if s == nil || s.backend() == nil {
		return
	}
	keys := articleOSSObjectKeys(s.cfg, art)
	if len(keys) == 0 {
		return
	}
	if err := s.backend().DeleteObjects(keys); err != nil {
		if s.log != nil {
			s.log.Warn("oss delete article objects",
				zap.Uint64("article_id", art.ID),
				zap.Strings("keys", keys),
				zap.Error(err),
			)
		}
		return
	}
	if s.log != nil {
		s.log.Info("oss deleted article objects",
			zap.Uint64("article_id", art.ID),
			zap.Strings("keys", keys),
		)
	}
}

var favoriteFolderCoverOSSExts = []string{"jpg", "jpeg", "png", "webp", "gif", "bmp"}

func favoriteFolderOSSObjectKeys(cfg *config.C, f video.FavoriteFolder) []string {
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

// PurgeFavoriteFolder deletes all OSS objects referenced by a favorite folder.
func (s *StorageService) PurgeFavoriteFolder(f video.FavoriteFolder) {
	if s == nil || s.backend() == nil {
		return
	}
	keys := favoriteFolderOSSObjectKeys(s.cfg, f)
	if len(keys) == 0 {
		return
	}
	if err := s.backend().DeleteObjects(keys); err != nil {
		if s.log != nil {
			s.log.Warn("oss delete favorite folder cover",
				zap.Uint64("folder_id", f.ID),
				zap.Uint64("user_id", f.UserID),
				zap.Strings("keys", keys),
				zap.Error(err),
			)
		}
		return
	}
	if s.log != nil {
		s.log.Info("oss deleted favorite folder cover",
			zap.Uint64("folder_id", f.ID),
			zap.Uint64("user_id", f.UserID),
			zap.Strings("keys", keys),
		)
	}
}

// PurgeFavoriteFolderCoverURL deletes the cover object of a favorite folder.
func (s *StorageService) PurgeFavoriteFolderCoverURL(coverURL string, uid, folderID uint64) {
	if s == nil {
		return
	}
	s.PurgeFavoriteFolder(video.FavoriteFolder{
		ID:       folderID,
		UserID:   uid,
		CoverURL: coverURL,
	})
}
