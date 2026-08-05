package favorite

import (
	"cakecake/internal/model/video"
	"cakecake/internal/service"
	vsvc "cakecake/internal/service/video"
	"context"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

const favoriteFolderCapacity = 999

// FavoriteService handles favorite folders and video favorites.
type FavoriteService struct {
	store FavoriteStore
	rdb   *redis.Client
	log   *zap.Logger

	// Domain providers (Phase 1: *gorm.DB impl; Phase 2+: gRPC clients)
	users  service.UserProvider
	videos vsvc.VideoProvider
}

// NewFavoriteService creates a FavoriteService with the given storage, cache,
// logger, and cross-domain user/video providers.
func NewFavoriteService(db *gorm.DB, rdb *redis.Client, log *zap.Logger, users service.UserProvider, videos vsvc.VideoProvider) *FavoriteService {
	return &FavoriteService{store: NewFavoriteStore(db), rdb: rdb, log: log, users: users, videos: videos}
}

// CreateFolder creates a new favorite folder.
func (s *FavoriteService) CreateFolder(ctx context.Context, folder *video.FavoriteFolder) error {
	return s.store.CreateFolder(ctx, folder)
}

// GetFolderByID returns a folder by ID.
func (s *FavoriteService) GetFolderByID(ctx context.Context, id uint64) (*video.FavoriteFolder, error) {
	return s.store.GetFolderByID(ctx, id)
}

// ListFoldersByUser returns all folders for a user.
func (s *FavoriteService) ListFoldersByUser(ctx context.Context, userID uint64) ([]video.FavoriteFolder, error) {
	return s.store.ListFoldersByUser(ctx, userID)
}

// UpdateFolder updates folder fields.
func (s *FavoriteService) UpdateFolder(ctx context.Context, id uint64, updates map[string]interface{}) error {
	return s.store.UpdateFolder(ctx, id, updates)
}

// DeleteFolder deletes a folder and its favorites.
func (s *FavoriteService) DeleteFolder(ctx context.Context, id uint64) error {
	return s.store.DeleteFolder(ctx, id)
}

// AddFavorite adds a video to a folder.
func (s *FavoriteService) AddFavorite(ctx context.Context, folderID, videoID, userID uint64) error {
	return s.store.AddFavorite(ctx, folderID, videoID, userID)
}

// RemoveFavorite removes a video from a folder.
func (s *FavoriteService) RemoveFavorite(ctx context.Context, folderID, videoID uint64) error {
	return s.store.RemoveFavorite(ctx, folderID, videoID)
}

// ListFavoritesByFolder returns video IDs in a folder with pagination.
func (s *FavoriteService) ListFavoritesByFolder(ctx context.Context, folderID uint64, page, pageSize int) ([]video.VideoFavorite, int64, error) {
	return s.store.ListFavoritesByFolder(ctx, folderID, page, pageSize)
}

// IsFavorited checks if a user has favorited a video.
func (s *FavoriteService) IsFavorited(ctx context.Context, userID, videoID uint64) bool {
	return s.store.IsFavorited(ctx, userID, videoID)
}

// CountByUser returns the total favorites for a user.
func (s *FavoriteService) CountByUser(ctx context.Context, userID uint64) int64 {
	return s.store.CountByUser(ctx, userID)
}

// BatchFavorited returns a map of video_id -> favorited for a user.
func (s *FavoriteService) BatchFavorited(ctx context.Context, userID uint64, videoIDs []uint64) map[uint64]bool {
	return s.store.BatchFavorited(ctx, userID, videoIDs)
}

// SearchFolders searches folders by name for autocomplete.
func (s *FavoriteService) SearchFolders(ctx context.Context, userID uint64, keyword string, limit int) ([]video.FavoriteFolder, error) {
	return s.store.SearchFolders(ctx, userID, keyword, limit)
}

// MigrateOrphanFavorites moves favorites with folder_id=0 to the target folder.
func (s *FavoriteService) MigrateOrphanFavorites(ctx context.Context, userID, targetFolderID uint64) error {
	return s.store.MigrateOrphanFavorites(ctx, userID, targetFolderID)
}

// DeleteFavoriteEntry removes a single favorite entry.
func (s *FavoriteService) DeleteFavoriteEntry(ctx context.Context, id uint64) error {
	return s.store.DeleteFavoriteEntry(ctx, id)
}

// UpdateFolderCover updates the cover URL of a folder.
func (s *FavoriteService) UpdateFolderCover(ctx context.Context, folderID uint64, coverURL string) error {
	return s.store.UpdateFolderCover(ctx, folderID, coverURL)
}

// CountFoldersByUser counts folders for a user.
func (s *FavoriteService) CountFoldersByUser(ctx context.Context, userID uint64) (int64, error) {
	return s.store.CountFoldersByUser(ctx, userID)
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
	return s.store.CheckFavoriteExists(ctx, userID, folderID, videoID)
}

// DeleteFavoriteByVideo removes a specific favorite by user/folder/video ids.
func (s *FavoriteService) DeleteFavoriteByVideo(ctx context.Context, userID, folderID, videoID uint64) error {
	return s.store.DeleteFavoriteByVideo(ctx, userID, folderID, videoID)
}

// DeleteFavoritesByFolder removes all favorites in a folder.
func (s *FavoriteService) DeleteFavoritesByFolder(ctx context.Context, folderID uint64) error {
	return s.store.DeleteFavoritesByFolder(ctx, folderID)
}

// CountFavoritesInFolder counts favorites entries in a folder.
func (s *FavoriteService) CountFavoritesInFolder(ctx context.Context, folderID uint64) (int64, error) {
	return s.store.CountFavoritesInFolder(ctx, folderID)
}

// FolderCoverFromVideos returns the cover URL of the latest video in the folder.
func (s *FavoriteService) FolderCoverFromVideos(ctx context.Context, folderID uint64) string {
	fav, err := s.store.LatestFavoriteInFolder(ctx, folderID)
	if err != nil {
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
		owned, err := s.store.CountOwnedFolders(ctx, userID, ids)
		if err != nil {
			return nil, err
		}
		if int(owned) != len(ids) {
			return nil, nil
		}
		for fid := range want {
			cnt, _ := s.store.CountFavoritesInFolder(ctx, fid)
			if cnt >= favoriteFolderCapacity {
				exists, _ := s.CheckFavoriteExists(ctx, userID, fid, videoID)
				if !exists {
					return nil, nil
				}
			}
		}
	}
	existing, err := s.store.FindFavoritesForUserVideo(ctx, userID, videoID)
	if err != nil {
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
		if err := s.store.CreateVideoFavorite(ctx, &row); err != nil {
			return nil, err
		}
	}
	for i := range existing {
		if want[existing[i].FolderID] {
			continue
		}
		if err := s.store.DeleteVideoFavorite(ctx, &existing[i]); err != nil {
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
	rows, total, err := s.store.ListUserFavoriteVideoRows(ctx, ownerID, limit, folderID, filterFolder)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return &UserFavoriteVideoResult{Items: []UserFavoriteVideoItem{}, Total: total}, nil
	}

	vids := make([]uint64, 0, len(rows))
	for i := range rows {
		vids = append(vids, rows[i].VideoID)
	}

	// Use vsvc.VideoProvider to fetch published video info
	videos := make(map[uint64]*vsvc.VideoInfo)
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

	// Use service.UserProvider to fetch user info
	users := make(map[uint64]service.UserInfo)
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
	return s.store.CountPublicFoldersByUser(ctx, userID)
}

// DeleteFolderCascade deletes a folder and all its favorited entries in a transaction.
func (s *FavoriteService) DeleteFolderCascade(ctx context.Context, folderID, userID uint64) error {
	return s.store.DeleteFolderCascade(ctx, folderID, userID)
}

// FilterPublishedVideoIDs returns which of the given IDs belong to published videos.
func (s *FavoriteService) FilterPublishedVideoIDs(ctx context.Context, videoIDs []uint64) ([]uint64, error) {
	if len(videoIDs) == 0 {
		return nil, nil
	}
	// Use vsvc.VideoProvider to filter published videos
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
	return s.store.FilterPublishedVideoIDs(ctx, videoIDs)
}

// ToggleFavorite adds or removes a favorite, returning the new state and video count.
func (s *FavoriteService) ToggleFavorite(ctx context.Context, userID, videoID uint64) (bool, uint64, error) {
	return s.store.ToggleFavorite(ctx, userID, videoID)
}

// ToggleVideoFavoriteWithFolder adds or removes a favorite in a specific folder, returning new state and fav_count.
func (s *FavoriteService) ToggleVideoFavoriteWithFolder(ctx context.Context, userID, videoID, folderID uint64) (bool, uint64, error) {
	return s.store.ToggleVideoFavoriteWithFolder(ctx, userID, videoID, folderID)
}

// MoveFavoritesBetweenFolders moves favorites from one folder to another.
func (s *FavoriteService) MoveFavoritesBetweenFolders(ctx context.Context, uid, fromFolderID, toFolderID uint64) (int64, error) {
	return s.store.MoveFavoritesBetweenFolders(ctx, uid, fromFolderID, toFolderID)
}

// VideoFavCount returns the fav_count for a video.
func (s *FavoriteService) VideoFavCount(ctx context.Context, videoID uint64) (uint64, error) {
	return s.store.VideoFavCount(ctx, videoID)
}

// AdjustVideoFavCount increments or decrements the fav_count on a video.
func (s *FavoriteService) AdjustVideoFavCount(ctx context.Context, videoID uint64, delta int) error {
	return s.store.AdjustVideoFavCount(ctx, videoID, delta)
}

// UserFavoriteCount returns the number of times a user has favorited a specific video.
func (s *FavoriteService) UserFavoriteCount(ctx context.Context, userID, videoID uint64) (int64, error) {
	return s.store.UserFavoriteCount(ctx, userID, videoID)
}
