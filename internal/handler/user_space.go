package handler

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"minibili/internal/errcode"
	"minibili/internal/middleware"
	"minibili/internal/model"
	"minibili/internal/pkg/resp"
	"minibili/internal/pkg/userlevel"
)

// GetUserPublic returns a minimal public profile for personal space (no auth).
func (a *API) GetUserPublic(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("userId"), 10, 64)
	if err != nil || id == 0 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	var u model.User
	if err := a.DB.First(&u, id).Error; err != nil {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	ensureUserCakeID(a.DB, &u)
	nick := strings.TrimSpace(u.Nickname)
	if nick == "" {
		nick = model.DisplayUsername(&u)
	}
	avatar := avatarURLForAPI(&u)
	sign := strings.TrimSpace(u.Sign)
	announcement := strings.TrimSpace(u.SpaceAnnouncement)
	gender := strings.TrimSpace(u.Gender)
	if gender != "male" && gender != "female" && gender != "secret" {
		gender = "secret"
	}
	if model.IsUserAnonymized(&u) {
		nick = "已注销用户"
		avatar = ""
		sign = ""
		announcement = ""
		gender = "secret"
	}
	viewer, viewerOK := middleware.UserID(c)
	if viewerOK && viewer != id && dmUsersBlocked(a.DB, viewer, id) {
		resp.Err(c, http.StatusForbidden, errcode.CodeUserBlocked)
		return
	}
	isOwner := viewerOK && viewer == id
	privacy := spacePrivacyFromUser(&u)
	birthday := ""
	if isOwner || u.PrivacyPublicBirthday {
		birthday = strings.TrimSpace(u.Birthday)
	}
	followingCnt, followerCnt := userFollowCounts(a.DB, id)
	payload := gin.H{
		"user_id":          u.ID,
		"nickname":         nick,
		"cake_id":          strings.TrimSpace(u.CakeID),
		"avatar_url":       avatar,
		"sign":             sign,
		"announcement":     announcement,
		"gender":           gender,
		"birthday":         birthday,
		"privacy":          privacy,
		"is_owner":         isOwner,
		"following_count":  followingCnt,
		"follower_count":   followerCnt,
		"published_count":  uploaderPublishedCount(a.DB, id),
		"followed_by_me":   false,
		"level_info":       userlevel.FromExperience(u.Experience),
	}
	if viewerOK && viewer != id {
		payload["followed_by_me"] = userFollows(a.DB, viewer, id)
	}
	resp.OK(c, payload)
}

// ListUserPublishedVideos lists published videos for a user (public, no auth).
func (a *API) ListUserPublishedVideos(c *gin.Context) {
	uid, err := strconv.ParseUint(c.Param("userId"), 10, 64)
	if err != nil || uid == 0 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	var u model.User
	if err := a.DB.First(&u, uid).Error; err != nil {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	if model.IsUserAnonymized(&u) {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	limit := 20
	if s := c.Query("limit"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 && n <= 50 {
			limit = n
		}
	}
	curID, _ := strconv.ParseUint(c.Query("cursor"), 10, 64)
	q := a.DB.Model(&model.Video{}).Where("user_id = ? AND status = ?", uid, "published")
	if curID > 0 {
		q = q.Where("id < ?", curID)
	}
	var list []model.Video
	if err := q.Order("id DESC").Limit(limit + 1).Find(&list).Error; err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	hasMore := len(list) > limit
	if hasMore {
		list = list[:limit]
	}
	up := model.DisplayUsername(&u)
	ctx := context.Background()
	var viewer uint64
	if uid, ok := middleware.UserID(c); ok {
		viewer = uid
	}
	ids := make([]uint64, 0, len(list))
	for _, v := range list {
		ids = append(ids, v.ID)
	}
	eng := videoEngagementByViewer(a.DB, viewer, ids)
	items := make([]gin.H, 0, len(list))
	for _, v := range list {
		pc, _ := a.Play.Display(ctx, &v)
		items = append(items, videoCard(v, up, pc, eng[v.ID]))
	}
	next := ""
	if hasMore && len(list) > 0 {
		last := list[len(list)-1]
		next = strconv.FormatUint(last.ID, 10)
	}
	resp.OK(c, gin.H{"items": items, "next_cursor": next})
}
