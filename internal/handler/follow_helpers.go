package handler

import (
	"context"

)

// getFollowCounts returns following and follower counts.
func (a *API) getFollowCounts(userID uint64) (following, followers int64) {
	c, err := a.FollowSvc.GetFollowCounts(context.Background(), userID)
	if err != nil {
		return 0, 0
	}
	return c.Following, c.Followers
}

// isFollowing checks if followerID follows followeeID.
func (a *API) isFollowing(followerID, followeeID uint64) bool {
	if followerID == 0 || followeeID == 0 || followerID == followeeID {
		return false
	}
	ok, _ := a.FollowSvc.IsFollowing(context.Background(), followerID, followeeID)
	return ok
}

// getUploaderPublishedCount returns total published content count.
func (a *API) getUploaderPublishedCount(userID uint64) int64 {
	c, _ := a.FollowSvc.GetUploaderPublishedCount(context.Background(), userID)
	return c
}

// isDMUsersBlocked checks if either user has blocked the other.
func (a *API) isDMUsersBlocked(aID, bID uint64) bool {
	if aID == 0 || bID == 0 || aID == bID {
		return false
	}
	blocked, _ := a.FollowSvc.UsersBlocked(context.Background(), aID, bID)
	return blocked
}
