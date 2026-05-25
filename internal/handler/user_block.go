package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"minibili/internal/errcode"
	"minibili/internal/middleware"
	"minibili/internal/model"
	"minibili/internal/pkg/resp"
)

func dmUsersBlocked(db *gorm.DB, a, b uint64) bool {
	if a == 0 || b == 0 || a == b {
		return false
	}
	var n int64
	_ = db.Model(&model.UserBlock{}).Where(
		"(blocker_id = ? AND blocked_id = ?) OR (blocker_id = ? AND blocked_id = ?)",
		a, b, b, a,
	).Count(&n).Error
	return n > 0
}

func unfollowBothWays(tx *gorm.DB, a, b uint64) error {
	if err := tx.Where("follower_id = ? AND followee_id = ?", a, b).
		Delete(&model.UserFollow{}).Error; err != nil {
		return err
	}
	if err := tx.Where("follower_id = ? AND followee_id = ?", b, a).
		Delete(&model.UserFollow{}).Error; err != nil {
		return err
	}
	if err := deleteFollowGroupMembersForFollowee(tx, a, b); err != nil {
		return err
	}
	return deleteFollowGroupMembersForFollowee(tx, b, a)
}

// BlockUser adds peer to the caller's blacklist and removes mutual follows.
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
	var exist model.UserBlock
	err = a.DB.Where("blocker_id = ? AND blocked_id = ?", uid, blockedID).First(&exist).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		row := model.UserBlock{BlockerID: uid, BlockedID: blockedID}
		if err := a.DB.Transaction(func(tx *gorm.DB) error {
			if err := tx.Create(&row).Error; err != nil {
				return err
			}
			return unfollowBothWays(tx, uid, blockedID)
		}); err != nil {
			resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
			return
		}
	} else {
		_ = a.DB.Transaction(func(tx *gorm.DB) error {
			return unfollowBothWays(tx, uid, blockedID)
		})
	}
	resp.OK(c, gin.H{"blocked": true, "user_id": blockedID})
}
