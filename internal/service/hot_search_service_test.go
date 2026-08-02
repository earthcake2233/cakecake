package service

import (
	"cakecake/internal/model/admin"
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func newHotSearchService(t *testing.T) (*HotSearchService, *SearchHotRecorder) {
	t.Helper()
	db := newAgentTestDB(t)
	mr, rdb := newAgentTestRedis(t)
	_ = mr
	rec := &SearchHotRecorder{Rdb: rdb}
	return NewHotSearchService(db, rec), rec
}

func TestHotSearchService_OpCRUD(t *testing.T) {
	s, _ := newHotSearchService(t)
	ctx := context.Background()

	// Empty list.
	rows, err := s.ListOps(ctx)
	require.NoError(t, err)
	require.Empty(t, rows)

	op := &admin.HotSearchOp{OpType: "pin", Keyword: "golang", DisplayTitle: "Go", PinRank: 1, Enabled: true}
	require.NoError(t, s.CreateOp(ctx, op))
	require.NotZero(t, op.ID)

	got, err := s.GetOp(ctx, op.ID)
	require.NoError(t, err)
	require.Equal(t, "golang", got.Keyword)
	_, err = s.GetOp(ctx, 999)
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)

	require.NoError(t, s.UpdateOp(ctx, op.ID, map[string]any{"badge": "热"}))
	got, err = s.GetOp(ctx, op.ID)
	require.NoError(t, err)
	require.Equal(t, "热", got.Badge)

	all, err := s.FindAllOps(ctx)
	require.NoError(t, err)
	require.Len(t, all, 1)

	rows, err = s.ListOps(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 1)

	require.NoError(t, s.DeleteOp(ctx, op.ID))
	_, err = s.GetOp(ctx, op.ID)
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)
}

func TestHotSearchService_QuickOp(t *testing.T) {
	s, _ := newHotSearchService(t)
	ctx := context.Background()

	// Create new.
	op, created, err := s.QuickOpCreateOrUpdate(ctx, "pin", "  golang ", "", "热", 2)
	require.NoError(t, err)
	require.True(t, created)
	require.Equal(t, "golang", op.Keyword)
	require.Equal(t, "golang", op.DisplayTitle)

	// Update existing (normalized keyword match).
	op2, created, err := s.QuickOpCreateOrUpdate(ctx, "manual", "GoLang", "Go 语言", "新", 5)
	require.NoError(t, err)
	require.False(t, created)
	require.Equal(t, op.ID, op2.ID)
	require.Equal(t, "Go 语言", op2.DisplayTitle)

	// Block op clears rank/badge.
	op3, created, err := s.QuickOpCreateOrUpdate(ctx, "block", "rust", "Rust", "热", 9)
	require.NoError(t, err)
	require.True(t, created)
	require.Zero(t, op3.PinRank)
	require.Empty(t, op3.Badge)
}

func TestHotSearchService_ReorderItems(t *testing.T) {
	s, _ := newHotSearchService(t)
	ctx := context.Background()

	manual := &admin.HotSearchOp{OpType: "manual", Keyword: "a", PinRank: 1, Enabled: true}
	require.NoError(t, s.CreateOp(ctx, manual))

	require.NoError(t, s.ReorderItems(ctx, []ReorderItem{
		{Keyword: "a", Title: "A", OpID: manual.ID},
		{Keyword: "b", Title: "B"},
		{Keyword: "", Title: ""},
	}))

	var layout admin.HotSearchDisplayLayout
	require.NoError(t, s.db.First(&layout, 1).Error)
	require.Contains(t, layout.OrderJSON, "b")
	got, err := s.GetOp(ctx, manual.ID)
	require.NoError(t, err)
	require.Equal(t, 1, got.PinRank)

	// Layout helpers.
	require.True(t, s.HasDisplayLayout(ctx))
	require.NoError(t, s.ClearDisplayLayout(ctx))
	require.False(t, s.HasDisplayLayout(ctx))
	require.NoError(t, s.SaveDisplayLayout(ctx, []HotSearchLayoutEntry{{Keyword: "x", Title: "X"}}))
	require.NoError(t, s.ApplyLayoutMove(ctx, "y", "Y", 1))
	require.NoError(t, s.RemoveLayoutEntry(ctx, "y"))
}

func TestHotSearchService_RedisPassthrough(t *testing.T) {
	s, rec := newHotSearchService(t)
	ctx := context.Background()

	require.NoError(t, rec.BoostKeyword(ctx, "golang", 3))
	require.NoError(t, rec.Record(ctx, 1, "", "golang"))

	rows, err := s.TopWithScores(ctx, 10)
	require.NoError(t, err)
	require.NotEmpty(t, rows)

	require.NoError(t, s.BoostKeyword(ctx, "golang", 5))
	require.NoError(t, s.RemoveKeywordFromRedis(ctx, "golang"))
	rows, err = s.TopWithScores(ctx, 10)
	require.NoError(t, err)
	require.Empty(t, rows)

	// Nil recorder short-circuits.
	s2 := NewHotSearchService(newAgentTestDB(t), nil)
	rows, err = s2.TopWithScores(ctx, 10)
	require.NoError(t, err)
	require.Empty(t, rows)
	require.NoError(t, s2.RemoveKeywordFromRedis(ctx, "x"))
	require.NoError(t, s2.BoostKeyword(ctx, "x", 1))
}

func TestHotSearchService_ValidateAndNormalize(t *testing.T) {
	s, _ := newHotSearchService(t)
	require.True(t, s.ValidateSuggestTerm("hello"))
	require.False(t, s.ValidateSuggestTerm(makeString(51, 'a')))
	require.Equal(t, "golang", s.NormalizeKeyword("  GoLang "))
}

func makeString(n int, r rune) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = r
	}
	return string(b)
}
