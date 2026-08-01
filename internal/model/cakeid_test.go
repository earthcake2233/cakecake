package model_test

import (
	"cakecake/internal/model/user"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFormatCakeID(t *testing.T) {
	require.Equal(t, "cake_00000000001", user.FormatCakeID(1))
	require.Equal(t, "cake_00000000123", user.FormatCakeID(123))
	require.Equal(t, "cake_91090742550", user.FormatCakeID(91090742550))
}

func TestFormatCakeID_Zero(t *testing.T) {
	require.Equal(t, "cake_00000000000", user.FormatCakeID(0))
}

func TestFormatCakeID_MaxUint64(t *testing.T) {
	id := user.FormatCakeID(^uint64(0))
	require.Contains(t, id, "cake_")
	require.Greater(t, len(id), 15)
}

func TestFormatCakeID_Consistency(t *testing.T) {
	// Same input should always produce same output
	a := user.FormatCakeID(12345)
	b := user.FormatCakeID(12345)
	require.Equal(t, a, b)
}
