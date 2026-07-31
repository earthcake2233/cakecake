package handler

import (
	"minibili/internal/model/dynamic"
	"minibili/internal/model/user"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"minibili/internal/errcode"
	"minibili/internal/middleware"
	"minibili/internal/pkg/resp"
)

func adminDynamicToJSON(d *dynamic.UserDynamic, authorName string) gin.H {
	imgs := parseDynamicImagesJSON(d.ImagesJSON)
	if imgs == nil {
		imgs = []string{}
	}
	cover := ""
	if len(imgs) > 0 {
		cover = imgs[0]
	}
	return gin.H{
		"id":            d.ID,
		"title":         d.Title,
		"content":       d.Content,
		"images":        imgs,
		"cover_url":     cover,
		"user_id":       d.UserID,
		"uploader_name": authorName,
		"like_count":    d.LikeCount,
		"comment_count": d.CommentCount,
		"created_at":    d.CreatedAt,
	}
}

// AdminListDynamics GET /api/v1/admin/dynamics — dynamics do not require review; ops can view and delete.

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

	result, err := a.DynamicSvc.AdminListDynamics(c.Request.Context(), q, page, pageSize)
	if err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}

	totalPages := int((result.Total + int64(pageSize) - 1) / int64(pageSize))
	if totalPages < 1 {
		totalPages = 1
	}
	if page > totalPages {
		page = totalPages
	}

	uids := make([]uint64, 0, len(result.Rows))
	for i := range result.Rows {
		uids = append(uids, result.Rows[i].UserID)
	}
	names := map[uint64]string{}
	if len(uids) > 0 {
		usersMap := a.UserSvc.BatchGetUsers(c.Request.Context(), uids)
		for id, u := range usersMap {
			name := user.DisplayUsername(u)
			if u.Nickname != "" && !user.IsUserAnonymized(u) {
				name = strings.TrimSpace(u.Nickname)
			}
			names[id] = name
		}
	}
	items := make([]gin.H, 0, len(result.Rows))
	for i := range result.Rows {
		items = append(items, adminDynamicToJSON(&result.Rows[i], names[result.Rows[i].UserID]))
	}
	resp.OK(c, gin.H{
		"items":       items,
		"page":        page,
		"page_size":   pageSize,
		"total":       result.Total,
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
	dyn, err := a.DynamicSvc.GetDynamicByID(c.Request.Context(), id)
	if err != nil {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	u, _ := a.UserSvc.GetUserByID(c.Request.Context(), dyn.UserID)
	name := user.DisplayUsername(u)
	if u.Nickname != "" && !user.IsUserAnonymized(u) {
		name = strings.TrimSpace(u.Nickname)
	}
	resp.OK(c, adminDynamicToJSON(dyn, name))
}

// AdminDeleteDynamic POST /api/v1/admin/dynamics/:id/delete or DELETE /api/v1/admin/dynamics/:id
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
	dyn, err := a.DynamicSvc.GetDynamicByID(c.Request.Context(), id)
	if err != nil {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	if err := a.DynamicSvc.AdminDeleteDynamicCascade(c.Request.Context(), id, func(tx *gorm.DB) error {
		return deleteUserDynamicCascade(tx, id)
	}); err != nil {
		a.Log.Error("admin delete dynamic", zap.Error(err), zap.Uint64("dynamic_id", id))
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	purgeDynamicOSSObjects(a.Cfg, a.OSS, a.Log, *dyn)
	a.Log.Info("admin deleted dynamic",
		zap.Uint64("dynamic_id", id),
		zap.Uint64("admin_id", adminID),
		zap.Uint64("user_id", dyn.UserID),
	)
	resp.OK(c, gin.H{"ok": true})
}
