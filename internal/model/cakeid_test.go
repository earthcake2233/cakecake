package model_test

import (
	"testing"

	"minibili/internal/model"

	"github.com/stretchr/testify/require"
)

func TestFormatCakeID(t *testing.T) {
	require.Equal(t, "cake_00000000001", model.FormatCakeID(1))
	require.Equal(t, "cake_00000000123", model.FormatCakeID(123))
	require.Equal(t, "cake_91090742550", model.FormatCakeID(91090742550))
}

func TestFormatCakeID_Zero(t *testing.T) {
	require.Equal(t, "cake_00000000000", model.FormatCakeID(0))
}

func TestFormatCakeID_MaxUint64(t *testing.T) {
	id := model.FormatCakeID(^uint64(0))
	require.Contains(t, id, "cake_")
	require.Greater(t, len(id), 15)
}

func TestFormatCakeID_Consistency(t *testing.T) {
	// Same input should always produce same output
	a := model.FormatCakeID(12345)
	b := model.FormatCakeID(12345)
	require.Equal(t, a, b)
}
