package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"minibili/internal/errcode"
	"minibili/internal/middleware"
	"minibili/internal/model"
	"minibili/internal/pkg/resp"
)

// ToggleVideoLike toggles the current user's like on a published video.
func (a *API) ToggleVideoLike(c *gin.Context) {
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
	var v model.Video
	if err := a.DB.First(&v, vid).Error; err != nil {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	if v.Status != "published" {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	var like model.VideoLike
	res := a.DB.Where("user_id = ? AND video_id = ?", uid, vid).Limit(1).Find(&like)
	if res.Error != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	if res.RowsAffected == 0 {
		lk := model.VideoLike{UserID: uid, VideoID: vid}
		if err := a.DB.Create(&lk).Error; err != nil {
			resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
			return
		}
		_ = a.DB.Model(&model.Video{}).Where("id = ?", vid).UpdateColumn("like_count", gorm.Expr("like_count + ?", 1)).Error
		resp.OK(c, gin.H{"liked": true})
		return
	}
	if err := a.DB.Delete(&like).Error; err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	_ = a.DB.Model(&model.Video{}).Where("id = ?", vid).UpdateColumn("like_count", gorm.Expr("GREATEST(like_count - ?, 0)", 1)).Error
	resp.OK(c, gin.H{"liked": false})
}
