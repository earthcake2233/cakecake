package service

import (
	"cakecake/internal/model/user"
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func newUserService(t *testing.T) *UserService {
	t.Helper()
	return NewUserService(newAgentTestDB(t), zapNop())
}

func TestUserService_GetMeAndUpdate(t *testing.T) {
	s := newUserService(t)
	ctx := context.Background()
	seedUser(t, s.db, 1, "alice")

	me, err := s.GetMe(ctx, 1)
	require.NoError(t, err)
	require.Equal(t, "alice", me.Username)
	require.Contains(t, me.CakeID, "cake_")

	require.NoError(t, s.UpdateProfile(ctx, 1, map[string]interface{}{
		"nickname": "A", "sign": "hi", "gender": "female", "birthday": "2000-01-01", "avatar_url": "ignored",
	}))
	me, err = s.GetMe(ctx, 1)
	require.NoError(t, err)
	require.Equal(t, "A", me.Nickname)
	require.Equal(t, "hi", me.Sign)
	require.Equal(t, "female", me.Gender)
	require.NotEqual(t, "ignored", me.AvatarURL)

	// Empty filtered updates -> no-op.
	require.NoError(t, s.UpdateProfile(ctx, 1, map[string]interface{}{"unknown": 1}))

	// GetUserPublic + brief.
	pub, err := s.GetUserPublic(ctx, 1)
	require.NoError(t, err)
	require.Equal(t, "alice", pub.Username)
	name, avatar, err := s.GetUserBrief(ctx, 1)
	require.NoError(t, err)
	require.Equal(t, "A", name)
	require.Empty(t, avatar)
	_, _, err = s.GetUserBrief(ctx, 999)
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)
}

func TestUserService_UsernamePassword(t *testing.T) {
	s := newUserService(t)
	ctx := context.Background()
	seedUser(t, s.db, 1, "alice")
	seedUser(t, s.db, 2, "bob")

	// Duplicate username rejected.
	err := s.UpdateUsername(ctx, 1, "bob")
	require.ErrorIs(t, err, ErrParamError)
	require.NoError(t, s.UpdateUsername(ctx, 1, "alice2"))

	require.NoError(t, s.UpdatePassword(ctx, 1, "newhash"))
	hash, err := s.GetPasswordHash(ctx, 1)
	require.NoError(t, err)
	require.Equal(t, "newhash", hash)
}

func TestUserService_BatchAndAvatar(t *testing.T) {
	s := newUserService(t)
	ctx := context.Background()
	seedUser(t, s.db, 1, "alice")
	seedUser(t, s.db, 2, "bob")

	users := s.BatchGetUsers(ctx, []uint64{1, 2, 99})
	require.Len(t, users, 2)
	require.Empty(t, s.BatchGetUsers(ctx, nil))

	require.NoError(t, s.UpdateAvatar(ctx, 1, "obj/1.jpg"))
	u, err := s.GetUserByID(ctx, 1)
	require.NoError(t, err)
	require.Equal(t, "obj/1.jpg", u.AvatarURL)

	require.NoError(t, s.UpdateAnnouncement(ctx, 1, "announce"))
	u, err = s.GetUserByID(ctx, 1)
	require.NoError(t, err)
	require.Equal(t, "announce", u.SpaceAnnouncement)
}

func TestUserService_CakeID(t *testing.T) {
	s := newUserService(t)
	ctx := context.Background()
	seedUser(t, s.db, 1, "alice")

	u, err := s.GetUserByID(ctx, 1)
	require.NoError(t, err)
	require.NoError(t, s.EnsureCakeID(ctx, u))
	require.NotEmpty(t, u.CakeID)
	// Already set -> no-op.
	before := u.CakeID
	require.NoError(t, s.EnsureCakeID(ctx, u))
	require.Equal(t, before, u.CakeID)
}

func TestUserService_DeletionAndPrivacy(t *testing.T) {
	s := newUserService(t)
	ctx := context.Background()
	seedUser(t, s.db, 1, "alice")

	reqAt := time.Now()
	effAt := reqAt.Add(7 * 24 * time.Hour)
	require.NoError(t, s.RequestDeletion(ctx, 1, reqAt, effAt))
	require.NoError(t, s.RevokeDeletion(ctx, 1))
	u, err := s.GetPrivacySettings(ctx, 1)
	require.NoError(t, err)
	require.Nil(t, u.DeletionRequestedAt)

	require.NoError(t, s.UpdatePrivacySettings(ctx, 1, map[string]interface{}{"privacy_public_fans": true}))
	u, err = s.GetPrivacySettings(ctx, 1)
	require.NoError(t, err)
	require.True(t, u.PrivacyPublicFans)

	// FinalizeDeletion commits the callback transaction.
	committed := false
	require.NoError(t, s.FinalizeDeletion(ctx, 1, func(tx *gorm.DB) error {
		committed = true
		return nil
	}))
	require.True(t, committed)
}

func TestUserService_CoinLedger(t *testing.T) {
	s := newUserService(t)
	ctx := context.Background()
	seedUser(t, s.db, 1, "alice")
	require.NoError(t, s.db.Create(&user.CoinLedger{UserID: 1, DeltaTenths: -10}).Error)
	require.NoError(t, s.db.Create(&user.CoinLedger{UserID: 1, DeltaTenths: 30}).Error)

	total, rows, err := s.ListCoinLedger(ctx, 1, time.Now().Add(-time.Hour), 10, 0)
	require.NoError(t, err)
	require.Equal(t, int64(2), total)
	require.Len(t, rows, 2)
}
