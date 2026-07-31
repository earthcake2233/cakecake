package storage

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDeleteObject_EmptyKey(t *testing.T) {
	o := &OSS{}
	err := o.DeleteObject("")
	require.NoError(t, err)
	err = o.DeleteObject("  ")
	require.NoError(t, err)
	err = o.DeleteObject("/")
	require.NoError(t, err)
}

func TestDeleteObjects_Empty(t *testing.T) {
	o := &OSS{}
	// Empty list should not error
	err := o.DeleteObjects(nil)
	require.NoError(t, err)
	err = o.DeleteObjects([]string{})
	require.NoError(t, err)
}

func TestDeleteObjects_SkipEmptyKeys(t *testing.T) {
	o := &OSS{}
	err := o.DeleteObjects([]string{"", "/", "  "})
	require.NoError(t, err)
}
