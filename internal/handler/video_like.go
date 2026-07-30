package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"minibili/internal/errcode"
	"minibili/internal/middleware"
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
	liked, err := a.VideoSvc.ToggleVideoLike(c.Request.Context(), uid, vid)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		} else {
			resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		}
		return
	}
	resp.OK(c, gin.H{"liked": liked})
}
