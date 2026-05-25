package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"minibili/internal/errcode"
	"minibili/internal/middleware"
	"minibili/internal/model"
	"minibili/internal/pkg/resp"
)

type spacePrivacyPayload struct {
	PublicFavorites   bool `json:"public_favorites"`
	PublicRecentCoins bool `json:"public_recent_coins"`
	PublicFollowing   bool `json:"public_following"`
	PublicFans        bool `json:"public_fans"`
	PublicBirthday    bool `json:"public_birthday"`
}

func spacePrivacyFromUser(u *model.User) spacePrivacyPayload {
	return spacePrivacyPayload{
		PublicFavorites:   u.PrivacyPublicFavorites,
		PublicRecentCoins: u.PrivacyPublicRecentCoins,
		PublicFollowing:   u.PrivacyPublicFollowing,
		PublicFans:        u.PrivacyPublicFans,
		PublicBirthday:    u.PrivacyPublicBirthday,
	}
}

func spaceViewerCanSee(ownerID uint64, viewerOK bool, viewerID uint64, allowed bool) bool {
	if viewerOK && viewerID == ownerID {
		return true
	}
	return allowed
}

// GetMeSpacePrivacy returns current user's space privacy toggles.
func (a *API) GetMeSpacePrivacy(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	var u model.User
	if err := a.DB.First(&u, uid).Error; err != nil {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	resp.OK(c, spacePrivacyFromUser(&u))
}

type updateSpacePrivacyReq struct {
	PublicFavorites   *bool `json:"public_favorites"`
	PublicRecentCoins *bool `json:"public_recent_coins"`
	PublicFollowing   *bool `json:"public_following"`
	PublicFans        *bool `json:"public_fans"`
	PublicBirthday    *bool `json:"public_birthday"`
}

// UpdateMeSpacePrivacy updates space privacy toggles (partial patch).
func (a *API) UpdateMeSpacePrivacy(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	var req updateSpacePrivacyReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	var u model.User
	if err := a.DB.First(&u, uid).Error; err != nil {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	updates := map[string]interface{}{}
	if req.PublicFavorites != nil {
		updates["privacy_public_favorites"] = *req.PublicFavorites
	}
	if req.PublicRecentCoins != nil {
		updates["privacy_public_recent_coins"] = *req.PublicRecentCoins
	}
	if req.PublicFollowing != nil {
		updates["privacy_public_following"] = *req.PublicFollowing
	}
	if req.PublicFans != nil {
		updates["privacy_public_fans"] = *req.PublicFans
	}
	if req.PublicBirthday != nil {
		updates["privacy_public_birthday"] = *req.PublicBirthday
	}
	if len(updates) == 0 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	if err := a.DB.Model(&u).Updates(updates).Error; err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	_ = a.DB.First(&u, uid)
	resp.OK(c, spacePrivacyFromUser(&u))
}
