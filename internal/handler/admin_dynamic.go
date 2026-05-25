package handler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"minibili/internal/errcode"
	"minibili/internal/middleware"
	"minibili/internal/model"
	"minibili/internal/pkg/resp"
)

func adminDynamicToJSON(d *model.UserDynamic, authorName string) gin.H {
	imgs := parseDynamicImagesJSON(d.ImagesJSON)
	if imgs == nil {
		imgs = []string{}
	}
	cover := ""
	if len(imgs) > 0 {
		cover = imgs[0]
	}
	return gin.H{
		"id":             d.ID,
		"title":          d.Title,
		"content":        d.Content,
		"images":         imgs,
		"cover_url":      cover,
		"user_id":        d.UserID,
		"uploader_name":  authorName,
		"like_count":     d.LikeCount,
		"comment_count": d.CommentCount,
		"created_at":    d.CreatedAt,
	}
}

// AdminListDynamics GET /api/v1/admin/dynamics — 动态无需审核，运营可查看与删除。
func (a *API) AdminListDynamics(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 50 {
		pageSize = 20
	}
	q := strings.TrimSpace(c.Query("q"))

	dbq := a.DB.Model(&model.UserDynamic{})
	if q != "" {
		dbq = dbq.Where("title LIKE ? OR content LIKE ?", "%"+q+"%", "%"+q+"%")
	}
	var total int64
	if err := dbq.Count(&total).Error; err != nil {
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
	var rows []model.UserDynamic
	if err := dbq.Order("created_at DESC, id DESC").Offset(offset).Limit(pageSize).Find(&rows).Error; err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	uids := make([]uint64, 0, len(rows))
	for i := range rows {
		uids = append(uids, rows[i].UserID)
	}
	names := map[uint64]string{}
	if len(uids) > 0 {
		var users []model.User
		_ = a.DB.Where("id IN ?", uids).Find(&users).Error
		for i := range users {
			names[users[i].ID] = model.DisplayUsername(&users[i])
			if users[i].Nickname != "" && !model.IsUserAnonymized(&users[i]) {
				names[users[i].ID] = strings.TrimSpace(users[i].Nickname)
			}
		}
	}
	items := make([]gin.H, 0, len(rows))
	for i := range rows {
		items = append(items, adminDynamicToJSON(&rows[i], names[rows[i].UserID]))
	}
	resp.OK(c, gin.H{
		"items":       items,
		"page":        page,
		"page_size":   pageSize,
		"total":       total,
		"total_pages": totalPages,
	})
}

// AdminGetDynamic GET /api/v1/admin/dynamics/:id
func (a *API) AdminGetDynamic(c *gin.Context) {
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
	var u model.User
	_ = a.DB.First(&u, dyn.UserID).Error
	name := model.DisplayUsername(&u)
	if u.Nickname != "" && !model.IsUserAnonymized(&u) {
		name = strings.TrimSpace(u.Nickname)
	}
	resp.OK(c, adminDynamicToJSON(&dyn, name))
}

// AdminDeleteDynamic POST /api/v1/admin/dynamics/:id/delete 或 DELETE /api/v1/admin/dynamics/:id
func (a *API) AdminDeleteDynamic(c *gin.Context) {
	adminID, ok := middleware.AdminID(c)
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
	if err := a.DB.Transaction(func(tx *gorm.DB) error {
		return deleteUserDynamicCascade(tx, id)
	}); err != nil {
		a.Log.Error("admin delete dynamic", zap.Error(err), zap.Uint64("dynamic_id", id))
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	purgeDynamicOSSObjects(a.Cfg, a.OSS, a.Log, dyn)
	a.Log.Info("admin deleted dynamic",
		zap.Uint64("dynamic_id", id),
		zap.Uint64("admin_id", adminID),
		zap.Uint64("user_id", dyn.UserID),
	)
	resp.OK(c, gin.H{"ok": true})
}
