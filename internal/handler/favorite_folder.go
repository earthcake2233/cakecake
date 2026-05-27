package handler

import (
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"minibili/internal/errcode"
	"minibili/internal/middleware"
	"minibili/internal/model"
	"minibili/internal/pkg/coverval"
	"minibili/internal/pkg/resp"
)

const defaultFavoriteFolderTitle = "默认收藏夹"

type createFavoriteFolderJSON struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	IsPublic    *bool  `json:"is_public"`
}

func (a *API) ensureDefaultFavoriteFolder(userID uint64) (model.FavoriteFolder, error) {
	var f model.FavoriteFolder
	err := a.DB.Where("user_id = ? AND is_default = ?", userID, true).First(&f).Error
	if err == nil {
		return f, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return f, err
	}
	f = model.FavoriteFolder{
		UserID:    userID,
		Title:     defaultFavoriteFolderTitle,
		IsPublic:  true,
		IsDefault: true,
	}
	if err := a.DB.Create(&f).Error; err != nil {
		return f, err
	}
	_ = a.DB.Model(&model.VideoFavorite{}).
		Where("user_id = ? AND folder_id = ?", userID, 0).
		Update("folder_id", f.ID).Error
	return f, nil
}

func folderCoverFromVideos(db *gorm.DB, folderID uint64) string {
	var fav model.VideoFavorite
	if err := db.Where("folder_id = ?", folderID).
		Order("created_at DESC, id DESC").
		Limit(1).
		Find(&fav).Error; err != nil || fav.ID == 0 {
		return ""
	}
	var v model.Video
	if err := db.Select("cover_url").First(&v, fav.VideoID).Error; err != nil {
		return ""
	}
	return strings.TrimSpace(v.CoverURL)
}

func resolveFolderCoverURL(db *gorm.DB, f *model.FavoriteFolder) string {
	if u := strings.TrimSpace(f.CoverURL); u != "" {
		return u
	}
	return folderCoverFromVideos(db, f.ID)
}

func folderRowPayload(db *gorm.DB, f *model.FavoriteFolder) gin.H {
	var cnt int64
	_ = db.Model(&model.VideoFavorite{}).Where("folder_id = ?", f.ID).Count(&cnt).Error
	cover := resolveFolderCoverURL(db, f)
	out := gin.H{
		"id":          f.ID,
		"title":       f.Title,
		"description": f.Description,
		"is_public":   f.IsPublic,
		"is_default":  f.IsDefault,
		"video_count": cnt,
	}
	if cover != "" {
		out["cover_url"] = cover
	} else {
		out["cover_url"] = nil
	}
	return out
}

func (a *API) folderListPayload(userID uint64, publicOnly bool) ([]gin.H, error) {
	if _, err := a.ensureDefaultFavoriteFolder(userID); err != nil {
		return nil, err
	}
	q := a.DB.Where("user_id = ?", userID)
	if publicOnly {
		q = q.Where("is_public = ?", true)
	}
	var folders []model.FavoriteFolder
	if err := q.Order("is_default DESC, created_at DESC, id DESC").Find(&folders).Error; err != nil {
		return nil, err
	}
	out := make([]gin.H, 0, len(folders))
	for i := range folders {
		out = append(out, folderRowPayload(a.DB, &folders[i]))
	}
	return out, nil
}

func parseFolderIsPublicForm(raw string) bool {
	v := strings.TrimSpace(strings.ToLower(raw))
	if v == "false" || v == "0" || v == "no" {
		return false
	}
	return true
}

func (a *API) uploadFavoriteFolderCover(uid, folderID uint64, fh *multipart.FileHeader) (string, int) {
	if fh == nil {
		return "", 0
	}
	if code := coverval.ValidateCoverHeader(fh); code != 0 {
		return "", code
	}
	if a.OSS == nil {
		return "", errcode.CodeInternalError
	}
	if err := os.MkdirAll(a.Cfg.TempUploadDir, 0o755); err != nil {
		return "", errcode.CodeInternalError
	}
	tmp := filepath.Join(a.Cfg.TempUploadDir, uuid.NewString()+filepath.Ext(fh.Filename))
	if err := saveUploadedFile(fh, tmp); err != nil {
		return "", errcode.CodeInternalError
	}
	defer os.Remove(tmp)
	ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(fh.Filename)), ".")
	if ext == "jpeg" {
		ext = "jpg"
	}
	key := fmt.Sprintf("favorite-folders/%d/%d.%s", uid, folderID, ext)
	if err := a.OSS.UploadFile(key, tmp); err != nil {
		a.Log.Error("oss favorite folder cover upload", zap.Error(err))
		return "", errcode.CodeInternalError
	}
	return a.Cfg.OSSObjectURL(key), 0
}

// ListMyFavoriteFolders returns the caller's favorite folders.
func (a *API) ListMyFavoriteFolders(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	items, err := a.folderListPayload(uid, false)
	if err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	resp.OK(c, gin.H{"items": items})
}

// CreateFavoriteFolder creates a new favorite folder for the caller.
// Accepts application/json or multipart/form-data (title, description, is_public, optional cover).
func (a *API) CreateFavoriteFolder(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	ct := strings.ToLower(c.GetHeader("Content-Type"))
	if strings.Contains(ct, "multipart/form-data") {
		a.createFavoriteFolderMultipart(c, uid)
		return
	}
	a.createFavoriteFolderJSON(c, uid)
}

func (a *API) createFavoriteFolderJSON(c *gin.Context, uid uint64) {
	var body createFavoriteFolderJSON
	if err := c.ShouldBindJSON(&body); err != nil {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	title := strings.TrimSpace(body.Title)
	if title == "" || utf8.RuneCountInString(title) > 20 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	desc := strings.TrimSpace(body.Description)
	if utf8.RuneCountInString(desc) > 200 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	isPublic := true
	if body.IsPublic != nil {
		isPublic = *body.IsPublic
	}
	if _, err := a.ensureDefaultFavoriteFolder(uid); err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	row := model.FavoriteFolder{
		UserID:      uid,
		Title:       title,
		Description: desc,
		IsPublic:    isPublic,
		IsDefault:   false,
	}
	if err := a.DB.Create(&row).Error; err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	resp.OK(c, folderRowPayload(a.DB, &row))
}

func (a *API) createFavoriteFolderMultipart(c *gin.Context, uid uint64) {
	if err := c.Request.ParseMultipartForm(12 << 20); err != nil {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	title := strings.TrimSpace(c.PostForm("title"))
	if title == "" || utf8.RuneCountInString(title) > 20 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	desc := strings.TrimSpace(c.PostForm("description"))
	if utf8.RuneCountInString(desc) > 200 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	isPublic := parseFolderIsPublicForm(c.PostForm("is_public"))
	if _, err := a.ensureDefaultFavoriteFolder(uid); err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	row := model.FavoriteFolder{
		UserID:      uid,
		Title:       title,
		Description: desc,
		IsPublic:    isPublic,
		IsDefault:   false,
	}
	if err := a.DB.Create(&row).Error; err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	if fh, err := c.FormFile("cover"); err == nil && fh != nil {
		url, code := a.uploadFavoriteFolderCover(uid, row.ID, fh)
		if code != 0 {
			_ = a.DB.Delete(&row).Error
			resp.Err(c, http.StatusBadRequest, code)
			return
		}
		if url != "" {
			if err := a.DB.Model(&row).Update("cover_url", url).Error; err != nil {
				purgeFavoriteFolderCoverURL(a.Cfg, a.OSS, a.Log, url, uid, row.ID)
				_ = a.DB.Delete(&row).Error
				resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
				return
			}
			row.CoverURL = url
		}
	}
	resp.OK(c, folderRowPayload(a.DB, &row))
}

// ListUserFavoriteFolders returns favorite folders for a user's space (public, or all if viewer is owner).
func (a *API) ListUserFavoriteFolders(c *gin.Context) {
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
	if !isOwner && !u.PrivacyPublicFavorites {
		resp.OK(c, gin.H{"items": []gin.H{}, "total": 0, "hidden_count": 0})
		return
	}
	publicOnly := !isOwner
	items, err := a.folderListPayload(ownerID, publicOnly)
	if err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	var total int64
	_ = a.DB.Model(&model.FavoriteFolder{}).Where("user_id = ?", ownerID).Count(&total).Error
	hiddenCount := int64(0)
	if viewerOK && viewer == ownerID {
		var publicCnt int64
		_ = a.DB.Model(&model.FavoriteFolder{}).
			Where("user_id = ? AND is_public = ?", ownerID, true).
			Count(&publicCnt).Error
		hiddenCount = total - publicCnt
		if hiddenCount < 0 {
			hiddenCount = 0
		}
	}
	displayTotal := total
	if publicOnly {
		displayTotal = int64(len(items))
	}
	resp.OK(c, gin.H{
		"items":        items,
		"total":        displayTotal,
		"hidden_count": hiddenCount,
	})
}

func parseFolderIDQuery(c *gin.Context) (uint64, bool, error) {
	raw := strings.TrimSpace(c.Query("folder_id"))
	if raw == "" {
		return 0, false, nil
	}
	id, err := strconv.ParseUint(raw, 10, 64)
	if err != nil || id == 0 {
		return 0, false, errors.New("bad folder_id")
	}
	return id, true, nil
}

func parseFolderIDParam(c *gin.Context) (uint64, bool) {
	id, err := strconv.ParseUint(c.Param("folderId"), 10, 64)
	if err != nil || id == 0 {
		return 0, false
	}
	return id, true
}

func (a *API) loadUserFavoriteFolder(uid, folderID uint64) (model.FavoriteFolder, int) {
	var f model.FavoriteFolder
	if err := a.DB.Where("id = ? AND user_id = ?", folderID, uid).First(&f).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return f, errcode.CodeNotFound
		}
		return f, errcode.CodeInternalError
	}
	return f, 0
}

// UpdateFavoriteFolder updates folder metadata (json or multipart).
func (a *API) UpdateFavoriteFolder(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	folderID, ok := parseFolderIDParam(c)
	if !ok {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	ct := strings.ToLower(c.GetHeader("Content-Type"))
	if strings.Contains(ct, "multipart/form-data") {
		a.updateFavoriteFolderMultipart(c, uid, folderID)
		return
	}
	a.updateFavoriteFolderJSON(c, uid, folderID)
}

func (a *API) updateFavoriteFolderJSON(c *gin.Context, uid, folderID uint64) {
	var body createFavoriteFolderJSON
	if err := c.ShouldBindJSON(&body); err != nil {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	row, code := a.loadUserFavoriteFolder(uid, folderID)
	if code != 0 {
		resp.Err(c, http.StatusNotFound, code)
		return
	}
	title := strings.TrimSpace(body.Title)
	if title == "" || utf8.RuneCountInString(title) > 20 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	desc := strings.TrimSpace(body.Description)
	if utf8.RuneCountInString(desc) > 200 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	isPublic := row.IsPublic
	if body.IsPublic != nil {
		isPublic = *body.IsPublic
	}
	updates := map[string]interface{}{
		"title":       title,
		"description": desc,
		"is_public":   isPublic,
	}
	if err := a.DB.Model(&row).Updates(updates).Error; err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	_ = a.DB.First(&row, row.ID).Error
	resp.OK(c, folderRowPayload(a.DB, &row))
}

func (a *API) updateFavoriteFolderMultipart(c *gin.Context, uid, folderID uint64) {
	if err := c.Request.ParseMultipartForm(12 << 20); err != nil {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	row, code := a.loadUserFavoriteFolder(uid, folderID)
	if code != 0 {
		resp.Err(c, http.StatusNotFound, code)
		return
	}
	title := strings.TrimSpace(c.PostForm("title"))
	if title == "" || utf8.RuneCountInString(title) > 20 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	desc := strings.TrimSpace(c.PostForm("description"))
	if utf8.RuneCountInString(desc) > 200 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	isPublic := parseFolderIsPublicForm(c.PostForm("is_public"))
	updates := map[string]interface{}{
		"title":       title,
		"description": desc,
		"is_public":   isPublic,
	}
	var uploadedCoverURL string
	if fh, err := c.FormFile("cover"); err == nil && fh != nil {
		url, coverCode := a.uploadFavoriteFolderCover(uid, row.ID, fh)
		if coverCode != 0 {
			resp.Err(c, http.StatusBadRequest, coverCode)
			return
		}
		if url != "" {
			uploadedCoverURL = url
			updates["cover_url"] = url
		}
	}
	if err := a.DB.Model(&row).Updates(updates).Error; err != nil {
		if uploadedCoverURL != "" {
			purgeFavoriteFolderCoverURL(a.Cfg, a.OSS, a.Log, uploadedCoverURL, uid, row.ID)
		}
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	_ = a.DB.First(&row, row.ID).Error
	resp.OK(c, folderRowPayload(a.DB, &row))
}

// DeleteFavoriteFolder removes a non-default folder and its favorites.
func (a *API) DeleteFavoriteFolder(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	folderID, ok := parseFolderIDParam(c)
	if !ok {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	row, code := a.loadUserFavoriteFolder(uid, folderID)
	if code != 0 {
		resp.Err(c, http.StatusNotFound, code)
		return
	}
	if row.IsDefault {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	err := a.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("user_id = ? AND folder_id = ?", uid, folderID).
			Delete(&model.VideoFavorite{}).Error; err != nil {
			return err
		}
		return tx.Delete(&row).Error
	})
	if err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	purgeFavoriteFolderOSSObjects(a.Cfg, a.OSS, a.Log, row)
	resp.OK(c, gin.H{"deleted": true})
}

func (a *API) validateFolderOwnedByUser(uid, folderID uint64) bool {
	var n int64
	_ = a.DB.Model(&model.FavoriteFolder{}).
		Where("id = ? AND user_id = ?", folderID, uid).
		Count(&n).Error
	return n > 0
}

// ClearInvalidFavoritesInFolder removes favorites whose videos are missing or not published.
func (a *API) ClearInvalidFavoritesInFolder(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	folderID, ok := parseFolderIDParam(c)
	if !ok {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	if !a.validateFolderOwnedByUser(uid, folderID) {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	var favs []model.VideoFavorite
	if err := a.DB.Where("user_id = ? AND folder_id = ?", uid, folderID).Find(&favs).Error; err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	if len(favs) == 0 {
		resp.OK(c, gin.H{"cleared": 0})
		return
	}
	vids := make([]uint64, 0, len(favs))
	for i := range favs {
		vids = append(vids, favs[i].VideoID)
	}
	var publishedIDs []uint64
	if err := a.DB.Model(&model.Video{}).
		Where("id IN ? AND status = ?", vids, "published").
		Pluck("id", &publishedIDs).Error; err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	pub := make(map[uint64]struct{}, len(publishedIDs))
	for _, id := range publishedIDs {
		pub[id] = struct{}{}
	}
	invalidVids := make([]uint64, 0)
	for _, id := range vids {
		if _, ok := pub[id]; !ok {
			invalidVids = append(invalidVids, id)
		}
	}
	if len(invalidVids) == 0 {
		resp.OK(c, gin.H{"cleared": 0})
		return
	}
	for _, vid := range invalidVids {
		before, err := a.userVideoFavoriteCount(uid, vid)
		if err != nil {
			resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
			return
		}
		if err := a.DB.Where("user_id = ? AND folder_id = ? AND video_id = ?", uid, folderID, vid).
			Delete(&model.VideoFavorite{}).Error; err != nil {
			resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
			return
		}
		after, _ := a.userVideoFavoriteCount(uid, vid)
		a.syncVideoFavCountAfterUserChange(vid, before, after)
	}
	resp.OK(c, gin.H{"cleared": len(invalidVids)})
}

type batchRemoveFavoritesJSON struct {
	VideoIDs []uint64 `json:"video_ids"`
}

// BatchRemoveVideosFromFavoriteFolder removes multiple videos from one folder.
func (a *API) BatchRemoveVideosFromFavoriteFolder(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	folderID, ok := parseFolderIDParam(c)
	if !ok {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	if !a.validateFolderOwnedByUser(uid, folderID) {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	var body batchRemoveFavoritesJSON
	if err := c.ShouldBindJSON(&body); err != nil || len(body.VideoIDs) == 0 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	ids := make([]uint64, 0, len(body.VideoIDs))
	seen := make(map[uint64]struct{})
	for _, raw := range body.VideoIDs {
		if raw == 0 {
			continue
		}
		if _, dup := seen[raw]; dup {
			continue
		}
		seen[raw] = struct{}{}
		ids = append(ids, raw)
	}
	if len(ids) == 0 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	removed := 0
	for _, vid := range ids {
		before, err := a.userVideoFavoriteCount(uid, vid)
		if err != nil {
			resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
			return
		}
		res := a.DB.Where("user_id = ? AND folder_id = ? AND video_id = ?", uid, folderID, vid).
			Delete(&model.VideoFavorite{})
		if res.Error != nil {
			resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
			return
		}
		if res.RowsAffected > 0 {
			removed++
		}
		after, _ := a.userVideoFavoriteCount(uid, vid)
		a.syncVideoFavCountAfterUserChange(vid, before, after)
	}
	resp.OK(c, gin.H{"removed": removed})
}
