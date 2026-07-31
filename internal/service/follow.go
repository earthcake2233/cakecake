package service

import (
	"context"
	"minibili/internal/model/article"
	"minibili/internal/model/dynamic"
	"minibili/internal/model/user"
	"minibili/internal/model/video"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// FollowService handles follow/unfollow business logic.
type FollowService struct {
	db  *gorm.DB
	log *zap.Logger
}

// NewFollowService creates a FollowService.
func NewFollowService(db *gorm.DB, log *zap.Logger) *FollowService {
	return &FollowService{db: db, log: log}
}

// FollowCounts holds following and follower counts.
type FollowCounts struct {
	Following int64
	Followers int64
}

// GetFollowCounts returns the following/follower counts for a user.
func (s *FollowService) GetFollowCounts(ctx context.Context, userID uint64) (FollowCounts, error) {
	var following, followers int64
	if err := s.db.WithContext(ctx).Model(&user.UserFollow{}).Where("follower_id = ?", userID).Count(&following).Error; err != nil {
		return FollowCounts{}, err
	}
	if err := s.db.WithContext(ctx).Model(&user.UserFollow{}).Where("followee_id = ?", userID).Count(&followers).Error; err != nil {
		return FollowCounts{}, err
	}
	return FollowCounts{Following: following, Followers: followers}, nil
}

// IsFollowing checks if followerID follows followeeID.
func (s *FollowService) IsFollowing(ctx context.Context, followerID, followeeID uint64) (bool, error) {
	if followerID == 0 || followeeID == 0 || followerID == followeeID {
		return false, nil
	}
	var n int64
	if err := s.db.WithContext(ctx).Model(&user.UserFollow{}).
		Where("follower_id = ? AND followee_id = ?", followerID, followeeID).
		Count(&n).Error; err != nil {
		return false, err
	}
	return n > 0, nil
}

// GetFollowingIDs returns a set of followee IDs that the follower follows among the candidates.
func (s *FollowService) GetFollowingIDs(ctx context.Context, followerID uint64, candidateIDs []uint64) (map[uint64]bool, error) {
	out := make(map[uint64]bool)
	if followerID == 0 || len(candidateIDs) == 0 {
		return out, nil
	}
	uniq := make([]uint64, 0, len(candidateIDs))
	seen := make(map[uint64]struct{}, len(candidateIDs))
	for _, id := range candidateIDs {
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
		return out, nil
	}
	var ids []uint64
	if err := s.db.WithContext(ctx).Model(&user.UserFollow{}).
		Where("follower_id = ? AND followee_id IN ?", followerID, uniq).
		Pluck("followee_id", &ids).Error; err != nil {
		return nil, err
	}
	for _, id := range ids {
		out[id] = true
	}
	return out, nil
}

// GetFollowingList returns the follow rows for following list.
func (s *FollowService) GetFollowingList(ctx context.Context, ownerID uint64, limit int, groupID uint64) ([]user.UserFollow, error) {
	q := s.db.WithContext(ctx).Where("follower_id = ?", ownerID)
	if groupID > 0 {
		var memberIDs []uint64
		if err := s.db.WithContext(ctx).Model(&user.UserFollowGroupMember{}).
			Where("group_id = ?", groupID).Pluck("followee_id", &memberIDs).Error; err != nil {
			return nil, err
		}
		if len(memberIDs) == 0 {
			return []user.UserFollow{}, nil
		}
		q = q.Where("followee_id IN ?", memberIDs)
	}
	var rows []user.UserFollow
	if err := q.Order("created_at DESC").Limit(limit).Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// GetFollowersList returns the follow rows for followers list.
func (s *FollowService) GetFollowersList(ctx context.Context, ownerID uint64, limit int) ([]user.UserFollow, error) {
	var rows []user.UserFollow
	if err := s.db.WithContext(ctx).Where("followee_id = ?", ownerID).
		Order("created_at DESC").Limit(limit).Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// ToggleFollow toggles a follow relationship.
func (s *FollowService) ToggleFollow(ctx context.Context, followerID, followeeID uint64) (bool, error) {
	var row user.UserFollow
	err := s.db.WithContext(ctx).Where("follower_id = ? AND followee_id = ?", followerID, followeeID).First(&row).Error
	if err == nil {
		// Unfollow
		if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			if err := tx.Delete(&row).Error; err != nil {
				return err
			}
			return tx.Where("group_id IN (SELECT id FROM user_follow_groups WHERE user_id = ?) AND followee_id = ?", followerID, followeeID).
				Delete(&user.UserFollowGroupMember{}).Error
		}); err != nil {
			return false, err
		}
		return false, nil
	}
	if err != gorm.ErrRecordNotFound {
		return false, err
	}
	// Follow
	row = user.UserFollow{FollowerID: followerID, FolloweeID: followeeID}
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		return false, err
	}
	return true, nil
}

// LoadUser returns a user model, checking for anonymization.
func (s *FollowService) LoadUser(ctx context.Context, userID uint64) (*user.User, error) {
	var u user.User
	if err := s.db.WithContext(ctx).First(&u, userID).Error; err != nil {
		return nil, err
	}
	if user.IsUserAnonymized(&u) {
		return nil, gorm.ErrRecordNotFound
	}
	return &u, nil
}

// UsersBlocked checks if either user has blocked the other.
func (s *FollowService) UsersBlocked(ctx context.Context, a, b uint64) (bool, error) {
	if a == 0 || b == 0 || a == b {
		return false, nil
	}
	var n int64
	if err := s.db.WithContext(ctx).Model(&user.UserBlock{}).
		Where("(blocker_id = ? AND blocked_id = ?) OR (blocker_id = ? AND blocked_id = ?)", a, b, b, a).
		Count(&n).Error; err != nil {
		return false, err
	}
	return n > 0, nil
}

// GetUploaderPublishedCount returns total published content count for a user.
func (s *FollowService) GetUploaderPublishedCount(ctx context.Context, userID uint64) (int64, error) {
	var videoN, articleN, dynN int64
	if err := s.db.WithContext(ctx).Model(&video.Video{}).
		Where("user_id = ? AND status = ?", userID, video.StatusPublished).
		Count(&videoN).Error; err != nil {
		return 0, err
	}
	if err := s.db.WithContext(ctx).Model(&article.Article{}).
		Where("user_id = ? AND status = ?", userID, article.StatusPublished).
		Count(&articleN).Error; err != nil {
		return 0, err
	}
	if err := s.db.WithContext(ctx).Model(&dynamic.UserDynamic{}).
		Where("user_id = ?", userID).
		Count(&dynN).Error; err != nil {
		return 0, err
	}
	return videoN + articleN + dynN, nil
}

// ??? Follow Groups ???

// ListGroups returns all follow groups for a user.
func (s *FollowService) ListGroups(ctx context.Context, userID uint64) ([]user.UserFollowGroup, error) {
	var groups []user.UserFollowGroup
	if err := s.db.WithContext(ctx).Where("user_id = ?", userID).Order("created_at ASC, id ASC").Find(&groups).Error; err != nil {
		return nil, err
	}
	return groups, nil
}

// GetGroup returns a single follow group, verifying ownership.
func (s *FollowService) GetGroup(ctx context.Context, userID, groupID uint64) (*user.UserFollowGroup, error) {
	if groupID == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	var g user.UserFollowGroup
	if err := s.db.WithContext(ctx).Where("id = ? AND user_id = ?", groupID, userID).First(&g).Error; err != nil {
		return nil, err
	}
	return &g, nil
}

// GetGroupMemberCounts returns member counts for the given group IDs.
func (s *FollowService) GetGroupMemberCounts(ctx context.Context, groupIDs []uint64) (map[uint64]int64, error) {
	out := make(map[uint64]int64, len(groupIDs))
	if len(groupIDs) == 0 {
		return out, nil
	}
	type row struct {
		GroupID uint64
		Cnt     int64
	}
	var rows []row
	if err := s.db.WithContext(ctx).Model(&user.UserFollowGroupMember{}).
		Select("group_id, COUNT(*) AS cnt").
		Where("group_id IN ?", groupIDs).
		Group("group_id").
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	for i := range rows {
		out[rows[i].GroupID] = rows[i].Cnt
	}
	return out, nil
}

// CreateGroup creates a new follow group for the user.
func (s *FollowService) CreateGroup(ctx context.Context, userID uint64, name string) (*user.UserFollowGroup, error) {
	var total int64
	if err := s.db.WithContext(ctx).Model(&user.UserFollowGroup{}).Where("user_id = ?", userID).Count(&total).Error; err != nil {
		return nil, err
	}
	if total >= 50 {
		return nil, ErrParamError
	}
	var dup int64
	if err := s.db.WithContext(ctx).Model(&user.UserFollowGroup{}).
		Where("user_id = ? AND name = ?", userID, name).
		Count(&dup).Error; err != nil {
		return nil, err
	}
	if dup > 0 {
		return nil, ErrParamError
	}
	g := user.UserFollowGroup{UserID: userID, Name: name}
	if err := s.db.WithContext(ctx).Create(&g).Error; err != nil {
		return nil, err
	}
	return &g, nil
}

// UpdateGroup renames a follow group, verifying ownership.
func (s *FollowService) UpdateGroup(ctx context.Context, userID, groupID uint64, name string) (*user.UserFollowGroup, error) {
	g, err := s.GetGroup(ctx, userID, groupID)
	if err != nil {
		return nil, err
	}
	if err := s.db.WithContext(ctx).Model(g).Update("name", name).Error; err != nil {
		return nil, err
	}
	return s.GetGroup(ctx, userID, groupID)
}

// DeleteGroup deletes a follow group and its members, verifying ownership.
func (s *FollowService) DeleteGroup(ctx context.Context, userID, groupID uint64) error {
	g, err := s.GetGroup(ctx, userID, groupID)
	if err != nil {
		return err
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("group_id = ?", g.ID).Delete(&user.UserFollowGroupMember{}).Error; err != nil {
			return err
		}
		return tx.Delete(&user.UserFollowGroup{}, g.ID).Error
	})
}

// GetFolloweeGroupIDs returns which of the user's follow groups include a followee.
func (s *FollowService) GetFolloweeGroupIDs(ctx context.Context, userID, followeeID uint64) ([]uint64, error) {
	var groupIDs []uint64
	err := s.db.WithContext(ctx).Model(&user.UserFollowGroupMember{}).
		Select("user_follow_group_members.group_id").
		Joins("JOIN user_follow_groups ON user_follow_groups.id = user_follow_group_members.group_id").
		Where("user_follow_groups.user_id = ? AND user_follow_group_members.followee_id = ?", userID, followeeID).
		Pluck("user_follow_group_members.group_id", &groupIDs).Error
	return groupIDs, err
}

// AddGroupMember adds a followee to a follow group.
func (s *FollowService) AddGroupMember(ctx context.Context, groupID, followeeID uint64) error {
	_, err := s.GetGroup(ctx, 0, groupID) // ownership check done by caller
	if err != nil {
		return err
	}
	var existing user.UserFollowGroupMember
	err = s.db.WithContext(ctx).Where("group_id = ? AND followee_id = ?", groupID, followeeID).First(&existing).Error
	if err == nil {
		return nil // already exists
	}
	if err != gorm.ErrRecordNotFound {
		return err
	}
	return s.db.WithContext(ctx).Create(&user.UserFollowGroupMember{GroupID: groupID, FolloweeID: followeeID}).Error
}

// RemoveGroupMember removes a followee from a follow group.
func (s *FollowService) RemoveGroupMember(ctx context.Context, groupID, followeeID uint64) error {
	return s.db.WithContext(ctx).Where("group_id = ? AND followee_id = ?", groupID, followeeID).
		Delete(&user.UserFollowGroupMember{}).Error
}

// RemoveFolloweeFromAllGroups removes a followee from all groups owned by the user.
func (s *FollowService) RemoveFolloweeFromAllGroups(ctx context.Context, ownerID, followeeID uint64) error {
	var groupIDs []uint64
	if err := s.db.WithContext(ctx).Model(&user.UserFollowGroup{}).
		Where("user_id = ?", ownerID).
		Pluck("id", &groupIDs).Error; err != nil {
		return err
	}
	if len(groupIDs) == 0 {
		return nil
	}
	return s.db.WithContext(ctx).Where("group_id IN ? AND followee_id = ?", groupIDs, followeeID).
		Delete(&user.UserFollowGroupMember{}).Error
}

// GetFolloweeIDsInGroup returns followee IDs in a group.
func (s *FollowService) GetFolloweeIDsInGroup(ctx context.Context, groupID uint64) ([]uint64, error) {
	var ids []uint64
	err := s.db.WithContext(ctx).Model(&user.UserFollowGroupMember{}).
		Where("group_id = ?", groupID).
		Pluck("followee_id", &ids).Error
	return ids, err
}

// BlockUser blocks targetID by uid, removing mutual follows.
func (s *FollowService) BlockUser(ctx context.Context, uid, targetID uint64) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		row := user.UserBlock{BlockerID: uid, BlockedID: targetID}
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
		return unfollowBothWays(tx, uid, targetID)
	})
}

// UnblockUser removes block and unfollows both ways.
func (s *FollowService) UnblockUser(ctx context.Context, uid, targetID uint64) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return unfollowBothWays(tx, uid, targetID)
	})
}

func unfollowBothWays(tx *gorm.DB, a, b uint64) error {
	if err := tx.Where("follower_id = ? AND followee_id = ?", a, b).
		Delete(&user.UserFollow{}).Error; err != nil {
		return err
	}
	if err := tx.Where("follower_id = ? AND followee_id = ?", b, a).
		Delete(&user.UserFollow{}).Error; err != nil {
		return err
	}
	// Remove from all follow groups for both directions
	for _, owner := range []uint64{a, b} {
		var groupIDs []uint64
		if err := tx.Model(&user.UserFollowGroup{}).
			Where("user_id = ?", owner).Pluck("id", &groupIDs).Error; err != nil {
			return err
		}
		if len(groupIDs) > 0 {
			followee := b
			if owner == b {
				followee = a
			}
			if err := tx.Where("group_id IN ? AND followee_id = ?", groupIDs, followee).
				Delete(&user.UserFollowGroupMember{}).Error; err != nil {
				return err
			}
		}
	}
	return nil
}
