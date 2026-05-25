package handler

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"minibili/internal/errcode"
	"minibili/internal/middleware"
	"minibili/internal/model"
	"minibili/internal/pkg/resp"
)

const maxFollowGroupsPerUser = 50

type followGroupNameJSON struct {
	Name string `json:"name"`
}

func parseFollowGroupID(c *gin.Context) (uint64, bool) {
	id, err := strconv.ParseUint(c.Param("groupId"), 10, 64)
	return id, err == nil && id > 0
}

func validateFollowGroupName(name string) bool {
	name = strings.TrimSpace(name)
	return name != "" && utf8.RuneCountInString(name) <= 16
}

func followGroupMemberCounts(db *gorm.DB, groupIDs []uint64) map[uint64]int64 {
	out := make(map[uint64]int64, len(groupIDs))
	if len(groupIDs) == 0 {
		return out
	}
	type row struct {
		GroupID uint64
		Cnt     int64
	}
	var rows []row
	_ = db.Model(&model.UserFollowGroupMember{}).
		Select("group_id, COUNT(*) AS cnt").
		Where("group_id IN ?", groupIDs).
		Group("group_id").
		Scan(&rows).Error
	for i := range rows {
		out[rows[i].GroupID] = rows[i].Cnt
	}
	return out
}

func followGroupPayload(db *gorm.DB, g *model.UserFollowGroup) gin.H {
	counts := followGroupMemberCounts(db, []uint64{g.ID})
	return gin.H{
		"id":           g.ID,
		"name":         g.Name,
		"member_count": counts[g.ID],
		"created_at":   g.CreatedAt,
	}
}

// ListMyFollowGroups lists the caller's custom following groups.
func (a *API) ListMyFollowGroups(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		resp.Err(c, http.StatusUnauthorized, errcode.CodeUnauthorized)
		return
	}
	var groups []model.UserFollowGroup
	if err := a.DB.Where("user_id = ?", uid).Order("created_at ASC, id ASC").Find(&groups).Error; err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	ids := make([]uint64, 0, len(groups))
	for i := range groups {
		ids = append(ids, groups[i].ID)
	}
	counts := followGroupMemberCounts(a.DB, ids)
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
	var total int64
	if err := a.DB.Model(&model.UserFollowGroup{}).Where("user_id = ?", uid).Count(&total).Error; err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	if total >= maxFollowGroupsPerUser {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	var dup int64
	if err := a.DB.Model(&model.UserFollowGroup{}).
		Where("user_id = ? AND name = ?", uid, name).
		Count(&dup).Error; err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	if dup > 0 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	row := model.UserFollowGroup{UserID: uid, Name: name}
	if err := a.DB.Create(&row).Error; err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	resp.OK(c, followGroupPayload(a.DB, &row))
}

// UpdateFollowGroup renames a custom following group for the caller.
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
	g, ok := loadFollowGroupForOwner(a.DB, uid, groupID)
	if !ok {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
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
	var dup int64
	if err := a.DB.Model(&model.UserFollowGroup{}).
		Where("user_id = ? AND name = ? AND id <> ?", uid, name, g.ID).
		Count(&dup).Error; err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	if dup > 0 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	g.Name = name
	if err := a.DB.Save(g).Error; err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	resp.OK(c, followGroupPayload(a.DB, g))
}

// DeleteFollowGroup removes a custom group and its member links only (does not unfollow).
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
	if _, ok := loadFollowGroupForOwner(a.DB, uid, groupID); !ok {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	err := a.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("group_id = ?", groupID).Delete(&model.UserFollowGroupMember{}).Error; err != nil {
			return err
		}
		return tx.Where("id = ? AND user_id = ?", groupID, uid).Delete(&model.UserFollowGroup{}).Error
	})
	if err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	resp.OK(c, gin.H{"deleted": true, "id": groupID})
}

func loadFollowGroupForOwner(db *gorm.DB, ownerID, groupID uint64) (*model.UserFollowGroup, bool) {
	if groupID == 0 {
		return nil, false
	}
	var g model.UserFollowGroup
	if err := db.Where("id = ? AND user_id = ?", groupID, ownerID).First(&g).Error; err != nil {
		return nil, false
	}
	return &g, true
}

func followeeIDsInGroup(db *gorm.DB, groupID uint64) ([]uint64, error) {
	var ids []uint64
	err := db.Model(&model.UserFollowGroupMember{}).
		Where("group_id = ?", groupID).
		Pluck("followee_id", &ids).Error
	return ids, err
}

type followGroupMemberJSON struct {
	FolloweeID uint64 `json:"followee_id"`
}

func parseFolloweeIDParam(c *gin.Context) (uint64, bool) {
	id, err := strconv.ParseUint(c.Param("followeeId"), 10, 64)
	return id, err == nil && id > 0
}

func deleteFollowGroupMembersForFollowee(db *gorm.DB, ownerID, followeeID uint64) error {
	var groupIDs []uint64
	if err := db.Model(&model.UserFollowGroup{}).
		Where("user_id = ?", ownerID).
		Pluck("id", &groupIDs).Error; err != nil {
		return err
	}
	if len(groupIDs) == 0 {
		return nil
	}
	return db.Where("group_id IN ? AND followee_id = ?", groupIDs, followeeID).
		Delete(&model.UserFollowGroupMember{}).Error
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
	if !userFollows(a.DB, uid, followeeID) {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	var groupIDs []uint64
	err := a.DB.Model(&model.UserFollowGroupMember{}).
		Select("user_follow_group_members.group_id").
		Joins("JOIN user_follow_groups ON user_follow_groups.id = user_follow_group_members.group_id").
		Where("user_follow_groups.user_id = ? AND user_follow_group_members.followee_id = ?", uid, followeeID).
		Pluck("user_follow_group_members.group_id", &groupIDs).Error
	if err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	resp.OK(c, gin.H{"group_ids": groupIDs})
}

// AddFollowGroupMember adds a followee into one of the caller's groups (must already follow them).
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
	if _, ok := loadFollowGroupForOwner(a.DB, uid, groupID); !ok {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	var body followGroupMemberJSON
	if err := c.ShouldBindJSON(&body); err != nil || body.FolloweeID == 0 {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	if body.FolloweeID == uid || !userFollows(a.DB, uid, body.FolloweeID) {
		resp.Err(c, http.StatusBadRequest, errcode.CodeParamError)
		return
	}
	var row model.UserFollowGroupMember
	err := a.DB.Where("group_id = ? AND followee_id = ?", groupID, body.FolloweeID).First(&row).Error
	if err == nil {
		resp.OK(c, gin.H{"added": false, "group_id": groupID, "followee_id": body.FolloweeID})
		return
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	row = model.UserFollowGroupMember{GroupID: groupID, FolloweeID: body.FolloweeID}
	if err := a.DB.Create(&row).Error; err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	resp.OK(c, gin.H{"added": true, "group_id": groupID, "followee_id": body.FolloweeID})
}

// RemoveFollowGroupMember removes a followee from one of the caller's groups.
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
	if _, ok := loadFollowGroupForOwner(a.DB, uid, groupID); !ok {
		resp.Err(c, http.StatusNotFound, errcode.CodeNotFound)
		return
	}
	if err := a.DB.Where("group_id = ? AND followee_id = ?", groupID, followeeID).
		Delete(&model.UserFollowGroupMember{}).Error; err != nil {
		resp.Err(c, http.StatusInternalServerError, errcode.CodeInternalError)
		return
	}
	resp.OK(c, gin.H{"removed": true, "group_id": groupID, "followee_id": followeeID})
}
