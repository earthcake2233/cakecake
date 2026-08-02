package handler

import (
	"cakecake/internal/model/user"
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"cakecake/internal/errcode"
	"cakecake/internal/middleware"
	"cakecake/internal/pkg/resp"
)

// followUserRows converts follow rows to typed items for API response.
type followUserItem struct {
	UserID     uint64 `json:"user_id"`
	Nickname   string `json:"nickname"`
	Sign       string `json:"sign"`
	AvatarURL  string `json:"avatar_url"`
	FollowedAt string `json:"followed_at"`
	Mutual     *bool  `json:"mutual,omitempty"`
}

type followListResponse struct {
	Items   []followUserItem `json:"items"`
	Total   int64            `json:"total"`
	GroupID uint64           `json:"group_id,omitempty"`
}

type followToggleResponse struct {
	Followed      bool  `json:"followed"`
	FollowerCount int64 `json:"follower_count"`
}

// buildFollowUserRows converts follow rows to typed items for API response.
func (a *API) buildFollowUserRows(
	ownerID uint64,
	rows []user.UserFollow,
	followerField bool,
) ([]followUserItem, error) {
	if len(rows) == 0 {
		return []followUserItem{}, nil
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
	users := a.UserSvc.BatchGetUsers(context.Background(), ids)
	umap := make(map[uint64]user.User, len(users))
	for id, u := range users {
		umap[id] = *u
	}
	mutual := make(map[uint64]bool)
	if followerField && len(ids) > 0 {
		following, _ := a.FollowSvc.GetFollowingIDs(context.Background(), ownerID, ids)
		for fid := range following {
			mutual[fid] = true
		}
	}
	items := make([]followUserItem, 0, len(rows))
	for i := range rows {
		var uid uint64
		if followerField {
			uid = rows[i].FolloweeID
		} else {
			uid = rows[i].FollowerID
		}
		u, ok := umap[uid]
		if !ok || user.IsUserAnonymized(&u) {
			continue
		}
		nick := strings.TrimSpace(u.Nickname)
		if nick == "" {
			nick = user.DisplayUsername(&u)
		}
		sign := strings.TrimSpace(u.Sign)
		if sign == "" {
			sign = ""
		}
		item := followUserItem{
			UserID:     u.ID,
			Nickname:   nick,
			Sign:       sign,
			AvatarURL:  avatarURLForAPI(&u),
			FollowedAt: created[uid].Format(time.RFC3339),
		}
		if followerField {
			m := mutual[uid]
			item.Mutual = &m
		}
		items = append(items, item)
	}
	return items, nil
}

func loadSpaceUserForFollow(a *API, userID uint64) (user.User, bool) {
	u, err := a.UserSvc.GetUserByID(context.Background(), userID)
	if err != nil || u == nil {
		return user.User{}, false
	}
	if user.IsUserAnonymized(u) {
		return *u, false
	}
	return *u, true
}

func canViewFollowingList(viewerOK bool, viewer, ownerID uint64, u *user.User) bool {
	return spaceViewerCanSee(ownerID, viewerOK, viewer, u.PrivacyPublicFollowing)
}

func canViewFollowersList(viewerOK bool, viewer, ownerID uint64, u *user.User) bool {
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
	limit := parseLimit(c, 200, 500)
	groupID, _ := strconv.ParseUint(strings.TrimSpace(c.Query("groupId")), 10, 64)
	rows, svcErr := a.FollowSvc.GetFollowingList(c.Request.Context(), ownerID, limit, groupID)
	if svcErr != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	items, err := a.buildFollowUserRows(ownerID, rows, true)
	if err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	counts, _ := a.FollowSvc.GetFollowCounts(c.Request.Context(), ownerID)
	total := counts.Following
	payload := followListResponse{Items: items, Total: total, GroupID: groupID}
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
	limit := parseLimit(c, 200, 500)
	rows, svcErr := a.FollowSvc.GetFollowersList(c.Request.Context(), ownerID, limit)
	if svcErr != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	items, err := a.buildFollowUserRows(ownerID, rows, false)
	if err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	counts, _ := a.FollowSvc.GetFollowCounts(c.Request.Context(), ownerID)
	resp.OK(c, followListResponse{Items: items, Total: counts.Followers})
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
	if a.isDMUsersBlocked(uid, followeeID) {
		resp.Err(c, http.StatusForbidden, errcode.CodeUserBlocked)
		return
	}
	followed, svcErr := a.FollowSvc.ToggleFollow(c.Request.Context(), uid, followeeID)
	if svcErr != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	counts, _ := a.FollowSvc.GetFollowCounts(c.Request.Context(), followeeID)
	resp.OK(c, followToggleResponse{Followed: followed, FollowerCount: counts.Followers})
}
