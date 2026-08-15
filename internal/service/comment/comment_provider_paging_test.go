package comment

import (
	"context"
	"testing"

	"cakecake/internal/model/comment"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newPagingCommentDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&comment.Comment{}))
	return db
}

func TestListCommentsPage_RootsWithReplies(t *testing.T) {
	db := newPagingCommentDB(t)
	ctx := context.Background()
	for i := 1; i <= 5; i++ {
		root := comment.Comment{VideoID: 10, UserID: 1, Content: "root", Level: 1, LikeCount: uint64(i), Approved: true}
		require.NoError(t, db.Create(&root).Error)
		child := comment.Comment{VideoID: 10, UserID: 2, ParentID: root.ID, Level: 2, Content: "child", Approved: true}
		require.NoError(t, db.Create(&child).Error)
		grand := comment.Comment{VideoID: 10, UserID: 3, ParentID: child.ID, Level: 3, Content: "grand", Approved: true}
		require.NoError(t, db.Create(&grand).Error)
	}

	p := NewCommentProvider(db)
	pg, err := p.ListCommentsPage(ctx, CommentVideo, 10, false, CommentListQuery{Page: 2, PageSize: 2, Sort: "hot"})
	require.NoError(t, err)
	require.Equal(t, int64(5), pg.Total)
	// Page 2 of hot order (like_count desc) = roots with like_count 3 and 2,
	// each with its child + grandchild: 2 * 3 = 6 rows.
	require.Len(t, pg.Rows, 6)

	var gotRoots []commentListRow
	for _, r := range pg.Rows {
		if r.ParentID == 0 {
			gotRoots = append(gotRoots, r)
		}
	}
	require.Len(t, gotRoots, 2)
	require.Equal(t, uint64(3), gotRoots[0].LikeCount)
	require.Equal(t, uint64(2), gotRoots[1].LikeCount)
}

func TestListCommentsPage_CuratedFiltersApproved(t *testing.T) {
	db := newPagingCommentDB(t)
	ctx := context.Background()
	root := comment.Comment{VideoID: 11, UserID: 1, Content: "approved", Level: 1, Approved: true}
	require.NoError(t, db.Create(&root).Error)
	require.NoError(t, db.Create(&comment.Comment{VideoID: 11, UserID: 1, Content: "pending", Level: 1, Approved: false}).Error)
	require.NoError(t, db.Create(&comment.Comment{VideoID: 11, UserID: 2, ParentID: root.ID, Level: 2, Content: "reply-pending", Approved: false}).Error)

	p := NewCommentProvider(db)
	pg, err := p.ListCommentsPage(ctx, CommentVideo, 11, true, CommentListQuery{Page: 1, PageSize: 20, Sort: "hot"})
	require.NoError(t, err)
	require.Equal(t, int64(1), pg.Total)
	require.Len(t, pg.Rows, 1)
	require.Equal(t, "approved", pg.Rows[0].Content)
}

func TestCommentTotalPages(t *testing.T) {
	require.Equal(t, 1, commentTotalPages(0, 20))
	require.Equal(t, 1, commentTotalPages(20, 20))
	require.Equal(t, 2, commentTotalPages(21, 20))
	require.Equal(t, 3, commentTotalPages(41, 20))
}
