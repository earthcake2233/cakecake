package service

import (
	"cakecake/internal/model/extra"
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func newSearchHistoryService(t *testing.T) *SearchHistoryService {
	t.Helper()
	return NewSearchHistoryService(newAgentTestDB(t), zapNop())
}

func TestSearchHistory_UpsertAndList(t *testing.T) {
	s := newSearchHistoryService(t)
	ctx := context.Background()

	// Blank keyword is a no-op.
	require.NoError(t, s.UpsertKeyword(ctx, 1, "   ", time.Now()))

	at := time.Now()
	require.NoError(t, s.UpsertKeyword(ctx, 1, "Golang", at))
	require.NoError(t, s.UpsertKeyword(ctx, 1, "golang", at.Add(time.Minute))) // dedup by norm, update

	// Duplicate normalized rows get collapsed to the first.
	require.NoError(t, s.db.Create(&extra.UserSearchHistory{
		UserID: 1, Keyword: "Rust", KeywordNorm: "rust", UpdatedAt: at,
	}).Error)
	require.NoError(t, s.UpsertKeyword(ctx, 1, "rust", at.Add(2*time.Minute)))
	var n int64
	require.NoError(t, s.db.Model(&extra.UserSearchHistory{}).Where("keyword_norm = ?", "rust").Count(&n).Error)
	require.Equal(t, int64(1), n)

	kws, err := s.ListKeywords(ctx, 1)
	require.NoError(t, err)
	require.Contains(t, kws, "golang")
	require.Contains(t, kws, "rust")
}

func TestSearchHistory_Trim(t *testing.T) {
	s := newSearchHistoryService(t)
	ctx := context.Background()
	at := time.Now()
	for i := 0; i < maxUserSearchHistory+5; i++ {
		require.NoError(t, s.UpsertKeyword(ctx, 1, string(rune('a'+i%26))+itoa(uint64(i)), at.Add(time.Duration(i)*time.Second)))
	}
	require.NoError(t, s.TrimHistory(ctx, 1))
	var n int64
	require.NoError(t, s.db.Model(&extra.UserSearchHistory{}).Where("user_id = ?", 1).Count(&n).Error)
	require.Equal(t, int64(maxUserSearchHistory), n)
}

func TestSearchHistory_Replace(t *testing.T) {
	s := newSearchHistoryService(t)
	ctx := context.Background()

	require.NoError(t, s.UpsertKeyword(ctx, 1, "old", time.Now()))
	require.NoError(t, s.ReplaceHistory(ctx, 1, []string{"new1", "", "new2"}))

	kws, err := s.ListKeywords(ctx, 1)
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"new1", "new2"}, kws)
}
