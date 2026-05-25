package handler

import (
	"errors"
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

func userFollowCounts(db *gorm.DB, userID uint64) (following, followers int64) {
	_ = db.Model(&model.UserFollow{}).Where("follower_id = ?", userID).Count(&following).Error
	_ = db.Model(&model.UserFollow{}).Where("followee_id = ?", userID).Count(&followers).Error
	return following, followers
}

func userFollows(db *gorm.DB, followerID, followeeID uint64) bool {
	if followerID == 0 || followeeID == 0 || followerID == followeeID {
		return false
	}
	var n int64
	_ = db.Model(&model.UserFollow{}).
		Where("follower_id = ? AND followee_id = ?", followerID, followeeID).
		Count(&n).Error
	return n > 0
}

// userFolloweeIDsSet returns followee IDs the viewer follows among the given candidates.
func userFolloweeIDsSet(db *gorm.DB, followerID uint64, followeeIDs []uint64) map[uint64]bool {
	out := make(map[uint64]bool)
	if followerID == 0 || len(followeeIDs) == 0 {
		return out
	}
	uniq := make([]uint64, 0, len(followeeIDs))
	seen := make(map[uint64]struct{}, len(followeeIDs))
	for _, id := range followeeIDs {
		if id == 0 || id == followerID {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		uniq = append(uniq, id)
	}
	if len(uniq) == 0 {
		return out
	}
	var ids []uint64
	_ = db.Model(&model.UserFollow{}).
		Where("follower_id = ? AND followee_id IN ?", followerID, uniq).
		Pluck("followee_id", &ids).Error
	for _, id := range ids {
		out[id] = true
	}
	return out
}

func uploaderPublishedCount(db *gorm.DB, userID uint64) int64 {
	var videoN, articleN, dynN int64
	_ = db.Model(&model.Video{}).
		Where("user_id = ? AND status = ?", userID, "published").
		Count(&videoN).Error
	_ = db.Model(&model.Article{}).
		Where("user_id = ? AND status = ?", userID, articleStatusPublished).
		Count(&articleN).Error
	_ = db.Model(&model.UserDynamic{}).
		Where("user_id = ?", userID).
		Count(&dynN).Error
	return videoN + articleN + dynN
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
			sign = "这个家伙很懒，什么都没有写"
		}
		item := gin.H{
			"user_id":    u.ID,
			"nickname":   nick,
			"sign":       sign,
			"avatar_url": avatarURLForAPI(&u),
			"followed_at": created[uid].Format(time.RFC3339),
		}
		if followerField {
			item["mutual"] = mutual[uid]
		}
		items = append(items, item)
	}
	return items, nil
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
	if groupID > 0 {
		g, ok := loadFollowGroupForOwner(a.DB, ownerID, groupID)
		if !ok {
			resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
			return
		}
		_ = g
	}
	var rows []model.UserFollow
	q := a.DB.Where("follower_id = ?", ownerID)
	if groupID > 0 {
		ids, err := followeeIDsInGroup(a.DB, groupID)
		if err != nil {
			resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
			return
		}
		if len(ids) == 0 {
			resp.OK(c, gin.H{"items": []gin.H{}, "total": 0, "group_id": groupID})
			return
		}
		q = q.Where("followee_id IN ?", ids)
	}
	if err := q.Order("created_at DESC").Limit(limit).Find(&rows).Error; err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	items, err := followUserRows(a.DB, ownerID, rows, true)
	if err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	total := int64(len(items))
	if groupID == 0 {
		following, _ := userFollowCounts(a.DB, ownerID)
		total = following
	}
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
	var rows []model.UserFollow
	if err := a.DB.Where("followee_id = ?", ownerID).
		Order("created_at DESC").
		Limit(limit).
		Find(&rows).Error; err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	items, err := followUserRows(a.DB, ownerID, rows, false)
	if err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	_, followers := userFollowCounts(a.DB, ownerID)
	resp.OK(c, gin.H{"items": items, "total": followers})
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
	var row model.UserFollow
	err = a.DB.Where("follower_id = ? AND followee_id = ?", uid, followeeID).First(&row).Error
	if err == nil {
		if err := a.DB.Transaction(func(tx *gorm.DB) error {
			if err := tx.Delete(&row).Error; err != nil {
				return err
			}
			return deleteFollowGroupMembersForFollowee(tx, uid, followeeID)
		}); err != nil {
			resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
			return
		}
		_, followers := userFollowCounts(a.DB, followeeID)
		resp.OK(c, gin.H{
			"followed":       false,
			"follower_count": followers,
		})
		return
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	row = model.UserFollow{FollowerID: uid, FolloweeID: followeeID}
	if err := a.DB.Create(&row).Error; err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	_, followers := userFollowCounts(a.DB, followeeID)
	resp.OK(c, gin.H{
		"followed":       true,
		"follower_count": followers,
	})
}
