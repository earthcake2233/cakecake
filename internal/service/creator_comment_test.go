package service

import (
	"cakecake/internal/model/article"
	"cakecake/internal/model/comment"
	"cakecake/internal/model/dynamic"
	"cakecake/internal/model/video"
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func newCreatorCommentService(t *testing.T) *CreatorCommentService {
	t.Helper()
	db := newAgentTestDB(t)
	_, rdb := newAgentTestRedis(t)
	return NewCreatorCommentService(db, rdb, zapNop())
}

func TestCreatorComment_VideoComments(t *testing.T) {
	s := newCreatorCommentService(t)
	ctx := context.Background()
	seedUser(t, s.db, 1, "creator")
	seedUser(t, s.db, 2, "viewer")
	require.NoError(t, s.db.Create(&video.Video{ID: 10, UserID: 1, Title: "my video", Status: video.StatusPublished}).Error)
	require.NoError(t, s.db.Create(&comment.Comment{VideoID: 10, UserID: 2, Content: "approved", Approved: true}).Error)
	require.NoError(t, s.db.Create(&comment.Comment{VideoID: 10, UserID: 2, Content: "pending", Approved: false}).Error)
	require.NoError(t, s.db.Model(&video.Video{}).Where("id = ?", 10).Update("comments_curated", true).Error)

	// Non-pending: only approved.
	res, err := s.ListCreatorVideoComments(ctx, CreatorVideoCommentQuery{UserID: 1, Page: 1, PageSize: 10})
	require.NoError(t, err)
	require.Equal(t, int64(1), res.Total)
	require.Len(t, res.Comments, 1)
	require.Equal(t, "approved", res.Comments[0].Content)

	// Pending unprocessed.
	res, err = s.ListCreatorVideoComments(ctx, CreatorVideoCommentQuery{UserID: 1, Page: 1, PageSize: 10, Pending: true})
	require.NoError(t, err)
	require.Equal(t, int64(1), res.Total)
	require.Equal(t, "pending", res.Comments[0].Content)

	// Pending ignored.
	require.NoError(t, s.db.Model(&comment.Comment{}).Where("content = ?", "pending").
		Update("curated_ignored", true).Error)
	res, err = s.ListCreatorVideoComments(ctx, CreatorVideoCommentQuery{
		UserID: 1, Page: 1, PageSize: 10, Pending: true, PendingStatus: "ignored",
	})
	require.NoError(t, err)
	require.Equal(t, int64(1), res.Total)

	// Keyword + likes sort.
	res, err = s.ListCreatorVideoComments(ctx, CreatorVideoCommentQuery{
		UserID: 1, Page: 1, PageSize: 10, Keyword: "approv", SortKey: "likes",
	})
	require.NoError(t, err)
	require.Equal(t, int64(1), res.Total)

	// FilterVideoID.
	res, err = s.ListCreatorVideoComments(ctx, CreatorVideoCommentQuery{UserID: 1, Page: 1, PageSize: 10, FilterVideoID: 10})
	require.NoError(t, err)
	require.Equal(t, int64(1), res.Total)

	// Viewer likes.
	var cid uint64
	require.NoError(t, s.db.Model(&comment.Comment{}).Where("content = ?", "approved").Pluck("id", &cid).Error)
	require.NoError(t, s.db.Create(&comment.CommentLike{UserID: 2, CommentID: cid}).Error)
	res, err = s.ListCreatorVideoComments(ctx, CreatorVideoCommentQuery{UserID: 1, Page: 1, PageSize: 10, ViewerID: 2})
	require.NoError(t, err)
	require.True(t, res.LikedByViewer[cid])

	// Ownership checks.
	ok, err := s.CheckVideoOwnership(ctx, 10, 1)
	require.NoError(t, err)
	require.True(t, ok)
	ok, err = s.CheckVideoOwnership(ctx, 10, 2)
	require.NoError(t, err)
	require.False(t, ok)
	ok, err = s.CheckVideoOwnership(ctx, 999, 1)
	require.NoError(t, err)
	require.False(t, ok)
}

func TestCreatorComment_ArticleAndDynamic(t *testing.T) {
	s := newCreatorCommentService(t)
	ctx := context.Background()
	require.NoError(t, s.db.Create(&article.Article{ID: 20, UserID: 1, Title: "a", Status: article.StatusPublished}).Error)
	require.NoError(t, s.db.Create(&dynamic.UserDynamic{ID: 30, UserID: 1, Title: "d"}).Error)
	require.NoError(t, s.db.Create(&comment.ArticleComment{ArticleID: 20, UserID: 2, Content: "ac", Approved: true}).Error)
	require.NoError(t, s.db.Create(&comment.DynamicComment{DynamicID: 30, UserID: 2, Content: "dc", Approved: true}).Error)

	ares, err := s.ListCreatorArticleComments(ctx, CreatorArticleCommentQuery{UserID: 1, Page: 1, PageSize: 10})
	require.NoError(t, err)
	require.Equal(t, int64(1), ares.Total)

	dres, err := s.ListCreatorDynamicComments(ctx, CreatorDynamicCommentQuery{UserID: 1, Page: 1, PageSize: 10})
	require.NoError(t, err)
	require.Equal(t, int64(1), dres.Total)

	ok, err := s.CheckArticleOwnership(ctx, 20, 1)
	require.NoError(t, err)
	require.True(t, ok)
	ok, err = s.CheckArticleOwnership(ctx, 20, 2)
	require.NoError(t, err)
	require.False(t, ok)
	ok, err = s.CheckDynamicOwnership(ctx, 30, 1)
	require.NoError(t, err)
	require.True(t, ok)
	ok, err = s.CheckDynamicOwnership(ctx, 30, 2)
	require.NoError(t, err)
	require.False(t, ok)
}

func TestCreatorComment_BatchFetchesAndCounts(t *testing.T) {
	s := newCreatorCommentService(t)
	ctx := context.Background()
	seedUser(t, s.db, 1, "u1")
	seedUser(t, s.db, 2, "u2")
	require.NoError(t, s.db.Create(&video.Video{ID: 10, UserID: 1, Title: "v", Status: video.StatusPublished}).Error)
	require.NoError(t, s.db.Create(&article.Article{ID: 20, UserID: 1, Title: "a", Status: article.StatusPublished}).Error)
	require.NoError(t, s.db.Create(&dynamic.UserDynamic{ID: 30, UserID: 1, Title: "d"}).Error)
	require.NoError(t, s.db.Create(&comment.Comment{VideoID: 10, UserID: 2, Content: "c1", Approved: true}).Error)
	cm := comment.Comment{VideoID: 10, UserID: 2, Content: "c2", Approved: true}
	require.NoError(t, s.db.Create(&cm).Error)
	require.NoError(t, s.db.Create(&comment.Comment{VideoID: 10, UserID: 2, Content: "reply", ParentID: cm.ID, Approved: true}).Error)
	require.NoError(t, s.db.Create(&comment.ArticleComment{ArticleID: 20, UserID: 2, Content: "ac", Approved: true}).Error)
	ac := comment.ArticleComment{ArticleID: 20, UserID: 2, Content: "ac2", Approved: true}
	require.NoError(t, s.db.Create(&ac).Error)
	require.NoError(t, s.db.Create(&comment.ArticleComment{ArticleID: 20, UserID: 2, Content: "areply", ParentID: ac.ID, Approved: true}).Error)
	require.NoError(t, s.db.Create(&comment.DynamicComment{DynamicID: 30, UserID: 2, Content: "dc", Approved: true}).Error)
	dc := comment.DynamicComment{DynamicID: 30, UserID: 2, Content: "dc2", Approved: true}
	require.NoError(t, s.db.Create(&dc).Error)
	require.NoError(t, s.db.Create(&comment.DynamicComment{DynamicID: 30, UserID: 2, Content: "dreply", ParentID: dc.ID, Approved: true}).Error)

	videos, err := s.BatchFetchVideos(ctx, []uint64{10})
	require.NoError(t, err)
	require.Contains(t, videos, uint64(10))
	users, err := s.BatchFetchUsers(ctx, []uint64{1, 2})
	require.NoError(t, err)
	require.Len(t, users, 2)
	comments, err := s.BatchFetchComments(ctx, []uint64{cm.ID})
	require.NoError(t, err)
	require.Contains(t, comments, cm.ID)
	acoms, err := s.BatchFetchArticleComments(ctx, []uint64{ac.ID})
	require.NoError(t, err)
	require.Contains(t, acoms, ac.ID)
	dcoms, err := s.BatchFetchDynamicComments(ctx, []uint64{dc.ID})
	require.NoError(t, err)
	require.Contains(t, dcoms, dc.ID)
	arts, err := s.BatchFetchArticles(ctx, []uint64{20})
	require.NoError(t, err)
	require.Contains(t, arts, uint64(20))
	dyns, err := s.BatchFetchDynamics(ctx, []uint64{30})
	require.NoError(t, err)
	require.Contains(t, dyns, uint64(30))

	// Like counts: 1 reply each.
	require.Equal(t, map[uint64]uint64{cm.ID: 1}, s.CommentReplyCounts(ctx, []uint64{cm.ID}))
	require.Equal(t, map[uint64]uint64{ac.ID: 1}, s.ArticleCommentReplyCounts(ctx, []uint64{ac.ID}))
	require.Equal(t, map[uint64]uint64{dc.ID: 1}, s.DynamicCommentReplyCounts(ctx, []uint64{dc.ID}))

	// Viewer like fetches.
	require.NoError(t, s.db.Create(&comment.CommentLike{UserID: 2, CommentID: cm.ID}).Error)
	likes, err := s.BatchFetchUserLikesForComments(ctx, 2, []uint64{cm.ID})
	require.NoError(t, err)
	require.Equal(t, map[uint64]bool{cm.ID: true}, likes)
	require.NoError(t, s.db.Create(&comment.ArticleCommentLike{UserID: 2, CommentID: ac.ID}).Error)
	alikes, err := s.BatchFetchUserLikesForArticleComments(ctx, 2, []uint64{ac.ID})
	require.NoError(t, err)
	require.Equal(t, map[uint64]bool{ac.ID: true}, alikes)
	require.NoError(t, s.db.Create(&comment.DynamicCommentLike{UserID: 2, CommentID: dc.ID}).Error)
	dlikes, err := s.BatchFetchUserLikesForDynamicComments(ctx, 2, []uint64{dc.ID})
	require.NoError(t, err)
	require.Equal(t, map[uint64]bool{dc.ID: true}, dlikes)
}
