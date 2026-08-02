package user

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestIsUserAnonymized(t *testing.T) {
	require.False(t, IsUserAnonymized(nil))
	require.False(t, IsUserAnonymized(&User{}))
	now := time.Now()
	require.True(t, IsUserAnonymized(&User{AnonymizedAt: &now}))
}

func TestDisplayUsername(t *testing.T) {
	require.Equal(t, "", DisplayUsername(nil))
	require.Equal(t, "alice", DisplayUsername(&User{Username: "alice"}))
	now := time.Now()
	require.Equal(t, "已注销用户", DisplayUsername(&User{Username: "alice", AnonymizedAt: &now}))
}

func TestFormatCakeID(t *testing.T) {
	require.Equal(t, "cake_00000000001", FormatCakeID(1))
	require.Equal(t, "cake_00000123456", FormatCakeID(123456))
}
