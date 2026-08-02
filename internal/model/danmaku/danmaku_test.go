package danmaku

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTableNames(t *testing.T) {
	require.Equal(t, "danmakus", (Danmaku{}).TableName())
	require.Equal(t, "danmaku_likes", (DanmakuLike{}).TableName())
}
