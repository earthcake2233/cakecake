package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"minibili/internal/errcode"
	"minibili/internal/middleware"
	"minibili/internal/model"
	"minibili/internal/pkg/dailyreward"
	"minibili/internal/pkg/resp"
	"minibili/internal/pkg/usercoin"
)

func videoFavsByViewer(db *gorm.DB, viewer uint64, ids []uint64) map[uint64]bool {
	out := make(map[uint64]bool)
	if viewer == 0 || len(ids) == 0 {
		return out
	}
	var rows []model.VideoFavorite
	if err := db.Where("user_id = ? AND video_id IN ?", viewer, ids).Find(&rows).Error; err != nil {
		return out
	}
	for i := range rows {
		out[rows[i].VideoID] = true
	}
	return out
}

func videoCoinsByViewer(db *gorm.DB, viewer uint64, ids []uint64) map[uint64]int {
	out := make(map[uint64]int)
	if viewer == 0 || len(ids) == 0 {
		return out
	}
	var rows []model.VideoCoin
	if err := db.Where("user_id = ? AND video_id IN ?", viewer, ids).Find(&rows).Error; err != nil {
		return out
	}
	for i := range rows {
		amt := rows[i].Amount
		if amt < 0 {
			amt = 0
		}
		if amt > 2 {
			amt = 2
		}
		out[rows[i].VideoID] = amt
	}
	return out
}

func videoEngagementByViewer(db *gorm.DB, viewer uint64, ids []uint64) map[uint64]videoEngagement {
	out := make(map[uint64]videoEngagement, len(ids))
	if viewer == 0 || len(ids) == 0 {
		return out
	}
	liked := videoLikesByViewer(db, viewer, ids)
	faved := videoFavsByViewer(db, viewer, ids)
	coined := videoCoinsByViewer(db, viewer, ids)
	later := watchLaterByViewer(db, viewer, ids)
	for _, id := range ids {
		coinAmt := coined[id]
		out[id] = videoEngagement{
			LikedByMe:     liked[id],
			FavoritedByMe: faved[id],
			CoinedByMe:    coinAmt > 0,
			MyCoinAmount:  coinAmt,
			InWatchLater:  later[id],
		}
	}
	return out
}

func watchLaterByViewer(db *gorm.DB, viewer uint64, ids []uint64) map[uint64]bool {
	out := make(map[uint64]bool)
	if viewer == 0 || len(ids) == 0 {
		return out
	}
	var rows []model.WatchLater
	if err := db.Where("user_id = ? AND video_id IN ?", viewer, ids).Find(&rows).Error; err != nil {
		return out
	}
	for i := range rows {
		out[rows[i].VideoID] = true
	}
	return out
}

func loadPublishedVideo(a *API, vid uint64) (model.Video, bool) {
	var v model.Video
	if err := a.DB.First(&v, vid).Error; err != nil {
		return v, false
	}
	if v.Status != "published" {
		return v, false
	}
	return v, true
}

// ToggleVideoFavorite toggles the current user's favorite on a published video.
func (a *API) ToggleVideoFavorite(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	vid, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || vid == 0 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	if _, ok := loadPublishedVideo(a, vid); !ok {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	var rows []model.VideoFavorite
	res := a.DB.Where("user_id = ? AND video_id = ?", uid, vid).Find(&rows)
	if res.Error != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	if len(rows) == 0 {
		def, err := a.ensureDefaultFavoriteFolder(uid)
		if err != nil {
			resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
			return
		}
		row := model.VideoFavorite{UserID: uid, VideoID: vid, FolderID: def.ID}
		if err := a.DB.Create(&row).Error; err != nil {
			resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
			return
		}
		_ = a.DB.Model(&model.Video{}).Where("id = ?", vid).UpdateColumn("fav_count", gorm.Expr("fav_count + ?", 1)).Error
		var v model.Video
		_ = a.DB.First(&v, vid).Error
		resp.OK(c, gin.H{"favorited": true, "fav_count": v.FavCount})
		return
	}
	if err := a.DB.Where("user_id = ? AND video_id = ?", uid, vid).Delete(&model.VideoFavorite{}).Error; err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	_ = a.DB.Model(&model.Video{}).Where("id = ?", vid).UpdateColumn("fav_count", gorm.Expr("GREATEST(fav_count - ?, 0)", 1)).Error
	var v model.Video
	_ = a.DB.First(&v, vid).Error
	resp.OK(c, gin.H{"favorited": false, "fav_count": v.FavCount})
}

const favoriteFolderCapacity = 999

type setVideoFavoriteFoldersJSON struct {
	FolderIDs []uint64 `json:"folder_ids"`
}

// GetVideoFavoritePicker returns folders for the collect dialog on the video page.
func (a *API) GetVideoFavoritePicker(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	vid, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || vid == 0 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	if _, ok := loadPublishedVideo(a, vid); !ok {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	folderRows, err := a.folderListPayload(uid, false)
	if err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	selected := make(map[uint64]bool)
	var favRows []model.VideoFavorite
	if err := a.DB.Where("user_id = ? AND video_id = ?", uid, vid).Find(&favRows).Error; err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	for i := range favRows {
		selected[favRows[i].FolderID] = true
	}
	items := make([]gin.H, 0, len(folderRows))
	for _, row := range folderRows {
		id, _ := row["id"].(uint64)
		isDefault, _ := row["is_default"].(bool)
		videoCount, _ := row["video_count"].(int64)
		countLabel := strconv.FormatInt(videoCount, 10)
		if !isDefault {
			countLabel = strconv.FormatInt(videoCount, 10) + "/" + strconv.Itoa(favoriteFolderCapacity)
		}
		items = append(items, gin.H{
			"id":           id,
			"title":        row["title"],
			"is_default":   isDefault,
			"video_count":  videoCount,
			"count_label":  countLabel,
			"selected":     selected[id],
		})
	}
	var v model.Video
	_ = a.DB.First(&v, vid).Error
	resp.OK(c, gin.H{
		"favorited":  len(favRows) > 0,
		"fav_count":  v.FavCount,
		"folder_ids": folderIDsFromMap(selected),
		"items":      items,
	})
}

func folderIDsFromMap(m map[uint64]bool) []uint64 {
	out := make([]uint64, 0, len(m))
	for id := range m {
		if id > 0 {
			out = append(out, id)
		}
	}
	return out
}

// SetVideoFavoriteFolders syncs which folders contain the video for the current user.
func (a *API) SetVideoFavoriteFolders(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	vid, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || vid == 0 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	if _, ok := loadPublishedVideo(a, vid); !ok {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	var body setVideoFavoriteFoldersJSON
	if err := c.ShouldBindJSON(&body); err != nil {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	want := make(map[uint64]bool)
	for _, fid := range body.FolderIDs {
		if fid > 0 {
			want[fid] = true
		}
	}
	if _, err := a.ensureDefaultFavoriteFolder(uid); err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	if len(want) > 0 {
		var owned int64
		ids := make([]uint64, 0, len(want))
		for fid := range want {
			ids = append(ids, fid)
		}
		if err := a.DB.Model(&model.FavoriteFolder{}).
			Where("user_id = ? AND id IN ?", uid, ids).
			Count(&owned).Error; err != nil {
			resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
			return
		}
		if int(owned) != len(ids) {
			resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
			return
		}
		for fid := range want {
			var cnt int64
			_ = a.DB.Model(&model.VideoFavorite{}).Where("folder_id = ?", fid).Count(&cnt).Error
			var already bool
			var row model.VideoFavorite
			if err := a.DB.Where("user_id = ? AND video_id = ? AND folder_id = ?", uid, vid, fid).
				Limit(1).Find(&row).Error; err == nil && row.ID > 0 {
				already = true
			}
			if !already && cnt >= favoriteFolderCapacity {
				resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
				return
			}
		}
	}
	var existing []model.VideoFavorite
	if err := a.DB.Where("user_id = ? AND video_id = ?", uid, vid).Find(&existing).Error; err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
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
		row := model.VideoFavorite{UserID: uid, VideoID: vid, FolderID: fid}
		if err := a.DB.Create(&row).Error; err != nil {
			resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
			return
		}
	}
	for i := range existing {
		if want[existing[i].FolderID] {
			continue
		}
		if err := a.DB.Delete(&existing[i]).Error; err != nil {
			resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
			return
		}
	}
	willFavorited := len(want) > 0
	if !wasFavorited && willFavorited {
		_ = a.DB.Model(&model.Video{}).Where("id = ?", vid).UpdateColumn("fav_count", gorm.Expr("fav_count + ?", 1)).Error
	} else if wasFavorited && !willFavorited {
		_ = a.DB.Model(&model.Video{}).Where("id = ?", vid).UpdateColumn("fav_count", gorm.Expr("GREATEST(fav_count - ?, 0)", 1)).Error
	}
	var v model.Video
	_ = a.DB.First(&v, vid).Error
	resp.OK(c, gin.H{
		"favorited":  willFavorited,
		"fav_count":  v.FavCount,
		"folder_ids": folderIDsFromMap(want),
	})
}

func (a *API) userVideoFavoriteCount(uid, vid uint64) (int64, error) {
	var cnt int64
	err := a.DB.Model(&model.VideoFavorite{}).
		Where("user_id = ? AND video_id = ?", uid, vid).
		Count(&cnt).Error
	return cnt, err
}

func (a *API) syncVideoFavCountAfterUserChange(vid uint64, before, after int64) {
	if before == 0 && after > 0 {
		_ = a.DB.Model(&model.Video{}).Where("id = ?", vid).
			UpdateColumn("fav_count", gorm.Expr("fav_count + ?", 1)).Error
	} else if before > 0 && after == 0 {
		_ = a.DB.Model(&model.Video{}).Where("id = ?", vid).
			UpdateColumn("fav_count", gorm.Expr("GREATEST(fav_count - ?, 0)", 1)).Error
	}
}

func (a *API) validateFolderOwned(uid, folderID uint64) bool {
	var cnt int64
	_ = a.DB.Model(&model.FavoriteFolder{}).
		Where("user_id = ? AND id = ?", uid, folderID).
		Count(&cnt).Error
	return cnt > 0
}

// RemoveVideoFromFavoriteFolder removes the video from one folder (current-folder unfavorite).
func (a *API) RemoveVideoFromFavoriteFolder(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	vid, folderID, ok := parseVideoFolderParams(c)
	if !ok {
		return
	}
	if _, ok := loadPublishedVideo(a, vid); !ok {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	if !a.validateFolderOwned(uid, folderID) {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	before, err := a.userVideoFavoriteCount(uid, vid)
	if err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	res := a.DB.Where("user_id = ? AND video_id = ? AND folder_id = ?", uid, vid, folderID).
		Delete(&model.VideoFavorite{})
	if res.Error != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	after, _ := a.userVideoFavoriteCount(uid, vid)
	a.syncVideoFavCountAfterUserChange(vid, before, after)
	resp.OK(c, gin.H{"ok": true, "removed": res.RowsAffected > 0})
}

// AddVideoToFavoriteFolder copies the video into another folder.
func (a *API) AddVideoToFavoriteFolder(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	vid, folderID, ok := parseVideoFolderParams(c)
	if !ok {
		return
	}
	if _, ok := loadPublishedVideo(a, vid); !ok {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	if !a.validateFolderOwned(uid, folderID) {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	var exists model.VideoFavorite
	if err := a.DB.Where("user_id = ? AND video_id = ? AND folder_id = ?", uid, vid, folderID).
		Limit(1).Find(&exists).Error; err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	if exists.ID > 0 {
		resp.OK(c, gin.H{"ok": true, "copied": false})
		return
	}
	var cnt int64
	_ = a.DB.Model(&model.VideoFavorite{}).Where("folder_id = ?", folderID).Count(&cnt).Error
	if cnt >= favoriteFolderCapacity {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	before, err := a.userVideoFavoriteCount(uid, vid)
	if err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	row := model.VideoFavorite{UserID: uid, VideoID: vid, FolderID: folderID}
	if err := a.DB.Create(&row).Error; err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	after, _ := a.userVideoFavoriteCount(uid, vid)
	a.syncVideoFavCountAfterUserChange(vid, before, after)
	resp.OK(c, gin.H{"ok": true, "copied": true})
}

type moveVideoFavoriteFolderJSON struct {
	FromFolderID uint64 `json:"from_folder_id"`
	ToFolderID   uint64 `json:"to_folder_id"`
}

// MoveVideoFavoriteFolder moves the video from one folder to another.
func (a *API) MoveVideoFavoriteFolder(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	vid, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || vid == 0 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	var body moveVideoFavoriteFolderJSON
	if err := c.ShouldBindJSON(&body); err != nil {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	if body.FromFolderID == 0 || body.ToFolderID == 0 || body.FromFolderID == body.ToFolderID {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	if _, ok := loadPublishedVideo(a, vid); !ok {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	if !a.validateFolderOwned(uid, body.FromFolderID) || !a.validateFolderOwned(uid, body.ToFolderID) {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	var inFrom model.VideoFavorite
	if err := a.DB.Where("user_id = ? AND video_id = ? AND folder_id = ?", uid, vid, body.FromFolderID).
		Limit(1).Find(&inFrom).Error; err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	if inFrom.ID == 0 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	var inTo model.VideoFavorite
	_ = a.DB.Where("user_id = ? AND video_id = ? AND folder_id = ?", uid, vid, body.ToFolderID).
		Limit(1).Find(&inTo).Error
	if inTo.ID > 0 {
		if err := a.DB.Delete(&inFrom).Error; err != nil {
			resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
			return
		}
		resp.OK(c, gin.H{"ok": true, "moved": true})
		return
	}
	var cnt int64
	_ = a.DB.Model(&model.VideoFavorite{}).Where("folder_id = ?", body.ToFolderID).Count(&cnt).Error
	if cnt >= favoriteFolderCapacity {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	if err := a.DB.Delete(&inFrom).Error; err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	row := model.VideoFavorite{UserID: uid, VideoID: vid, FolderID: body.ToFolderID}
	if err := a.DB.Create(&row).Error; err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	resp.OK(c, gin.H{"ok": true, "moved": true})
}

func parseVideoFolderParams(c *gin.Context) (vid, folderID uint64, ok bool) {
	vid, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || vid == 0 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return 0, 0, false
	}
	folderID, err = strconv.ParseUint(c.Param("folderId"), 10, 64)
	if err != nil || folderID == 0 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return 0, 0, false
	}
	return vid, folderID, true
}

type videoCoinJSON struct {
	Amount int `json:"amount"`
}

// PostVideoCoin adds 1 or 2 coins from the current user (max 2 per video; second visit adds 1 only).
func (a *API) PostVideoCoin(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	vid, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || vid == 0 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	v, ok := loadPublishedVideo(a, vid)
	if !ok {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	if v.UserID == uid {
		resp.Err(c, http.StatusBadRequest, errcode.CodeCannotCoinSelf)
		return
	}
	var body videoCoinJSON
	_ = c.ShouldBindJSON(&body)
	amount := body.Amount
	if amount != 1 && amount != 2 {
		amount = 1
	}
	var exist model.VideoCoin
	res := a.DB.Where("user_id = ? AND video_id = ?", uid, vid).Limit(1).Find(&exist)
	if res.Error != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	coinBefore := dailyreward.CoinProgress(a.DB, uid)
	var viewer model.User
	var spentAmount int
	var myCoinAmount int

	if res.RowsAffected > 0 {
		if exist.Amount >= 2 {
			resp.Err(c, http.StatusBadRequest, errcode.CodeAlreadyCoined)
			return
		}
		spentAmount = 1
		myCoinAmount = 2
		if err := a.DB.Transaction(func(tx *gorm.DB) error {
			if err := usercoin.SpendOnVideoCoin(tx, uid, v.UserID, vid, spentAmount); err != nil {
				return err
			}
			if err := tx.Model(&exist).Update("amount", 2).Error; err != nil {
				return err
			}
			return tx.Model(&model.Video{}).Where("id = ?", vid).
				UpdateColumn("coin_count", gorm.Expr("coin_count + ?", 1)).Error
		}); err != nil {
			if errors.Is(err, usercoin.ErrInsufficientCoins) {
				resp.Err(c, http.StatusBadRequest, errcode.CodeInsufficientCoins)
				return
			}
			a.Log.Error("post video coin add", zap.Error(err))
			resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
			return
		}
	} else {
		if amount != 1 && amount != 2 {
			amount = 1
		}
		spentAmount = amount
		myCoinAmount = amount
		if err := a.DB.Transaction(func(tx *gorm.DB) error {
			if err := usercoin.SpendOnVideoCoin(tx, uid, v.UserID, vid, spentAmount); err != nil {
				return err
			}
			row := model.VideoCoin{UserID: uid, VideoID: vid, Amount: amount}
			if err := tx.Create(&row).Error; err != nil {
				return err
			}
			return tx.Model(&model.Video{}).Where("id = ?", vid).
				UpdateColumn("coin_count", gorm.Expr("coin_count + ?", amount)).Error
		}); err != nil {
			if errors.Is(err, usercoin.ErrInsufficientCoins) {
				resp.Err(c, http.StatusBadRequest, errcode.CodeInsufficientCoins)
				return
			}
			a.Log.Error("post video coin", zap.Error(err))
			resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
			return
		}
	}

	coinAfter := dailyreward.CoinProgress(a.DB, uid)
	_ = dailyreward.GrantCoinExp(a.DB, uid, coinBefore, coinAfter)
	_ = a.DB.First(&v, vid).Error
	_ = a.DB.First(&viewer, uid).Error
	resp.OK(c, gin.H{
		"coined":                  true,
		"coin_count":              v.CoinCount,
		"amount":                  spentAmount,
		"my_coin_amount":          myCoinAmount,
		"coined_by_me":            true,
		"coin_balance":            usercoin.BalanceFloat(viewer.CoinBalanceTenths),
		"daily_coin_exp_progress": coinAfter,
		"daily_coin_exp_max":      dailyreward.ExpCoinMax,
	})
}

// ToggleWatchLater toggles the current user's watch-later entry for a published video.
func (a *API) ToggleWatchLater(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	vid, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || vid == 0 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	if _, ok := loadPublishedVideo(a, vid); !ok {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	var wl model.WatchLater
	res := a.DB.Where("user_id = ? AND video_id = ?", uid, vid).Limit(1).Find(&wl)
	if res.Error != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	if res.RowsAffected == 0 {
		row := model.WatchLater{UserID: uid, VideoID: vid}
		if err := a.DB.Create(&row).Error; err != nil {
			resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
			return
		}
		resp.OK(c, gin.H{"in_watch_later": true})
		return
	}
	if err := a.DB.Delete(&wl).Error; err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	resp.OK(c, gin.H{"in_watch_later": false})
}

const watchLaterMaxItems = 100

// ListMyWatchLater returns the caller's watch-later queue (newest first).
func (a *API) ListMyWatchLater(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	var total int64
	if err := a.DB.Model(&model.WatchLater{}).Where("user_id = ?", uid).Count(&total).Error; err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	limit := watchLaterMaxItems
	if raw := c.Query("limit"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 && n <= watchLaterMaxItems {
			limit = n
		}
	}
	var rows []model.WatchLater
	if err := a.DB.Where("user_id = ?", uid).
		Order("created_at DESC, id DESC").
		Limit(limit).
		Find(&rows).Error; err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	if len(rows) == 0 {
		resp.OK(c, gin.H{
			"items":     []gin.H{},
			"total":     total,
			"max_limit": watchLaterMaxItems,
		})
		return
	}
	vids := make([]uint64, 0, len(rows))
	for i := range rows {
		vids = append(vids, rows[i].VideoID)
	}
	var videos []model.Video
	if err := a.DB.Where("id IN ? AND status = ?", vids, "published").Find(&videos).Error; err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	byID := make(map[uint64]model.Video, len(videos))
	uids := make([]uint64, 0, len(videos))
	for i := range videos {
		byID[videos[i].ID] = videos[i]
		uids = append(uids, videos[i].UserID)
	}
	var users []model.User
	if len(uids) > 0 {
		_ = a.DB.Where("id IN ?", uids).Find(&users).Error
	}
	userByID := map[uint64]model.User{}
	for i := range users {
		userByID[users[i].ID] = users[i]
	}
	items := make([]gin.H, 0, len(rows))
	for i := range rows {
		v, ok := byID[rows[i].VideoID]
		if !ok {
			continue
		}
		pc, _ := a.Play.Display(c.Request.Context(), &v)
		u := userByID[v.UserID]
		items = append(items, gin.H{
			"id":                  v.ID,
			"title":               v.Title,
			"cover_url":           v.CoverURL,
			"play_count":          pc,
			"danmaku_count":       v.DanmakuCount,
			"duration":            v.DurationSec,
			"uploader":            model.DisplayUsername(&u),
			"uploader_id":         v.UserID,
			"uploader_avatar_url": uploaderAvatarForAPI(&u),
			"watched":             rows[i].Watched,
			"created_at":          v.CreatedAt.Format("2006-01-02 15:04:05"),
			"added_at":            rows[i].CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}
	resp.OK(c, gin.H{
		"items":     items,
		"total":     total,
		"max_limit": watchLaterMaxItems,
	})
}

// ClearMyWatchLater removes all watch-later entries for the current user.
func (a *API) ClearMyWatchLater(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	if err := a.DB.Where("user_id = ?", uid).Delete(&model.WatchLater{}).Error; err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	resp.OK(c, gin.H{"ok": true})
}

// ClearWatchedWatchLater removes watched entries from the user's watch-later queue.
func (a *API) ClearWatchedWatchLater(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	if err := a.DB.Where("user_id = ? AND watched = ?", uid, true).Delete(&model.WatchLater{}).Error; err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	resp.OK(c, gin.H{"ok": true})
}

// MarkWatchLaterWatched marks a watch-later item as watched.
func (a *API) MarkWatchLaterWatched(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	vid, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || vid == 0 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	res := a.DB.Model(&model.WatchLater{}).
		Where("user_id = ? AND video_id = ?", uid, vid).
		Update("watched", true)
	if res.Error != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	if res.RowsAffected == 0 {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	resp.OK(c, gin.H{"watched": true})
}

func (a *API) favoriteListItems(c *gin.Context, ownerID uint64, limit int, folderID uint64, filterFolder bool) ([]gin.H, int64, error) {
	if _, err := a.ensureDefaultFavoriteFolder(ownerID); err != nil {
		return nil, 0, err
	}
	base := a.DB.Model(&model.VideoFavorite{}).Where("user_id = ?", ownerID)
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
	if limit <= 0 || limit > 200 {
		limit = 200
	}
	q := a.DB.Where("user_id = ?", ownerID)
	if filterFolder {
		q = q.Where("folder_id = ?", folderID)
	}
	var rows []model.VideoFavorite
	if err := q.Order("created_at DESC, id DESC").
		Limit(limit).
		Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	if len(rows) == 0 {
		return []gin.H{}, total, nil
	}
	vids := make([]uint64, 0, len(rows))
	for i := range rows {
		vids = append(vids, rows[i].VideoID)
	}
	var videos []model.Video
	if err := a.DB.Where("id IN ? AND status = ?", vids, "published").Find(&videos).Error; err != nil {
		return nil, 0, err
	}
	byID := make(map[uint64]model.Video, len(videos))
	uids := make([]uint64, 0, len(videos))
	for i := range videos {
		byID[videos[i].ID] = videos[i]
		uids = append(uids, videos[i].UserID)
	}
	var users []model.User
	if len(uids) > 0 {
		_ = a.DB.Where("id IN ?", uids).Find(&users).Error
	}
	userByID := map[uint64]model.User{}
	for i := range users {
		userByID[users[i].ID] = users[i]
	}
	items := make([]gin.H, 0, len(rows))
	seenVideo := make(map[uint64]struct{})
	for i := range rows {
		v, ok := byID[rows[i].VideoID]
		if !ok {
			continue
		}
		if !filterFolder {
			if _, dup := seenVideo[rows[i].VideoID]; dup {
				continue
			}
			seenVideo[rows[i].VideoID] = struct{}{}
		}
		pc, _ := a.Play.Display(c.Request.Context(), &v)
		u := userByID[v.UserID]
		items = append(items, gin.H{
			"id":                  v.ID,
			"title":               v.Title,
			"cover_url":           v.CoverURL,
			"play_count":          pc,
			"danmaku_count":       v.DanmakuCount,
			"duration":            v.DurationSec,
			"uploader":            model.DisplayUsername(&u),
			"uploader_id":         v.UserID,
			"uploader_avatar_url": uploaderAvatarForAPI(&u),
			"created_at":          v.CreatedAt.Format("2006-01-02 15:04:05"),
			"favorited_at":        rows[i].CreatedAt.Format("2006-01-02 15:04:05"),
			"folder_id":           rows[i].FolderID,
		})
	}
	return items, total, nil
}

// ListMyFavorites returns the caller's favorited published videos (newest favorite first).
func (a *API) ListMyFavorites(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	limit := 200
	if raw := c.Query("limit"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 && n <= 200 {
			limit = n
		}
	}
	folderID, filterFolder, err := parseFolderIDQuery(c)
	if err != nil {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	items, total, err := a.favoriteListItems(c, uid, limit, folderID, filterFolder)
	if err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	resp.OK(c, gin.H{"items": items, "total": total})
}

// ListUserFavorites returns a user's favorited published videos for their public space.
func (a *API) ListUserFavorites(c *gin.Context) {
	ownerID, err := strconv.ParseUint(c.Param("userId"), 10, 64)
	if err != nil || ownerID == 0 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	var u model.User
	if err := a.DB.First(&u, ownerID).Error; err != nil {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	viewer, viewerOK := middleware.UserID(c)
	if !spaceViewerCanSee(ownerID, viewerOK, viewer, u.PrivacyPublicFavorites) {
		resp.OK(c, gin.H{"items": []gin.H{}, "total": 0})
		return
	}
	limit := 200
	if raw := c.Query("limit"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 && n <= 200 {
			limit = n
		}
	}
	folderID, filterFolder, err := parseFolderIDQuery(c)
	if err != nil {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	if filterFolder {
		var folder model.FavoriteFolder
		if err := a.DB.Where("id = ? AND user_id = ?", folderID, ownerID).First(&folder).Error; err != nil {
			resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
			return
		}
		if !folder.IsPublic {
			resp.Err(c, http.StatusForbidden, errcode.CodeForbidden)
			return
		}
	}
	items, total, err := a.favoriteListItems(c, ownerID, limit, folderID, filterFolder)
	if err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	resp.OK(c, gin.H{"items": items, "total": total})
}

func (a *API) coinRecentListItems(c *gin.Context, ownerID uint64, limit int) ([]gin.H, int64, error) {
	var coins []model.VideoCoin
	if err := a.DB.Where("user_id = ?", ownerID).
		Order("created_at DESC").
		Limit(limit).
		Find(&coins).Error; err != nil {
		return nil, 0, err
	}
	var total int64
	_ = a.DB.Model(&model.VideoCoin{}).Where("user_id = ?", ownerID).Count(&total).Error
	if len(coins) == 0 {
		return []gin.H{}, total, nil
	}
	vids := make([]uint64, 0, len(coins))
	seen := make(map[uint64]struct{}, len(coins))
	for i := range coins {
		vid := coins[i].VideoID
		if _, ok := seen[vid]; ok {
			continue
		}
		seen[vid] = struct{}{}
		vids = append(vids, vid)
	}
	var videos []model.Video
	if err := a.DB.Where("id IN ? AND status = ?", vids, "published").Find(&videos).Error; err != nil {
		return nil, 0, err
	}
	vmap := make(map[uint64]model.Video, len(videos))
	uids := make([]uint64, 0, len(videos))
	uidSeen := make(map[uint64]struct{})
	for i := range videos {
		vmap[videos[i].ID] = videos[i]
		if _, ok := uidSeen[videos[i].UserID]; !ok {
			uidSeen[videos[i].UserID] = struct{}{}
			uids = append(uids, videos[i].UserID)
		}
	}
	users := make(map[uint64]model.User)
	if len(uids) > 0 {
		var urows []model.User
		_ = a.DB.Where("id IN ?", uids).Find(&urows).Error
		for i := range urows {
			users[urows[i].ID] = urows[i]
		}
	}
	viewer, viewerOK := middleware.UserID(c)
	var viewerID uint64
	if viewerOK {
		viewerID = viewer
	}
	eng := videoEngagementByViewer(a.DB, viewerID, vids)
	items := make([]gin.H, 0, len(coins))
	for i := range coins {
		v, ok := vmap[coins[i].VideoID]
		if !ok {
			continue
		}
		u := users[v.UserID]
		e := eng[v.ID]
		items = append(items, gin.H{
			"id":                  v.ID,
			"title":               v.Title,
			"cover_url":           v.CoverURL,
			"play_count":          v.PlayCount,
			"danmaku_count":       v.DanmakuCount,
			"comment_count":       v.CommentCount,
			"duration":            v.DurationSec,
			"uploader":            uploaderNameForAPI(&u),
			"uploader_avatar_url": uploaderAvatarForAPI(&u),
			"created_at":          v.CreatedAt,
			"coined_at":           coins[i].CreatedAt,
			"liked_by_me":         e.LikedByMe,
			"favorited_by_me":     e.FavoritedByMe,
			"coined_by_me":        e.CoinedByMe,
			"in_watch_later":      e.InWatchLater,
		})
	}
	return items, total, nil
}

// ListUserRecentCoinVideos returns videos the user recently coined (owner-only).
func (a *API) ListUserRecentCoinVideos(c *gin.Context) {
	ownerID, err := strconv.ParseUint(c.Param("userId"), 10, 64)
	if err != nil || ownerID == 0 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	var u model.User
	if err := a.DB.First(&u, ownerID).Error; err != nil {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	viewer, viewerOK := middleware.UserID(c)
	isOwner := viewerOK && viewer == ownerID
	if !isOwner && !u.PrivacyPublicRecentCoins {
		resp.OK(c, gin.H{"items": []gin.H{}, "total": 0})
		return
	}
	limit := 20
	if raw := c.Query("limit"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 && n <= 50 {
			limit = n
		}
	}
	items, total, err := a.coinRecentListItems(c, ownerID, limit)
	if err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	resp.OK(c, gin.H{"items": items, "total": total})
}
