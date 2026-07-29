package handler

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"minibili/internal/errcode"
	"minibili/internal/middleware"
	"minibili/internal/model"
	"minibili/internal/pkg/resp"
)

// followUserRows converts follow rows to gin.H items for API response.
func followUserRows(
	db *gorm.DB,
	ownerID uint64,
	rows []model.UserFollow,
	followerField bool,
) ([]gin.H, error) {
	if len(rows) == 0 {
		return []gin.H{}, nil
	}
	ids := make([]uint64, 0, len(rows))
	created := make(map[uint64]time.Time, len(rows))
	for i := range rows {
		var uid uint64
		if followerField {
			uid = rows[i].FolloweeID
		} else {
			uid = rows[i].FollowerID
		}
		ids = append(ids, uid)
		created[uid] = rows[i].CreatedAt
	}
	var users []model.User
	if err := db.Where("id IN ?", ids).Find(&users).Error; err != nil {
		return nil, err
	}
	umap := make(map[uint64]model.User, len(users))
	for i := range users {
		umap[users[i].ID] = users[i]
	}
	mutual := make(map[uint64]bool)
	if followerField && len(ids) > 0 {
		var back []model.UserFollow
		_ = db.Where("follower_id IN ? AND followee_id = ?", ids, ownerID).Find(&back).Error
		for i := range back {
			mutual[back[i].FollowerID] = true
		}
	}
	items := make([]gin.H, 0, len(rows))
	for i := range rows {
		var uid uint64
		if followerField {
			uid = rows[i].FolloweeID
		} else {
			uid = rows[i].FollowerID
		}
		u, ok := umap[uid]
		if !ok || model.IsUserAnonymized(&u) {
			continue
		}
		nick := strings.TrimSpace(u.Nickname)
		if nick == "" {
			nick = model.DisplayUsername(&u)
		}
		sign := strings.TrimSpace(u.Sign)
		if sign == "" {
			sign = "?????????????"
		}
		item := gin.H{
			"user_id":     u.ID,
			"nickname":    nick,
			"sign":        sign,
			"avatar_url":  avatarURLForAPI(&u),
			"followed_at": created[uid].Format(time.RFC3339),
		}
		if followerField {
			item["mutual"] = mutual[uid]
		}
		items = append(items, item)
	}
	return items, nil
}

func loadSpaceUserForFollow(a *API, userID uint64) (model.User, bool) {
	var u model.User
	if err := a.DB.First(&u, userID).Error; err != nil {
		return u, false
	}
	if model.IsUserAnonymized(&u) {
		return u, false
	}
	return u, true
}

func canViewFollowingList(viewerOK bool, viewer, ownerID uint64, u *model.User) bool {
	return spaceViewerCanSee(ownerID, viewerOK, viewer, u.PrivacyPublicFollowing)
}

func canViewFollowersList(viewerOK bool, viewer, ownerID uint64, u *model.User) bool {
	return spaceViewerCanSee(ownerID, viewerOK, viewer, u.PrivacyPublicFans)
}

// ListUserFollowing lists users that owner follows (respects privacy).
func (a *API) ListUserFollowing(c *gin.Context) {
	ownerID, err := strconv.ParseUint(c.Param("userId"), 10, 64)
	if err != nil || ownerID == 0 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	u, ok := loadSpaceUserForFollow(a, ownerID)
	if !ok {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	viewer, viewerOK := middleware.UserID(c)
	if !canViewFollowingList(viewerOK, viewer, ownerID, &u) {
		resp.Err(c, http.StatusForbidden, errcode.CodeForbidden)
		return
	}
	limit := 200
	if raw := c.Query("limit"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 && n <= 500 {
			limit = n
		}
	}
	groupID, _ := strconv.ParseUint(strings.TrimSpace(c.Query("groupId")), 10, 64)
	rows, svcErr := a.FollowSvc.GetFollowingList(c.Request.Context(), ownerID, limit, groupID)
	if svcErr != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	items, err := followUserRows(a.DB, ownerID, rows, true)
	if err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	counts, _ := a.FollowSvc.GetFollowCounts(c.Request.Context(), ownerID)
	total := counts.Following
	payload := gin.H{"items": items, "total": total}
	if groupID > 0 {
		payload["group_id"] = groupID
	}
	resp.OK(c, payload)
}

// ListUserFollowers lists users who follow owner (respects privacy).
func (a *API) ListUserFollowers(c *gin.Context) {
	ownerID, err := strconv.ParseUint(c.Param("userId"), 10, 64)
	if err != nil || ownerID == 0 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	u, ok := loadSpaceUserForFollow(a, ownerID)
	if !ok {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	viewer, viewerOK := middleware.UserID(c)
	if !canViewFollowersList(viewerOK, viewer, ownerID, &u) {
		resp.Err(c, http.StatusForbidden, errcode.CodeForbidden)
		return
	}
	limit := 200
	if raw := c.Query("limit"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 && n <= 500 {
			limit = n
		}
	}
	rows, svcErr := a.FollowSvc.GetFollowersList(c.Request.Context(), ownerID, limit)
	if svcErr != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	items, err := followUserRows(a.DB, ownerID, rows, false)
	if err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	counts, _ := a.FollowSvc.GetFollowCounts(c.Request.Context(), ownerID)
	resp.OK(c, gin.H{"items": items, "total": counts.Followers})
}

// ToggleFollowUser toggles the caller's follow on another user.
func (a *API) ToggleFollowUser(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	followeeID, err := strconv.ParseUint(c.Param("userId"), 10, 64)
	if err != nil || followeeID == 0 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	if uid == followeeID {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	if _, ok := loadSpaceUserForFollow(a, followeeID); !ok {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	if dmUsersBlocked(a.DB, uid, followeeID) {
		resp.Err(c, http.StatusForbidden, errcode.CodeUserBlocked)
		return
	}
	followed, svcErr := a.FollowSvc.ToggleFollow(c.Request.Context(), uid, followeeID)
	if svcErr != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	counts, _ := a.FollowSvc.GetFollowCounts(c.Request.Context(), followeeID)
	resp.OK(c, gin.H{
		"followed":       followed,
		"follower_count": counts.Followers,
	})
}
