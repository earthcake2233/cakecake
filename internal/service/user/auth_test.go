package user

import (
	"cakecake/internal/model/admin"
	"cakecake/internal/model/user"
	"cakecake/internal/pkg/jwttoken"
	"cakecake/internal/service"
	"cakecake/internal/service/servicetest"
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func newAuthService(t *testing.T) (*AuthService, *gorm.DB) {
	t.Helper()
	db := servicetest.NewDB(t)
	_, rdb := servicetest.NewRedis(t)
	mgr, err := jwttoken.NewManager("test-secret-test-secret")
	require.NoError(t, err)
	return NewAuthService(db, rdb, servicetest.ZapNop(), mgr, AuthConfig{AgentBotUsername: "agent"}), db
}

func TestAuthService_RegisterAuthenticate(t *testing.T) {
	s, db := newAuthService(t)
	ctx := context.Background()

	// Invalid input.
	_, err := s.Register(ctx, "ab", "short")
	require.ErrorIs(t, err, service.ErrParamError)
	_, err = s.Register(ctx, "ai_bot", "password123")
	require.ErrorIs(t, err, service.ErrParamError)
	_, err = s.Register(ctx, "agent", "password123")
	require.ErrorIs(t, err, service.ErrParamError)

	res, err := s.Register(ctx, "alice", "password123")
	require.NoError(t, err)
	require.NotZero(t, res.UserID)
	require.Equal(t, "alice", res.Username)
	require.Contains(t, res.CakeID, "cake_")

	// Duplicate username.
	_, err = s.Register(ctx, "alice", "password123")
	require.Error(t, err)
	require.Equal(t, 40006, err.(*service.SvcError).Code)

	// Authenticate.
	authRes, err := s.Authenticate(ctx, res.UserID, "password123")
	require.NoError(t, err)
	require.Equal(t, "alice", authRes.Username)
	require.NotEmpty(t, authRes.AccessToken)
	require.NotEmpty(t, authRes.RefreshToken)

	_, err = s.Authenticate(ctx, res.UserID, "wrong")
	require.Equal(t, 40100, err.(*service.SvcError).Code)
	_, err = s.Authenticate(ctx, 999, "password123")
	require.Equal(t, 40100, err.(*service.SvcError).Code)

	// Anonymized user cannot authenticate.
	now := time.Now()
	require.NoError(t, db.Model(&user.User{}).Where("id = ?", res.UserID).
		Update("anonymized_at", &now).Error)
	_, err = s.Authenticate(ctx, res.UserID, "password123")
	require.Equal(t, 40100, err.(*service.SvcError).Code)
}

func TestAuthService_Refresh(t *testing.T) {
	s, _ := newAuthService(t)
	ctx := context.Background()
	res, err := s.Register(ctx, "alice", "password123")
	require.NoError(t, err)

	authRes, err := s.Authenticate(ctx, res.UserID, "password123")
	require.NoError(t, err)

	ref, err := s.Refresh(ctx, authRes.RefreshToken)
	require.NoError(t, err)
	require.NotEmpty(t, ref.AccessToken)

	// Old refresh token is invalidated (rotation).
	_, err = s.Refresh(ctx, authRes.RefreshToken)
	require.ErrorIs(t, err, service.ErrUnauthorized)
	// Garbage token.
	_, err = s.Refresh(ctx, "garbage")
	require.ErrorIs(t, err, service.ErrUnauthorized)
}

func TestAuthService_Logout(t *testing.T) {
	s, _ := newAuthService(t)
	ctx := context.Background()
	res, err := s.Register(ctx, "alice", "password123")
	require.NoError(t, err)
	authRes, err := s.Authenticate(ctx, res.UserID, "password123")
	require.NoError(t, err)

	// Logout invalidates the refresh token server-side.
	require.NoError(t, s.Logout(ctx, authRes.RefreshToken))
	_, err = s.Refresh(ctx, authRes.RefreshToken)
	require.ErrorIs(t, err, service.ErrUnauthorized)

	// Idempotent: logging out again is still a success.
	require.NoError(t, s.Logout(ctx, authRes.RefreshToken))
	// Garbage token: no error, client-side cleanup still proceeds.
	require.NoError(t, s.Logout(ctx, "not-a-jwt"))
}

func TestAuthService_PasswordChangeInvalidatesRefresh(t *testing.T) {
	s, _ := newAuthService(t)
	ctx := context.Background()
	res, err := s.Register(ctx, "alice", "password123")
	require.NoError(t, err)
	authRes, err := s.Authenticate(ctx, res.UserID, "password123")
	require.NoError(t, err)

	// Rotate once: ref1 is a fresh token at epoch 0, not yet blacklisted.
	ref1, err := s.Refresh(ctx, authRes.RefreshToken)
	require.NoError(t, err)

	// Password change bumps the epoch: ref1 (epoch 0) must now be rejected.
	require.NoError(t, s.BumpRefreshEpoch(ctx, res.UserID))
	_, err = s.Refresh(ctx, ref1.RefreshToken)
	require.ErrorIs(t, err, service.ErrUnauthorized)

	// A token minted after the bump refreshes normally.
	auth2, err := s.Authenticate(ctx, res.UserID, "password123")
	require.NoError(t, err)
	ref2, err := s.Refresh(ctx, auth2.RefreshToken)
	require.NoError(t, err)
	require.NotEmpty(t, ref2.RefreshToken)
}

func TestAuthService_AdminTokens(t *testing.T) {
	s, db := newAuthService(t)
	ctx := context.Background()

	require.False(t, s.AdminRefreshTokenInvalid(ctx, "tid1"))
	require.NoError(t, s.MarkAdminRefreshTokenInvalid(ctx, "tid1"))
	require.True(t, s.AdminRefreshTokenInvalid(ctx, "tid1"))

	// LookupUser.
	servicetest.SeedUser(t, db, 1, "alice")
	brief, err := s.LookupUser(ctx, " alice ")
	require.NoError(t, err)
	require.Equal(t, uint64(1), brief.ID)
	_, err = s.LookupUser(ctx, "nobody")
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)
}

func TestAuthService_AdminCRUD(t *testing.T) {
	s, db := newAuthService(t)
	ctx := context.Background()
	hash, _ := bcrypt.GenerateFromPassword([]byte("x"), bcrypt.MinCost)
	adm := admin.Admin{Username: "root", PasswordHash: string(hash)}
	require.NoError(t, db.Create(&adm).Error)

	got, err := s.FindAdminByUsername(ctx, "root")
	require.NoError(t, err)
	require.Equal(t, adm.ID, got.ID)
	got, err = s.GetAdminByID(ctx, adm.ID)
	require.NoError(t, err)
	require.Equal(t, "root", got.Username)

	now := time.Now()
	require.NoError(t, s.UpdateAdminLoginTime(ctx, adm.ID, now))
	got, err = s.GetAdminByID(ctx, adm.ID)
	require.NoError(t, err)
	require.NotNil(t, got.LastLoginAt)

	_, err = s.FindAdminByUsername(ctx, "nope")
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)
}
