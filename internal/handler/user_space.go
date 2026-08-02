package handler

import (
	"cakecake/internal/model/user"
	"context"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"cakecake/internal/errcode"
	"cakecake/internal/middleware"
	"cakecake/internal/pkg/resp"
	"cakecake/internal/pkg/userlevel"
)

type spaceProfileResponse struct {
	UserID         uint64              `json:"user_id"`
	Nickname       string              `json:"nickname"`
	CakeID         string              `json:"cake_id"`
	AvatarURL      string              `json:"avatar_url"`
	Sign           string              `json:"sign"`
	Announcement   string              `json:"announcement"`
	Gender         string              `json:"gender"`
	Birthday       string              `json:"birthday"`
	Privacy        spacePrivacyPayload `json:"privacy"`
	IsOwner        bool                `json:"is_owner"`
	FollowingCount int64               `json:"following_count"`
	FollowerCount  int64               `json:"follower_count"`
	PublishedCount int64               `json:"published_count"`
	FollowedByMe   bool                `json:"followed_by_me"`
	LevelInfo      userlevel.Info      `json:"level_info"`
}

type userVideoListResponse struct {
	Items      []videoCardDTO `json:"items"`
	NextCursor string         `json:"next_cursor"`
}

// GetUserPublic returns a minimal public profile for personal space (no auth).
// GetUserPublic godoc
// @Summary      Get user public profile
// @Description  Get user public space by user ID
// @Tags        Users
// @Produce     json
// @Param       userId path int true "User ID"
// @Success     200 {object} map[string]interface{}
// @Router      /space/{userId} [get]
func (a *API) GetUserPublic(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("userId"), 10, 64)
	if err != nil || id == 0 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	u, err := a.UserSvc.GetUserByID(c.Request.Context(), id)
	if err != nil {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	_ = a.UserSvc.EnsureCakeID(c.Request.Context(), u)
	nick := strings.TrimSpace(u.Nickname)
	if nick == "" {
		nick = user.DisplayUsername(u)
	}
	avatar := avatarURLForAPI(u)
	sign := strings.TrimSpace(u.Sign)
	announcement := strings.TrimSpace(u.SpaceAnnouncement)
	gender := strings.TrimSpace(u.Gender)
	if gender != "male" && gender != "female" && gender != "secret" {
		gender = "secret"
	}
	if user.IsUserAnonymized(u) {
		nick = user.DisplayUsername(u)
		avatar = ""
		sign = ""
		announcement = ""
		gender = "secret"
	}
	viewer, viewerOK := middleware.UserID(c)
	if viewerOK && viewer != id {
		blocked, _ := a.FollowSvc.UsersBlocked(c.Request.Context(), viewer, id)
		if blocked {
			resp.Err(c, http.StatusForbidden, errcode.CodeUserBlocked)
			return
		}
	}
	isOwner := viewerOK && viewer == id
	privacy := spacePrivacyFromUser(u)
	birthday := ""
	if isOwner || u.PrivacyPublicBirthday {
		birthday = strings.TrimSpace(u.Birthday)
	}
	counts, _ := a.FollowSvc.GetFollowCounts(c.Request.Context(), id)
	pubCount, _ := a.FollowSvc.GetUploaderPublishedCount(c.Request.Context(), id)
	payload := spaceProfileResponse{
		UserID:         u.ID,
		Nickname:       nick,
		CakeID:         strings.TrimSpace(u.CakeID),
		AvatarURL:      avatar,
		Sign:           sign,
		Announcement:   announcement,
		Gender:         gender,
		Birthday:       birthday,
		Privacy:        privacy,
		IsOwner:        isOwner,
		FollowingCount: counts.Following,
		FollowerCount:  counts.Followers,
		PublishedCount: pubCount,
		FollowedByMe:   false,
		LevelInfo:      userlevel.FromExperience(u.Experience),
	}
	if viewerOK && viewer != id {
		following, _ := a.FollowSvc.IsFollowing(c.Request.Context(), viewer, id)
		payload.FollowedByMe = following
	}
	resp.OK(c, payload)
}

// ListUserPublishedVideos lists published videos for a user (public, no auth).
// ListUserPublishedVideos godoc
// @Summary      List user videos
// @Description  Get paginated published videos for a user space
// @Tags         Users
// @Produce      json
// @Param        userId path int true "User ID"
// @Param        page query int false "Page number" default(1)
// @Param        page_size query int false "Page size" default(20)
// @Success      200 {object} map[string]interface{}
// @Router       /space/{userId}/videos [get]
func (a *API) ListUserPublishedVideos(c *gin.Context) {
	uid, err := strconv.ParseUint(c.Param("userId"), 10, 64)
	if err != nil || uid == 0 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	u, err := a.UserSvc.GetUserByID(c.Request.Context(), uid)
	if err != nil {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	if user.IsUserAnonymized(u) {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	limit := parseLimit(c, 20, 50)
	curID, _ := strconv.ParseUint(c.Query("cursor"), 10, 64)
	list, err := a.VideoSvc.ListUserPublishedVideosCursor(c.Request.Context(), uid, curID, limit+1)
	if err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	hasMore := len(list) > limit
	if hasMore {
		list = list[:limit]
	}
	up := user.DisplayUsername(u)
	ctx := context.Background()
	var viewer uint64
	if uid, ok := middleware.UserID(c); ok {
		viewer = uid
	}
	ids := make([]uint64, 0, len(list))
	for _, v := range list {
		ids = append(ids, v.ID)
	}
	eng := a.engagementByViewer(viewer, ids)
	items := make([]videoCardDTO, 0, len(list))
	for _, v := range list {
		pc, _ := a.Play.Display(ctx, &v)
		items = append(items, videoCard(v, up, pc, eng[v.ID]))
	}
	next := ""
	if hasMore && len(list) > 0 {
		last := list[len(list)-1]
		next = strconv.FormatUint(last.ID, 10)
	}
	resp.OK(c, userVideoListResponse{Items: items, NextCursor: next})
}
