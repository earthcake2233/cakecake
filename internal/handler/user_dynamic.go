package handler

import (
	"cakecake/internal/model/comment"
	"cakecake/internal/model/dynamic"
	"cakecake/internal/model/user"
	"context"
	"encoding/json"
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

	"cakecake/internal/errcode"
	"cakecake/internal/middleware"
	"cakecake/internal/pkg/coverval"
	"cakecake/internal/pkg/resp"
	"cakecake/internal/service"
)

const (
	maxDynamicTitleRunes   = 20
	maxDynamicContentRunes = 233
	maxDynamicImages       = 9
)

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

func userDynamicAuthorName(author *user.User) string {
	if author == nil || user.IsUserAnonymized(author) {
		return ""
	}
	if author.Nickname != "" {
		return strings.TrimSpace(author.Nickname)
	}
	return user.DisplayUsername(author)
}

type userDynamicItemDTO struct {
	ID              uint64   `json:"id"`
	Title           string   `json:"title"`
	Content         string   `json:"content"`
	Images          []string `json:"images"`
	LikeCount       uint64   `json:"like_count"`
	CommentCount    uint64   `json:"comment_count"`
	LikedByMe       bool     `json:"liked_by_me"`
	CommentsClosed  bool     `json:"comments_closed"`
	CommentsCurated bool     `json:"comments_curated"`
	CreatedAt       string   `json:"created_at"`
}

type userDynamicReadDTO struct {
	userDynamicItemDTO
	UserID       uint64 `json:"user_id"`
	AuthorName   string `json:"author_name"`
	AuthorAvatar string `json:"author_avatar"`
	IsAuthor     bool   `json:"is_author"`
}

type dynamicPlaybackResponse struct {
	CommentsClosed  bool `json:"comments_closed"`
	CommentsCurated bool `json:"comments_curated"`
}

type dynamicLikeResponse struct {
	Liked          bool `json:"liked"`
	LikeCountDelta int  `json:"like_count_delta"`
}

type userDynamicListResponse struct {
	Items      []userDynamicItemDTO `json:"items"`
	Page       int                  `json:"page"`
	PageSize   int                  `json:"page_size"`
	Total      int64                `json:"total"`
	TotalPages int                  `json:"total_pages"`
}

type userDynamicCursorListResponse struct {
	Items      []userDynamicItemDTO `json:"items"`
	NextCursor string               `json:"next_cursor"`
}

func userDynamicReadPayload(d *dynamic.UserDynamic, author *user.User, likedByMe bool, viewer uint64) userDynamicReadDTO {
	return userDynamicReadDTO{
		userDynamicItemDTO: userDynamicPayload(d, likedByMe),
		UserID:             d.UserID,
		AuthorName:         userDynamicAuthorName(author),
		AuthorAvatar:       uploaderAvatarForAPI(author),
		IsAuthor:           viewer > 0 && viewer == d.UserID,
	}
}

// GetUserDynamic returns a single user dynamic for reading (public).
// GetUserDynamic godoc
// @Summary      Get dynamic detail
// @Description  Get full content of a dynamic post
// @Tags         Dynamics
// @Produce      json
// @Param        id path int true "Dynamic ID"
// @Success      200 {object} map[string]interface{}
// @Router       /user-dynamics/{id} [get]
func (a *API) GetUserDynamic(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	dyn, ok := loadUserDynamic(a, id)
	if !ok {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	var author user.User
	aP, _ := a.UserSvc.GetUserPublic(c.Request.Context(), dyn.UserID)
	if aP != nil {
		author.ID = aP.ID
		author.Username = aP.Username
		author.AvatarURL = aP.AvatarURL
	}
	var viewer uint64
	if uid, ok := middleware.UserID(c); ok {
		viewer = uid
	}
	likedMap := a.DynamicSvc.BatchCheckLiked(c.Request.Context(), viewer, []uint64{id})
	resp.OK(c, userDynamicReadPayload(dyn, &author, likedMap[id], viewer))
}

func userDynamicPayload(d *dynamic.UserDynamic, likedByMe bool) userDynamicItemDTO {
	imgs := parseDynamicImagesJSON(d.ImagesJSON)
	if imgs == nil {
		imgs = []string{}
	}
	return userDynamicItemDTO{
		ID:              d.ID,
		Title:           d.Title,
		Content:         d.Content,
		Images:          imgs,
		LikeCount:       d.LikeCount,
		CommentCount:    d.CommentCount,
		LikedByMe:       likedByMe,
		CommentsClosed:  d.CommentsClosed,
		CommentsCurated: d.CommentsCurated,
		CreatedAt:       d.CreatedAt.Format("2006-01-02 15:04:05"),
	}
}

type userDynamicPlaybackPatch struct {
	CommentsClosed  *bool `json:"comments_closed"`
	CommentsCurated *bool `json:"comments_curated"`
}

// PatchUserDynamicPlayback toggles comment settings on the caller's dynamic (owner only).
func (a *API) PatchUserDynamicPlayback(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	dyn, err := a.DynamicSvc.GetDynamicByID(context.Background(), id)
	if err != nil {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	if dyn.UserID != uid {
		resp.Err(c, http.StatusForbidden, errcode.CodeForbidden)
		return
	}
	var req userDynamicPlaybackPatch
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	if req.CommentsClosed == nil && req.CommentsCurated == nil {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	updates := map[string]interface{}{}
	if req.CommentsClosed != nil {
		updates["comments_closed"] = *req.CommentsClosed
	}
	if req.CommentsCurated != nil {
		updates["comments_curated"] = *req.CommentsCurated
	}
	if err := a.DynamicSvc.UpdateDynamic(context.Background(), id, updates); err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	d, e := a.DynamicSvc.GetDynamicByID(context.Background(), id)
	if e != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	dyn = d
	resp.OK(c, dynamicPlaybackResponse{
		CommentsClosed:  dyn.CommentsClosed,
		CommentsCurated: dyn.CommentsCurated,
	})
}

func (a *API) uploadDynamicImage(uid uint64, fh *multipart.FileHeader) (string, int) {
	if fh == nil {
		return "", errcode.CodeParamError
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
	if ext == "" {
		ext = "jpg"
	}
	key := fmt.Sprintf("dynamics/%d/%s.%s", uid, uuid.NewString(), ext)
	if err := a.OSS.UploadFile(key, tmp); err != nil {
		a.Log.Error("oss dynamic image upload", zap.Error(err))
		return "", errcode.CodeInternalError
	}
	return a.Cfg.OSSObjectURL(key), 0
}

// PostUserDynamic publishes an image/text dynamic (multipart: title, content, images[]).
func (a *API) PostUserDynamic(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	if err := c.Request.ParseMultipartForm(32 << 20); err != nil {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	title := strings.TrimSpace(c.PostForm("title"))
	content := strings.TrimSpace(c.PostForm("content"))
	if n := utf8.RuneCountInString(title); n > maxDynamicTitleRunes {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	if n := utf8.RuneCountInString(content); n > maxDynamicContentRunes {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	var files []*multipart.FileHeader
	if c.Request.MultipartForm != nil {
		files = c.Request.MultipartForm.File["images"]
	}
	if len(files) > maxDynamicImages {
		files = files[:maxDynamicImages]
	}
	if title == "" && content == "" && len(files) == 0 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	imageURLs := make([]string, 0, len(files))
	for _, fh := range files {
		url, code := a.uploadDynamicImage(uid, fh)
		if code != 0 {
			resp.Err(c, http.StatusBadRequest, code)
			return
		}
		imageURLs = append(imageURLs, url)
	}
	imgsJSON, err := json.Marshal(imageURLs)
	if err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	dyn := dynamic.UserDynamic{
		UserID:     uid,
		Title:      title,
		Content:    content,
		ImagesJSON: string(imgsJSON),
	}
	if err := a.DynamicSvc.CreateDynamic(context.Background(), &dyn); err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	resp.OK(c, userDynamicPayload(&dyn, false))
}

func parseKeepDynamicImagesForm(raw string) []string {
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

// PutMyUserDynamic updates the caller's image/text dynamic (multipart).
func (a *API) PutMyUserDynamic(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	dyn, err := a.DynamicSvc.GetDynamicByID(context.Background(), id)
	if err != nil {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	if dyn.UserID != uid {
		resp.Err(c, http.StatusForbidden, errcode.CodeForbidden)
		return
	}
	if err := c.Request.ParseMultipartForm(32 << 20); err != nil {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	title := strings.TrimSpace(c.PostForm("title"))
	content := strings.TrimSpace(c.PostForm("content"))
	if n := utf8.RuneCountInString(title); n > maxDynamicTitleRunes {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	if n := utf8.RuneCountInString(content); n > maxDynamicContentRunes {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	keepURLs := parseKeepDynamicImagesForm(c.PostForm("keep_images"))
	var files []*multipart.FileHeader
	if c.Request.MultipartForm != nil {
		files = c.Request.MultipartForm.File["images"]
	}
	remain := maxDynamicImages - len(keepURLs)
	if remain < 0 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	if len(files) > remain {
		files = files[:remain]
	}
	imageURLs := append([]string(nil), keepURLs...)
	for _, fh := range files {
		url, code := a.uploadDynamicImage(uid, fh)
		if code != 0 {
			resp.Err(c, http.StatusBadRequest, code)
			return
		}
		imageURLs = append(imageURLs, url)
	}
	if title == "" && content == "" && len(imageURLs) == 0 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	oldURLs := parseDynamicImagesJSON(dyn.ImagesJSON)
	imgsJSON, err := json.Marshal(imageURLs)
	if err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	dyn.Title = title
	updates := map[string]interface{}{
		"content":     content,
		"images_json": string(imgsJSON),
	}
	if err := a.DynamicSvc.UpdateDynamic(context.Background(), id, updates); err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	purgeRemovedDynamicImageURLs(a.Cfg, a.OSS, a.Log, oldURLs, imageURLs)
	resp.OK(c, userDynamicPayload(dyn, false))
}

// ToggleDynamicLike toggles the current user's like on a user dynamic.
func (a *API) ToggleDynamicLike(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	did, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || did == 0 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	liked, err := a.DynamicSvc.ToggleDynamicLike(context.Background(), uid, did)
	if err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	if liked {
		resp.OK(c, dynamicLikeResponse{Liked: true, LikeCountDelta: 1})
	} else {
		resp.OK(c, dynamicLikeResponse{Liked: false, LikeCountDelta: -1})
	}
}

func deleteUserDynamicCascade(tx *gorm.DB, id uint64) error {
	var cids []uint64
	_ = tx.Model(&comment.DynamicComment{}).Where("dynamic_id = ?", id).Pluck("id", &cids).Error
	if len(cids) > 0 {
		if err := tx.Where("comment_id IN ?", cids).Delete(&comment.DynamicCommentLike{}).Error; err != nil {
			return err
		}
		if err := tx.Where("comment_id IN ?", cids).Delete(&comment.DynamicCommentDislike{}).Error; err != nil {
			return err
		}
		if err := tx.Where("dynamic_id = ?", id).Delete(&comment.DynamicComment{}).Error; err != nil {
			return err
		}
	}
	if err := tx.Where("dynamic_id = ?", id).Delete(&comment.UserDynamicLike{}).Error; err != nil {
		return err
	}
	return tx.Where("id = ?", id).Delete(&dynamic.UserDynamic{}).Error
}

// DeleteMyDynamic deletes the caller's own image/text dynamic.
func (a *API) DeleteMyDynamic(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	dyn, err := a.DynamicSvc.GetDynamicByID(context.Background(), id)
	if err != nil {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	if dyn.UserID != uid {
		resp.Err(c, http.StatusForbidden, errcode.CodeForbidden)
		return
	}
	if err := a.DynamicSvc.DeleteDynamic(context.Background(), id); err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	purgeDynamicOSSObjects(a.Cfg, a.OSS, a.Log, *dyn)
	resp.OK(c, okResponse{OK: true})
}

// ListMyDynamics lists the current user's image/text dynamics (content management).
// Query: page, page_size, sort(time|reply|like), q(title or content).
func (a *API) ListMyDynamics(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	page, pageSize := parsePagination(c, 10)
	sortKey := strings.TrimSpace(c.DefaultQuery("sort", "time"))
	titleQ := strings.TrimSpace(c.Query("q"))

	resDyn, err := a.DynamicSvc.ListMyDynamicsAdvanced(c.Request.Context(), service.MyDynamicFilter{
		UserID: uid, TitleQ: titleQ, SortKey: sortKey, Page: page, PageSize: pageSize,
	})
	if err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	items := make([]userDynamicItemDTO, 0, len(resDyn.Dynamics))
	for i := range resDyn.Dynamics {
		items = append(items, userDynamicPayload(&resDyn.Dynamics[i], false))
	}
	resp.OK(c, userDynamicListResponse{
		Items:      items,
		Page:       page,
		PageSize:   pageSize,
		Total:      resDyn.Total,
		TotalPages: resDyn.TotalPages,
	})
}

// ListUserPublishedDynamics lists a user's image/text dynamics (public).
// ListUserPublishedDynamics godoc
// @Summary      List user dynamics
// @Description  Get paginated dynamics for a user space
// @Tags         Dynamics
// @Produce      json
// @Param        userId path int true "User ID"
// @Param        page query int false "Page number" default(1)
// @Param        page_size query int false "Page size" default(20)
// @Success      200 {object} map[string]interface{}
// @Router       /space/{userId}/dynamics [get]
func (a *API) ListUserPublishedDynamics(c *gin.Context) {
	uid, err := strconv.ParseUint(c.Param("userId"), 10, 64)
	if err != nil || uid == 0 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	var u user.User
	uP, err := a.UserSvc.GetUserPublic(c.Request.Context(), uid)
	if err != nil || uP == nil {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	u.ID = uP.ID
	u.Username = uP.Username
	u.AvatarURL = uP.AvatarURL
	if user.IsUserAnonymized(&u) {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	limit := parseLimit(c, 20, 50)
	curID, _ := strconv.ParseUint(c.Query("cursor"), 10, 64)
	list, err := a.DynamicSvc.ListUserDynamicsCursor(c.Request.Context(), uid, curID, limit+1)
	if err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	hasMore := len(list) > limit
	if hasMore {
		list = list[:limit]
	}
	var viewer uint64
	if uid, ok := middleware.UserID(c); ok {
		viewer = uid
	}
	ids := make([]uint64, 0, len(list))
	for _, d := range list {
		ids = append(ids, d.ID)
	}
	likedMap := a.DynamicSvc.BatchCheckLiked(c.Request.Context(), viewer, ids)
	items := make([]userDynamicItemDTO, 0, len(list))
	for i := range list {
		items = append(items, userDynamicPayload(&list[i], likedMap[list[i].ID]))
	}
	nextCursor := ""
	if hasMore && len(list) > 0 {
		nextCursor = strconv.FormatUint(list[len(list)-1].ID, 10)
	}
	resp.OK(c, userDynamicCursorListResponse{
		Items:      items,
		NextCursor: nextCursor,
	})
}
