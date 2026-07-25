package handler

import (
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"minibili/internal/model"
)

func newBlockDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.UserBlock{}, &model.UserFollow{}, &model.UserFollowGroup{}, &model.UserFollowGroupMember{}))
	return db
}

func TestDmUsersBlocked_SameUser(t *testing.T) {
	db := newBlockDB(t)
	got := dmUsersBlocked(db, 42, 42)
	require.False(t, got)
}

func TestDmUsersBlocked_ZeroIDs(t *testing.T) {
	db := newBlockDB(t)
	got := dmUsersBlocked(db, 0, 42)
	require.False(t, got)
	got = dmUsersBlocked(db, 42, 0)
	require.False(t, got)
}

func TestDmUsersBlocked_Blocked(t *testing.T) {
	db := newBlockDB(t)
	block := model.UserBlock{BlockerID: 1, BlockedID: 2}
	require.NoError(t, db.Create(&block).Error)

	got := dmUsersBlocked(db, 1, 2)
	require.True(t, got)
}

func TestDmUsersBlocked_ReverseBlocked(t *testing.T) {
	db := newBlockDB(t)
	block := model.UserBlock{BlockerID: 2, BlockedID: 1}
	require.NoError(t, db.Create(&block).Error)

	got := dmUsersBlocked(db, 1, 2)
	require.True(t, got)
}

func TestDmUsersBlocked_NotBlocked(t *testing.T) {
	db := newBlockDB(t)
	got := dmUsersBlocked(db, 1, 2)
	require.False(t, got)
}

func TestUnfollowBothWays_Success(t *testing.T) {
	db := newBlockDB(t)
	f1 := model.UserFollow{FollowerID: 1, FolloweeID: 2}
	f2 := model.UserFollow{FollowerID: 2, FolloweeID: 1}
	require.NoError(t, db.Create(&f1).Error)
	require.NoError(t, db.Create(&f2).Error)

	err := unfollowBothWays(db, 1, 2)
	require.NoError(t, err)

	var count int64
	db.Model(&model.UserFollow{}).Count(&count)
	require.Equal(t, int64(0), count)
}

func TestUnfollowBothWays_NoFollows(t *testing.T) {
	db := newBlockDB(t)
	err := unfollowBothWays(db, 1, 2)
	require.NoError(t, err)
}
