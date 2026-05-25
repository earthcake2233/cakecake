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
