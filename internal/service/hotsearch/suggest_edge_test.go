package hotsearch

import (
	"cakecake/internal/model/admin"
	"cakecake/internal/model/extra"
	"context"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// ---------- SearchSuggest ----------

func TestSearchSuggest_EmptyDB(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&extra.UserSearchHistory{}, &admin.HotSearchOp{}))

	ctx := context.Background()
	results := SearchSuggest(ctx, db, nil, 0, "", 10)
	require.NotNil(t, results)
	require.Empty(t, results)
}

func TestSearchSuggest_WithUserHistory(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&extra.UserSearchHistory{}, &admin.HotSearchOp{}))

	// Add user search history
	h := extra.UserSearchHistory{
		UserID:  1,
		Keyword: "golang testing",
	}
	require.NoError(t, db.Create(&h).Error)

	ctx := context.Background()
	results := SearchSuggest(ctx, db, nil, 1, "golang", 10)
	require.NotNil(t, results)
	require.GreaterOrEqual(t, len(results), 1)
	require.Contains(t, results[0].Name, "golang")
	require.Contains(t, results[0].Value, "golang")
}

func TestSearchSuggest_TermTooLong(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&extra.UserSearchHistory{}, &admin.HotSearchOp{}))

	ctx := context.Background()
	longTerm := string(make([]rune, 60))
	results := SearchSuggest(ctx, db, nil, 0, longTerm, 10)
	require.NotNil(t, results)
	require.Empty(t, results)
}

func TestSearchSuggest_LimitBounds(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&extra.UserSearchHistory{}, &admin.HotSearchOp{}))

	ctx := context.Background()

	// Zero limit should become 10
	results := SearchSuggest(ctx, db, nil, 0, "", 0)
	require.NotNil(t, results)
	require.LessOrEqual(t, len(results), 10)

	// Large limit should be capped at 20
	results = SearchSuggest(ctx, db, nil, 0, "", 100)
	require.NotNil(t, results)
	require.LessOrEqual(t, len(results), 20)
}

// ---------- ValidateSuggestTerm ----------

func TestValidateSuggestTerm_Edge(t *testing.T) {
	require.True(t, ValidateSuggestTerm("short"))
	require.True(t, ValidateSuggestTerm(""))
	require.True(t, ValidateSuggestTerm("  "))
	require.True(t, ValidateSuggestTerm("a"))
	term50 := string(make([]rune, 50))
	require.True(t, ValidateSuggestTerm(term50))
	term51 := string(make([]rune, 51))
	require.False(t, ValidateSuggestTerm(term51))
}

// ---------- highlightSuggestKeyword ----------

func TestHighlightSuggestKeyword_Edge(t *testing.T) {
	require.Empty(t, highlightSuggestKeyword("", "x"))
	require.Empty(t, highlightSuggestKeyword("  ", "x"))
	require.Equal(t, "hello", highlightSuggestKeyword("hello", ""))
	require.Equal(t, "<em class=\"suggest_high_light\">he</em>llo", highlightSuggestKeyword("hello", "he"))
	require.Equal(t, "h<em class=\"suggest_high_light\">el</em>lo", highlightSuggestKeyword("hello", "el"))
	require.Equal(t, "hel<em class=\"suggest_high_light\">lo</em>", highlightSuggestKeyword("hello", "lo"))
	// Case insensitive
	require.Equal(t, "<em class=\"suggest_high_light\">HE</em>LLO", highlightSuggestKeyword("HELLO", "he"))
}

// ---------- escapeHTML ----------

func TestEscapeHTML_Edge(t *testing.T) {
	require.Equal(t, "a&amp;b", escapeHTML("a&b"))
	require.Equal(t, "&lt;tag&gt;", escapeHTML("<tag>"))
	require.Equal(t, "&quot;quote&quot;", escapeHTML(`"quote"`))
	require.Equal(t, "no change", escapeHTML("no change"))
	require.Empty(t, escapeHTML(""))
}
