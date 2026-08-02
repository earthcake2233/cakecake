package service

import (
	"cakecake/internal/model/article"
	"cakecake/internal/model/dynamic"
	"cakecake/internal/model/user"
	"cakecake/internal/model/video"
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func zapNop() *zap.Logger {
	return zap.NewNop()
}

func newFollowService(t *testing.T) *FollowService {
	t.Helper()
	return NewFollowService(newAgentTestDB(t), zapNop())
}

func seedUser(t *testing.T, db *gorm.DB, id uint64, username string) *user.User {
	t.Helper()
	u := &user.User{ID: id, Username: username, PasswordHash: "x", CakeID: user.FormatCakeID(id)}
	require.NoError(t, db.Create(u).Error)
	return u
}

func seedFollow(t *testing.T, db *gorm.DB, followerID, followeeID uint64) {
	t.Helper()
	require.NoError(t, db.Create(&user.UserFollow{FollowerID: followerID, FolloweeID: followeeID}).Error)
}

func TestFollowService_GetFollowCounts(t *testing.T) {
	s := newFollowService(t)
	ctx := context.Background()

	seedFollow(t, s.db, 1, 2)
	seedFollow(t, s.db, 1, 3)
	seedFollow(t, s.db, 4, 1)

	got, err := s.GetFollowCounts(ctx, 1)
	require.NoError(t, err)
	require.Equal(t, FollowCounts{Following: 2, Followers: 1}, got)
}

func TestFollowService_IsFollowing(t *testing.T) {
	s := newFollowService(t)
	ctx := context.Background()

	// Invalid ids short-circuit to false.
	ok, err := s.IsFollowing(ctx, 0, 1)
	require.NoError(t, err)
	require.False(t, ok)
	ok, err = s.IsFollowing(ctx, 1, 1)
	require.NoError(t, err)
	require.False(t, ok)

	seedFollow(t, s.db, 1, 2)
	ok, err = s.IsFollowing(ctx, 1, 2)
	require.NoError(t, err)
	require.True(t, ok)

	ok, err = s.IsFollowing(ctx, 2, 1)
	require.NoError(t, err)
	require.False(t, ok)
}

func TestFollowService_GetFollowingIDs(t *testing.T) {
	s := newFollowService(t)
	ctx := context.Background()

	// No-op inputs.
	out, err := s.GetFollowingIDs(ctx, 0, []uint64{1, 2})
	require.NoError(t, err)
	require.Empty(t, out)
	out, err = s.GetFollowingIDs(ctx, 1, nil)
	require.NoError(t, err)
	require.Empty(t, out)

	seedFollow(t, s.db, 1, 2)
	seedFollow(t, s.db, 1, 3)
	out, err = s.GetFollowingIDs(ctx, 1, []uint64{2, 3, 4, 3, 0, 1})
	require.NoError(t, err)
	require.Equal(t, map[uint64]bool{2: true, 3: true}, out)
}

func TestFollowService_GetFollowingList(t *testing.T) {
	s := newFollowService(t)
	ctx := context.Background()

	seedFollow(t, s.db, 1, 2)
	seedFollow(t, s.db, 1, 3)

	rows, err := s.GetFollowingList(ctx, 1, 10, 0)
	require.NoError(t, err)
	require.Len(t, rows, 2)

	// Group filter with no members yields empty list.
	rows, err = s.GetFollowingList(ctx, 1, 10, 99)
	require.NoError(t, err)
	require.Empty(t, rows)

	// Group filter with member.
	g := user.UserFollowGroup{UserID: 1, Name: "g1"}
	require.NoError(t, s.db.Create(&g).Error)
	require.NoError(t, s.db.Create(&user.UserFollowGroupMember{GroupID: g.ID, FolloweeID: 2}).Error)
	rows, err = s.GetFollowingList(ctx, 1, 10, g.ID)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, uint64(2), rows[0].FolloweeID)
}

func TestFollowService_GetFollowersList(t *testing.T) {
	s := newFollowService(t)
	ctx := context.Background()

	seedFollow(t, s.db, 1, 2)
	seedFollow(t, s.db, 3, 2)
	rows, err := s.GetFollowersList(ctx, 2, 10)
	require.NoError(t, err)
	require.Len(t, rows, 2)
}

func TestFollowService_ToggleFollow(t *testing.T) {
	s := newFollowService(t)
	ctx := context.Background()

	// Follow.
	ok, err := s.ToggleFollow(ctx, 1, 2)
	require.NoError(t, err)
	require.True(t, ok)

	// Follow again -> unfollow.
	ok, err = s.ToggleFollow(ctx, 1, 2)
	require.NoError(t, err)
	require.False(t, ok)
	var n int64
	require.NoError(t, s.db.Model(&user.UserFollow{}).Count(&n).Error)
	require.Zero(t, n)

	// Unfollow also removes group memberships for the pair.
	g := user.UserFollowGroup{UserID: 1, Name: "g1"}
	require.NoError(t, s.db.Create(&g).Error)
	require.NoError(t, s.db.Create(&user.UserFollowGroupMember{GroupID: g.ID, FolloweeID: 2}).Error)
	_, err = s.ToggleFollow(ctx, 1, 2)
	require.NoError(t, err)
	_, err = s.ToggleFollow(ctx, 1, 2)
	require.NoError(t, err)
	var members int64
	require.NoError(t, s.db.Model(&user.UserFollowGroupMember{}).Count(&members).Error)
	require.Zero(t, members)
}

func TestFollowService_LoadUser(t *testing.T) {
	s := newFollowService(t)
	ctx := context.Background()

	// Missing user.
	_, err := s.LoadUser(ctx, 42)
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)

	seedUser(t, s.db, 1, "alice")
	u, err := s.LoadUser(ctx, 1)
	require.NoError(t, err)
	require.Equal(t, "alice", u.Username)

	// Anonymized users are treated as not found.
	now := time.Now()
	require.NoError(t, s.db.Model(&user.User{}).Where("id = ?", 1).
		Update("anonymized_at", &now).Error)
	_, err = s.LoadUser(ctx, 1)
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)
}

func TestFollowService_UsersBlocked(t *testing.T) {
	s := newFollowService(t)
	ctx := context.Background()

	ok, err := s.UsersBlocked(ctx, 0, 1)
	require.NoError(t, err)
	require.False(t, ok)
	ok, err = s.UsersBlocked(ctx, 5, 5)
	require.NoError(t, err)
	require.False(t, ok)

	require.NoError(t, s.db.Create(&user.UserBlock{BlockerID: 1, BlockedID: 2}).Error)
	ok, err = s.UsersBlocked(ctx, 1, 2)
	require.NoError(t, err)
	require.True(t, ok)
	// Blocking is symmetric.
	ok, err = s.UsersBlocked(ctx, 2, 1)
	require.NoError(t, err)
	require.True(t, ok)
}

func TestFollowService_GetUploaderPublishedCount(t *testing.T) {
	s := newFollowService(t)
	ctx := context.Background()

	require.NoError(t, s.db.Create(&video.Video{UserID: 1, Title: "v1", Status: video.StatusPublished}).Error)
	require.NoError(t, s.db.Create(&video.Video{UserID: 1, Title: "v2", Status: video.StatusDraft}).Error)
	require.NoError(t, s.db.Create(&article.Article{UserID: 1, Title: "a1", Status: article.StatusPublished}).Error)
	require.NoError(t, s.db.Create(&dynamic.UserDynamic{UserID: 1, Title: "d1"}).Error)

	n, err := s.GetUploaderPublishedCount(ctx, 1)
	require.NoError(t, err)
	require.Equal(t, int64(3), n)
}

func TestFollowService_GroupsCRUD(t *testing.T) {
	s := newFollowService(t)
	ctx := context.Background()

	// List empty.
	groups, err := s.ListGroups(ctx, 1)
	require.NoError(t, err)
	require.Empty(t, groups)

	g1, err := s.CreateGroup(ctx, 1, "first")
	require.NoError(t, err)
	require.Equal(t, uint64(1), g1.UserID)

	// Duplicate name rejected.
	_, err = s.CreateGroup(ctx, 1, "first")
	require.ErrorIs(t, err, ErrParamError)

	// Update.
	upd, err := s.UpdateGroup(ctx, 1, g1.ID, "renamed")
	require.NoError(t, err)
	require.Equal(t, "renamed", upd.Name)

	// GetGroup verifies ownership.
	_, err = s.GetGroup(ctx, 2, g1.ID)
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)
	got, err := s.GetGroup(ctx, 1, g1.ID)
	require.NoError(t, err)
	require.Equal(t, g1.ID, got.ID)
	_, err = s.GetGroup(ctx, 1, 0)
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)

	// Member counts.
	require.NoError(t, s.db.Create(&user.UserFollowGroupMember{GroupID: g1.ID, FolloweeID: 2}).Error)
	require.NoError(t, s.db.Create(&user.UserFollowGroupMember{GroupID: g1.ID, FolloweeID: 3}).Error)
	counts, err := s.GetGroupMemberCounts(ctx, []uint64{g1.ID, 999})
	require.NoError(t, err)
	require.Equal(t, int64(2), counts[g1.ID])
	counts, err = s.GetGroupMemberCounts(ctx, nil)
	require.NoError(t, err)
	require.Empty(t, counts)

	// AddGroupMember: create + idempotent + ownership check.
	require.NoError(t, s.AddGroupMember(ctx, g1.ID, 4))
	require.NoError(t, s.AddGroupMember(ctx, g1.ID, 4))
	err = s.AddGroupMember(ctx, 999, 4)
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)

	ids, err := s.GetFolloweeIDsInGroup(ctx, g1.ID)
	require.NoError(t, err)
	require.ElementsMatch(t, []uint64{2, 3, 4}, ids)

	// GetFolloweeGroupIDs.
	gIDs, err := s.GetFolloweeGroupIDs(ctx, 1, 2)
	require.NoError(t, err)
	require.Equal(t, []uint64{g1.ID}, gIDs)

	// RemoveGroupMember.
	require.NoError(t, s.RemoveGroupMember(ctx, g1.ID, 4))
	ids, err = s.GetFolloweeIDsInGroup(ctx, g1.ID)
	require.NoError(t, err)
	require.NotContains(t, ids, uint64(4))

	// RemoveFolloweeFromAllGroups.
	require.NoError(t, s.RemoveFolloweeFromAllGroups(ctx, 1, 3))
	ids, err = s.GetFolloweeIDsInGroup(ctx, g1.ID)
	require.NoError(t, err)
	require.NotContains(t, ids, uint64(3))

	// DeleteGroup removes members too.
	require.NoError(t, s.DeleteGroup(ctx, 1, g1.ID))
	groups, err = s.ListGroups(ctx, 1)
	require.NoError(t, err)
	require.Empty(t, groups)
	var members int64
	require.NoError(t, s.db.Model(&user.UserFollowGroupMember{}).Count(&members).Error)
	require.Zero(t, members)
}

func TestFollowService_BlockUnblock(t *testing.T) {
	s := newFollowService(t)
	ctx := context.Background()

	seedFollow(t, s.db, 1, 2)
	seedFollow(t, s.db, 2, 1)
	g := user.UserFollowGroup{UserID: 1, Name: "g1"}
	require.NoError(t, s.db.Create(&g).Error)
	require.NoError(t, s.db.Create(&user.UserFollowGroupMember{GroupID: g.ID, FolloweeID: 2}).Error)

	require.NoError(t, s.BlockUser(ctx, 1, 2))
	blocked, err := s.UsersBlocked(ctx, 1, 2)
	require.NoError(t, err)
	require.True(t, blocked)
	// Mutual follows removed.
	var n int64
	require.NoError(t, s.db.Model(&user.UserFollow{}).Count(&n).Error)
	require.Zero(t, n)
	// Group member rows cleaned for owner 1.
	var members int64
	require.NoError(t, s.db.Model(&user.UserFollowGroupMember{}).Count(&members).Error)
	require.Zero(t, members)

	require.NoError(t, s.UnblockUser(ctx, 1, 2))
	blocked, err = s.UsersBlocked(ctx, 1, 2)
	require.NoError(t, err)
	require.False(t, blocked)
}
