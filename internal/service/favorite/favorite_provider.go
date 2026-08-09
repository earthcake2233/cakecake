package favorite

import (
	"cakecake/internal/model/video"
	"context"

	"gorm.io/gorm"
)

// FavoriteStore is the favorite-domain storage boundary.
// Phase 1: *gorm.DB impl. Phase 2+: replaced by gRPC client / per-domain store.
type FavoriteStore interface {
	CreateFolder(ctx context.Context, folder *video.FavoriteFolder) error
	GetFolderByID(ctx context.Context, id uint64) (*video.FavoriteFolder, error)
	ListFoldersByUser(ctx context.Context, userID uint64) ([]video.FavoriteFolder, error)
	UpdateFolder(ctx context.Context, id uint64, updates map[string]interface{}) error
	DeleteFolder(ctx context.Context, id uint64) error
	AddFavorite(ctx context.Context, folderID, videoID, userID uint64) error
	RemoveFavorite(ctx context.Context, folderID, videoID uint64) error
	ListFavoritesByFolder(ctx context.Context, folderID uint64, page, pageSize int) ([]video.VideoFavorite, int64, error)
	IsFavorited(ctx context.Context, userID, videoID uint64) bool
	CountByUser(ctx context.Context, userID uint64) int64
	BatchFavorited(ctx context.Context, userID uint64, videoIDs []uint64) map[uint64]bool
	SearchFolders(ctx context.Context, userID uint64, keyword string, limit int) ([]video.FavoriteFolder, error)
	MigrateOrphanFavorites(ctx context.Context, userID, targetFolderID uint64) error
	DeleteFavoriteEntry(ctx context.Context, id uint64) error
	UpdateFolderCover(ctx context.Context, folderID uint64, coverURL string) error
	CountFoldersByUser(ctx context.Context, userID uint64) (int64, error)
	CheckFavoriteExists(ctx context.Context, userID, folderID, videoID uint64) (bool, error)
	DeleteFavoriteByVideo(ctx context.Context, userID, folderID, videoID uint64) error
	DeleteFavoritesByFolder(ctx context.Context, folderID uint64) error
	CountFavoritesInFolder(ctx context.Context, folderID uint64) (int64, error)
	LatestFavoriteInFolder(ctx context.Context, folderID uint64) (*video.VideoFavorite, error)
	CountOwnedFolders(ctx context.Context, userID uint64, ids []uint64) (int64, error)
	FindFavoritesForUserVideo(ctx context.Context, userID, videoID uint64) ([]video.VideoFavorite, error)
	CreateVideoFavorite(ctx context.Context, row *video.VideoFavorite) error
	DeleteVideoFavorite(ctx context.Context, row *video.VideoFavorite) error
	ListUserFavoriteVideoRows(ctx context.Context, ownerID uint64, limit int, folderID uint64, filterFolder bool) ([]video.VideoFavorite, int64, error)
	CountPublicFoldersByUser(ctx context.Context, userID uint64) (int64, error)
	DeleteFolderCascade(ctx context.Context, folderID, userID uint64) error
	FilterPublishedVideoIDs(ctx context.Context, videoIDs []uint64) ([]uint64, error)
	ToggleFavorite(ctx context.Context, userID, videoID uint64) (bool, uint64, error)
	ToggleVideoFavoriteWithFolder(ctx context.Context, userID, videoID, folderID uint64) (bool, uint64, error)
	MoveFavoritesBetweenFolders(ctx context.Context, uid, fromFolderID, toFolderID uint64) (int64, error)
	VideoFavCount(ctx context.Context, videoID uint64) (uint64, error)
	AdjustVideoFavCount(ctx context.Context, videoID uint64, delta int) error
	UserFavoriteCount(ctx context.Context, userID, videoID uint64) (int64, error)
}

// FavoriteStoreImpl implements FavoriteStore using *gorm.DB (Phase 1 monolith).
type FavoriteStoreImpl struct {
	db *gorm.DB
}

var _ FavoriteStore = (*FavoriteStoreImpl)(nil)

// NewFavoriteStore creates a gorm-backed FavoriteStore implementation.
func NewFavoriteStore(db *gorm.DB) *FavoriteStoreImpl {
	return &FavoriteStoreImpl{db: db}
}

// CreateFolder inserts a favorite folder row.
func (p *FavoriteStoreImpl) CreateFolder(ctx context.Context, folder *video.FavoriteFolder) error {
	return p.db.WithContext(ctx).Create(folder).Error
}

// GetFolderByID loads a favorite folder by id.
func (p *FavoriteStoreImpl) GetFolderByID(ctx context.Context, id uint64) (*video.FavoriteFolder, error) {
	var f video.FavoriteFolder
	if err := p.db.WithContext(ctx).First(&f, id).Error; err != nil {
		return nil, err
	}
	return &f, nil
}

// ListFoldersByUser lists a user's favorite folders in creation order.
func (p *FavoriteStoreImpl) ListFoldersByUser(ctx context.Context, userID uint64) ([]video.FavoriteFolder, error) {
	var folders []video.FavoriteFolder
	if err := p.db.WithContext(ctx).Where("user_id = ?", userID).Order("id ASC").Find(&folders).Error; err != nil {
		return nil, err
	}
	return folders, nil
}

// UpdateFolder applies partial updates to a favorite folder.
func (p *FavoriteStoreImpl) UpdateFolder(ctx context.Context, id uint64, updates map[string]interface{}) error {
	return p.db.WithContext(ctx).Model(&video.FavoriteFolder{}).Where("id = ?", id).Updates(updates).Error
}

// DeleteFolder deletes a folder and its favorite entries atomically.
func (p *FavoriteStoreImpl) DeleteFolder(ctx context.Context, id uint64) error {
	return p.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		_ = tx.Where("folder_id = ?", id).Delete(&video.VideoFavorite{}).Error
		return tx.Delete(&video.FavoriteFolder{}, id).Error
	})
}

// AddFavorite adds a video to a folder (idempotent first-or-create).
func (p *FavoriteStoreImpl) AddFavorite(ctx context.Context, folderID, videoID, userID uint64) error {
	fav := video.VideoFavorite{FolderID: folderID, VideoID: videoID, UserID: userID}
	return p.db.WithContext(ctx).Where("folder_id = ? AND video_id = ?", folderID, videoID).
		FirstOrCreate(&fav).Error
}

// RemoveFavorite removes a video from a folder.
func (p *FavoriteStoreImpl) RemoveFavorite(ctx context.Context, folderID, videoID uint64) error {
	return p.db.WithContext(ctx).Where("folder_id = ? AND video_id = ?", folderID, videoID).
		Delete(&video.VideoFavorite{}).Error
}

// ListFavoritesByFolder pages the favorite entries of a folder (newest first).
func (p *FavoriteStoreImpl) ListFavoritesByFolder(ctx context.Context, folderID uint64, page, pageSize int) ([]video.VideoFavorite, int64, error) {
	var total int64
	_ = p.db.WithContext(ctx).Model(&video.VideoFavorite{}).Where("folder_id = ?", folderID).Count(&total).Error
	offset := (page - 1) * pageSize
	var favs []video.VideoFavorite
	if err := p.db.WithContext(ctx).Where("folder_id = ?", folderID).Order("id DESC").
		Offset(offset).Limit(pageSize).Find(&favs).Error; err != nil {
		return nil, 0, err
	}
	return favs, total, nil
}

// IsFavorited reports whether the user favorited the video in any folder.
func (p *FavoriteStoreImpl) IsFavorited(ctx context.Context, userID, videoID uint64) bool {
	var cnt int64
	p.db.WithContext(ctx).Model(&video.VideoFavorite{}).Where("user_id = ? AND video_id = ?", userID, videoID).Count(&cnt)
	return cnt > 0
}

// CountByUser returns the total number of favorite entries for a user.
func (p *FavoriteStoreImpl) CountByUser(ctx context.Context, userID uint64) int64 {
	var cnt int64
	_ = p.db.WithContext(ctx).Model(&video.VideoFavorite{}).Where("user_id = ?", userID).Count(&cnt).Error
	return cnt
}

// BatchFavorited maps video ids to whether the user favorited them.
func (p *FavoriteStoreImpl) BatchFavorited(ctx context.Context, userID uint64, videoIDs []uint64) map[uint64]bool {
	result := make(map[uint64]bool)
	if userID == 0 || len(videoIDs) == 0 {
		return result
	}
	var favs []video.VideoFavorite
	p.db.WithContext(ctx).Where("user_id = ? AND video_id IN ?", userID, videoIDs).Find(&favs)
	for _, f := range favs {
		result[f.VideoID] = true
	}
	return result
}

// SearchFolders searches a user's folders by title keyword.
func (p *FavoriteStoreImpl) SearchFolders(ctx context.Context, userID uint64, keyword string, limit int) ([]video.FavoriteFolder, error) {
	var folders []video.FavoriteFolder
	if err := p.db.WithContext(ctx).Where("user_id = ? AND title LIKE ?", userID, "%"+keyword+"%").
		Limit(limit).Find(&folders).Error; err != nil {
		return nil, err
	}
	return folders, nil
}

// MigrateOrphanFavorites moves legacy folder-less favorites into a target folder.
func (p *FavoriteStoreImpl) MigrateOrphanFavorites(ctx context.Context, userID, targetFolderID uint64) error {
	return p.db.WithContext(ctx).Model(&video.VideoFavorite{}).
		Where("user_id = ? AND folder_id = ?", userID, 0).
		Update("folder_id", targetFolderID).Error
}

// DeleteFavoriteEntry deletes a single favorite entry by id.
func (p *FavoriteStoreImpl) DeleteFavoriteEntry(ctx context.Context, id uint64) error {
	return p.db.WithContext(ctx).Delete(&video.VideoFavorite{}, id).Error
}

// UpdateFolderCover sets a folder's cover URL.
func (p *FavoriteStoreImpl) UpdateFolderCover(ctx context.Context, folderID uint64, coverURL string) error {
	return p.db.WithContext(ctx).Model(&video.FavoriteFolder{}).Where("id = ?", folderID).Update("cover_url", coverURL).Error
}

// CountFoldersByUser counts a user's favorite folders.
func (p *FavoriteStoreImpl) CountFoldersByUser(ctx context.Context, userID uint64) (int64, error) {
	var total int64
	err := p.db.WithContext(ctx).Model(&video.FavoriteFolder{}).Where("user_id = ?", userID).Count(&total).Error
	return total, err
}

// CheckFavoriteExists reports whether a specific favorite entry exists.
func (p *FavoriteStoreImpl) CheckFavoriteExists(ctx context.Context, userID, folderID, videoID uint64) (bool, error) {
	var cnt int64
	err := p.db.WithContext(ctx).Model(&video.VideoFavorite{}).
		Where("user_id = ? AND folder_id = ? AND video_id = ?", userID, folderID, videoID).
		Count(&cnt).Error
	return cnt > 0, err
}

// DeleteFavoriteByVideo removes a specific favorite entry.
func (p *FavoriteStoreImpl) DeleteFavoriteByVideo(ctx context.Context, userID, folderID, videoID uint64) error {
	return p.db.WithContext(ctx).
		Where("user_id = ? AND folder_id = ? AND video_id = ?", userID, folderID, videoID).
		Delete(&video.VideoFavorite{}).Error
}

// DeleteFavoritesByFolder removes all favorite entries of a folder.
func (p *FavoriteStoreImpl) DeleteFavoritesByFolder(ctx context.Context, folderID uint64) error {
	return p.db.WithContext(ctx).Where("folder_id = ?", folderID).Delete(&video.VideoFavorite{}).Error
}

// CountFavoritesInFolder counts favorite entries in a folder.
func (p *FavoriteStoreImpl) CountFavoritesInFolder(ctx context.Context, folderID uint64) (int64, error) {
	var cnt int64
	err := p.db.WithContext(ctx).Model(&video.VideoFavorite{}).Where("folder_id = ?", folderID).Count(&cnt).Error
	return cnt, err
}

// LatestFavoriteInFolder returns the newest favorite entry in a folder.
func (p *FavoriteStoreImpl) LatestFavoriteInFolder(ctx context.Context, folderID uint64) (*video.VideoFavorite, error) {
	var fav video.VideoFavorite
	if err := p.db.WithContext(ctx).Where("folder_id = ?", folderID).
		Order("created_at DESC, id DESC").Limit(1).Find(&fav).Error; err != nil {
		return nil, err
	}
	if fav.ID == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return &fav, nil
}

// CountOwnedFolders counts how many of the given folder ids belong to the user.
func (p *FavoriteStoreImpl) CountOwnedFolders(ctx context.Context, userID uint64, ids []uint64) (int64, error) {
	var owned int64
	err := p.db.WithContext(ctx).Model(&video.FavoriteFolder{}).
		Where("user_id = ? AND id IN ?", userID, ids).
		Count(&owned).Error
	return owned, err
}

// FindFavoritesForUserVideo lists a user's favorite entries for one video.
func (p *FavoriteStoreImpl) FindFavoritesForUserVideo(ctx context.Context, userID, videoID uint64) ([]video.VideoFavorite, error) {
	var existing []video.VideoFavorite
	if err := p.db.WithContext(ctx).Where("user_id = ? AND video_id = ?", userID, videoID).Find(&existing).Error; err != nil {
		return nil, err
	}
	return existing, nil
}

// CreateVideoFavorite inserts a favorite row.
func (p *FavoriteStoreImpl) CreateVideoFavorite(ctx context.Context, row *video.VideoFavorite) error {
	return p.db.WithContext(ctx).Create(row).Error
}

// DeleteVideoFavorite removes a favorite row.
func (p *FavoriteStoreImpl) DeleteVideoFavorite(ctx context.Context, row *video.VideoFavorite) error {
	return p.db.WithContext(ctx).Delete(row).Error
}

// ListUserFavoriteVideoRows lists a user's favorite rows (optionally in one folder) with total count.
func (p *FavoriteStoreImpl) ListUserFavoriteVideoRows(ctx context.Context, ownerID uint64, limit int, folderID uint64, filterFolder bool) ([]video.VideoFavorite, int64, error) {
	base := p.db.WithContext(ctx).Model(&video.VideoFavorite{}).Where("user_id = ?", ownerID)
	if filterFolder {
		base = base.Where("folder_id = ?", folderID)
	}
	var total int64
	if filterFolder {
		if err := base.Count(&total).Error; err != nil {
			return nil, 0, err
		}
	} else {
		if err := base.Select("COUNT(DISTINCT video_id)").Scan(&total).Error; err != nil {
			return nil, 0, err
		}
	}
	q := p.db.WithContext(ctx).Where("user_id = ?", ownerID)
	if filterFolder {
		q = q.Where("folder_id = ?", folderID)
	}
	var rows []video.VideoFavorite
	if err := q.Order("created_at DESC, id DESC").Limit(limit).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

// CountPublicFoldersByUser counts a user's public folders.
func (p *FavoriteStoreImpl) CountPublicFoldersByUser(ctx context.Context, userID uint64) (int64, error) {
	var cnt int64
	err := p.db.WithContext(ctx).Model(&video.FavoriteFolder{}).
		Where("user_id = ? AND is_public = ?", userID, true).
		Count(&cnt).Error
	return cnt, err
}

// DeleteFolderCascade deletes a user's folder and its favorite entries atomically.
func (p *FavoriteStoreImpl) DeleteFolderCascade(ctx context.Context, folderID, userID uint64) error {
	return p.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.WithContext(ctx).Where("user_id = ? AND folder_id = ?", userID, folderID).
			Delete(&video.VideoFavorite{}).Error; err != nil {
			return err
		}
		return tx.WithContext(ctx).Delete(&video.FavoriteFolder{}, folderID).Error
	})
}

// FilterPublishedVideoIDs returns the subset of video ids that are published.
func (p *FavoriteStoreImpl) FilterPublishedVideoIDs(ctx context.Context, videoIDs []uint64) ([]uint64, error) {
	if len(videoIDs) == 0 {
		return nil, nil
	}
	var publishedIDs []uint64
	err := p.db.WithContext(ctx).Model(&video.Video{}).
		Where("id IN ? AND status = ?", videoIDs, video.StatusPublished).
		Pluck("id", &publishedIDs).Error
	return publishedIDs, err
}

// ToggleFavorite toggles a user's favorite on a video, returning the new state and fav count.
func (p *FavoriteStoreImpl) ToggleFavorite(ctx context.Context, userID, videoID uint64) (bool, uint64, error) {
	var rows []video.VideoFavorite
	res := p.db.WithContext(ctx).Where("user_id = ? AND video_id = ?", userID, videoID).Find(&rows)
	if res.Error != nil {
		return false, 0, res.Error
	}
	if len(rows) == 0 {
		row := video.VideoFavorite{UserID: userID, VideoID: videoID}
		if err := p.db.WithContext(ctx).Create(&row).Error; err != nil {
			return false, 0, err
		}
		_ = p.db.WithContext(ctx).Model(&video.Video{}).Where("id = ?", videoID).UpdateColumn("fav_count", gorm.Expr("fav_count + ?", 1)).Error
		var v video.Video
		if err := p.db.WithContext(ctx).First(&v, videoID).Error; err != nil {
			return true, 0, nil
		}
		return true, v.FavCount, nil
	}
	if err := p.db.WithContext(ctx).Where("user_id = ? AND video_id = ?", userID, videoID).Delete(&video.VideoFavorite{}).Error; err != nil {
		return false, 0, err
	}
	_ = p.db.WithContext(ctx).Model(&video.Video{}).Where("id = ?", videoID).UpdateColumn("fav_count", gorm.Expr("CASE WHEN fav_count - ? < 0 THEN 0 ELSE fav_count - ? END", 1, 1)).Error
	var v video.Video
	if err := p.db.WithContext(ctx).First(&v, videoID).Error; err != nil {
		return false, 0, nil
	}
	return false, v.FavCount, nil
}

// MoveFavoritesBetweenFolders moves a user's favorites between folders, returning remaining count in the source.
func (p *FavoriteStoreImpl) MoveFavoritesBetweenFolders(ctx context.Context, uid, fromFolderID, toFolderID uint64) (int64, error) {
	var favs []video.VideoFavorite
	if err := p.db.WithContext(ctx).Where("user_id = ? AND folder_id = ?", uid, fromFolderID).
		Order("created_at ASC").Find(&favs).Error; err != nil {
		return 0, err
	}
	if len(favs) == 0 {
		return 0, nil
	}
	for _, fav := range favs {
		var already video.VideoFavorite
		_ = p.db.WithContext(ctx).Where("user_id = ? AND video_id = ?", uid, fav.VideoID).Limit(1).Find(&already).Error
		if already.ID > 0 && already.FolderID == toFolderID {
			_ = p.db.WithContext(ctx).Delete(&fav).Error
		} else {
			_ = p.db.WithContext(ctx).Model(&fav).Update("folder_id", toFolderID).Error
		}
	}
	var remaining int64
	_ = p.db.WithContext(ctx).Model(&video.VideoFavorite{}).Where("user_id = ? AND folder_id = ?", uid, fromFolderID).Count(&remaining).Error
	return remaining, nil
}

// VideoFavCount returns a video's stored favorite count.
func (p *FavoriteStoreImpl) VideoFavCount(ctx context.Context, videoID uint64) (uint64, error) {
	var v video.Video
	if err := p.db.WithContext(ctx).Select("fav_count").First(&v, videoID).Error; err != nil {
		return 0, err
	}
	return v.FavCount, nil
}

// ToggleVideoFavoriteWithFolder toggles a favorite in a specific folder, returning the new state and fav count.
func (p *FavoriteStoreImpl) ToggleVideoFavoriteWithFolder(ctx context.Context, userID, videoID, folderID uint64) (bool, uint64, error) {
	var rows []video.VideoFavorite
	res := p.db.WithContext(ctx).Where("user_id = ? AND video_id = ?", userID, videoID).Find(&rows)
	if res.Error != nil {
		return false, 0, res.Error
	}
	if len(rows) == 0 {
		row := video.VideoFavorite{UserID: userID, VideoID: videoID, FolderID: folderID}
		if err := p.db.WithContext(ctx).Create(&row).Error; err != nil {
			return false, 0, err
		}
		_ = p.db.WithContext(ctx).Model(&video.Video{}).Where("id = ?", videoID).UpdateColumn("fav_count", gorm.Expr("fav_count + ?", 1)).Error
		var v video.Video
		if err := p.db.WithContext(ctx).First(&v, videoID).Error; err != nil {
			return true, 0, nil
		}
		return true, v.FavCount, nil
	}
	if err := p.db.WithContext(ctx).Where("user_id = ? AND video_id = ?", userID, videoID).Delete(&video.VideoFavorite{}).Error; err != nil {
		return false, 0, err
	}
	_ = p.db.WithContext(ctx).Model(&video.Video{}).Where("id = ?", videoID).UpdateColumn("fav_count", gorm.Expr("CASE WHEN fav_count - ? < 0 THEN 0 ELSE fav_count - ? END", 1, 1)).Error
	var v video.Video
	if err := p.db.WithContext(ctx).First(&v, videoID).Error; err != nil {
		return false, 0, nil
	}
	return false, v.FavCount, nil
}

// AdjustVideoFavCount adjusts a video's fav count by delta (negative clamps at zero).
func (p *FavoriteStoreImpl) AdjustVideoFavCount(ctx context.Context, videoID uint64, delta int) error {
	if delta >= 0 {
		return p.db.WithContext(ctx).Model(&video.Video{}).Where("id = ?", videoID).
			UpdateColumn("fav_count", gorm.Expr("fav_count + ?", delta)).Error
	}
	return p.db.WithContext(ctx).Model(&video.Video{}).Where("id = ?", videoID).
		UpdateColumn("fav_count", gorm.Expr("CASE WHEN fav_count - ? < 0 THEN 0 ELSE fav_count - ? END", -delta, -delta)).Error
}

// UserFavoriteCount counts a user's favorite entries for a video.
func (p *FavoriteStoreImpl) UserFavoriteCount(ctx context.Context, userID, videoID uint64) (int64, error) {
	var cnt int64
	err := p.db.WithContext(ctx).Model(&video.VideoFavorite{}).Where("user_id = ? AND video_id = ?", userID, videoID).Count(&cnt).Error
	return cnt, err
}
