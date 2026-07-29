package service

import (
	"context"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"minibili/internal/model"
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
	if err := s.db.WithContext(ctx).Model(&model.UserFollow{}).Where("follower_id = ?", userID).Count(&following).Error; err != nil {
		return FollowCounts{}, err
	}
	if err := s.db.WithContext(ctx).Model(&model.UserFollow{}).Where("followee_id = ?", userID).Count(&followers).Error; err != nil {
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
	if err := s.db.WithContext(ctx).Model(&model.UserFollow{}).
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
	if err := s.db.WithContext(ctx).Model(&model.UserFollow{}).
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
func (s *FollowService) GetFollowingList(ctx context.Context, ownerID uint64, limit int, groupID uint64) ([]model.UserFollow, error) {
	q := s.db.WithContext(ctx).Where("follower_id = ?", ownerID)
	if groupID > 0 {
		var memberIDs []uint64
		if err := s.db.WithContext(ctx).Model(&model.UserFollowGroupMember{}).
			Where("group_id = ?", groupID).Pluck("followee_id", &memberIDs).Error; err != nil {
			return nil, err
		}
		if len(memberIDs) == 0 {
			return []model.UserFollow{}, nil
		}
		q = q.Where("followee_id IN ?", memberIDs)
	}
	var rows []model.UserFollow
	if err := q.Order("created_at DESC").Limit(limit).Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// GetFollowersList returns the follow rows for followers list.
func (s *FollowService) GetFollowersList(ctx context.Context, ownerID uint64, limit int) ([]model.UserFollow, error) {
	var rows []model.UserFollow
	if err := s.db.WithContext(ctx).Where("followee_id = ?", ownerID).
		Order("created_at DESC").Limit(limit).Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// ToggleFollow toggles a follow relationship.
func (s *FollowService) ToggleFollow(ctx context.Context, followerID, followeeID uint64) (bool, error) {
	var row model.UserFollow
	err := s.db.WithContext(ctx).Where("follower_id = ? AND followee_id = ?", followerID, followeeID).First(&row).Error
	if err == nil {
		// Unfollow
		if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			if err := tx.Delete(&row).Error; err != nil {
				return err
			}
			return tx.Where("group_id IN (SELECT id FROM follow_groups WHERE user_id = ?) AND followee_id = ?", followerID, followeeID).
				Delete(&model.UserFollowGroupMember{}).Error
		}); err != nil {
			return false, err
		}
		return false, nil
	}
	if err != gorm.ErrRecordNotFound {
		return false, err
	}
	// Follow
	row = model.UserFollow{FollowerID: followerID, FolloweeID: followeeID}
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		return false, err
	}
	return true, nil
}

// LoadUser returns a user model, checking for anonymization.
func (s *FollowService) LoadUser(ctx context.Context, userID uint64) (*model.User, error) {
	var u model.User
	if err := s.db.WithContext(ctx).First(&u, userID).Error; err != nil {
		return nil, err
	}
	if model.IsUserAnonymized(&u) {
		return nil, gorm.ErrRecordNotFound
	}
	return &u, nil
}

// GetUploaderPublishedCount returns total published content count for a user.
func (s *FollowService) GetUploaderPublishedCount(ctx context.Context, userID uint64) (int64, error) {
	var videoN, articleN, dynN int64
	if err := s.db.WithContext(ctx).Model(&model.Video{}).
		Where("user_id = ? AND status = ?", userID, "published").
		Count(&videoN).Error; err != nil {
		return 0, err
	}
	if err := s.db.WithContext(ctx).Model(&model.Article{}).
		Where("user_id = ? AND status = ?", userID, "published").
		Count(&articleN).Error; err != nil {
		return 0, err
	}
	if err := s.db.WithContext(ctx).Model(&model.UserDynamic{}).
		Where("user_id = ?", userID).
		Count(&dynN).Error; err != nil {
		return 0, err
	}
	return videoN + articleN + dynN, nil
}
