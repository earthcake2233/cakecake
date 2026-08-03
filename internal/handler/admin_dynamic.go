package handler

import (
	"time"

	"cakecake/internal/model/dynamic"
	"cakecake/internal/model/user"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"cakecake/internal/errcode"
	"cakecake/internal/middleware"
	"cakecake/internal/pkg/resp"
)

type adminDynamicItem struct {
	ID           uint64    `json:"id"`
	Title        string    `json:"title"`
	Content      string    `json:"content"`
	Images       []string  `json:"images"`
	CoverURL     string    `json:"cover_url"`
	UserID       uint64    `json:"user_id"`
	UploaderName string    `json:"uploader_name"`
	LikeCount    uint64    `json:"like_count"`
	CommentCount uint64    `json:"comment_count"`
	CreatedAt    time.Time `json:"created_at"`
}

type adminDynamicListResponse struct {
	Items      []adminDynamicItem `json:"items"`
	Page       int                `json:"page"`
	PageSize   int                `json:"page_size"`
	Total      int64              `json:"total"`
	TotalPages int                `json:"total_pages"`
}

func adminDynamicToJSON(d *dynamic.UserDynamic, authorName string) adminDynamicItem {
	imgs := parseDynamicImagesJSON(d.ImagesJSON)
	if imgs == nil {
		imgs = []string{}
	}
	cover := ""
	if len(imgs) > 0 {
		cover = imgs[0]
	}
	return adminDynamicItem{
		ID:           d.ID,
		Title:        d.Title,
		Content:      d.Content,
		Images:       imgs,
		CoverURL:     cover,
		UserID:       d.UserID,
		UploaderName: authorName,
		LikeCount:    d.LikeCount,
		CommentCount: d.CommentCount,
		CreatedAt:    d.CreatedAt,
	}
}

// AdminListDynamics GET /api/v1/admin/dynamics — dynamics do not require review; ops can view and delete.

func (a *API) AdminListDynamics(c *gin.Context) {
	page, pageSize := parsePagination(c, 20)
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
	items := make([]adminDynamicItem, 0, len(result.Rows))
	for i := range result.Rows {
		items = append(items, adminDynamicToJSON(&result.Rows[i], names[result.Rows[i].UserID]))
	}
	resp.OK(c, adminDynamicListResponse{
		Items:      items,
		Page:       page,
		PageSize:   pageSize,
		Total:      result.Total,
		TotalPages: totalPages,
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
	a.StorageSvc.PurgeDynamic(*dyn)
	a.Log.Info("admin deleted dynamic",
		zap.Uint64("dynamic_id", id),
		zap.Uint64("admin_id", adminID),
		zap.Uint64("user_id", dyn.UserID),
	)
	resp.OK(c, okResponse{OK: true})
}
