package banner

import (
	"cakecake/internal/model/admin"
	"cakecake/internal/service/servicetest"
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestBannerService_CRUD(t *testing.T) {
	s := NewBannerService(servicetest.NewDB(t))
	ctx := context.Background()
	b := &admin.HomeBanner{Title: "b", ImageURL: "u", Enabled: true, SortOrder: 1}
	require.NoError(t, s.CreateBanner(ctx, b))
	require.NotZero(t, b.ID)

	got, err := s.GetBanner(ctx, b.ID)
	require.NoError(t, err)
	require.Equal(t, "b", got.Title)

	active, err := s.ListActiveBanners(ctx)
	require.NoError(t, err)
	require.Len(t, active, 1)
	all, err := s.ListBanners(ctx)
	require.NoError(t, err)
	require.Len(t, all, 1)

	require.NoError(t, s.UpdateBanner(ctx, b.ID, map[string]interface{}{"title": "b2"}))
	require.NoError(t, s.DeleteBanner(ctx, b.ID))
	_, err = s.GetBanner(ctx, b.ID)
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)
}
