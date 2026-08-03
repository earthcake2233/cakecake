package handler

import (
	"context"
)

// getFollowCounts returns following and follower counts.
func (a *API) getFollowCounts(ctx context.Context, userID uint64) (following, followers int64) {
	c, err := a.FollowSvc.GetFollowCounts(ctx, userID)
	if err != nil {
		return 0, 0
	}
	return c.Following, c.Followers
}

// isFollowing checks if followerID follows followeeID.
func (a *API) isFollowing(ctx context.Context, followerID, followeeID uint64) bool {
	if followerID == 0 || followeeID == 0 || followerID == followeeID {
		return false
	}
	ok, _ := a.FollowSvc.IsFollowing(ctx, followerID, followeeID)
	return ok
}

// getUploaderPublishedCount returns total published content count.
func (a *API) getUploaderPublishedCount(ctx context.Context, userID uint64) int64 {
	c, _ := a.FollowSvc.GetUploaderPublishedCount(ctx, userID)
	return c
}

// isDMUsersBlocked checks if either user has blocked the other.
func (a *API) isDMUsersBlocked(ctx context.Context, aID, bID uint64) bool {
	if aID == 0 || bID == 0 || aID == bID {
		return false
	}
	blocked, _ := a.FollowSvc.UsersBlocked(ctx, aID, bID)
	return blocked
}
