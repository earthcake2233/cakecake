package service

import (
	"cakecake/internal/model/comment"
	"cakecake/internal/model/dynamic"
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func newDynamicService(t *testing.T) *DynamicService {
	t.Helper()
	db := newAgentTestDB(t)
	_, rdb := newAgentTestRedis(t)
	return NewDynamicService(db, rdb, zapNop())
}

func TestDynamicService_CRUD(t *testing.T) {
	s := newDynamicService(t)
	ctx := context.Background()

	d := &dynamic.UserDynamic{UserID: 1, Title: "my dynamic", Content: "hello"}
	require.NoError(t, s.CreateDynamic(ctx, d))
	require.NotZero(t, d.ID)

	got, err := s.GetDynamicByID(ctx, d.ID)
	require.NoError(t, err)
	require.Equal(t, "my dynamic", got.Title)
	_, err = s.GetDynamicByID(ctx, 999)
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)

	require.NoError(t, s.UpdateDynamic(ctx, d.ID, map[string]interface{}{"title": "renamed"}))
	got, err = s.GetDynamicByID(ctx, d.ID)
	require.NoError(t, err)
	require.Equal(t, "renamed", got.Title)

	// Cascade delete removes likes and comments.
	require.NoError(t, s.db.Create(&comment.UserDynamicLike{UserID: 2, DynamicID: d.ID}).Error)
	cm := comment.DynamicComment{DynamicID: d.ID, UserID: 2, Content: "c1"}
	require.NoError(t, s.db.Create(&cm).Error)
	require.NoError(t, s.db.Create(&comment.DynamicCommentLike{UserID: 3, CommentID: cm.ID}).Error)
	require.NoError(t, s.db.Create(&comment.DynamicCommentDislike{UserID: 4, CommentID: cm.ID}).Error)

	require.NoError(t, s.DeleteDynamic(ctx, d.ID))
	var n int64
	require.NoError(t, s.db.Model(&dynamic.UserDynamic{}).Count(&n).Error)
	require.Zero(t, n)
	require.NoError(t, s.db.Model(&comment.UserDynamicLike{}).Count(&n).Error)
	require.Zero(t, n)
	require.NoError(t, s.db.Model(&comment.DynamicComment{}).Count(&n).Error)
	require.Zero(t, n)
	require.NoError(t, s.db.Model(&comment.DynamicCommentLike{}).Count(&n).Error)
	require.Zero(t, n)
}

func TestDynamicService_Lists(t *testing.T) {
	s := newDynamicService(t)
	ctx := context.Background()
	for i := 0; i < 3; i++ {
		require.NoError(t, s.CreateDynamic(ctx, &dynamic.UserDynamic{UserID: 1, Title: "d", Content: "x"}))
	}
	require.NoError(t, s.CreateDynamic(ctx, &dynamic.UserDynamic{UserID: 2, Title: "other", Content: "x"}))

	n, err := s.CountUserDynamics(ctx, 1)
	require.NoError(t, err)
	require.Equal(t, int64(3), n)

	list, total, err := s.ListUserDynamicsPaginated(ctx, 1, 1, 2, "")
	require.NoError(t, err)
	require.Equal(t, int64(3), total)
	require.Len(t, list, 2)

	list, total, err = s.ListDynamics(ctx, 1, 1, 10)
	require.NoError(t, err)
	require.Len(t, list, 3)

	rows, err := s.ListUserDynamicsCursor(ctx, 1, 0, 2)
	require.NoError(t, err)
	require.Len(t, rows, 2)
	rows, err = s.ListUserDynamicsCursor(ctx, 1, rows[0].ID, 10)
	require.NoError(t, err)
	require.NotEmpty(t, rows)
}

func TestDynamicService_Like(t *testing.T) {
	s := newDynamicService(t)
	ctx := context.Background()
	d := &dynamic.UserDynamic{UserID: 1, Title: "d"}
	require.NoError(t, s.CreateDynamic(ctx, d))

	liked, err := s.ToggleDynamicLike(ctx, 2, d.ID)
	require.NoError(t, err)
	require.True(t, liked)
	var got dynamic.UserDynamic
	require.NoError(t, s.db.First(&got, d.ID).Error)
	require.Equal(t, uint64(1), got.LikeCount)

	liked, err = s.ToggleDynamicLike(ctx, 2, d.ID)
	require.NoError(t, err)
	require.False(t, liked)
	require.NoError(t, s.db.First(&got, d.ID).Error)
	require.Zero(t, got.LikeCount)

	// BatchCheckLiked.
	_, err = s.ToggleDynamicLike(ctx, 3, d.ID)
	require.NoError(t, err)
	require.Equal(t, map[uint64]bool{d.ID: true}, s.BatchCheckLiked(ctx, 3, []uint64{d.ID}))
	require.Empty(t, s.BatchCheckLiked(ctx, 0, []uint64{d.ID}))
	require.Empty(t, s.BatchCheckLiked(ctx, 3, nil))
}

func TestDynamicService_AdvancedAndAdmin(t *testing.T) {
	s := newDynamicService(t)
	ctx := context.Background()
	for i := 0; i < 3; i++ {
		require.NoError(t, s.CreateDynamic(ctx, &dynamic.UserDynamic{UserID: 1, Title: "t", Content: "c"}))
	}

	res, err := s.ListMyDynamicsAdvanced(ctx, MyDynamicFilter{UserID: 1, TitleQ: "t", Page: 1, PageSize: 2, SortKey: "like"})
	require.NoError(t, err)
	require.Equal(t, int64(3), res.Total)
	require.Equal(t, 2, res.TotalPages)
	require.Len(t, res.Dynamics, 2)
	// Reply and default sort orders.
	res, err = s.ListMyDynamicsAdvanced(ctx, MyDynamicFilter{UserID: 1, Page: 1, PageSize: 10, SortKey: "reply"})
	require.NoError(t, err)
	require.Equal(t, int64(3), res.Total)
	res, err = s.ListMyDynamicsAdvanced(ctx, MyDynamicFilter{UserID: 1, Page: 1, PageSize: 10})
	require.NoError(t, err)
	require.Equal(t, int64(3), res.Total)
	// Page beyond last page clamps to last page.
	res, err = s.ListMyDynamicsAdvanced(ctx, MyDynamicFilter{UserID: 1, Page: 99, PageSize: 2})
	require.NoError(t, err)
	require.Equal(t, 2, res.TotalPages)

	adminRes, err := s.AdminListDynamics(ctx, "c", 1, 10)
	require.NoError(t, err)
	require.Equal(t, int64(3), adminRes.Total)

	called := false
	require.NoError(t, s.AdminDeleteDynamicCascade(ctx, 1, func(tx *gorm.DB) error {
		called = true
		return nil
	}))
	require.True(t, called)
}
