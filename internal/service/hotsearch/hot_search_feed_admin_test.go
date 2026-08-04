package hotsearch

import (
	"cakecake/internal/model/admin"
	"cakecake/internal/service/servicetest"
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestListHotSearchMerged(t *testing.T) {
	s, rec, db := newHotSearchService(t)
	ctx := context.Background()

	// Inactive op is skipped.
	start := time.Now().Add(time.Hour)
	require.NoError(t, s.CreateOp(ctx, &admin.HotSearchOp{
		OpType: "pin", Keyword: "future", PinRank: 1, Enabled: true, StartAt: &start,
	}))
	// Blocked keyword.
	require.NoError(t, s.CreateOp(ctx, &admin.HotSearchOp{OpType: "block", Keyword: "bad", Enabled: true}))
	// Manual op.
	require.NoError(t, s.CreateOp(ctx, &admin.HotSearchOp{OpType: "manual", Keyword: "golang", DisplayTitle: "Go", Badge: "热", PinRank: 1, Enabled: true}))
	// Redis auto rank.
	require.NoError(t, rec.Record(ctx, 1, "", "rust"))
	require.NoError(t, rec.Record(ctx, 2, "", "rust"))

	items, err := ListHotSearchMerged(ctx, db, rec, 10)
	require.NoError(t, err)
	require.NotEmpty(t, items)
	require.Equal(t, "Go", items[0].Title)

	// Detail variant includes source annotations.
	details, err := ListHotSearchMergedDetail(ctx, db, rec, 10)
	require.NoError(t, err)
	require.NotEmpty(t, details)

	// Ops+Redis merge path directly.
	legacy, err := listHotSearchMergedDetailFromOps(ctx, db, rec, 0)
	require.NoError(t, err)
	require.NotEmpty(t, legacy)

	// Flag map.
	flags := ActiveHotSearchOpFlags(db)
	require.True(t, flags["golang"].Manual)
	require.True(t, flags["bad"].Blocked)
	require.Empty(t, ActiveHotSearchOpFlags(nil))
}

func TestListHotSearchMergedWithLayout(t *testing.T) {
	s, rec, db := newHotSearchService(t)
	ctx := context.Background()
	require.NoError(t, s.CreateOp(ctx, &admin.HotSearchOp{OpType: "manual", Keyword: "golang", PinRank: 1, Enabled: true}))
	require.NoError(t, s.SaveDisplayLayout(ctx, []HotSearchLayoutEntry{{Keyword: "golang", Title: "Go"}}))
	require.NoError(t, rec.Record(ctx, 1, "", "golang"))

	items, err := ListHotSearchMerged(ctx, db, rec, 10)
	require.NoError(t, err)
	require.Equal(t, "golang", items[0].Title)
}

func TestSearchSuggest_WithHistoryAndOps(t *testing.T) {
	s, rec, db := newHotSearchService(t)
	ctx := context.Background()
	servicetest.SeedUser(t, db, 1, "alice")
	require.NoError(t, s.CreateOp(ctx, &admin.HotSearchOp{OpType: "pin", Keyword: "golang", DisplayTitle: "Go 语言", PinRank: 1, Enabled: true}))
	require.NoError(t, db.Exec(
		"INSERT INTO user_search_histories (user_id, keyword, updated_at, keyword_norm) VALUES (1, 'rust', ?, 'rust')",
		time.Now(),
	).Error)
	require.NoError(t, rec.Record(ctx, 1, "", "spring"))

	tags := s.SearchSuggest(ctx, 1, "go", 10)
	require.NotEmpty(t, tags)
	tags = s.SearchSuggest(ctx, 0, "spring", 5)
	require.NotEmpty(t, tags)
	// No matching candidates falls back to the merged hot-search list.
	require.NotEmpty(t, s.SearchSuggest(ctx, 0, "zzz-no-match", 10))
}
