package service

import (
	"cakecake/internal/model/admin"
	"cakecake/internal/model/user"
	"cakecake/internal/pkg/jwttoken"
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func newAuthService(t *testing.T) *AuthService {
	t.Helper()
	db := newAgentTestDB(t)
	_, rdb := newAgentTestRedis(t)
	mgr, err := jwttoken.NewManager("test-secret-test-secret")
	require.NoError(t, err)
	return NewAuthService(db, rdb, zapNop(), mgr, AuthConfig{AgentBotUsername: "agent"})
}

func TestAuthService_RegisterAuthenticate(t *testing.T) {
	s := newAuthService(t)
	ctx := context.Background()

	// Invalid input.
	_, err := s.Register(ctx, "ab", "short")
	require.ErrorIs(t, err, ErrParamError)
	_, err = s.Register(ctx, "ai_bot", "password123")
	require.ErrorIs(t, err, ErrParamError)
	_, err = s.Register(ctx, "agent", "password123")
	require.ErrorIs(t, err, ErrParamError)

	res, err := s.Register(ctx, "alice", "password123")
	require.NoError(t, err)
	require.NotZero(t, res.UserID)
	require.Equal(t, "alice", res.Username)
	require.Contains(t, res.CakeID, "cake_")

	// Duplicate username.
	_, err = s.Register(ctx, "alice", "password123")
	require.Error(t, err)
	require.Equal(t, 40006, err.(*SvcError).Code)

	// Authenticate.
	authRes, err := s.Authenticate(ctx, res.UserID, "password123")
	require.NoError(t, err)
	require.Equal(t, "alice", authRes.Username)
	require.NotEmpty(t, authRes.AccessToken)
	require.NotEmpty(t, authRes.RefreshToken)

	_, err = s.Authenticate(ctx, res.UserID, "wrong")
	require.Equal(t, 40100, err.(*SvcError).Code)
	_, err = s.Authenticate(ctx, 999, "password123")
	require.Equal(t, 40100, err.(*SvcError).Code)

	// Anonymized user cannot authenticate.
	now := time.Now()
	require.NoError(t, s.db.Model(&user.User{}).Where("id = ?", res.UserID).
		Update("anonymized_at", &now).Error)
	_, err = s.Authenticate(ctx, res.UserID, "password123")
	require.Equal(t, 40100, err.(*SvcError).Code)
}

func TestAuthService_Refresh(t *testing.T) {
	s := newAuthService(t)
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
	require.ErrorIs(t, err, ErrUnauthorized)
	// Garbage token.
	_, err = s.Refresh(ctx, "garbage")
	require.ErrorIs(t, err, ErrUnauthorized)
}

func TestAuthService_AdminTokens(t *testing.T) {
	s := newAuthService(t)
	ctx := context.Background()

	require.False(t, s.AdminRefreshTokenInvalid(ctx, "tid1"))
	require.NoError(t, s.MarkAdminRefreshTokenInvalid(ctx, "tid1"))
	require.True(t, s.AdminRefreshTokenInvalid(ctx, "tid1"))

	// LookupUser.
	seedUser(t, s.db, 1, "alice")
	brief, err := s.LookupUser(ctx, " alice ")
	require.NoError(t, err)
	require.Equal(t, uint64(1), brief.ID)
	_, err = s.LookupUser(ctx, "nobody")
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)
}

func TestAuthService_AdminCRUD(t *testing.T) {
	s := newAuthService(t)
	ctx := context.Background()
	hash, _ := bcrypt.GenerateFromPassword([]byte("x"), bcrypt.MinCost)
	adm := admin.Admin{Username: "root", PasswordHash: string(hash)}
	require.NoError(t, s.db.Create(&adm).Error)

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
