package handler

import (
	"minibili/internal/model/comment"
	"minibili/internal/model/video"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCommentDelete_Cascade(t *testing.T) {
	api, _, _ := newTestAPI(t)

	video := video.Video{
		Title:       "cascade test",
		Description: "desc",
		Status:      "published",
		UserID:      1,
	}
	require.NoError(t, api.DB.Create(&video).Error)

	root := comment.Comment{VideoID: video.ID, UserID: 1, Content: "root"}
	require.NoError(t, api.DB.Create(&root).Error)

	child1 := comment.Comment{VideoID: video.ID, UserID: 1, Content: "child1", ParentID: root.ID}
	require.NoError(t, api.DB.Create(&child1).Error)

	grandchild := comment.Comment{VideoID: video.ID, UserID: 1, Content: "grandchild", ParentID: child1.ID}
	require.NoError(t, api.DB.Create(&grandchild).Error)

	var count int64
	api.DB.Model(&comment.Comment{}).Where("video_id = ?", video.ID).Count(&count)
	require.Equal(t, int64(3), count)

	// Delete bottom-up: grandchildren, children, root
	api.DB.Where("parent_id = ?", child1.ID).Delete(&comment.Comment{})
	api.DB.Where("parent_id = ?", root.ID).Delete(&comment.Comment{})
	api.DB.Delete(&root)

	api.DB.Model(&comment.Comment{}).Where("video_id = ?", video.ID).Count(&count)
	require.Equal(t, int64(0), count)
}

func TestCommentLike_Toggle(t *testing.T) {
	api, _, _ := newTestAPI(t)

	video := video.Video{
		Title:       "like test",
		Description: "desc",
		Status:      "published",
		UserID:      1,
	}
	require.NoError(t, api.DB.Create(&video).Error)

	cm := comment.Comment{VideoID: video.ID, UserID: 1, Content: "nice"}
	require.NoError(t, api.DB.Create(&cm).Error)

	// Like
	like := comment.CommentLike{UserID: 2, CommentID: cm.ID}
	require.NoError(t, api.DB.Create(&like).Error)
	api.DB.Model(&comment.Comment{}).Where("id = ?", cm.ID).
		Update("like_count", uint64(1))

	var c comment.Comment
	api.DB.First(&c, cm.ID)
	require.Equal(t, uint64(1), c.LikeCount)

	// Unlike - delete like row and decrement
	api.DB.Where("user_id = ? AND comment_id = ?", 2, cm.ID).Delete(&comment.CommentLike{})
	api.DB.Model(&comment.Comment{}).Where("id = ?", cm.ID).
		Update("like_count", uint64(0))

	api.DB.First(&c, cm.ID)
	require.Equal(t, uint64(0), c.LikeCount)
}

func TestCommentPin(t *testing.T) {
	api, _, _ := newTestAPI(t)

	video := video.Video{
		Title:       "pin test",
		Description: "desc",
		Status:      "published",
		UserID:      1,
	}
	require.NoError(t, api.DB.Create(&video).Error)

	c1 := comment.Comment{VideoID: video.ID, UserID: 1, Content: "pinned comment", Pinned: true}
	require.NoError(t, api.DB.Create(&c1).Error)
	c2 := comment.Comment{VideoID: video.ID, UserID: 1, Content: "normal comment", Pinned: false}
	require.NoError(t, api.DB.Create(&c2).Error)

	var pinned []comment.Comment
	api.DB.Where("video_id = ? AND pinned = ?", video.ID, true).Find(&pinned)
	require.Len(t, pinned, 1)
	require.Equal(t, c1.ID, pinned[0].ID)
}
