package handler

import (
	"context"
	"errors"
	"minibili/internal/model/video"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"minibili/internal/errcode"
	"minibili/internal/middleware"
	"minibili/internal/pkg/resp"
	"minibili/internal/pkg/usercoin"
)

func (a *API) userVideoFavoriteCount(uid, vid uint64) (int64, error) {
	return a.EngagementSvc.UserFavoriteCount(context.Background(), uid, vid)
}

func videoCoinsByViewer(db *gorm.DB, viewer uint64, ids []uint64) map[uint64]int {
	out := make(map[uint64]int)
	if viewer == 0 || len(ids) == 0 {
		return out
	}
	var rows []video.VideoCoin
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

func (a *API) engagementByViewer(viewer uint64, ids []uint64) map[uint64]videoEngagement {
	out := make(map[uint64]videoEngagement, len(ids))
	if viewer == 0 || len(ids) == 0 {
		return out
	}
	liked := a.EngagementSvc.BatchVideoLikes(context.Background(), viewer, ids)
	faved := a.EngagementSvc.BatchFavoritedByUser(context.Background(), viewer, ids)
	coined := a.EngagementSvc.BatchCoinedByUser(context.Background(), viewer, ids)
	later := a.EngagementSvc.BatchWatchLater(context.Background(), viewer, ids)
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
	var rows []video.WatchLater
	if err := db.Where("user_id = ? AND video_id IN ?", viewer, ids).Find(&rows).Error; err != nil {
		return out
	}
	for i := range rows {
		out[rows[i].VideoID] = true
	}
	return out
}

func loadPublishedVideo(a *API, vid uint64) (*video.Video, bool) {
	v, err := a.VideoSvc.GetPublishedVideo(context.Background(), vid)
	if err != nil {
		return nil, false
	}
	return v, true
}

// ToggleVideoFavorite toggles the current user's favorite on a published video.
// ToggleVideoFavorite godoc
// @Summary      Favorite/unfavorite a video
// @Description  Toggle favorite status on a video
// @Tags         Videos
// @Produce      json
// @Param        id path int true "Video ID"
// @Success      200 {object} map[string]interface{}
// @Router       /videos/{id}/favorite [post]
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
	def, err := a.ensureDefaultFavoriteFolder(uid)
	if err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	favorited, favCount, err := a.EngagementSvc.ToggleVideoFavoriteWithFolder(context.Background(), uid, vid, def.ID)
	if err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	resp.OK(c, gin.H{"favorited": favorited, "fav_count": favCount})
}

const favoriteFolderCapacity = 999

type setVideoFavoriteFoldersJSON struct {
	FolderIDs []uint64 `json:"folder_ids"`
}

// GetVideoFavoritePicker returns folders for the collect dialog on the video page.
// GetVideoFavoritePicker godoc
// @Summary      Get favorite folder picker
// @Description  Get favorite folders for the video picker UI
// @Tags         Videos
// @Produce      json
// @Param        id path int true "Video ID"
// @Success      200 {object} map[string]interface{}
// @Router       /videos/{id}/favorite-picker [get]
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
	items := make([]gin.H, 0, len(folderRows))
	for _, row := range folderRows {
		id, _ := row["id"].(uint64)
		isDefault, _ := row["is_default"].(bool)
		videoCount, _ := row["video_count"].(int64)
		countLabel := strconv.FormatInt(videoCount, 10)
		if !isDefault {
			countLabel = strconv.FormatInt(videoCount, 10) + "/" + strconv.Itoa(favoriteFolderCapacity)
		}
		inFolder, _ := a.FavoriteSvc.CheckFavoriteExists(context.Background(), uid, id, vid)
		if inFolder {
			selected[id] = true
		}
		items = append(items, gin.H{"id": id, "title": row["title"],
			"is_default": isDefault, "video_count": videoCount,
			"count_label": countLabel, "selected": selected[id],
		})
	}
	v, _ := a.VideoSvc.GetPublishedVideo(context.Background(), vid)
	var favCount uint64
	if v != nil {
		favCount = v.FavCount
	}
	resp.OK(c, gin.H{"favorited": len(selected) > 0, "fav_count": favCount, "folder_ids": folderIDsFromMap(selected), "items": items})
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
// SetVideoFavoriteFolders godoc
// @Summary      Set video favorite folders
// @Description  Replace all favorite folders for a video
// @Tags         Videos
// @Produce      json
// @Param        id path int true "Video ID"
// @Param        body body object{folder_ids=array} true "Folder IDs"
// @Success      200 {object} map[string]interface{}
// @Router       /videos/{id}/favorite-folders [put]
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
	if _, err := a.ensureDefaultFavoriteFolder(uid); err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	result, err := a.FavoriteSvc.SetVideoFavoriteFolders(context.Background(), uid, vid, body.FolderIDs)
	if err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	if result == nil {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	resp.OK(c, gin.H{"favorited": result.Favorited, "fav_count": result.FavCount, "folder_ids": result.FolderIDs})
}

func (a *API) syncVideoFavCountAfterUserChange(vid uint64, before, after int64) {
	if before == 0 && after > 0 {
		_ = a.EngagementSvc.AdjustVideoFavCount(context.Background(), vid, 1)
	} else if before > 0 && after == 0 {
		_ = a.EngagementSvc.AdjustVideoFavCount(context.Background(), vid, -1)
	}
}

func (a *API) validateFolderOwned(uid, folderID uint64) bool {
	f, err := a.FavoriteSvc.GetFolderByID(context.Background(), folderID)
	return err == nil && f.UserID == uid
}

// RemoveVideoFromFavoriteFolder removes the video from one folder (current-folder unfavorite).
// RemoveVideoFromFavoriteFolder godoc
// @Summary      Remove video from favorite folder
// @Description  Remove a video from a specific favorite folder
// @Tags         Videos
// @Produce      json
// @Param        id path int true "Video ID"
// @Param        folderId path int true "Folder ID"
// @Success      200 {object} map[string]interface{}
// @Router       /videos/{id}/favorite-folders/{folderId} [delete]
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
	if err := a.FavoriteSvc.RemoveFavorite(context.Background(), folderID, vid); err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	after, _ := a.userVideoFavoriteCount(uid, vid)
	a.syncVideoFavCountAfterUserChange(vid, before, after)
	resp.OK(c, gin.H{"ok": true, "removed": after < before})
}

// AddVideoToFavoriteFolder copies the video into another folder.
// AddVideoToFavoriteFolder godoc
// @Summary      Add video to favorite folder
// @Description  Add a video to a specific favorite folder
// @Tags         Videos
// @Produce      json
// @Param        id path int true "Video ID"
// @Param        folderId path int true "Folder ID"
// @Success      200 {object} map[string]interface{}
// @Router       /videos/{id}/favorite-folders/{folderId} [post]
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
	exists, err := a.FavoriteSvc.CheckFavoriteExists(context.Background(), uid, folderID, vid)
	if err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	if exists {
		resp.OK(c, gin.H{"ok": true, "copied": false})
		return
	}
	cnt, err := a.FavoriteSvc.CountFavoritesInFolder(context.Background(), folderID)
	if err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	if cnt >= favoriteFolderCapacity {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	before, err := a.userVideoFavoriteCount(uid, vid)
	if err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	if err := a.FavoriteSvc.AddFavorite(context.Background(), folderID, vid, uid); err != nil {
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
// MoveVideoFavoriteFolder godoc
// @Summary      Move video between favorite folders
// @Description  Move a video from one favorite folder to another
// @Tags         Videos
// @Produce      json
// @Param        id path int true "Video ID"
// @Success      200 {object} map[string]interface{}
// @Router       /videos/{id}/favorite-folders/move [put]
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
	inFrom, err := a.FavoriteSvc.CheckFavoriteExists(context.Background(), uid, body.FromFolderID, vid)
	if err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	if !inFrom {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	inTo, _ := a.FavoriteSvc.CheckFavoriteExists(context.Background(), uid, body.ToFolderID, vid)
	if inTo {
		if err := a.FavoriteSvc.DeleteFavoriteByVideo(context.Background(), uid, body.FromFolderID, vid); err != nil {
			resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
			return
		}
		resp.OK(c, gin.H{"ok": true, "moved": true})
		return
	}
	cnt, err := a.FavoriteSvc.CountFavoritesInFolder(context.Background(), body.ToFolderID)
	if err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	if cnt >= favoriteFolderCapacity {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	if err := a.FavoriteSvc.DeleteFavoriteByVideo(context.Background(), uid, body.FromFolderID, vid); err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	if err := a.FavoriteSvc.AddFavorite(context.Background(), body.ToFolderID, vid, uid); err != nil {
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
// PostVideoCoin godoc
// @Summary      Coin a video
// @Description  Award coins to a video (1 or 2 coins)
// @Tags         Videos
// @Produce      json
// @Param        id path int true "Video ID"
// @Param        body body object{coins=int} true "Number of coins (1 or 2)"
// @Success      200 {object} map[string]interface{}
// @Router       /videos/{id}/coin [post]
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
	result, err := a.EngagementSvc.PostVideoCoin(context.Background(), uid, vid, v.UserID, amount)
	if err != nil {
		if errors.Is(err, usercoin.ErrInsufficientCoins) {
			resp.Err(c, http.StatusBadRequest, errcode.CodeInsufficientCoins)
			return
		}
		a.Log.Error("post video coin", zap.Error(err))
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	if result == nil {
		resp.Err(c, http.StatusBadRequest, errcode.CodeAlreadyCoined)
		return
	}
	resp.OK(c, gin.H{
		"coined":                  true,
		"coin_count":              result.CoinCount,
		"amount":                  result.Amount,
		"my_coin_amount":          result.MyCoinAmount,
		"coined_by_me":            true,
		"coin_balance":            result.CoinBalance,
		"daily_coin_exp_progress": result.DailyProgress,
		"daily_coin_exp_max":      result.DailyMax,
	})
}

// ToggleWatchLater toggles the current user's watch-later entry for a published video.
// ToggleWatchLater godoc
// @Summary      Add/remove watch later
// @Description  Toggle watch later status for a video
// @Tags         Videos
// @Produce      json
// @Param        id path int true "Video ID"
// @Success      200 {object} map[string]interface{}
// @Router       /videos/{id}/watch-later [post]
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
	added, err := a.EngagementSvc.ToggleWatchLater(context.Background(), uid, vid)
	if err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	resp.OK(c, gin.H{"in_watch_later": added})
}

const watchLaterMaxItems = 100

// ListMyWatchLater returns the caller's watch-later queue (newest first).
// ListMyWatchLater godoc
// @Summary      List watch later
// @Description  Get paginated list of videos saved for later
// @Tags         Videos
// @Produce      json
// @Param        page query int false "Page number" default(1)
// @Param        page_size query int false "Page size" default(20)
// @Success      200 {object} map[string]interface{}
// @Router       /users/me/watch-later [get]
func (a *API) ListMyWatchLater(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	page, pageSize := parsePagination(c, 20)
	items, total, err := a.EngagementSvc.ListWatchLaterWithVideos(context.Background(), uid, page, pageSize)
	if err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	resp.OK(c, gin.H{"items": items, "total": total, "max_limit": watchLaterMaxItems})
}

// ClearMyWatchLater removes all watch-later entries for the current user.
// ClearMyWatchLater godoc
// @Summary      Clear watch later
// @Description  Remove all videos from watch later list
// @Tags         Videos
// @Produce      json
// @Success      200 {object} map[string]interface{}
// @Router       /users/me/watch-later [delete]
func (a *API) ClearMyWatchLater(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	if err := a.EngagementSvc.ClearWatchLater(context.Background(), uid); err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	resp.OK(c, gin.H{"msg": "ok"})
}

// ClearWatchedWatchLater removes watched entries from the user's watch-later queue.
// ClearWatchedWatchLater godoc
// @Summary      Clear watched from watch later
// @Description  Remove already-watched videos from watch later
// @Tags         Videos
// @Produce      json
// @Success      200 {object} map[string]interface{}
// @Router       /users/me/watch-later/watched [delete]
func (a *API) ClearWatchedWatchLater(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	if err := a.EngagementSvc.ClearWatchedWatchLater(context.Background(), uid); err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	resp.OK(c, gin.H{"msg": "ok"})
}

// MarkWatchLaterWatched marks a watch-later item as watched.
// MarkWatchLaterWatched godoc
// @Summary      Mark watch later as watched
// @Description  Mark a specific watch later item as watched
// @Tags         Videos
// @Produce      json
// @Param        id path int true "Video ID"
// @Success      200 {object} map[string]interface{}
// @Router       /users/me/watch-later/{id}/watched [post]
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
	if err := a.EngagementSvc.MarkWatchLaterWatched(context.Background(), uid, vid); err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	resp.OK(c, gin.H{"msg": "ok"})
}

func (a *API) favoriteListItems(ctx context.Context, ownerID uint64, limit int, folderID uint64, filterFolder bool) ([]gin.H, int64, error) {
	if _, err := a.ensureDefaultFavoriteFolder(ownerID); err != nil {
		return nil, 0, err
	}
	result, err := a.FavoriteSvc.ListUserFavoriteVideos(ctx, ownerID, limit, folderID, filterFolder)
	if err != nil {
		return nil, 0, err
	}
	items := make([]gin.H, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, gin.H{
			"id": item.ID, "title": item.Title, "cover_url": item.CoverURL,
			"play_count": item.PlayCount, "danmaku_count": item.DanmakuCount, "duration": item.Duration,
			"uploader": item.UploaderName, "uploader_id": item.UploaderID,
			"uploader_avatar_url": item.UploaderAvatar,
			"created_at":          item.CreatedAt, "favorited_at": item.FavoritedAt,
			"folder_id": item.FolderID,
		})
	}
	return items, result.Total, nil
}

// ListMyFavorites returns the caller's favorited published videos (newest favorite first).
func (a *API) ListMyFavorites(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	limit := parseLimit(c, 200, 200)
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
	resp.OK(c, gin.H{"items": items, "total": total, "max_limit": watchLaterMaxItems})
}

// ListUserFavorites returns a user's favorited published videos for their public space.
func (a *API) ListUserFavorites(c *gin.Context) {
	ownerID, err := strconv.ParseUint(c.Param("userId"), 10, 64)
	if err != nil || ownerID == 0 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	up, err := a.UserSvc.GetUserPublic(context.Background(), ownerID)
	if err != nil {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	viewer, viewerOK := middleware.UserID(c)
	if !spaceViewerCanSee(ownerID, viewerOK, viewer, up.PrivacyPublicFavorites) {
		resp.OK(c, gin.H{"items": []gin.H{}, "total": 0})
		return
	}
	limit := parseLimit(c, 200, 200)
	folderID, filterFolder, err := parseFolderIDQuery(c)
	if err != nil {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	if filterFolder {
		f, err := a.FavoriteSvc.GetFolderByID(context.Background(), folderID)
		if err != nil || f.UserID != ownerID {
			resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
			return
		}
		if !f.IsPublic {
			resp.Err(c, http.StatusForbidden, errcode.CodeForbidden)
			return
		}
	}
	items, total, err := a.favoriteListItems(context.Background(), ownerID, limit, folderID, filterFolder)
	if err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	resp.OK(c, gin.H{"items": items, "total": total, "max_limit": watchLaterMaxItems})
}

func (a *API) coinRecentListItems(ctx context.Context, ownerID uint64, limit int) ([]gin.H, int64, error) {
	items, total, err := a.EngagementSvc.ListUserCoinedVideos(ctx, ownerID, limit)
	if err != nil {
		return nil, 0, err
	}
	result := make([]gin.H, 0, len(items))
	for _, item := range items {
		result = append(result, gin.H{
			"id": item.ID, "title": item.Title, "cover_url": item.CoverURL,
			"play_count": item.PlayCount, "danmaku_count": item.DanmakuCount,
			"comment_count": item.CommentCount, "duration": item.Duration,
			"uploader": item.UploaderName, "uploader_avatar_url": item.UploaderAvatar,
			"created_at": item.CreatedAt, "coined_at": item.CoinedAt,
		})
	}
	return result, total, nil
}

// ListUserRecentCoinVideos returns videos the user recently coined (owner-only).
func (a *API) ListUserRecentCoinVideos(c *gin.Context) {
	ownerID, err := strconv.ParseUint(c.Param("userId"), 10, 64)
	if err != nil || ownerID == 0 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	up, err := a.UserSvc.GetUserPublic(context.Background(), ownerID)
	if err != nil {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	viewer, viewerOK := middleware.UserID(c)
	isOwner := viewerOK && viewer == ownerID
	if !isOwner && !up.PrivacyPublicRecentCoins {
		resp.OK(c, gin.H{"items": []gin.H{}, "total": 0})
		return
	}
	limit := parseLimit(c, 20, 50)
	items, total, err := a.coinRecentListItems(c, ownerID, limit)
	if err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	resp.OK(c, gin.H{"items": items, "total": total, "max_limit": watchLaterMaxItems})
}
