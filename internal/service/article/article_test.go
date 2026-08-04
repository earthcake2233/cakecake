package article

import (
	"cakecake/internal/model/article"
	"cakecake/internal/model/comment"
	"cakecake/internal/service/servicetest"
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func newArticleService(t *testing.T) (*ArticleService, *gorm.DB) {
	t.Helper()
	db := servicetest.NewDB(t)
	_, rdb := servicetest.NewRedis(t)
	return NewArticleService(db, rdb, servicetest.ZapNop(), nil), db
}

func seedArticle(t *testing.T, s *ArticleService, id, userID uint64, status string) *article.Article {
	t.Helper()
	a := &article.Article{ID: id, UserID: userID, Title: "article", Status: status}
	require.NoError(t, s.CreateArticle(context.Background(), a))
	return a
}

func TestArticleService_CRUD(t *testing.T) {
	s, _ := newArticleService(t)
	ctx := context.Background()
	a := seedArticle(t, s, 10, 1, article.StatusDraft)

	got, err := s.GetArticleByID(ctx, 10)
	require.NoError(t, err)
	require.Equal(t, "article", got.Title)

	// Not published -> GetPublishedArticle fails.
	_, err = s.GetPublishedArticle(ctx, 10)
	require.Error(t, err)

	require.NoError(t, s.Publish(ctx, 10, nil))
	pub, err := s.GetPublishedArticle(ctx, 10)
	require.NoError(t, err)
	require.Equal(t, article.StatusPublished, pub.Status)

	// Publish idempotent.
	require.NoError(t, s.Publish(ctx, 10, nil))

	// Ownership.
	owned, err := s.GetOwnedArticle(ctx, 10, 1)
	require.NoError(t, err)
	require.Equal(t, a.ID, owned.ID)
	_, err = s.GetOwnedArticle(ctx, 10, 2)
	require.Error(t, err)

	require.NoError(t, s.UpdateArticle(ctx, 10, map[string]interface{}{"title": "renamed"}))
	require.NoError(t, s.IncrementArticleView(ctx, 10))
	got, err = s.GetArticleByID(ctx, 10)
	require.NoError(t, err)
	require.Equal(t, "renamed", got.Title)
	require.Equal(t, uint64(1), got.ViewCount)
}

func TestArticleService_Engagement(t *testing.T) {
	s, db := newArticleService(t)
	ctx := context.Background()
	servicetest.SeedUser(t, db, 1, "alice")
	servicetest.SeedUser(t, db, 2, "bob")
	seedArticle(t, s, 10, 2, article.StatusPublished)
	seedArticle(t, s, 11, 2, article.StatusPublished)

	faved, count, err := s.ToggleArticleFavorite(ctx, 1, 10)
	require.NoError(t, err)
	require.True(t, faved)
	require.Equal(t, uint64(1), count)
	faved, _, err = s.ToggleArticleFavorite(ctx, 1, 10)
	require.NoError(t, err)
	require.False(t, faved)

	// Coin flow.
	exist, err := s.HasArticleCoin(ctx, 1, 10)
	require.NoError(t, err)
	require.Nil(t, exist)
	res, err := s.PostArticleCoin(ctx, 1, 10, 1)
	require.NoError(t, err)
	require.True(t, res.Coined)
	require.Equal(t, 1, res.MyCoinAmount)
	res, err = s.PostArticleCoin(ctx, 1, 10, 1)
	require.NoError(t, err)
	require.Equal(t, 2, res.MyCoinAmount)
	res, err = s.PostArticleCoin(ctx, 1, 10, 2)
	require.NoError(t, err)
	require.True(t, res.AlreadyCoined)

	// Engagement batch.
	eng := s.BatchArticleEngagementByViewer(ctx, 1, []uint64{10, 11})
	require.True(t, eng[10].CoinedByMe)
	require.False(t, eng[11].FavoritedByMe)
	require.Empty(t, s.BatchArticleEngagementByViewer(ctx, 0, []uint64{10}))
}

func TestArticleService_Lists(t *testing.T) {
	s, _ := newArticleService(t)
	ctx := context.Background()
	for i := 0; i < 3; i++ {
		seedArticle(t, s, uint64(20+i), 1, article.StatusPublished)
	}
	seedArticle(t, s, 30, 1, article.StatusDraft)
	seedArticle(t, s, 31, 2, article.StatusPublished)

	res, err := s.ListArticlesCursor(ctx, 0, 2, "")
	require.NoError(t, err)
	require.True(t, res.HasMore)
	require.Len(t, res.Items, 2)

	res, err = s.ListMyArticlesCursor(ctx, 1, 0, 10, "passed", "art", "view")
	require.NoError(t, err)
	require.Len(t, res.Items, 3)

	page, err := s.ListMyArticlesPage(ctx, 1, 1, 10, "", "", "time")
	require.NoError(t, err)
	require.Equal(t, int64(3), page.Total)

	userRes, err := s.ListUserPublishedArticlesCursor(ctx, 1, 0, 10)
	require.NoError(t, err)
	require.Len(t, userRes.Items, 3)

	counts := s.CountArticlesByStatus(ctx, 1)
	require.Equal(t, int64(3), counts["passed"])
	require.Equal(t, int64(1), counts["draft"])

	// Favorited articles.
	_, _, err = s.ToggleArticleFavorite(ctx, 1, 20)
	require.NoError(t, err)
	favs, hasMore, err := s.ListFavoritedArticlesV2(ctx, 1, 0, 10)
	require.NoError(t, err)
	require.False(t, hasMore)
	require.Len(t, favs, 1)
	n, err := s.CountFavoritedArticles(ctx, 1, true)
	require.NoError(t, err)
	require.Equal(t, int64(1), n)

	// Batch fetch.
	articles := s.BatchFetchArticles(ctx, []uint64{20, 21}, true)
	require.Len(t, articles, 2)
}

func TestArticleService_AdminAndDelete(t *testing.T) {
	s, db := newArticleService(t)
	ctx := context.Background()
	seedArticle(t, s, 10, 1, article.StatusPublished)
	seedArticle(t, s, 11, 1, article.StatusPublished)
	require.NoError(t, db.Create(&comment.ArticleComment{ArticleID: 10, UserID: 2, Content: "c", Approved: true}).Error)

	n, err := s.CountByStatus(ctx, article.StatusPublished)
	require.NoError(t, err)
	require.Equal(t, int64(2), n)

	require.NoError(t, s.AdminUpdateArticle(ctx, 11, map[string]interface{}{"status": article.StatusRejected}))
	adminRes, err := s.AdminListArticles(ctx, []string{article.StatusPublished}, "", 1, 10)
	require.NoError(t, err)
	require.Equal(t, int64(1), adminRes.Total)

	// Delete with cascade.
	require.NoError(t, s.DeleteArticle(ctx, 10))
	var cn int64
	require.NoError(t, db.Model(&comment.ArticleComment{}).Where("article_id = ?", 10).Count(&cn).Error)
	require.Zero(t, cn)
	_, err = s.GetArticleByID(ctx, 10)
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)

	// Admin delete cascade callback.
	called := false
	require.NoError(t, s.AdminDeleteArticleCascade(ctx, 11, func(tx *gorm.DB) error {
		called = true
		return nil
	}))
	require.True(t, called)
}
