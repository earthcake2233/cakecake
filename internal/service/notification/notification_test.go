package notification

import (
	"cakecake/internal/model/comment"
	"cakecake/internal/model/notification"
	"cakecake/internal/service"
	"cakecake/internal/service/servicetest"
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func newNotificationService(t *testing.T) (*NotificationService, *gorm.DB) {
	t.Helper()
	db := servicetest.NewDB(t)
	_, rdb := servicetest.NewRedis(t)
	return NewNotificationService(db, rdb, servicetest.ZapNop(), service.NewUserProvider(db)), db
}

func TestNotification_NotifyVideoComment(t *testing.T) {
	ns, db := newNotificationService(t)
	ctx := context.Background()

	// Self-reply / zero uploader -> no-op.
	ns.NotifyVideoComment(ctx, 1, 1, comment.Comment{VideoID: 10, UserID: 1, Content: "x"})
	ns.NotifyVideoComment(ctx, 0, 1, comment.Comment{VideoID: 10, UserID: 1, Content: "x"})
	var n int64
	require.NoError(t, db.Model(&notification.Notification{}).Count(&n).Error)
	require.Zero(t, n)

	ns.NotifyVideoComment(ctx, 2, 1, comment.Comment{VideoID: 10, UserID: 1, Content: "hello world"})
	require.NoError(t, db.Model(&notification.Notification{}).Count(&n).Error)
	require.Equal(t, int64(1), n)
	var row notification.Notification
	require.NoError(t, db.First(&row).Error)
	require.Equal(t, uint64(2), row.RecipientID)
	require.Equal(t, "reply", row.Type)
	require.Equal(t, "hello world", row.CommentPreview)
}

func TestNotification_NotifyCommentReply(t *testing.T) {
	ns, db := newNotificationService(t)
	ctx := context.Background()
	parent := comment.Comment{VideoID: 10, UserID: 2, Content: "parent"}
	require.NoError(t, db.Create(&parent).Error)

	// Missing parent -> no-op.
	ns.NotifyCommentReply(ctx, 10, 1, &comment.Comment{ID: 99, VideoID: 10, UserID: 1}, 8888)
	// Self-reply -> no-op.
	ns.NotifyCommentReply(ctx, 10, 2, &comment.Comment{ID: 99, VideoID: 10, UserID: 2}, parent.ID)

	reply := &comment.Comment{VideoID: 10, UserID: 1, Content: "a very long reply content"}
	ns.NotifyCommentReply(ctx, 10, 1, reply, parent.ID)
	var n int64
	require.NoError(t, db.Model(&notification.Notification{}).Count(&n).Error)
	require.Equal(t, int64(1), n)
	var row notification.Notification
	require.NoError(t, db.First(&row).Error)
	require.Equal(t, uint64(2), row.RecipientID)
	require.Equal(t, "a very long rep", row.CommentPreview)
}

func TestNotification_NotifyCommentLike(t *testing.T) {
	ns, db := newNotificationService(t)
	ctx := context.Background()
	servicetest.SeedUser(t, db, 1, "alice")
	servicetest.SeedUser(t, db, 2, "bob")
	servicetest.SeedUser(t, db, 3, "carol")
	servicetest.SeedUser(t, db, 4, "dave")

	// Self-like / zero owner -> no-op.
	ns.NotifyCommentLike(ctx, comment.Comment{ID: 5, UserID: 1, Content: "c"}, 1)
	ns.NotifyCommentLike(ctx, comment.Comment{ID: 5, UserID: 0, Content: "c"}, 2)
	var n int64
	require.NoError(t, db.Model(&notification.Notification{}).Count(&n).Error)
	require.Zero(t, n)

	// Aggregation creation then update.
	ns.NotifyCommentLike(ctx, comment.Comment{ID: 5, UserID: 1, Content: "nice"}, 2)
	ns.NotifyCommentLike(ctx, comment.Comment{ID: 5, UserID: 1, Content: "nice"}, 3)
	require.NoError(t, db.Model(&notification.Notification{}).Count(&n).Error)
	require.Equal(t, int64(1), n)
	var row notification.Notification
	require.NoError(t, db.First(&row).Error)
	require.Equal(t, "like_aggregation", row.Type)
	require.Equal(t, 2, row.TotalLikes)

	// Muted recipient -> no new notification.
	require.NoError(t, db.Create(&notification.LikeNotifMute{RecipientID: 1, CommentID: 5}).Error)
	ns.NotifyCommentLike(ctx, comment.Comment{ID: 5, UserID: 1, Content: "nice"}, 4)
	require.NoError(t, db.Model(&notification.Notification{}).Count(&n).Error)
	require.Equal(t, int64(1), n)
}

func TestNotification_Article(t *testing.T) {
	ns, db := newNotificationService(t)
	ctx := context.Background()
	parent := comment.ArticleComment{ArticleID: 20, UserID: 2, Content: "p"}
	require.NoError(t, db.Create(&parent).Error)

	ns.NotifyArticleComment(ctx, 1, 2, comment.ArticleComment{ArticleID: 20, UserID: 2, Content: "hello"})
	ns.NotifyArticleCommentReply(ctx, 20, 3, &comment.ArticleComment{ArticleID: 20, UserID: 3, Content: "r"}, parent.ID)
	ns.NotifyArticleCommentReply(ctx, 20, 2, &comment.ArticleComment{ArticleID: 20, UserID: 2, Content: "r"}, parent.ID) // self

	var n int64
	require.NoError(t, db.Model(&notification.Notification{}).Count(&n).Error)
	require.Equal(t, int64(2), n)
}

func TestNotification_ListAndRead(t *testing.T) {
	ns, db := newNotificationService(t)
	ctx := context.Background()
	require.NoError(t, db.Create(&notification.Notification{RecipientID: 1, Type: "reply"}).Error)
	require.NoError(t, db.Create(&notification.Notification{RecipientID: 1, Type: "reply_like"}).Error)
	require.NoError(t, db.Create(&notification.Notification{RecipientID: 1, Type: "at"}).Error)
	require.NoError(t, db.Create(&notification.Notification{RecipientID: 1, Type: "like_aggregation"}).Error)
	require.NoError(t, db.Create(&notification.Notification{RecipientID: 1, Type: "system"}).Error)
	require.NoError(t, db.Create(&notification.Notification{RecipientID: 1, Type: "dm"}).Error)

	summary := ns.UnreadSummary(ctx, 1)
	require.Equal(t, int64(2), summary["reply"])
	require.Equal(t, int64(1), summary["at"])
	require.Equal(t, int64(1), summary["like"])
	require.Equal(t, int64(1), summary["system"])
	require.Equal(t, int64(1), summary["dm"])
	require.Equal(t, int64(0), ns.UnreadSummary(ctx, 0)["reply"])

	list, total, err := ns.ListNotifications(ctx, 1, "reply", 1, 10)
	require.NoError(t, err)
	require.Equal(t, int64(2), total)
	require.Len(t, list, 2)
	_, total, err = ns.ListNotifications(ctx, 1, "dm", 0, 0)
	require.NoError(t, err)
	require.Equal(t, int64(1), total)

	require.NoError(t, ns.MarkCategoryRead(ctx, 1, "reply"))
	require.NoError(t, ns.MarkNotificationsRead(ctx, 1, []uint64{list[0].ID, list[1].ID}))
	summary = ns.UnreadSummary(ctx, 1)
	require.Zero(t, summary["reply"])

	// Get / Delete with ownership.
	one := list[0]
	got, err := ns.GetNotification(ctx, one.ID, 1)
	require.NoError(t, err)
	require.Equal(t, one.ID, got.ID)
	_, err = ns.GetNotification(ctx, one.ID, 99)
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)
	require.NoError(t, ns.DeleteNotification(ctx, 1, one.ID))
	_, err = ns.GetNotification(ctx, one.ID, 1)
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)
}

func TestNotification_MuteAndLikers(t *testing.T) {
	ns, db := newNotificationService(t)
	ctx := context.Background()
	servicetest.SeedUser(t, db, 1, "alice")
	servicetest.SeedUser(t, db, 2, "bob")
	require.NoError(t, db.Create(&comment.CommentLike{UserID: 2, CommentID: 5}).Error)
	require.NoError(t, db.Create(&notification.Notification{
		RecipientID: 1, Type: "like_aggregation", RelatedID: 5, PayloadJSON: "like_comment:5",
	}).Error)

	// Not a like aggregation -> param error.
	require.NoError(t, db.Create(&notification.Notification{RecipientID: 1, Type: "reply", RelatedID: 6}).Error)
	var replyID uint64
	require.NoError(t, db.Model(&notification.Notification{}).Where("type = ?", "reply").Pluck("id", &replyID).Error)
	err := ns.MuteLikeNotification(ctx, 1, replyID)
	require.ErrorIs(t, err, service.ErrParamError)

	// Mute aggregation -> creates mute row.
	var aggID uint64
	require.NoError(t, db.Model(&notification.Notification{}).Where("type = ?", "like_aggregation").Pluck("id", &aggID).Error)
	require.NoError(t, ns.MuteLikeNotification(ctx, 1, aggID))

	// ListNotificationLikers.
	likers, err := ns.ListNotificationLikers(ctx, 1, aggID)
	require.NoError(t, err)
	require.Len(t, likers, 1)
	require.Equal(t, uint64(2), likers[0].ID)

	// Missing notification -> error.
	_, err = ns.ListNotificationLikers(ctx, 1, 99999)
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)
}
