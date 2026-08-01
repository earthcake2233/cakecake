package service

import (
	"cakecake/internal/model/video"
	"context"
	"fmt"
	"strings"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

const favoriteFolderCapacity = 999

type FavoriteService struct {
	db  *gorm.DB
	rdb *redis.Client
	log *zap.Logger

	// Domain providers (Phase 1: *gorm.DB impl; Phase 2+: gRPC clients)
	users  UserProvider
	videos VideoProvider
}

func NewFavoriteService(db *gorm.DB, rdb *redis.Client, log *zap.Logger, users UserProvider, videos VideoProvider) *FavoriteService {
	return &FavoriteService{db: db, rdb: rdb, log: log, users: users, videos: videos}
}

// CreateFolder creates a new favorite folder.
func (s *FavoriteService) CreateFolder(ctx context.Context, folder *video.FavoriteFolder) error {
	return s.db.WithContext(ctx).Create(folder).Error
}

// GetFolderByID returns a folder by ID.
func (s *FavoriteService) GetFolderByID(ctx context.Context, id uint64) (*video.FavoriteFolder, error) {
	var f video.FavoriteFolder
	if err := s.db.WithContext(ctx).First(&f, id).Error; err != nil {
		return nil, err
	}
	return &f, nil
}

// ListFoldersByUser returns all folders for a user.
func (s *FavoriteService) ListFoldersByUser(ctx context.Context, userID uint64) ([]video.FavoriteFolder, error) {
	var folders []video.FavoriteFolder
	if err := s.db.WithContext(ctx).Where("user_id = ?", userID).Order("id ASC").Find(&folders).Error; err != nil {
		return nil, err
	}
	return folders, nil
}

// UpdateFolder updates folder fields.
func (s *FavoriteService) UpdateFolder(ctx context.Context, id uint64, updates map[string]interface{}) error {
	return s.db.WithContext(ctx).Model(&video.FavoriteFolder{}).Where("id = ?", id).Updates(updates).Error
}

// DeleteFolder deletes a folder and its favorites.
func (s *FavoriteService) DeleteFolder(ctx context.Context, id uint64) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		_ = tx.Where("folder_id = ?", id).Delete(&video.VideoFavorite{}).Error
		return tx.Delete(&video.FavoriteFolder{}, id).Error
	})
}

// AddFavorite adds a video to a folder.
func (s *FavoriteService) AddFavorite(ctx context.Context, folderID, videoID, userID uint64) error {
	fav := video.VideoFavorite{FolderID: folderID, VideoID: videoID, UserID: userID}
	return s.db.WithContext(ctx).Where("folder_id = ? AND video_id = ?", folderID, videoID).
		FirstOrCreate(&fav).Error
}

// RemoveFavorite removes a video from a folder.
func (s *FavoriteService) RemoveFavorite(ctx context.Context, folderID, videoID uint64) error {
	return s.db.WithContext(ctx).Where("folder_id = ? AND video_id = ?", folderID, videoID).
		Delete(&video.VideoFavorite{}).Error
}

// ListFavoritesByFolder returns video IDs in a folder with pagination.
func (s *FavoriteService) ListFavoritesByFolder(ctx context.Context, folderID uint64, page, pageSize int) ([]video.VideoFavorite, int64, error) {
	var total int64
	_ = s.db.WithContext(ctx).Model(&video.VideoFavorite{}).Where("folder_id = ?", folderID).Count(&total).Error
	offset := (page - 1) * pageSize
	var favs []video.VideoFavorite
	if err := s.db.WithContext(ctx).Where("folder_id = ?", folderID).Order("id DESC").
		Offset(offset).Limit(pageSize).Find(&favs).Error; err != nil {
		return nil, 0, err
	}
	return favs, total, nil
}

// IsFavorited checks if a user has favorited a video.
func (s *FavoriteService) IsFavorited(ctx context.Context, userID, videoID uint64) bool {
	var cnt int64
	s.db.WithContext(ctx).Model(&video.VideoFavorite{}).Where("user_id = ? AND video_id = ?", userID, videoID).Count(&cnt)
	return cnt > 0
}

// CountByUser returns the total favorites for a user.
func (s *FavoriteService) CountByUser(ctx context.Context, userID uint64) int64 {
	var cnt int64
	_ = s.db.WithContext(ctx).Model(&video.VideoFavorite{}).Where("user_id = ?", userID).Count(&cnt).Error
	return cnt
}

// BatchFavorited returns a map of video_id -> favorited for a user.
func (s *FavoriteService) BatchFavorited(ctx context.Context, userID uint64, videoIDs []uint64) map[uint64]bool {
	result := make(map[uint64]bool)
	if userID == 0 || len(videoIDs) == 0 {
		return result
	}
	var favs []video.VideoFavorite
	s.db.WithContext(ctx).Where("user_id = ? AND video_id IN ?", userID, videoIDs).Find(&favs)
	for _, f := range favs {
		result[f.VideoID] = true
	}
	return result
}

// SearchFolders searches folders by name for autocomplete.
func (s *FavoriteService) SearchFolders(ctx context.Context, userID uint64, keyword string, limit int) ([]video.FavoriteFolder, error) {
	var folders []video.FavoriteFolder
	if err := s.db.WithContext(ctx).Where("user_id = ? AND name LIKE ?", userID, "%"+keyword+"%").
		Limit(limit).Find(&folders).Error; err != nil {
		return nil, err
	}
	return folders, nil
}

// MigrateOrphanFavorites moves favorites with folder_id=0 to the target folder.
func (s *FavoriteService) MigrateOrphanFavorites(ctx context.Context, userID, targetFolderID uint64) error {
	return s.db.WithContext(ctx).Model(&video.VideoFavorite{}).
		Where("user_id = ? AND folder_id = ?", userID, 0).
		Update("folder_id", targetFolderID).Error
}

// DeleteFavoriteEntry removes a single favorite entry.
func (s *FavoriteService) DeleteFavoriteEntry(ctx context.Context, id uint64) error {
	return s.db.WithContext(ctx).Delete(&video.VideoFavorite{}, id).Error
}

// UpdateFolderCover updates the cover URL of a folder.
func (s *FavoriteService) UpdateFolderCover(ctx context.Context, folderID uint64, coverURL string) error {
	return s.db.WithContext(ctx).Model(&video.FavoriteFolder{}).Where("id = ?", folderID).Update("cover_url", coverURL).Error
}

// CountFoldersByUser counts folders for a user.
func (s *FavoriteService) CountFoldersByUser(ctx context.Context, userID uint64) (int64, error) {
	var total int64
	err := s.db.WithContext(ctx).Model(&video.FavoriteFolder{}).Where("user_id = ?", userID).Count(&total).Error
	return total, err
}

// ListFavoritesByFolderWithVideoIds is like ListFavoritesByFolder but also returns video IDs.
func (s *FavoriteService) ListFavoritesByFolderWithVideoIds(ctx context.Context, folderID uint64, page, pageSize int) ([]video.VideoFavorite, int64, []uint64, error) {
	favs, total, err := s.ListFavoritesByFolder(ctx, folderID, page, pageSize)
	if err != nil {
		return nil, 0, nil, err
	}
	videoIDs := make([]uint64, 0, len(favs))
	for _, f := range favs {
		videoIDs = append(videoIDs, f.VideoID)
	}
	return favs, total, videoIDs, nil
}

// CheckFavoriteExists checks if a favorite entry exists for a user/folder/video.
func (s *FavoriteService) CheckFavoriteExists(ctx context.Context, userID, folderID, videoID uint64) (bool, error) {
	var cnt int64
	err := s.db.WithContext(ctx).Model(&video.VideoFavorite{}).
		Where("user_id = ? AND folder_id = ? AND video_id = ?", userID, folderID, videoID).
		Count(&cnt).Error
	return cnt > 0, err
}

// DeleteFavoriteByVideo removes a specific favorite by user/folder/video ids.
func (s *FavoriteService) DeleteFavoriteByVideo(ctx context.Context, userID, folderID, videoID uint64) error {
	return s.db.WithContext(ctx).
		Where("user_id = ? AND folder_id = ? AND video_id = ?", userID, folderID, videoID).
		Delete(&video.VideoFavorite{}).Error
}

// DeleteFavoritesByFolder removes all favorites in a folder.
func (s *FavoriteService) DeleteFavoritesByFolder(ctx context.Context, folderID uint64) error {
	return s.db.WithContext(ctx).Where("folder_id = ?", folderID).Delete(&video.VideoFavorite{}).Error
}

// CountFavoritesInFolder counts favorites entries in a folder.
func (s *FavoriteService) CountFavoritesInFolder(ctx context.Context, folderID uint64) (int64, error) {
	var cnt int64
	err := s.db.WithContext(ctx).Model(&video.VideoFavorite{}).Where("folder_id = ?", folderID).Count(&cnt).Error
	return cnt, err
}

// FolderCoverFromVideos returns the cover URL of the latest video in the folder.
func (s *FavoriteService) FolderCoverFromVideos(ctx context.Context, folderID uint64) string {
	var fav video.VideoFavorite
	if err := s.db.WithContext(ctx).Where("folder_id = ?", folderID).
		Order("created_at DESC, id DESC").Limit(1).Find(&fav).Error; err != nil || fav.ID == 0 {
		return ""
	}
	if s.videos == nil {
		return ""
	}
	videos, err := s.videos.BatchGetPublishedVideos(ctx, []uint64{fav.VideoID})
	if err != nil || len(videos) == 0 {
		return ""
	}
	return videos[fav.VideoID].CoverURL
}

// SetVideoFavoriteFoldersResult holds the result of batch-setting favorite folders.
type SetVideoFavoriteFoldersResult struct {
	Favorited bool
	FavCount  uint64
	FolderIDs []uint64
}

// SetVideoFavoriteFolders sets the exact set of favorite folders for a user-video pair.
func (s *FavoriteService) SetVideoFavoriteFolders(ctx context.Context, userID, videoID uint64, wantedIDs []uint64) (*SetVideoFavoriteFoldersResult, error) {
	want := make(map[uint64]bool)
	for _, fid := range wantedIDs {
		if fid > 0 {
			want[fid] = true
		}
	}
	if len(want) > 0 {
		ids := make([]uint64, 0, len(want))
		for fid := range want {
			ids = append(ids, fid)
		}
		var owned int64
		if err := s.db.WithContext(ctx).Model(&video.FavoriteFolder{}).
			Where("user_id = ? AND id IN ?", userID, ids).
			Count(&owned).Error; err != nil {
			return nil, err
		}
		if int(owned) != len(ids) {
			return nil, nil
		}
		for fid := range want {
			var cnt int64
			_ = s.db.WithContext(ctx).Model(&video.VideoFavorite{}).Where("folder_id = ?", fid).Count(&cnt).Error
			if cnt >= favoriteFolderCapacity {
				exists, _ := s.CheckFavoriteExists(ctx, userID, fid, videoID)
				if !exists {
					return nil, nil
				}
			}
		}
	}
	var existing []video.VideoFavorite
	if err := s.db.WithContext(ctx).Where("user_id = ? AND video_id = ?", userID, videoID).Find(&existing).Error; err != nil {
		return nil, err
	}
	existingSet := make(map[uint64]bool, len(existing))
	for i := range existing {
		existingSet[existing[i].FolderID] = true
	}
	wasFavorited := len(existing) > 0
	for fid := range want {
		if existingSet[fid] {
			continue
		}
		row := video.VideoFavorite{UserID: userID, VideoID: videoID, FolderID: fid}
		if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
			return nil, err
		}
	}
	for i := range existing {
		if want[existing[i].FolderID] {
			continue
		}
		if err := s.db.WithContext(ctx).Delete(&existing[i]).Error; err != nil {
			return nil, err
		}
	}
	willFavorited := len(want) > 0
	if !wasFavorited && willFavorited {
		if s.videos != nil {
			_ = s.videos.IncrFavCount(ctx, videoID, 1)
		}
	} else if wasFavorited && !willFavorited {
		if s.videos != nil {
			_ = s.videos.IncrFavCount(ctx, videoID, -1)
		}
	}
	var favCount uint64
	if s.videos != nil {
		if vmap, err := s.videos.BatchGetPublishedVideos(ctx, []uint64{videoID}); err == nil && len(vmap) > 0 {
			favCount = vmap[videoID].FavCount
		}
	}
	fids := make([]uint64, 0, len(want))
	for fid := range want {
		fids = append(fids, fid)
	}
	return &SetVideoFavoriteFoldersResult{
		Favorited: willFavorited, FavCount: favCount, FolderIDs: fids,
	}, nil
}

// UserFavoriteVideoItem is a structured favorite item for display.
type UserFavoriteVideoItem struct {
	ID             uint64
	Title          string
	CoverURL       string
	PlayCount      uint64
	DanmakuCount   uint64
	Duration       uint64
	UploaderName   string
	UploaderAvatar string
	UploaderID     uint64
	CreatedAt      string
	FavoritedAt    string
	FolderID       uint64
}

// UserFavoriteVideoResult holds paginated favorite video results.
type UserFavoriteVideoResult struct {
	Items []UserFavoriteVideoItem
	Total int64
}

// ListUserFavoriteVideos returns favorited videos for a user with pagination.
func (s *FavoriteService) ListUserFavoriteVideos(ctx context.Context, ownerID uint64, limit int, folderID uint64, filterFolder bool) (*UserFavoriteVideoResult, error) {
	if limit <= 0 || limit > 200 {
		limit = 200
	}

	base := s.db.WithContext(ctx).Model(&video.VideoFavorite{}).Where("user_id = ?", ownerID)
	if filterFolder {
		base = base.Where("folder_id = ?", folderID)
	}

	var total int64
	if filterFolder {
		if err := base.Count(&total).Error; err != nil {
			return nil, err
		}
	} else {
		if err := base.Select("COUNT(DISTINCT video_id)").Scan(&total).Error; err != nil {
			return nil, err
		}
	}

	q := s.db.WithContext(ctx).Where("user_id = ?", ownerID)
	if filterFolder {
		q = q.Where("folder_id = ?", folderID)
	}

	var rows []video.VideoFavorite
	if err := q.Order("created_at DESC, id DESC").Limit(limit).Find(&rows).Error; err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return &UserFavoriteVideoResult{Items: []UserFavoriteVideoItem{}, Total: total}, nil
	}

	vids := make([]uint64, 0, len(rows))
	for i := range rows {
		vids = append(vids, rows[i].VideoID)
	}

	// Use VideoProvider to fetch published video info
	videos := make(map[uint64]*VideoInfo)
	if s.videos != nil {
		vmap, err := s.videos.BatchGetPublishedVideos(ctx, vids)
		if err != nil {
			return nil, err
		}
		videos = vmap
	}

	uids := make([]uint64, 0, len(videos))
	for _, v := range videos {
		uids = append(uids, v.UserID)
	}

	// Use UserProvider to fetch user info
	users := make(map[uint64]UserInfo)
	if s.users != nil && len(uids) > 0 {
		umap, err := s.users.GetUsersByIDs(ctx, uids)
		if err != nil {
			return nil, err
		}
		users = umap
	}

	items := make([]UserFavoriteVideoItem, 0, len(rows))
	seenVideo := make(map[uint64]struct{})
	for i := range rows {
		v, ok := videos[rows[i].VideoID]
		if !ok {
			continue
		}
		if !filterFolder {
			if _, dup := seenVideo[rows[i].VideoID]; dup {
				continue
			}
			seenVideo[rows[i].VideoID] = struct{}{}
		}
		uploaderName := ""
		uploaderAvatar := ""
		if u, uok := users[v.UserID]; uok {
			uploaderName = u.Nickname
			uploaderAvatar = u.AvatarURL
		}
		items = append(items, UserFavoriteVideoItem{
			ID: v.ID, Title: v.Title, CoverURL: v.CoverURL,
			PlayCount: v.PlayCount, DanmakuCount: v.DanmakuCount, Duration: uint64(v.DurationSec),
			UploaderName: uploaderName, UploaderID: v.UserID, UploaderAvatar: uploaderAvatar,
			CreatedAt:   v.CreatedAt.Format("2006-01-02 15:04:05"),
			FavoritedAt: rows[i].CreatedAt.Format("2006-01-02 15:04:05"),
			FolderID:    rows[i].FolderID,
		})
	}
	return &UserFavoriteVideoResult{Items: items, Total: total}, nil
}

// CountPublicFoldersByUser returns the count of public folders for a user.
func (s *FavoriteService) CountPublicFoldersByUser(ctx context.Context, userID uint64) (int64, error) {
	var cnt int64
	err := s.db.WithContext(ctx).Model(&video.FavoriteFolder{}).
		Where("user_id = ? AND is_public = ?", userID, true).
		Count(&cnt).Error
	return cnt, err
}

// DeleteFolderCascade deletes a folder and all its favorited entries in a transaction.
func (s *FavoriteService) DeleteFolderCascade(ctx context.Context, folderID, userID uint64) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.WithContext(ctx).Where("user_id = ? AND folder_id = ?", userID, folderID).
			Delete(&video.VideoFavorite{}).Error; err != nil {
			return err
		}
		return tx.WithContext(ctx).Delete(&video.FavoriteFolder{}, folderID).Error
	})
}

// FilterPublishedVideoIDs returns which of the given IDs belong to published videos.
func (s *FavoriteService) FilterPublishedVideoIDs(ctx context.Context, videoIDs []uint64) ([]uint64, error) {
	if len(videoIDs) == 0 {
		return nil, nil
	}
	// Use VideoProvider to filter published videos
	if s.videos != nil {
		vmap, err := s.videos.BatchGetPublishedVideos(ctx, videoIDs)
		if err != nil {
			return nil, err
		}
		result := make([]uint64, 0, len(vmap))
		for id := range vmap {
			result = append(result, id)
		}
		return result, nil
	}
	// Fallback to direct query (should not happen in production)
	var publishedIDs []uint64
	err := s.db.WithContext(ctx).Model(&video.Video{}).
		Where("id IN ? AND status = ?", videoIDs, video.StatusPublished).
		Pluck("id", &publishedIDs).Error
	return publishedIDs, err
}

// ensure unused imports are referenced
var _ = fmt.Sprintf
var _ = strings.TrimSpace
