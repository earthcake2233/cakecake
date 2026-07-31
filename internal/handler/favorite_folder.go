package handler

import (
	"context"
	"errors"
	"fmt"
	"mime/multipart"
	"minibili/internal/model/video"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"minibili/internal/errcode"
	"minibili/internal/middleware"
	"minibili/internal/pkg/coverval"
	"minibili/internal/pkg/resp"
)

const defaultFavoriteFolderTitle = "默认收藏夹"

type createFavoriteFolderJSON struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	IsPublic    *bool  `json:"is_public"`
}

func (a *API) ensureDefaultFavoriteFolder(userID uint64) (video.FavoriteFolder, error) {
	folders, err := a.FavoriteSvc.ListFoldersByUser(context.Background(), userID)
	if err != nil {
		return video.FavoriteFolder{}, err
	}
	for _, ff := range folders {
		if ff.IsDefault {
			return ff, nil
		}
	}
	f := video.FavoriteFolder{
		UserID:    userID,
		Title:     defaultFavoriteFolderTitle,
		IsPublic:  true,
		IsDefault: true,
	}
	if err := a.FavoriteSvc.CreateFolder(context.Background(), &f); err != nil {
		return f, err
	}
	_ = a.FavoriteSvc.MigrateOrphanFavorites(context.Background(), userID, f.ID)
	return f, nil
}

func (a *API) folderCoverFromVideos(folderID uint64) string {
	return a.FavoriteSvc.FolderCoverFromVideos(context.Background(), folderID)
}

func (a *API) resolveFolderCoverURL(f *video.FavoriteFolder) string {
	if u := strings.TrimSpace(f.CoverURL); u != "" {
		return u
	}
	return a.folderCoverFromVideos(f.ID)
}

func (a *API) folderRowPayload(f *video.FavoriteFolder) gin.H {
	vCnt, _ := a.FavoriteSvc.CountFavoritesInFolder(context.Background(), f.ID)
	cover := a.resolveFolderCoverURL(f)
	out := gin.H{
		"id":          f.ID,
		"title":       f.Title,
		"description": f.Description,
		"is_public":   f.IsPublic,
		"is_default":  f.IsDefault,
		"video_count": vCnt,
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
	folders, err := a.FavoriteSvc.ListFoldersByUser(context.Background(), userID)
	if err != nil {
		return nil, err
	}
	out := make([]gin.H, 0, len(folders))
	for i := range folders {
		if publicOnly && !folders[i].IsPublic {
			continue
		}
		out = append(out, a.folderRowPayload(&folders[i]))
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
// ListMyFavoriteFolders godoc
// @Summary      List my favorite folders
// @Description  Get all favorite folders for the current user
// @Tags         Favorites
// @Produce      json
// @Success      200 {object} map[string]interface{}
// @Router       /users/me/favorite-folders [get]
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
// CreateFavoriteFolder godoc
// @Summary      Create a favorite folder
// @Description  Create a new folder for organizing favorites
// @Tags         Favorites
// @Produce      json
// @Param        body body object{name=string} true "Folder name"
// @Success      200 {object} map[string]interface{}
// @Router       /users/me/favorite-folders [post]
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
	row := video.FavoriteFolder{
		UserID:      uid,
		Title:       title,
		Description: desc,
		IsPublic:    isPublic,
		IsDefault:   false,
	}
	if err := a.FavoriteSvc.CreateFolder(context.Background(), &row); err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	resp.OK(c, a.folderRowPayload(&row))
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
	row := video.FavoriteFolder{
		UserID:      uid,
		Title:       title,
		Description: desc,
		IsPublic:    isPublic,
		IsDefault:   false,
	}
	if err := a.FavoriteSvc.CreateFolder(context.Background(), &row); err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	if fh, err := c.FormFile("cover"); err == nil && fh != nil {
		url, code := a.uploadFavoriteFolderCover(uid, row.ID, fh)
		if code != 0 {
			_ = a.FavoriteSvc.DeleteFolder(context.Background(), row.ID)
			resp.Err(c, http.StatusBadRequest, code)
			return
		}
		if url != "" {
			if err := a.FavoriteSvc.UpdateFolderCover(context.Background(), row.ID, url); err != nil {
				purgeFavoriteFolderCoverURL(a.Cfg, a.OSS, a.Log, url, uid, row.ID)
				_ = a.FavoriteSvc.DeleteFolder(context.Background(), row.ID)
				resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
				return
			}
			row.CoverURL = url
		}
	}
	resp.OK(c, a.folderRowPayload(&row))
}

// ListUserFavoriteFolders returns favorite folders for a user's space (public, or all if viewer is owner).
func (a *API) ListUserFavoriteFolders(c *gin.Context) {
	ownerID, err := strconv.ParseUint(c.Param("userId"), 10, 64)
	if err != nil || ownerID == 0 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	u, err := a.UserSvc.GetUserPublic(context.Background(), ownerID)
	if err != nil {
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
	total, _ = a.FavoriteSvc.CountFoldersByUser(context.Background(), ownerID)
	hiddenCount := int64(0)
	if viewerOK && viewer == ownerID {
		publicCnt, _ := a.FavoriteSvc.CountPublicFoldersByUser(context.Background(), ownerID)
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

func (a *API) loadUserFavoriteFolder(uid, folderID uint64) (video.FavoriteFolder, int) {
	f, err := a.FavoriteSvc.GetFolderByID(context.Background(), folderID)
	if err != nil || f.UserID != uid {
		return video.FavoriteFolder{}, errcode.CodeNotFound
	}
	return *f, 0
}

// UpdateFavoriteFolder updates folder metadata (json or multipart).
// UpdateFavoriteFolder godoc
// @Summary      Update a favorite folder
// @Description  Update the name of a favorite folder
// @Tags         Favorites
// @Produce      json
// @Param        folderId path int true "Folder ID"
// @Param        body body object{name=string} true "New folder name"
// @Success      200 {object} map[string]interface{}
// @Router       /users/me/favorite-folders/{folderId} [put]
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
	if err := a.FavoriteSvc.UpdateFolder(context.Background(), row.ID, updates); err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	ptr, _ := a.FavoriteSvc.GetFolderByID(context.Background(), row.ID)
	if ptr != nil {
		row = *ptr
	}
	resp.OK(c, a.folderRowPayload(&row))
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
	if err := a.FavoriteSvc.UpdateFolder(context.Background(), row.ID, updates); err != nil {
		if uploadedCoverURL != "" {
			purgeFavoriteFolderCoverURL(a.Cfg, a.OSS, a.Log, uploadedCoverURL, uid, row.ID)
		}
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	ptr, _ := a.FavoriteSvc.GetFolderByID(context.Background(), row.ID)
	if ptr != nil {
		row = *ptr
	}
	resp.OK(c, a.folderRowPayload(&row))
}

// DeleteFavoriteFolder removes a non-default folder and its favorites.
// DeleteFavoriteFolder godoc
// @Summary      Delete a favorite folder
// @Description  Remove a favorite folder (favorites inside are ungrouped)
// @Tags         Favorites
// @Produce      json
// @Param        folderId path int true "Folder ID"
// @Success      200 {object} map[string]interface{}
// @Router       /users/me/favorite-folders/{folderId} [delete]
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
	err := a.FavoriteSvc.DeleteFolderCascade(context.Background(), folderID, uid)
	if err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	purgeFavoriteFolderOSSObjects(a.Cfg, a.OSS, a.Log, row)
	resp.OK(c, gin.H{"deleted": true})
}

func (a *API) validateFolderOwnedByUser(uid, folderID uint64) bool {
	f, err := a.FavoriteSvc.GetFolderByID(context.Background(), folderID)
	return err == nil && f.UserID == uid
}

// @Summary      Clear invalid favorites in folder
// @Description  Remove references to deleted videos from a folder
// @Tags         Favorites
// @Produce      json
// @Param        folderId path int true "Folder ID"
// @Success      200 {object} map[string]interface{}
// @Router       /users/me/favorite-folders/{folderId}/invalid-favorites [delete]
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
	favs, _, err := a.FavoriteSvc.ListFavoritesByFolder(context.Background(), folderID, 0, 0)
	if err != nil {
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
	publishedIDs, err := a.FavoriteSvc.FilterPublishedVideoIDs(context.Background(), vids)
	if err != nil {
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
		if err := a.FavoriteSvc.DeleteFavoriteByVideo(context.Background(), uid, folderID, vid); err != nil {
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
// BatchRemoveVideosFromFavoriteFolder godoc
// @Summary      Batch remove videos from folder
// @Description  Remove multiple videos from a favorite folder at once
// @Tags         Favorites
// @Produce      json
// @Param        folderId path int true "Folder ID"
// @Param        body body object{video_ids=array} true "Video IDs to remove"
// @Success      200 {object} map[string]interface{}
// @Router       /users/me/favorite-folders/{folderId}/batch-remove [post]
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
		err = a.FavoriteSvc.DeleteFavoriteByVideo(context.Background(), uid, folderID, vid)
		if err != nil {
			resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
			return
		}
		if err == nil {
			removed++
		}
		after, _ := a.userVideoFavoriteCount(uid, vid)
		a.syncVideoFavCountAfterUserChange(vid, before, after)
	}
	resp.OK(c, gin.H{"removed": removed})
}
