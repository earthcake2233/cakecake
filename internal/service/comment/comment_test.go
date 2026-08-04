package comment

import (
	"cakecake/internal/model/article"
	"cakecake/internal/model/comment"
	"cakecake/internal/model/dynamic"
	"cakecake/internal/model/video"
	"cakecake/internal/pkg/sensitive"
	"cakecake/internal/service"
	"cakecake/internal/service/notification"
	"cakecake/internal/service/servicetest"
	vsvc "cakecake/internal/service/video"
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func newCommentService(t *testing.T) (*CommentService, *gorm.DB) {
	t.Helper()
	db := servicetest.NewDB(t)
	_, rdb := servicetest.NewRedis(t)

	f, err := os.CreateTemp("", "sensitive-*.txt")
	require.NoError(t, err)
	t.Cleanup(func() { os.Remove(f.Name()) })
	_, err = f.WriteString("badword\n")
	require.NoError(t, err)
	require.NoError(t, f.Close())
	filter := sensitive.NewFilter(f.Name(), zap.NewNop())
	require.NoError(t, filter.Reload())

	notif := notification.NewNotificationService(db, rdb, servicetest.ZapNop(), service.NewUserProvider(db))
	return NewCommentService(db, rdb, servicetest.ZapNop(), filter, notif,
		service.NewUserProvider(db), vsvc.NewVideoProvider(db), service.NewArticleProvider(db), service.NewDynamicProvider(db)), db
}

func seedCommentTargets(t *testing.T, db *gorm.DB) {
	t.Helper()
	servicetest.SeedUser(t, db, 1, "owner")
	servicetest.SeedUser(t, db, 2, "viewer")
	require.NoError(t, db.Create(&video.Video{ID: 10, UserID: 1, Title: "v", Status: video.StatusPublished}).Error)
	require.NoError(t, db.Create(&article.Article{ID: 20, UserID: 1, Title: "a", Status: article.StatusPublished}).Error)
	require.NoError(t, db.Create(&dynamic.UserDynamic{ID: 30, UserID: 1, Title: "d"}).Error)
}

func TestCommentService_PostAndList(t *testing.T) {
	s, db := newCommentService(t)
	ctx := context.Background()
	seedCommentTargets(t, db)

	// Invalid content.
	_, err := s.PostComment(ctx, 2, 10, PostCommentReq{Content: ""}, "")
	require.ErrorIs(t, err, service.ErrParamError)
	_, err = s.PostComment(ctx, 2, 10, PostCommentReq{Content: "has badword"}, "")
	require.ErrorIs(t, err, service.ErrCommentSensitive)
	// Missing video.
	_, err = s.PostComment(ctx, 2, 999, PostCommentReq{Content: "hi"}, "")
	require.ErrorIs(t, err, service.ErrNotFound)

	// Success + notification created.
	cm, err := s.PostComment(ctx, 2, 10, PostCommentReq{Content: "hello"}, "CN")
	require.NoError(t, err)
	require.True(t, cm.Approved)
	require.Equal(t, 1, cm.Level)

	// Reply to missing parent -> not found.
	_, err = s.PostComment(ctx, 2, 10, PostCommentReq{Content: "reply", ParentID: 999}, "")
	require.ErrorIs(t, err, service.ErrNotFound)

	// Reply to parent on another video -> param error.
	other := comment.Comment{VideoID: 11, UserID: 2, Content: "other"}
	require.NoError(t, db.Create(&other).Error)
	_, err = s.PostComment(ctx, 2, 10, PostCommentReq{Content: "reply", ParentID: other.ID}, "")
	require.ErrorIs(t, err, service.ErrParamError)

	// Reply success.
	reply, err := s.PostComment(ctx, 2, 10, PostCommentReq{Content: "nice", ParentID: cm.ID}, "")
	require.NoError(t, err)
	require.Equal(t, 2, reply.Level)

	// Comments closed.
	require.NoError(t, db.Model(&video.Video{}).Where("id = ?", 10).Update("comments_closed", true).Error)
	_, err = s.PostComment(ctx, 2, 10, PostCommentReq{Content: "x"}, "")
	require.ErrorIs(t, err, service.ErrCommentsClosed)
	res, err := s.ListComments(ctx, 10, 2)
	require.NoError(t, err)
	require.True(t, res.CommentsClosed)
	require.Empty(t, res.Items)
	require.NoError(t, db.Model(&video.Video{}).Where("id = ?", 10).Update("comments_closed", false).Error)

	// List with reactions.
	require.NoError(t, db.Create(&comment.CommentLike{UserID: 2, CommentID: cm.ID}).Error)
	res, err = s.ListComments(ctx, 10, 2)
	require.NoError(t, err)
	require.Len(t, res.Items, 2)
	require.False(t, res.CommentsClosed)

	// Curated video: pending comments hidden from list.
	require.NoError(t, db.Model(&video.Video{}).Where("id = ?", 10).Update("comments_curated", true).Error)
	curated, err := s.PostComment(ctx, 2, 10, PostCommentReq{Content: "pending"}, "")
	require.NoError(t, err)
	require.False(t, curated.Approved)
	res, err = s.ListComments(ctx, 10, 2)
	require.NoError(t, err)
	require.True(t, res.CommentsCurated)
	require.Len(t, res.Items, 2) // only approved
}

func TestCommentService_DeletePin(t *testing.T) {
	s, db := newCommentService(t)
	ctx := context.Background()
	seedCommentTargets(t, db)

	cm, err := s.PostComment(ctx, 2, 10, PostCommentReq{Content: "root"}, "")
	require.NoError(t, err)
	reply, err := s.PostComment(ctx, 2, 10, PostCommentReq{Content: "child", ParentID: cm.ID}, "")
	require.NoError(t, err)
	require.NoError(t, db.Create(&comment.CommentLike{UserID: 3, CommentID: cm.ID}).Error)
	require.NoError(t, db.Create(&comment.CommentLike{UserID: 3, CommentID: reply.ID}).Error)

	// Non-owner, non-uploader -> forbidden.
	servicetest.SeedUser(t, db, 5, "intruder")
	err = s.DeleteComment(ctx, 5, cm.ID, false)
	require.ErrorIs(t, err, service.ErrForbidden)
	// Missing comment.
	err = s.DeleteComment(ctx, 2, 999, false)
	require.ErrorIs(t, err, service.ErrNotFound)

	// Uploader deletes root -> cascade removes reply + likes.
	err = s.DeleteComment(ctx, 1, cm.ID, true)
	require.NoError(t, err)
	var n int64
	require.NoError(t, db.Model(&comment.Comment{}).Count(&n).Error)
	require.Zero(t, n)
	require.NoError(t, db.Model(&comment.CommentLike{}).Count(&n).Error)
	require.Zero(t, n)

	// Pin flow.
	cm2, err := s.PostComment(ctx, 2, 10, PostCommentReq{Content: "pin me"}, "")
	require.NoError(t, err)
	// Target video missing.
	_, err = s.PinComment(ctx, 999, cm2.ID)
	require.ErrorIs(t, err, service.ErrNotFound)
	// Comment belongs to another video.
	other := comment.Comment{VideoID: 12, UserID: 2, Content: "other"}
	require.NoError(t, db.Create(&other).Error)
	_, err = s.PinComment(ctx, 10, other.ID)
	require.ErrorIs(t, err, service.ErrParamError)

	pinned, err := s.PinComment(ctx, 10, cm2.ID)
	require.NoError(t, err)
	require.True(t, pinned)
	pinned, err = s.PinComment(ctx, 10, cm2.ID)
	require.NoError(t, err)
	require.False(t, pinned)
}

func TestCommentService_Reactions(t *testing.T) {
	s, db := newCommentService(t)
	ctx := context.Background()
	seedCommentTargets(t, db)
	cm, err := s.PostComment(ctx, 2, 10, PostCommentReq{Content: "like me"}, "")
	require.NoError(t, err)

	// Missing comment.
	_, _, err = s.ToggleCommentLike(ctx, 2, 999)
	require.ErrorIs(t, err, service.ErrNotFound)
	_, err = s.ToggleCommentDislike(ctx, 2, 999)
	require.ErrorIs(t, err, service.ErrNotFound)

	liked, count, err := s.ToggleCommentLike(ctx, 2, cm.ID)
	require.NoError(t, err)
	require.True(t, liked)
	require.Equal(t, 1, count)
	liked, count, err = s.ToggleCommentLike(ctx, 2, cm.ID)
	require.NoError(t, err)
	require.False(t, liked)
	require.Zero(t, count)

	disliked, err := s.ToggleCommentDislike(ctx, 2, cm.ID)
	require.NoError(t, err)
	require.True(t, disliked)
	disliked, err = s.ToggleCommentDislike(ctx, 2, cm.ID)
	require.NoError(t, err)
	require.False(t, disliked)

	// Like after dislike clears dislike.
	_, err = s.ToggleCommentDislike(ctx, 2, cm.ID)
	require.NoError(t, err)
	liked, _, err = s.ToggleCommentLike(ctx, 2, cm.ID)
	require.NoError(t, err)
	require.True(t, liked)

	// Approve + ignore.
	require.NoError(t, s.ApproveComment(ctx, cm.ID))
	require.NoError(t, s.IgnoreCuratedComment(ctx, cm.ID))
	var row comment.Comment
	require.NoError(t, db.First(&row, cm.ID).Error)
	require.True(t, row.Approved)
	require.True(t, row.CuratedIgnored)

	// Get by ID.
	got, err := s.GetCommentByID(ctx, cm.ID)
	require.NoError(t, err)
	require.Equal(t, cm.ID, got.ID)
	_, err = s.GetCommentByID(ctx, 999)
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)
	_, err = s.GetDynamicCommentByID(ctx, 999)
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)
}

func TestCommentService_ArticleAndDynamic(t *testing.T) {
	s, db := newCommentService(t)
	ctx := context.Background()
	seedCommentTargets(t, db)

	ac, err := s.PostArticleComment(ctx, 2, 20, PostCommentReq{Content: "article cmt"}, "")
	require.NoError(t, err)
	require.NotZero(t, ac.ID)
	ares, err := s.ListArticleComments(ctx, 20, 2)
	require.NoError(t, err)
	require.Len(t, ares.Items, 1)

	// Missing article.
	_, err = s.PostArticleComment(ctx, 2, 999, PostCommentReq{Content: "x"}, "")
	require.ErrorIs(t, err, service.ErrNotFound)
	_, err = s.ListArticleComments(ctx, 999, 2)
	require.ErrorIs(t, err, service.ErrNotFound)

	pinned, err := s.PinArticleComment(ctx, 20, ac.ID)
	require.NoError(t, err)
	require.True(t, pinned)
	liked, _, err := s.ToggleArticleCommentLike(ctx, 2, ac.ID)
	require.NoError(t, err)
	require.True(t, liked)
	disliked, err := s.ToggleArticleCommentDislike(ctx, 2, ac.ID)
	require.NoError(t, err)
	require.True(t, disliked)
	require.NoError(t, s.ApproveArticleComment(ctx, ac.ID))
	require.NoError(t, s.IgnoreArticleComment(ctx, ac.ID))
	gotAC, err := s.GetArticleComment(ctx, ac.ID)
	require.NoError(t, err)
	require.Equal(t, ac.ID, gotAC.ID)
	err = s.DeleteArticleComment(ctx, 2, ac.ID, false)
	require.NoError(t, err)

	dc, err := s.PostDynamicComment(ctx, 2, 30, PostCommentReq{Content: "dyn cmt"}, "")
	require.NoError(t, err)
	dres, err := s.ListDynamicComments(ctx, 30, 2)
	require.NoError(t, err)
	require.Len(t, dres.Items, 1)
	_, err = s.PostDynamicComment(ctx, 2, 999, PostCommentReq{Content: "x"}, "")
	require.ErrorIs(t, err, service.ErrNotFound)
	_, err = s.ListDynamicComments(ctx, 999, 2)
	require.ErrorIs(t, err, service.ErrNotFound)

	liked, count, err := s.ToggleDynamicCommentReaction(ctx, 2, dc.ID, true)
	require.NoError(t, err)
	require.True(t, liked)
	require.Equal(t, 1, count)
	disliked, count, err = s.ToggleDynamicCommentReaction(ctx, 2, dc.ID, false)
	require.NoError(t, err)
	require.True(t, disliked)
	require.Equal(t, 1, count)
	require.NoError(t, s.ApproveDynComment(ctx, dc.ID))
	require.NoError(t, s.IgnoreDynComment(ctx, dc.ID))
	err = s.DeleteDynamicComment(ctx, 2, dc.ID, false)
	require.NoError(t, err)
}
