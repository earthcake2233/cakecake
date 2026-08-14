package traceid

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFromEmptyContext(t *testing.T) {
	require.Empty(t, FromContext(context.Background()))
}

func TestWithAndFrom(t *testing.T) {
	ctx := WithContext(context.Background(), "trace-1")
	require.Equal(t, "trace-1", FromContext(ctx))
}

func TestWithEmptyIsNoop(t *testing.T) {
	ctx := WithContext(context.Background(), "")
	require.Empty(t, FromContext(ctx))
}

func TestNewGeneratesUniqueIDs(t *testing.T) {
	a := New()
	b := New()
	require.NotEmpty(t, a)
	require.NotEqual(t, a, b)
}
