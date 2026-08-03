package handler

import (
	"cakecake/internal/model/video"
	"context"
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

	"cakecake/internal/errcode"
	"cakecake/internal/middleware"
	"cakecake/internal/pkg/coverval"
	"cakecake/internal/pkg/resp"
)

const defaultFavoriteFolderTitle = "默认收藏夹"

type createFavoriteFolderJSON struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	IsPublic    *bool  `json:"is_public"`
}

func (a *API) ensureDefaultFavoriteFolder(ctx context.Context, userID uint64) (video.FavoriteFolder, error) {
	folders, err := a.FavoriteSvc.ListFoldersByUser(ctx, userID)
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
	if err := a.FavoriteSvc.CreateFolder(ctx, &f); err != nil {
		return f, err
	}
	_ = a.FavoriteSvc.MigrateOrphanFavorites(ctx, userID, f.ID)
	return f, nil
}

func (a *API) folderCoverFromVideos(ctx context.Context, folderID uint64) string {
	return a.FavoriteSvc.FolderCoverFromVideos(ctx, folderID)
}

func (a *API) resolveFolderCoverURL(ctx context.Context, f *video.FavoriteFolder) string {
	if u := strings.TrimSpace(f.CoverURL); u != "" {
		return u
	}
	return a.folderCoverFromVideos(ctx, f.ID)
}

type folderItemDTO struct {
	ID          uint64  `json:"id"`
	Title       string  `json:"title"`
	Description string  `json:"description"`
	IsPublic    bool    `json:"is_public"`
	IsDefault   bool    `json:"is_default"`
	VideoCount  int64   `json:"video_count"`
	CoverURL    *string `json:"cover_url"`
}

type folderListResponse struct {
	Items []folderItemDTO `json:"items"`
}

type folderListWithCountsResponse struct {
	Items       []folderItemDTO `json:"items"`
	Total       int64           `json:"total"`
	HiddenCount int64           `json:"hidden_count"`
}

type clearedResponse struct {
	Cleared int `json:"cleared"`
}

type removedResponse struct {
	Removed int `json:"removed"`
}

func (a *API) folderRowPayload(ctx context.Context, f *video.FavoriteFolder) folderItemDTO {
	vCnt, _ := a.FavoriteSvc.CountFavoritesInFolder(ctx, f.ID)
	cover := a.resolveFolderCoverURL(ctx, f)
	out := folderItemDTO{
		ID:          f.ID,
		Title:       f.Title,
		Description: f.Description,
		IsPublic:    f.IsPublic,
		IsDefault:   f.IsDefault,
		VideoCount:  vCnt,
	}
	if cover != "" {
		out.CoverURL = &cover
	}
	return out
}

func (a *API) folderListPayload(ctx context.Context, userID uint64, publicOnly bool) ([]folderItemDTO, error) {
	if _, err := a.ensureDefaultFavoriteFolder(ctx, userID); err != nil {
		return nil, err
	}
	folders, err := a.FavoriteSvc.ListFoldersByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]folderItemDTO, 0, len(folders))
	for i := range folders {
		if publicOnly && !folders[i].IsPublic {
			continue
		}
		out = append(out, a.folderRowPayload(ctx, &folders[i]))
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
	if !a.StorageSvc.Enabled() {
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
	if err := a.StorageSvc.UploadFile(key, tmp); err != nil {
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
	items, err := a.folderListPayload(c.Request.Context(), uid, false)
	if err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	resp.OK(c, folderListResponse{Items: items})
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
	if _, err := a.ensureDefaultFavoriteFolder(c.Request.Context(), uid); err != nil {
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
	if err := a.FavoriteSvc.CreateFolder(c.Request.Context(), &row); err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	resp.OK(c, a.folderRowPayload(c.Request.Context(), &row))
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
	if _, err := a.ensureDefaultFavoriteFolder(c.Request.Context(), uid); err != nil {
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
	if err := a.FavoriteSvc.CreateFolder(c.Request.Context(), &row); err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	if fh, err := c.FormFile("cover"); err == nil && fh != nil {
		url, code := a.uploadFavoriteFolderCover(uid, row.ID, fh)
		if code != 0 {
			_ = a.FavoriteSvc.DeleteFolder(c.Request.Context(), row.ID)
			resp.Err(c, http.StatusBadRequest, code)
			return
		}
		if url != "" {
			if err := a.FavoriteSvc.UpdateFolderCover(c.Request.Context(), row.ID, url); err != nil {
				a.StorageSvc.PurgeFavoriteFolderCoverURL(url, uid, row.ID)
				_ = a.FavoriteSvc.DeleteFolder(c.Request.Context(), row.ID)
				resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
				return
			}
			row.CoverURL = url
		}
	}
	resp.OK(c, a.folderRowPayload(c.Request.Context(), &row))
}

// ListUserFavoriteFolders returns favorite folders for a user's space (public, or all if viewer is owner).
func (a *API) ListUserFavoriteFolders(c *gin.Context) {
	ownerID, err := strconv.ParseUint(c.Param("userId"), 10, 64)
	if err != nil || ownerID == 0 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	u, err := a.UserSvc.GetUserPublic(c.Request.Context(), ownerID)
	if err != nil {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	viewer, viewerOK := middleware.UserID(c)
	isOwner := viewerOK && viewer == ownerID
	if !isOwner && !u.PrivacyPublicFavorites {
		resp.OK(c, folderListWithCountsResponse{Items: []folderItemDTO{}, Total: 0, HiddenCount: 0})
		return
	}
	publicOnly := !isOwner
	items, err := a.folderListPayload(c.Request.Context(), ownerID, publicOnly)
	if err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	var total int64
	total, _ = a.FavoriteSvc.CountFoldersByUser(c.Request.Context(), ownerID)
	hiddenCount := int64(0)
	if viewerOK && viewer == ownerID {
		publicCnt, _ := a.FavoriteSvc.CountPublicFoldersByUser(c.Request.Context(), ownerID)
		hiddenCount = total - publicCnt
		if hiddenCount < 0 {
			hiddenCount = 0
		}
	}
	displayTotal := total
	if publicOnly {
		displayTotal = int64(len(items))
	}
	resp.OK(c, folderListWithCountsResponse{
		Items:       items,
		Total:       displayTotal,
		HiddenCount: hiddenCount,
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

func (a *API) loadUserFavoriteFolder(ctx context.Context, uid, folderID uint64) (video.FavoriteFolder, int) {
	f, err := a.FavoriteSvc.GetFolderByID(ctx, folderID)
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
	row, code := a.loadUserFavoriteFolder(c.Request.Context(), uid, folderID)
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
	if err := a.FavoriteSvc.UpdateFolder(c.Request.Context(), row.ID, updates); err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	ptr, _ := a.FavoriteSvc.GetFolderByID(c.Request.Context(), row.ID)
	if ptr != nil {
		row = *ptr
	}
	resp.OK(c, a.folderRowPayload(c.Request.Context(), &row))
}

func (a *API) updateFavoriteFolderMultipart(c *gin.Context, uid, folderID uint64) {
	if err := c.Request.ParseMultipartForm(12 << 20); err != nil {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	row, code := a.loadUserFavoriteFolder(c.Request.Context(), uid, folderID)
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
	if err := a.FavoriteSvc.UpdateFolder(c.Request.Context(), row.ID, updates); err != nil {
		if uploadedCoverURL != "" {
			a.StorageSvc.PurgeFavoriteFolderCoverURL(uploadedCoverURL, uid, row.ID)
		}
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	ptr, _ := a.FavoriteSvc.GetFolderByID(c.Request.Context(), row.ID)
	if ptr != nil {
		row = *ptr
	}
	resp.OK(c, a.folderRowPayload(c.Request.Context(), &row))
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
	row, code := a.loadUserFavoriteFolder(c.Request.Context(), uid, folderID)
	if code != 0 {
		resp.Err(c, http.StatusNotFound, code)
		return
	}
	if row.IsDefault {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	err := a.FavoriteSvc.DeleteFolderCascade(c.Request.Context(), folderID, uid)
	if err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	a.StorageSvc.PurgeFavoriteFolder(row)
	resp.OK(c, deletedResponse{Deleted: true})
}

func (a *API) validateFolderOwnedByUser(ctx context.Context, uid, folderID uint64) bool {
	f, err := a.FavoriteSvc.GetFolderByID(ctx, folderID)
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
	if !a.validateFolderOwnedByUser(c.Request.Context(), uid, folderID) {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	favs, _, err := a.FavoriteSvc.ListFavoritesByFolder(c.Request.Context(), folderID, 0, 0)
	if err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	if len(favs) == 0 {
		resp.OK(c, clearedResponse{Cleared: 0})
		return
	}
	vids := make([]uint64, 0, len(favs))
	for i := range favs {
		vids = append(vids, favs[i].VideoID)
	}
	publishedIDs, err := a.FavoriteSvc.FilterPublishedVideoIDs(c.Request.Context(), vids)
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
		resp.OK(c, clearedResponse{Cleared: 0})
		return
	}
	for _, vid := range invalidVids {
		before, err := a.userVideoFavoriteCount(c.Request.Context(), uid, vid)
		if err != nil {
			resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
			return
		}
		if err := a.FavoriteSvc.DeleteFavoriteByVideo(c.Request.Context(), uid, folderID, vid); err != nil {
			resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
			return
		}
		after, _ := a.userVideoFavoriteCount(c.Request.Context(), uid, vid)
		a.syncVideoFavCountAfterUserChange(c.Request.Context(), vid, before, after)
	}
	resp.OK(c, clearedResponse{Cleared: len(invalidVids)})
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
	if !a.validateFolderOwnedByUser(c.Request.Context(), uid, folderID) {
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
		before, err := a.userVideoFavoriteCount(c.Request.Context(), uid, vid)
		if err != nil {
			resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
			return
		}
		err = a.FavoriteSvc.DeleteFavoriteByVideo(c.Request.Context(), uid, folderID, vid)
		if err != nil {
			resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
			return
		}
		if err == nil {
			removed++
		}
		after, _ := a.userVideoFavoriteCount(c.Request.Context(), uid, vid)
		a.syncVideoFavCountAfterUserChange(c.Request.Context(), vid, before, after)
	}
	resp.OK(c, removedResponse{Removed: removed})
}
