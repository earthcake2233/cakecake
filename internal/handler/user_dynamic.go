package handler

import (
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

	"minibili/internal/errcode"
	"minibili/internal/middleware"
	"minibili/internal/model"
	"minibili/internal/pkg/coverval"
	"minibili/internal/pkg/resp"
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

func userDynamicAuthorName(author *model.User) string {
	if author == nil || model.IsUserAnonymized(author) {
		return ""
	}
	if author.Nickname != "" {
		return strings.TrimSpace(author.Nickname)
	}
	return model.DisplayUsername(author)
}

func userDynamicReadPayload(d *model.UserDynamic, author *model.User, likedByMe bool, viewer uint64) gin.H {
	out := userDynamicPayload(d, likedByMe)
	out["user_id"] = d.UserID
	out["author_name"] = userDynamicAuthorName(author)
	out["author_avatar"] = uploaderAvatarForAPI(author)
	out["is_author"] = viewer > 0 && viewer == d.UserID
	return out
}

// GetUserDynamic returns a single user dynamic for reading (public).
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
	var author model.User
	_ = a.DB.First(&author, dyn.UserID).Error
	var viewer uint64
	if uid, ok := middleware.UserID(c); ok {
		viewer = uid
	}
	likedMap := dynamicLikesByViewer(a.DB, viewer, []uint64{id})
	resp.OK(c, userDynamicReadPayload(dyn, &author, likedMap[id], viewer))
}

func userDynamicPayload(d *model.UserDynamic, likedByMe bool) gin.H {
	imgs := parseDynamicImagesJSON(d.ImagesJSON)
	if imgs == nil {
		imgs = []string{}
	}
	return gin.H{
		"id":               d.ID,
		"title":            d.Title,
		"content":          d.Content,
		"images":           imgs,
		"like_count":       d.LikeCount,
		"comment_count":    d.CommentCount,
		"liked_by_me":      likedByMe,
		"comments_closed":  d.CommentsClosed,
		"comments_curated": d.CommentsCurated,
		"created_at":       d.CreatedAt.Format("2006-01-02 15:04:05"),
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
	var dyn model.UserDynamic
	if err := a.DB.First(&dyn, id).Error; err != nil {
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
	if err := a.DB.Model(&dyn).Updates(updates).Error; err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	if err := a.DB.First(&dyn, id).Error; err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	resp.OK(c, gin.H{
		"comments_closed":  dyn.CommentsClosed,
		"comments_curated": dyn.CommentsCurated,
	})
}

func dynamicLikesByViewer(db *gorm.DB, viewer uint64, ids []uint64) map[uint64]bool {
	out := make(map[uint64]bool)
	if viewer == 0 || len(ids) == 0 {
		return out
	}
	var rows []model.UserDynamicLike
	_ = db.Where("user_id = ? AND dynamic_id IN ?", viewer, ids).Find(&rows).Error
	for _, r := range rows {
		out[r.DynamicID] = true
	}
	return out
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
	dyn := model.UserDynamic{
		UserID:     uid,
		Title:      title,
		Content:    content,
		ImagesJSON: string(imgsJSON),
	}
	if err := a.DB.Create(&dyn).Error; err != nil {
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
	var dyn model.UserDynamic
	if err := a.DB.First(&dyn, id).Error; err != nil {
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
	dyn.Content = content
	dyn.ImagesJSON = string(imgsJSON)
	if err := a.DB.Save(&dyn).Error; err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	purgeRemovedDynamicImageURLs(a.Cfg, a.OSS, a.Log, oldURLs, imageURLs)
	resp.OK(c, userDynamicPayload(&dyn, false))
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
	if _, ok := loadUserDynamic(a, did); !ok {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	var like model.UserDynamicLike
	res := a.DB.Where("user_id = ? AND dynamic_id = ?", uid, did).Limit(1).Find(&like)
	if res.Error != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	if res.RowsAffected == 0 {
		if err := a.DB.Create(&model.UserDynamicLike{UserID: uid, DynamicID: did}).Error; err != nil {
			resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
			return
		}
		_ = a.DB.Model(&model.UserDynamic{}).Where("id = ?", did).
			UpdateColumn("like_count", gorm.Expr("like_count + ?", 1)).Error
		resp.OK(c, gin.H{"liked": true, "like_count_delta": 1})
		return
	}
	if err := a.DB.Delete(&like).Error; err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	_ = a.DB.Model(&model.UserDynamic{}).Where("id = ?", did).
		UpdateColumn("like_count", gorm.Expr("GREATEST(like_count - ?, 0)", 1)).Error
	resp.OK(c, gin.H{"liked": false, "like_count_delta": -1})
}

func deleteUserDynamicCascade(tx *gorm.DB, id uint64) error {
	var cids []uint64
	_ = tx.Model(&model.DynamicComment{}).Where("dynamic_id = ?", id).Pluck("id", &cids).Error
	if len(cids) > 0 {
		if err := tx.Where("comment_id IN ?", cids).Delete(&model.DynamicCommentLike{}).Error; err != nil {
			return err
		}
		if err := tx.Where("comment_id IN ?", cids).Delete(&model.DynamicCommentDislike{}).Error; err != nil {
			return err
		}
		if err := tx.Where("dynamic_id = ?", id).Delete(&model.DynamicComment{}).Error; err != nil {
			return err
		}
	}
	if err := tx.Where("dynamic_id = ?", id).Delete(&model.UserDynamicLike{}).Error; err != nil {
		return err
	}
	return tx.Where("id = ?", id).Delete(&model.UserDynamic{}).Error
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
	var dyn model.UserDynamic
	if err := a.DB.First(&dyn, id).Error; err != nil {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	if dyn.UserID != uid {
		resp.Err(c, http.StatusForbidden, errcode.CodeForbidden)
		return
	}
	if err := a.DB.Transaction(func(tx *gorm.DB) error {
		return deleteUserDynamicCascade(tx, id)
	}); err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	purgeDynamicOSSObjects(a.Cfg, a.OSS, a.Log, dyn)
	resp.OK(c, gin.H{"ok": true})
}

func orderClauseForMyDynamics(sort string) string {
	switch strings.TrimSpace(sort) {
	case "reply":
		return "comment_count DESC, id DESC"
	case "like":
		return "like_count DESC, id DESC"
	default:
		return "id DESC"
	}
}

// ListMyDynamics lists the current user's image/text dynamics (稿件管理).
// Query: page, page_size, sort(time|reply|like), q(title or content).
func (a *API) ListMyDynamics(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 50 {
		pageSize = 10
	}
	sortKey := strings.TrimSpace(c.DefaultQuery("sort", "time"))
	titleQ := strings.TrimSpace(c.Query("q"))

	filtered := a.DB.Model(&model.UserDynamic{}).Where("user_id = ?", uid)
	if titleQ != "" {
		filtered = filtered.Where("title LIKE ? OR content LIKE ?", "%"+titleQ+"%", "%"+titleQ+"%")
	}
	var total int64
	if err := filtered.Count(&total).Error; err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	totalPages := int((total + int64(pageSize) - 1) / int64(pageSize))
	if totalPages < 1 {
		totalPages = 1
	}
	if page > totalPages {
		page = totalPages
	}
	offset := (page - 1) * pageSize
	var list []model.UserDynamic
	if err := filtered.Order(orderClauseForMyDynamics(sortKey)).
		Offset(offset).Limit(pageSize).Find(&list).Error; err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	items := make([]gin.H, 0, len(list))
	for i := range list {
		items = append(items, userDynamicPayload(&list[i], false))
	}
	resp.OK(c, gin.H{
		"items":       items,
		"page":        page,
		"page_size":   pageSize,
		"total":       total,
		"total_pages": totalPages,
	})
}

// ListUserPublishedDynamics lists a user's image/text dynamics (public).
func (a *API) ListUserPublishedDynamics(c *gin.Context) {
	uid, err := strconv.ParseUint(c.Param("userId"), 10, 64)
	if err != nil || uid == 0 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	var u model.User
	if err := a.DB.First(&u, uid).Error; err != nil {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	if model.IsUserAnonymized(&u) {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	limit := 20
	if s := c.Query("limit"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 && n <= 50 {
			limit = n
		}
	}
	curID, _ := strconv.ParseUint(c.Query("cursor"), 10, 64)
	q := a.DB.Model(&model.UserDynamic{}).Where("user_id = ?", uid)
	if curID > 0 {
		q = q.Where("id < ?", curID)
	}
	var list []model.UserDynamic
	if err := q.Order("id DESC").Limit(limit + 1).Find(&list).Error; err != nil {
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
	likedMap := dynamicLikesByViewer(a.DB, viewer, ids)
	items := make([]gin.H, 0, len(list))
	for i := range list {
		items = append(items, userDynamicPayload(&list[i], likedMap[list[i].ID]))
	}
	nextCursor := ""
	if hasMore && len(list) > 0 {
		nextCursor = strconv.FormatUint(list[len(list)-1].ID, 10)
	}
	resp.OK(c, gin.H{
		"items":       items,
		"next_cursor": nextCursor,
	})
}
