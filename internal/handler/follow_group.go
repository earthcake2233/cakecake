package handler

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/gin-gonic/gin"

	"cakecake/internal/errcode"
	"cakecake/internal/middleware"
	"cakecake/internal/pkg/resp"
	"cakecake/internal/service"
)

type followGroupNameJSON struct {
	Name string `json:"name"`
}

type followGroupMemberJSON struct {
	FolloweeID uint64 `json:"followee_id"`
}

func parseFollowGroupID(c *gin.Context) (uint64, bool) {
	id, err := strconv.ParseUint(c.Param("groupId"), 10, 64)
	return id, err == nil && id > 0
}

func parseFolloweeIDParam(c *gin.Context) (uint64, bool) {
	id, err := strconv.ParseUint(c.Param("followeeId"), 10, 64)
	return id, err == nil && id > 0
}

func validateFollowGroupName(name string) bool {
	name = strings.TrimSpace(name)
	return name != "" && utf8.RuneCountInString(name) <= 16
}

// ListMyFollowGroups lists the caller's custom following groups.
// ListMyFollowGroups godoc
// @Summary      List my follow groups
// @Description  Get all follow groups for current user
// @Tags         Users
// @Produce      json
// @Success      200 {object} map[string]interface{}
// @Router       /users/me/follow-groups [get]
func (a *API) ListMyFollowGroups(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	groups, err := a.FollowSvc.ListGroups(c.Request.Context(), uid)
	if err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	ids := make([]uint64, 0, len(groups))
	for i := range groups {
		ids = append(ids, groups[i].ID)
	}
	counts, err := a.FollowSvc.GetGroupMemberCounts(c.Request.Context(), ids)
	if err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	items := make([]gin.H, 0, len(groups))
	for i := range groups {
		items = append(items, gin.H{
			"id":           groups[i].ID,
			"name":         groups[i].Name,
			"member_count": counts[groups[i].ID],
			"created_at":   groups[i].CreatedAt,
		})
	}
	resp.OK(c, gin.H{"items": items})
}

// CreateFollowGroup creates a custom following group for the caller.
// CreateFollowGroup godoc
// @Summary      Create a follow group
// @Description  Create a new group to organize followed users
// @Tags         Users
// @Produce      json
// @Param        body body object{name=string} true "Group name"
// @Success      200 {object} map[string]interface{}
// @Router       /users/me/follow-groups [post]
func (a *API) CreateFollowGroup(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	var body followGroupNameJSON
	if err := c.ShouldBindJSON(&body); err != nil {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	name := strings.TrimSpace(body.Name)
	if !validateFollowGroupName(name) {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	g, err := a.FollowSvc.CreateGroup(c.Request.Context(), uid, name)
	if err != nil {
		if errors.Is(err, service.ErrParamError) {
			resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		} else {
			resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		}
		return
	}
	cnt, _ := a.FollowSvc.GetGroupMemberCounts(c.Request.Context(), []uint64{g.ID})
	resp.OK(c, gin.H{
		"id":           g.ID,
		"name":         g.Name,
		"member_count": cnt[g.ID],
		"created_at":   g.CreatedAt,
	})
}

// UpdateFollowGroup renames a custom following group for the caller.
// UpdateFollowGroup godoc
// @Summary      Update a follow group
// @Description  Rename a follow group
// @Tags         Users
// @Produce      json
// @Param        groupId path int true "Group ID"
// @Param        body body object{name=string} true "New group name"
// @Success      200 {object} map[string]interface{}
// @Router       /users/me/follow-groups/{groupId} [put]
func (a *API) UpdateFollowGroup(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	groupID, ok := parseFollowGroupID(c)
	if !ok {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	var body followGroupNameJSON
	if err := c.ShouldBindJSON(&body); err != nil {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	name := strings.TrimSpace(body.Name)
	if !validateFollowGroupName(name) {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	g, err := a.FollowSvc.UpdateGroup(c.Request.Context(), uid, groupID, name)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		} else {
			resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		}
		return
	}
	cnt, _ := a.FollowSvc.GetGroupMemberCounts(c.Request.Context(), []uint64{g.ID})
	resp.OK(c, gin.H{
		"id":           g.ID,
		"name":         g.Name,
		"member_count": cnt[g.ID],
		"created_at":   g.CreatedAt,
	})
}

// DeleteFollowGroup deletes a custom following group for the caller.
// DeleteFollowGroup godoc
// @Summary      Delete a follow group
// @Description  Delete a follow group and its members
// @Tags         Users
// @Produce      json
// @Param        groupId path int true "Group ID"
// @Success      200 {object} map[string]interface{}
// @Router       /users/me/follow-groups/{groupId} [delete]
func (a *API) DeleteFollowGroup(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	groupID, ok := parseFollowGroupID(c)
	if !ok {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	if err := a.FollowSvc.DeleteGroup(c.Request.Context(), uid, groupID); err != nil {
		if errors.Is(err, service.ErrNotFound) {
			resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		} else {
			resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		}
		return
	}
	resp.OK(c, gin.H{"deleted": true})
}

// ListFolloweeGroupIDs lists which of the caller's follow groups include a followee.
func (a *API) ListFolloweeGroupIDs(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	followeeID, ok := parseFolloweeIDParam(c)
	if !ok {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	following, err := a.FollowSvc.IsFollowing(c.Request.Context(), uid, followeeID)
	if err != nil || !following {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	groupIDs, err := a.FollowSvc.GetFolloweeGroupIDs(c.Request.Context(), uid, followeeID)
	if err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	resp.OK(c, gin.H{"group_ids": groupIDs})
}

// AddFollowGroupMember adds a followee into one of the caller's groups (must already follow them).
// AddFollowGroupMember godoc
// @Summary      Add user to follow group
// @Description  Add a followed user to a group
// @Tags         Users
// @Produce      json
// @Param        groupId path int true "Group ID"
// @Param        body body object{followee_id=int} true "User ID to add"
// @Success      200 {object} map[string]interface{}
// @Router       /users/me/follow-groups/{groupId}/members [post]
func (a *API) AddFollowGroupMember(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	groupID, ok := parseFollowGroupID(c)
	if !ok {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	_, err := a.FollowSvc.GetGroup(c.Request.Context(), uid, groupID)
	if err != nil {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	var body followGroupMemberJSON
	if err := c.ShouldBindJSON(&body); err != nil || body.FolloweeID == 0 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	if body.FolloweeID == uid {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	following, err := a.FollowSvc.IsFollowing(c.Request.Context(), uid, body.FolloweeID)
	if err != nil || !following {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	err = a.FollowSvc.AddGroupMember(c.Request.Context(), groupID, body.FolloweeID)
	if err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	resp.OK(c, gin.H{"added": true, "group_id": groupID, "followee_id": body.FolloweeID})
}

// RemoveFollowGroupMember removes a followee from one of the caller's groups.
// RemoveFollowGroupMember godoc
// @Summary      Remove user from follow group
// @Description  Remove a followed user from their group assignment
// @Tags         Users
// @Produce      json
// @Param        groupId path int true "Group ID"
// @Param        followeeId path int true "Followee User ID"
// @Success      200 {object} map[string]interface{}
// @Router       /users/me/follow-groups/{groupId}/members/{followeeId} [delete]
func (a *API) RemoveFollowGroupMember(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	groupID, ok := parseFollowGroupID(c)
	if !ok {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	followeeID, ok := parseFolloweeIDParam(c)
	if !ok {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	_, err := a.FollowSvc.GetGroup(c.Request.Context(), uid, groupID)
	if err != nil {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	if err := a.FollowSvc.RemoveGroupMember(c.Request.Context(), groupID, followeeID); err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	resp.OK(c, gin.H{"removed": true, "group_id": groupID, "followee_id": followeeID})
}
