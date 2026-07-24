package storage

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUploadFile_NoSuchFile(t *testing.T) {
	o := &OSS{}
	err := o.UploadFile("key", "/nonexistent/path/file.mp4")
	require.Error(t, err)
}
