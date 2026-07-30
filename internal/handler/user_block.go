package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"minibili/internal/errcode"
	"minibili/internal/middleware"
	"minibili/internal/pkg/resp"
)

// BlockUser adds peer to the caller's blacklist and removes mutual follows.
// BlockUser godoc
// @Summary      Block/unblock a user
// @Description  Toggle block status for a user
// @Tags         Users
// @Produce      json
// @Param        userId path int true "User ID to block/unblock"
// @Success      200 {object} map[string]interface{}
// @Router       /users/{userId}/block [post]
func (a *API) BlockUser(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	blockedID, err := strconv.ParseUint(c.Param("userId"), 10, 64)
	if err != nil || blockedID == 0 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	if uid == blockedID {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	if _, ok := loadSpaceUserForFollow(a, blockedID); !ok {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	// Check if already blocked
	blocked, _ := a.FollowSvc.UsersBlocked(c.Request.Context(), uid, blockedID)
	if !blocked {
		if err := a.FollowSvc.BlockUser(c.Request.Context(), uid, blockedID); err != nil {
			a.Log.Error("block user", zap.Error(err))
			resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
			return
		}
	} else {
		if err := a.FollowSvc.UnblockUser(c.Request.Context(), uid, blockedID); err != nil {
			a.Log.Error("unblock user", zap.Error(err))
			resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
			return
		}
	}
	resp.OK(c, gin.H{"blocked": true, "user_id": blockedID})
}
