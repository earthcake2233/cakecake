package service

import (
	"cakecake/internal/search"
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSearchService_Disabled(t *testing.T) {
	s := NewSearchService(nil, newAgentTestDB(t), nil, zapNop())
	ctx := context.Background()
	require.False(t, s.Enabled())
	res, err := s.SearchAll(ctx, search.SearchParams{Keyword: "x"})
	require.NoError(t, err)
	require.Nil(t, res)
	require.NoError(t, s.IndexVideoFromDB(ctx, 1))
	require.NoError(t, s.DeleteVideo(ctx, 1))
	require.NoError(t, s.IndexArticleFromDB(ctx, 1))
	require.NoError(t, s.DeleteArticle(ctx, 1))
	require.NoError(t, s.IndexUserFromDB(ctx, 1))
}

func TestSearchService_Cache(t *testing.T) {
	_, rdb := newAgentTestRedis(t)
	s := NewSearchService(nil, newAgentTestDB(t), rdb, zapNop())
	ctx := context.Background()

	require.NoError(t, s.CacheSet(ctx, "k", "v", time.Minute))
	got, err := s.CacheGet(ctx, "k")
	require.NoError(t, err)
	require.Equal(t, "v", got)

	// Nil redis no-ops.
	s2 := NewSearchService(nil, nil, nil, zapNop())
	got, err = s2.CacheGet(ctx, "k")
	require.NoError(t, err)
	require.Empty(t, got)
	require.NoError(t, s2.CacheSet(ctx, "k", "v", time.Minute))
}

func TestSearchService_EnrichUserHits(t *testing.T) {
	db := newAgentTestDB(t)
	seedUser(t, db, 1, "alice")
	s := NewSearchService(nil, db, nil, zapNop())
	ctx := context.Background()

	hits := s.EnrichUserHits(ctx, 1, []search.UserHit{})
	require.Empty(t, hits)

	// Nil db returns input unchanged.
	s2 := NewSearchService(nil, nil, nil, zapNop())
	require.Empty(t, s2.EnrichUserHits(ctx, 1, nil))
}
