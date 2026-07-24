package search

import (
	"math"
	"net/http"
	"strings"
	"testing"
)

func TestStripHTML(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"<p>Hello</p>", "Hello"},
		{`<em class="keyword">foo</em> bar`, "foo bar"},
		{"<b>bold</b> and <i>italic</i>", "bold and italic"},
		{"no tags here", "no tags here"},
		{"<div><span>nested</span></div>", "nested"},
		{"", ""},
		{"<br/>", ""},
	}
	for _, tc := range tests {
		got := stripHTML(tc.input)
		if got != tc.want {
			t.Errorf("stripHTML(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestEscapeQueryString(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"hello", "hello"},
		{"hello world", "hello world"},
		{"foo+bar", `foo\+bar`},
		{"a-b", `a\-b`},
		{"(parent)", `\(parent\)`},
		{"test:value", `test\:value`},
		{`foo^2`, `foo\^2`},
		{`"quote"`, `\"quote\"`},
		{"a~b", `a\~b`},
		{"a*b", `a\*b`},
		{"a?b", `a\?b`},
		{"a|b", `a\|b`},
		{"a&b", `a\&b`},
		{"a/b", `a\/b`},
		{"a<b", `a\<b`},
		{"a>b", `a\>b`},
		{"", ""},
	}
	for _, tc := range tests {
		got := escapeQueryString(tc.input)
		if got != tc.want {
			t.Errorf("escapeQueryString(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestValidateKeyword(t *testing.T) {
	tests := []struct {
		keyword string
		wantErr bool
	}{
		{"hello", false},
		{"", true},
		{"   ", true},
		{"a", false},
		{strings.Repeat("a", 50), false},
		{strings.Repeat("a", 51), true},
	}
	for _, tc := range tests {
		err := ValidateKeyword(tc.keyword)
		if tc.wantErr && err == nil {
			t.Errorf("ValidateKeyword(%q) expected error, got nil", tc.keyword)
		}
		if !tc.wantErr && err != nil {
			t.Errorf("ValidateKeyword(%q) unexpected error: %v", tc.keyword, err)
		}
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		sec  float64
		want string
	}{
		{0, "00:00"},
		{30, "00:30"},
		{60, "01:00"},
		{90.7, "01:31"},
		{3600, "1:00:00"},
		{3661, "1:01:01"},
		{86399, "23:59:59"},
		{-5, "00:00"},
		{3599.2, "59:59"},
	}
	for _, tc := range tests {
		got := formatDuration(tc.sec)
		if got != tc.want {
			t.Errorf("formatDuration(%v) = %q, want %q", tc.sec, got, tc.want)
		}
	}
}

func TestNormalizeVideoZoneForSearch(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"", ""},
		{"   ", ""},
		{"Animation", "Animation"},
		{"Life-Daily", "Life-Daily"},
		{"Tech→Code", "Tech-Code"},
		{"  Game  ", "Game"},
	}
	for _, tc := range tests {
		got := normalizeVideoZoneForSearch(tc.input)
		if got != tc.want {
			t.Errorf("normalizeVideoZoneForSearch(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestSplitVideoZoneForSearch(t *testing.T) {
	tests := []struct {
		zone       string
		wantParent string
		wantChild  string
	}{
		{"", "", ""},
		{"Animation", "Animation", ""},
		{"Life-Daily", "Life", "Daily"},
		{"  Music  -  Remix  ", "Music", "Remix"},
	}
	for _, tc := range tests {
		parent, child := splitVideoZoneForSearch(tc.zone)
		if parent != tc.wantParent || child != tc.wantChild {
			t.Errorf("splitVideoZoneForSearch(%q) = (%q, %q), want (%q, %q)",
				tc.zone, parent, child, tc.wantParent, tc.wantChild)
		}
	}
}

func TestSearchTypeName(t *testing.T) {
	tests := []struct {
		zone string
		want string
	}{
		{"", ""},
		{"Animation", "Animation"},
		{"Life-Daily", "LifeDaily"},
	}
	for _, tc := range tests {
		got := videoSearchTypeName(tc.zone)
		if got != tc.want {
			t.Errorf("videoSearchTypeName(%q) = %q, want %q", tc.zone, got, tc.want)
		}
	}
}

func TestFirstTagLabel(t *testing.T) {
	tests := []struct {
		tagsJSON string
		want     string
	}{
		{"", "专栏"},
		{"[]", "专栏"},
		{`["Music"]`, "Music"},
		{`["Foo", "Bar"]`, "Foo"},
		{"invalid json", "专栏"},
	}
	for _, tc := range tests {
		got := firstTagLabel(tc.tagsJSON)
		if got != tc.want {
			t.Errorf("firstTagLabel(%q) = %q, want %q", tc.tagsJSON, got, tc.want)
		}
	}
}

func TestArticleExcerpt(t *testing.T) {
	tests := []struct {
		bodyMD string
		max    int
		want   string
	}{
		{"", 10, ""},
		{"Simple text", 50, "Simple text"},
		{"# Header", 50, "Header"},
		{"**bold** text", 50, "bold text"},
		{"`code` here", 50, "code here"},
		{"> quote", 50, "quote"},
		{"[link](url)", 50, "linkurl"},
		{"Hello world", 5, "Hello…"},
		{"Hello world and more", 11, "Hello world…"},
	}
	for _, tc := range tests {
		got := articleExcerpt(tc.bodyMD, tc.max)
		if got != tc.want {
			t.Errorf("articleExcerpt(%q, %d) = %q, want %q", tc.bodyMD, tc.max, got, tc.want)
		}
	}
}

func TestTagsPlain(t *testing.T) {
	tests := []struct {
		tagsJSON string
		want     string
	}{
		{"", ""},
		{"[]", ""},
		{`["a", "b", "c"]`, "a b c"},
		{"invalid", ""},
	}
	for _, tc := range tests {
		got := tagsPlain(tc.tagsJSON)
		if got != tc.want {
			t.Errorf("tagsPlain(%q) = %q, want %q", tc.tagsJSON, got, tc.want)
		}
	}
}

func TestParseVideoFilter(t *testing.T) {
	vf := ParseVideoFilter(" click ", " lt10 ", " 动画 ")
	if vf.Order != "click" {
		t.Errorf("Order = %q, want %q", vf.Order, "click")
	}
	if vf.Duration != "lt10" {
		t.Errorf("Duration = %q, want %q", vf.Duration, "lt10")
	}
	if vf.Zone != "动画" {
		t.Errorf("Zone = %q, want %q", vf.Zone, "动画")
	}
}

func TestDurationRange(t *testing.T) {
	tests := []struct {
		duration string
		wantMin  float64
		wantMax  float64
		wantOk   bool
	}{
		{"", 0, 0, false},
		{"all", 0, 0, false},
		{"lt10", 0, 600, true},
		{"m10_30", 600, 1800, true},
		{"m30_60", 1800, 3600, true},
		{"gt60", 3600, 0, true},
		{"bogus", 0, 0, false},
	}
	for _, tc := range tests {
		vf := VideoFilter{Duration: tc.duration}
		min, max, ok := vf.durationRange()
		if min != tc.wantMin || max != tc.wantMax || ok != tc.wantOk {
			t.Errorf("durationRange(%q) = (%v, %v, %v), want (%v, %v, %v)",
				tc.duration, min, max, ok, tc.wantMin, tc.wantMax, tc.wantOk)
		}
	}
}

func TestVideoSortClause(t *testing.T) {
	for _, order := range []string{"click", "", "pubdate", "dm", "fav", "bogus"} {
		sort := videoSortClause(order)
		if len(sort) != 2 {
			t.Errorf("videoSortClause(%q) = %v, want 2 clauses", order, sort)
		}
	}
}

func TestZoneKeywordTermStructure(t *testing.T) {
	zkt := zoneKeywordTerm("zone", "动画")
	boolClause, ok := zkt["bool"].(map[string]any)
	if !ok {
		t.Fatal("expected bool clause")
	}
	should, ok := boolClause["should"].([]any)
	if !ok || len(should) < 1 {
		t.Fatal("expected should clauses")
	}
}

func TestZoneParentMatchFilterStructure(t *testing.T) {
	f := zoneParentMatchFilter("放映厅")
	boolClause, ok := f["bool"].(map[string]any)
	if !ok {
		t.Fatal("expected bool clause")
	}
	should, ok := boolClause["should"].([]any)
	if !ok || len(should) < 1 {
		t.Fatal("expected should clauses")
	}
}

func TestAppendVideoFiltersEmpty(t *testing.T) {
	filters := appendVideoFilters(nil, VideoFilter{})
	if len(filters) != 0 {
		t.Errorf("expected 0 filters, got %d", len(filters))
	}
}

func TestAppendVideoFiltersDuration(t *testing.T) {
	filters := appendVideoFilters(nil, VideoFilter{Duration: "lt10"})
	if len(filters) != 1 {
		t.Errorf("expected 1 filter, got %d", len(filters))
	}
}

func TestAppendVideoFiltersBoth(t *testing.T) {
	filters := appendVideoFilters(nil, VideoFilter{Duration: "gt60", Zone: "生活"})
	if len(filters) != 2 {
		t.Errorf("expected 2 filters, got %d", len(filters))
	}
}

func TestStripElasticsearchCompatHeaders(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		value   string
		wantVal string
	}{
		{"strip compatible-with=8", "Content-Type", "application/json; compatible-with=8", "application/json"},
		{"strip elasticsearch+json", "Accept", "application/vnd.elasticsearch+json", "application/json"},
		{"keep normal", "Content-Type", "application/json", "application/json"},
		{"keep unrelated", "Accept", "text/html", "text/html"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := http.Header{}
			h.Set(tc.key, tc.value)
			stripElasticsearchCompatHeaders(h)
			got := h.Get(tc.key)
			if got != tc.wantVal {
				t.Errorf("header %q = %q, want %q", tc.key, got, tc.wantVal)
			}
		})
	}
}

func TestArticleExcerptMaxRunes(t *testing.T) {
	s := "Hello, this is a test with more than 20 runes here"
	got := articleExcerpt(s, 20)
	rs := []rune(got)
	if len(rs) > 23 {
		t.Errorf("excerpt length %d > 23", len(rs))
	}
}

func TestFormatDurationNaN(t *testing.T) {
	got := formatDuration(math.NaN())
	if got == "" {
		t.Error("expected non-empty duration for NaN")
	}
}

func TestFirstTagLabelEmptyStrings(t *testing.T) {
	got := firstTagLabel(`["  ", "  "]`)
	if got != "专栏" {
		t.Errorf("empty strings = %q", got)
	}
}

func TestFirstTagLabelFirstNonEmpty(t *testing.T) {
	got := firstTagLabel(`["", "Music"]`)
	if got != "Music" {
		t.Errorf("expected Music, got %q", got)
	}
}
