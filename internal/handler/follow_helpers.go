package handler

import (
	"gorm.io/gorm"

	"minibili/internal/model"
)

// userFollowCounts returns following and follower counts (kept in handler for other handler consumers).
func userFollowCounts(db *gorm.DB, userID uint64) (following, followers int64) {
	_ = db.Model(&model.UserFollow{}).Where("follower_id = ?", userID).Count(&following).Error
	_ = db.Model(&model.UserFollow{}).Where("followee_id = ?", userID).Count(&followers).Error
	return following, followers
}

// userFollows checks if followerID follows followeeID (kept in handler for other handler consumers).
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

// uploaderPublishedCount returns total published content count (kept in handler for other handler consumers).
func uploaderPublishedCount(db *gorm.DB, userID uint64) int64 {
	var videoN, articleN, dynN int64
	_ = db.Model(&model.Video{}).
		Where("user_id = ? AND status = ?", userID, "published").
		Count(&videoN).Error
	_ = db.Model(&model.Article{}).
		Where("user_id = ? AND status = ?", userID, "published").
		Count(&articleN).Error
	_ = db.Model(&model.UserDynamic{}).
		Where("user_id = ?", userID).
		Count(&dynN).Error
	return videoN + articleN + dynN
}
