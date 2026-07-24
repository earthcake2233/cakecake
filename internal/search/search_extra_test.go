package search

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestClientClose(t *testing.T) {
	c := &Client{}
	require.NoError(t, c.Close())
}

func TestSearchStatusFromCount(t *testing.T) {
	require.Equal(t, "empty", searchStatusFromCount(0))
	require.Equal(t, "empty", searchStatusFromCount(-1))
	require.Equal(t, "ok", searchStatusFromCount(1))
}
