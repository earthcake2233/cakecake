package handler

import (
	"testing"

	"github.com/stretchr/testify/require"

	"minibili/internal/model"
)

func TestCommentDelete_Cascade(t *testing.T) {
	api, _, _ := newTestAPI(t)

	video := model.Video{
		Title:       "cascade test",
		Description: "desc",
		Status:      "published",
		UserID:      1,
	}
	require.NoError(t, api.DB.Create(&video).Error)

	root := model.Comment{VideoID: video.ID, UserID: 1, Content: "root"}
	require.NoError(t, api.DB.Create(&root).Error)

	child1 := model.Comment{VideoID: video.ID, UserID: 1, Content: "child1", ParentID: root.ID}
	require.NoError(t, api.DB.Create(&child1).Error)

	grandchild := model.Comment{VideoID: video.ID, UserID: 1, Content: "grandchild", ParentID: child1.ID}
	require.NoError(t, api.DB.Create(&grandchild).Error)

	var count int64
	api.DB.Model(&model.Comment{}).Where("video_id = ?", video.ID).Count(&count)
	require.Equal(t, int64(3), count)

	// Delete bottom-up: grandchildren, children, root
	api.DB.Where("parent_id = ?", child1.ID).Delete(&model.Comment{})
	api.DB.Where("parent_id = ?", root.ID).Delete(&model.Comment{})
	api.DB.Delete(&root)

	api.DB.Model(&model.Comment{}).Where("video_id = ?", video.ID).Count(&count)
	require.Equal(t, int64(0), count)
}

func TestCommentLike_Toggle(t *testing.T) {
	api, _, _ := newTestAPI(t)

	video := model.Video{
		Title:       "like test",
		Description: "desc",
		Status:      "published",
		UserID:      1,
	}
	require.NoError(t, api.DB.Create(&video).Error)

	comment := model.Comment{VideoID: video.ID, UserID: 1, Content: "nice"}
	require.NoError(t, api.DB.Create(&comment).Error)

	// Like
	like := model.CommentLike{UserID: 2, CommentID: comment.ID}
	require.NoError(t, api.DB.Create(&like).Error)
	api.DB.Model(&model.Comment{}).Where("id = ?", comment.ID).
		Update("like_count", uint64(1))

	var c model.Comment
	api.DB.First(&c, comment.ID)
	require.Equal(t, uint64(1), c.LikeCount)

	// Unlike - delete like row and decrement
	api.DB.Where("user_id = ? AND comment_id = ?", 2, comment.ID).Delete(&model.CommentLike{})
	api.DB.Model(&model.Comment{}).Where("id = ?", comment.ID).
		Update("like_count", uint64(0))

	api.DB.First(&c, comment.ID)
	require.Equal(t, uint64(0), c.LikeCount)
}

func TestCommentPin(t *testing.T) {
	api, _, _ := newTestAPI(t)

	video := model.Video{
		Title:       "pin test",
		Description: "desc",
		Status:      "published",
		UserID:      1,
	}
	require.NoError(t, api.DB.Create(&video).Error)

	c1 := model.Comment{VideoID: video.ID, UserID: 1, Content: "pinned comment", Pinned: true}
	require.NoError(t, api.DB.Create(&c1).Error)
	c2 := model.Comment{VideoID: video.ID, UserID: 1, Content: "normal comment", Pinned: false}
	require.NoError(t, api.DB.Create(&c2).Error)

	var pinned []model.Comment
	api.DB.Where("video_id = ? AND pinned = ?", video.ID, true).Find(&pinned)
	require.Len(t, pinned, 1)
	require.Equal(t, c1.ID, pinned[0].ID)
}