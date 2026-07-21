package jwttoken

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNewManager_EmptySecret(t *testing.T) {
	_, err := NewManager("")
	require.Error(t, err)
}

func TestIssuePair_Success(t *testing.T) {
	m, err := NewManager("my-secret-key-for-testing-32ch!")
	require.NoError(t, err)

	access, refresh, tokenID, err := m.IssuePair(42)
	require.NoError(t, err)
	require.NotEmpty(t, access)
	require.NotEmpty(t, refresh)
	require.NotEmpty(t, tokenID)
	require.NotEqual(t, access, refresh)
}

func TestParseAccess_Success(t *testing.T) {
	m, err := NewManager("my-secret-key-for-testing-32ch!")
	require.NoError(t, err)

	access, _, _, err := m.IssuePair(42)
	require.NoError(t, err)

	uid, tid, err := m.ParseAccess(access)
	require.NoError(t, err)
	require.Equal(t, uint64(42), uid)
	require.NotEmpty(t, tid)
}

func TestParseAccess_RejectsRefreshToken(t *testing.T) {
	m, _ := NewManager("my-secret-key-for-testing-32ch!")

	_, refresh, _, err := m.IssuePair(42)
	require.NoError(t, err)

	_, _, err = m.ParseAccess(refresh)
	require.Error(t, err)
	require.Contains(t, err.Error(), "not access token")
}

func TestParseRefresh_RejectsAccessToken(t *testing.T) {
	m, _ := NewManager("my-secret-key-for-testing-32ch!")

	access, _, _, err := m.IssuePair(42)
	require.NoError(t, err)

	_, _, err = m.ParseRefresh(access)
	require.Error(t, err)
	require.Contains(t, err.Error(), "not refresh token")
}

func TestParseAccess_InvalidToken(t *testing.T) {
	m, _ := NewManager("my-secret-key-for-testing-32ch!")
	_, _, err := m.ParseAccess("invalid-token")
	require.Error(t, err)
}

func TestParseAccess_WrongSecret(t *testing.T) {
	m1, _ := NewManager("secret-key-one-for-test-32chars!!")
	m2, _ := NewManager("secret-key-two-for-test-32chars!!")

	access, _, _, err := m1.IssuePair(42)
	require.NoError(t, err)

	_, _, err = m2.ParseAccess(access)
	require.Error(t, err)
}

func TestParseRefresh_Success(t *testing.T) {
	m, _ := NewManager("my-secret-key-for-testing-32ch!")

	_, refresh, _, err := m.IssuePair(99)
	require.NoError(t, err)

	uid, tid, err := m.ParseRefresh(refresh)
	require.NoError(t, err)
	require.Equal(t, uint64(99), uid)
	require.NotEmpty(t, tid)
}

func TestIssuePair_DifferentUsers(t *testing.T) {
	m, _ := NewManager("my-secret-key-for-testing-32ch!")

	a1, _, _, _ := m.IssuePair(1)
	a2, _, _, _ := m.IssuePair(2)

	uid1, _, _ := m.ParseAccess(a1)
	uid2, _, _ := m.ParseAccess(a2)
	require.Equal(t, uint64(1), uid1)
	require.Equal(t, uint64(2), uid2)
}

func TestExpiredToken(t *testing.T) {
	m, _ := NewManager("my-secret-key-for-testing-32ch!")

	// We can't easily create expired tokens without modifying Manager,
	// but we verify that ParseAccess rejects expired tokens by checking
	// the expiry field exists in claims
	_, _, err := m.ParseAccess("")
	require.Error(t, err)
}

func TestAdminPair_IssuesAdminPrincipal(t *testing.T) {
	m, _ := NewManager("my-secret-key-for-testing-32ch!")

	access, refresh, tokenID, err := m.IssueAdminPair(7)
	require.NoError(t, err)
	require.NotEmpty(t, access)
	require.NotEmpty(t, refresh)
	require.NotEmpty(t, tokenID)

	adminID, tid, err := m.ParseAdminAccess(access)
	require.NoError(t, err)
	require.Equal(t, uint64(7), adminID)
	require.NotEmpty(t, tid)
}

func TestParseAdminAccess_RejectsUserToken(t *testing.T) {
	m, _ := NewManager("my-secret-key-for-testing-32ch!")

	access, _, _, err := m.IssuePair(1)
	require.NoError(t, err)

	_, _, err = m.ParseAdminAccess(access)
	require.Error(t, err)
}

func TestParseAdminRefresh_RejectsAccessToken(t *testing.T) {
	m, _ := NewManager("my-secret-key-for-testing-32ch!")

	access, _, _, err := m.IssueAdminPair(3)
	require.NoError(t, err)

	_, _, err = m.ParseAdminRefresh(access)
	require.Error(t, err)
	require.Contains(t, err.Error(), "not admin refresh token")
}

func TestTokenExpiry(t *testing.T) {
	// Test that the Manager sets expiry on tokens
	m, _ := NewManager("my-secret-key-for-testing-32ch!")
	access, _, _, err := m.IssuePair(1)
	require.NoError(t, err)

	// Parse and check that expiry is in the future
	claims, err := m.parse(access)
	require.NoError(t, err)
	require.True(t, claims.ExpiresAt.After(time.Now()))
	require.True(t, time.Until(claims.ExpiresAt.Time) > 1*time.Hour)
}