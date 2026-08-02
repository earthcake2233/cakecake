package notification

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLikeNotifMuteTableName(t *testing.T) {
	require.Equal(t, "like_notif_mutes", (LikeNotifMute{}).TableName())
}
