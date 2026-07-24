package search

import (
	"net/http"
	"strings"
	"testing"
)

func TestStripHTML_edge(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"<p>  spaced </p>", "spaced"},
		{"<div>a<br/>b</div>", "ab"},
		{"<script>alert(1)</script>", "alert(1)"},
		{"<img src=\"x\"/>", ""},
		{"a < b > c", "a  c"},
		{"   ", ""},
		{"<a><b><c>deep</c></b></a>", "deep"},
		{"line1\nline2", "line1\nline2"},
		{"<p>multi\nline</p>", "multi\nline"},
	}
	for _, tc := range tests {
		got := stripHTML(tc.input)
		if got != tc.want {
			t.Errorf("stripHTML(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestEscapeQueryString_edge(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"a+b=c", "a\\+b=c"},
		{"[bracket]", "\\[bracket\\]"},
		{"{braces}", "\\{braces\\}"},
		{"term~0.5", "term\\~0.5"},
		{"a||b", "a\\|\\|b"},
		{"&&", "\\&\\&"},
		{"!important", "\\!important"},
		{"(nested (parens))", "\\(nested \\(parens\\)\\)"},
		{"mixed!specials+here", "mixed\\!specials\\+here"},
		{"back\\slash", "back\\\\slash"},
	}
	for _, tc := range tests {
		got := escapeQueryString(tc.input)
		if got != tc.want {
			t.Errorf("escapeQueryString(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestValidateKeyword_edge(t *testing.T) {
	tests := []struct {
		keyword string
		wantErr bool
		errMsg  string
	}{
		{"  hello  ", false, ""},
		{"\t\n", true, "empty"},
		{"a", false, ""},
		{strings.Repeat("a", 50), false, ""},
		{strings.Repeat("a", 51), true, "too long"},
		{strings.Repeat("测", 50), false, ""},
	}
	for _, tc := range tests {
		err := ValidateKeyword(tc.keyword)
		if tc.wantErr && err == nil {
			t.Errorf("ValidateKeyword(%q) expected error", tc.keyword)
		}
		if !tc.wantErr && err != nil {
			t.Errorf("ValidateKeyword(%q) unexpected error: %v", tc.keyword, err)
		}
		if tc.wantErr && err != nil && tc.errMsg != "" && !strings.Contains(err.Error(), tc.errMsg) {
			t.Errorf("ValidateKeyword(%q) error = %q, want substring %q", tc.keyword, err.Error(), tc.errMsg)
		}
	}
}

func TestFormatDuration_edge(t *testing.T) {
	tests := []struct {
		sec  float64
		want string
	}{
		{0.4, "00:00"},
		{0.5, "00:01"},
		{1.0, "00:01"},
		{59.4, "00:59"},
		{59.5, "01:00"},
		{3599.4, "59:59"},
		{3599.5, "1:00:00"},
		{3600, "1:00:00"},
		{3661.7, "1:01:02"},
		{7200, "2:00:00"},
		{86400, "24:00:00"},
		{999999, "277:46:39"},
	}
	for _, tc := range tests {
		got := formatDuration(tc.sec)
		if got != tc.want {
			t.Errorf("formatDuration(%v) = %q, want %q", tc.sec, got, tc.want)
		}
	}
}

func TestNormalizeVideoZoneForSearch_edge(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"  ", ""},
		{"Animation", "Animation"},
		// → (U+2192) replaces → and →→ same
		{"Tech→Code", "Tech-Code"},
		{"Life→Daily", "Life-Daily"},
		{"→leading", "-leading"},
		{"trailing→", "trailing-"},
		{"multi→→arrow", "multi--arrow"},
		// — (U+2014 EM DASH) replaces — alone (keeps surrounding spaces)
		{"A — B", "A - B"},
		// Space+→+Space gets replaced with single dash
		{"A → B", "A-B"},
	}
	for _, tc := range tests {
		got := normalizeVideoZoneForSearch(tc.input)
		if got != tc.want {
			t.Errorf("normalizeVideoZoneForSearch(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestSplitVideoZoneForSearch_edge(t *testing.T) {
	tests := []struct {
		zone       string
		wantParent string
		wantChild  string
	}{
		{"  ", "", ""},
		{"Animation", "Animation", ""},
		{"Music-Remix", "Music", "Remix"},
		{"Tech-Code", "Tech", "Code"},
		{"→foo", "-foo", ""},
		{"a-", "a", ""},
		{"-only", "-only", ""},
	}
	for _, tc := range tests {
		parent, child := splitVideoZoneForSearch(tc.zone)
		if parent != tc.wantParent || child != tc.wantChild {
			t.Errorf("splitVideoZoneForSearch(%q) = (%q, %q), want (%q, %q)",
				tc.zone, parent, child, tc.wantParent, tc.wantChild)
		}
	}
}

func TestVideoSearchTypeName_edge(t *testing.T) {
	tests := []struct {
		zone string
		want string
	}{
		{"  ", ""},
		{"Music", "Music"},
		{"Music-Remix", "MusicRemix"},
		{"→foo", "-foo"},
	}
	for _, tc := range tests {
		got := videoSearchTypeName(tc.zone)
		if got != tc.want {
			t.Errorf("videoSearchTypeName(%q) = %q, want %q", tc.zone, got, tc.want)
		}
	}
}

func TestFirstTagLabel_edge(t *testing.T) {
	tests := []struct {
		tagsJSON string
		want     string
	}{
		{"  ", "专栏"},
		{"  []  ", "专栏"},
		{"  [\"Music\"]  ", "Music"},
		{"[\"\", \"First\", \"Second\"]", "First"},
		{"[\"  \", \"  \"]", "专栏"},
		{"[\"Music\", \"Gaming\"]", "Music"},
		{"garbage json", "专栏"},
		{"null", "专栏"},
	}
	for _, tc := range tests {
		got := firstTagLabel(tc.tagsJSON)
		if got != tc.want {
			t.Errorf("firstTagLabel(%q) = %q, want %q", tc.tagsJSON, got, tc.want)
		}
	}
}

func TestArticleExcerpt_edge(t *testing.T) {
	tests := []struct {
		bodyMD string
		max    int
		want   string
	}{
		{"  ", 10, ""},
		{"  Some text  ", 50, "Some text"},
		{"### H3 Title", 50, "H3 Title"},
		{"**bold** and *italic*", 50, "bold and italic"},
		{"`inline code` here", 50, "inline code here"},
		{"> blockquote", 50, "blockquote"},
		{"[text](url)", 50, "texturl"},
		{"[text](url) and ![img](img.jpg)", 50, "texturl and !imgimg.jpg"},
		{"  multiple   spaces   ", 50, "multiple spaces"},
		{"Hello 世界", 6, "Hello …"},
		{"Hello 世界", 7, "Hello 世…"},
		{"Hello 世界", 8, "Hello 世界"},
		{"abcde", 3, "abc…"},
		{"abcde", 5, "abcde"},
		{"abcde", 6, "abcde"},
		{"line1\nline2\nline3", 50, "line1 line2 line3"},
	}
	for _, tc := range tests {
		got := articleExcerpt(tc.bodyMD, tc.max)
		if got != tc.want {
			t.Errorf("articleExcerpt(%q, %d) = %q, want %q", tc.bodyMD, tc.max, got, tc.want)
		}
	}
}

func TestTagsPlain_edge(t *testing.T) {
	tests := []struct {
		tagsJSON string
		want     string
	}{
		{"  ", ""},
		{"  []  ", ""},
		{"[\"a\"]", "a"},
		{"[\"foo\", \"bar\", \"baz\"]", "foo bar baz"},
		{"null", ""},
		{"garbage", ""},
		{"[\"single\"]", "single"},
	}
	for _, tc := range tests {
		got := tagsPlain(tc.tagsJSON)
		if got != tc.want {
			t.Errorf("tagsPlain(%q) = %q, want %q", tc.tagsJSON, got, tc.want)
		}
	}
}

func TestParseVideoFilter_edge(t *testing.T) {
	tests := []struct {
		order    string
		duration string
		zone     string
		want     VideoFilter
	}{
		{" click ", " lt10 ", " 动画 ", VideoFilter{Order: "click", Duration: "lt10", Zone: "动画"}},
		{"", "", "", VideoFilter{Order: "", Duration: "", Zone: ""}},
		{"  ", "  ", "  ", VideoFilter{Order: "", Duration: "", Zone: ""}},
		{"pubdate", "gt60", "生活", VideoFilter{Order: "pubdate", Duration: "gt60", Zone: "生活"}},
	}
	for _, tc := range tests {
		got := ParseVideoFilter(tc.order, tc.duration, tc.zone)
		if got != tc.want {
			t.Errorf("ParseVideoFilter(%q,%q,%q) = %#v, want %#v", tc.order, tc.duration, tc.zone, got, tc.want)
		}
	}
}

func TestDurationRange_ChineseLabels(t *testing.T) {
	tests := []struct {
		duration string
		wantMin  float64
		wantMax  float64
		wantOk   bool
	}{
		{"全部时长", 0, 0, false},
		{"10分钟以下", 0, 600, true},
		{"10-30分钟", 600, 1800, true},
		{"30-60分钟", 1800, 3600, true},
		{"60分钟以上", 3600, 0, true},
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

func TestVideoSortClause_values(t *testing.T) {
	orders := []string{"click", "pubdate", "dm", "fav", "default", "bogus", ""}
	for _, order := range orders {
		sort := videoSortClause(order)
		if len(sort) != 2 {
			t.Errorf("videoSortClause(%q) len = %d, want 2", order, len(sort))
		}
		for i, s := range sort {
			m, ok := s.(map[string]any)
			if !ok {
				t.Errorf("videoSortClause(%q)[%d] is not map", order, i)
				continue
			}
			for _, v := range m {
				vm, ok2 := v.(map[string]any)
				if !ok2 {
					t.Errorf("videoSortClause(%q)[%d] inner not map", order, i)
				} else if vm["order"] != "desc" {
					t.Errorf("videoSortClause(%q)[%d] order=%v", order, i, vm["order"])
				}
			}
		}
	}
}

func TestVideoSortClause_ChineseLabels(t *testing.T) {
	orders := []string{"最多点击", "最新发布", "最多弹幕", "最多收藏"}
	for _, order := range orders {
		sort := videoSortClause(order)
		if len(sort) != 2 {
			t.Errorf("videoSortClause(%q) len = %d, want 2", order, len(sort))
		}
	}
}

func TestZoneKeywordTerm_values(t *testing.T) {
	zkt := zoneKeywordTerm("zone", "动画")
	boolClause, ok := zkt["bool"].(map[string]any)
	if !ok {
		t.Fatal("expected bool clause")
	}
	should, ok := boolClause["should"].([]any)
	if !ok || len(should) != 2 {
		t.Fatalf("expected 2 should, got %d", len(should))
	}
	term1, ok := should[0].(map[string]any)["term"].(map[string]any)
	if !ok || term1["zone"] != "动画" {
		t.Fatalf("first should = %#v", should[0])
	}
	term2, ok := should[1].(map[string]any)["term"].(map[string]any)
	if !ok || term2["zone.keyword"] != "动画" {
		t.Fatalf("second should = %#v", should[1])
	}
	zkt2 := zoneKeywordTerm("zone.keyword", "music")
	boolClause2, ok := zkt2["bool"].(map[string]any)
	if !ok {
		t.Fatal("expected bool clause")
	}
	should2, ok := boolClause2["should"].([]any)
	if !ok || len(should2) != 1 {
		t.Fatalf("expected 1 should for zone.keyword, got %d", len(should2))
	}
}

func TestZoneParentMatchFilter_values(t *testing.T) {
	f := zoneParentMatchFilter("放映厅")
	boolClause, ok := f["bool"].(map[string]any)
	if !ok {
		t.Fatal("expected bool clause")
	}
	should, ok := boolClause["should"].([]any)
	if !ok || len(should) != 3 {
		t.Fatalf("expected 3 should, got %d", len(should))
	}
	prefixClause, ok := should[2].(map[string]any)["bool"].(map[string]any)
	if !ok {
		t.Fatalf("expected bool for prefix, got %#v", should[2])
	}
	prefixShould, ok := prefixClause["should"].([]any)
	if !ok || len(prefixShould) != 2 {
		t.Fatalf("expected 2 prefix shoulds, got %d", len(prefixShould))
	}
}

func TestAppendVideoFilters_zoneSpecific(t *testing.T) {
	tests := []struct {
		name   string
		filter VideoFilter
		minLen int
	}{
		{"duration only", VideoFilter{Duration: "m10_30"}, 1},
		{"zone full match (纪录片)", VideoFilter{Zone: "纪录片"}, 1},
		{"zone parent match (动画)", VideoFilter{Zone: "动画"}, 1},
		{"both lt10 + 动画", VideoFilter{Duration: "lt10", Zone: "动画"}, 2},
		{"gt60 with zone", VideoFilter{Duration: "gt60", Zone: "生活"}, 2},
		{"empty zone", VideoFilter{Zone: ""}, 0},
		{"all partition", VideoFilter{Zone: "全部分区"}, 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			filters := appendVideoFilters(nil, tc.filter)
			if len(filters) < tc.minLen {
				t.Errorf("expected >= %d filters, got %d: %#v", tc.minLen, len(filters), filters)
			}
		})
	}
}

func TestAppendVideoFilters_preserveExisting(t *testing.T) {
	existing := []any{
		map[string]any{"term": map[string]any{"status": "published"}},
	}
	filters := appendVideoFilters(existing, VideoFilter{Duration: "lt10"})
	if len(filters) != 2 {
		t.Fatalf("expected 2 filters, got %d", len(filters))
	}
	term, ok := filters[0].(map[string]any)["term"].(map[string]any)
	if !ok || term["status"] != "published" {
		t.Errorf("first filter not preserved: %#v", filters[0])
	}
}

func TestStripElasticsearchCompatHeaders_edge(t *testing.T) {
	tests := []struct {
		name  string
		key   string
		value string
		want  string
	}{
		{"accept compatible-with", "Accept", "application/json; compatible-with=8", "application/json"},
		{"content-type compatible-with", "Content-Type", "text/plain; compatible-with=8", "application/json"},
		{"accept elasticsearch+json", "Accept", "application/vnd.elasticsearch+json; compatible-with=8", "application/json"},
		{"normal accept", "Accept", "application/json", "application/json"},
		{"normal content-type", "Content-Type", "text/plain", "text/plain"},
		{"empty value", "Content-Type", "", ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := http.Header{}
			h.Set(tc.key, tc.value)
			stripElasticsearchCompatHeaders(h)
			got := h.Get(tc.key)
			if got != tc.want {
				t.Errorf("%s: header %q = %q, want %q", tc.name, tc.key, got, tc.want)
			}
		})
	}
}

func TestStripElasticsearchCompatHeaders_multiple(t *testing.T) {
	h := http.Header{}
	h.Set("Content-Type", "application/json; compatible-with=8")
	h.Set("Accept", "application/vnd.elasticsearch+json")
	h.Set("X-Custom", "keep-me")
	stripElasticsearchCompatHeaders(h)
	if h.Get("Content-Type") != "application/json" {
		t.Errorf("Content-Type = %q", h.Get("Content-Type"))
	}
	if h.Get("Accept") != "application/json" {
		t.Errorf("Accept = %q", h.Get("Accept"))
	}
	if h.Get("X-Custom") != "keep-me" {
		t.Errorf("X-Custom = %q, should be unchanged", h.Get("X-Custom"))
	}
}

func TestZoneParentChild_edge(t *testing.T) {
	tests := []struct {
		label    string
		wantP    string
		wantFull string
	}{
		{"", "", ""},
		{"  ", "", ""},
		{"全部分区", "", ""},
		{"动画", "动画", ""},
		{"番剧相关", "番剧", ""},
		{"纪录片", "放映厅", "放映厅-纪录片"},
		{"电影", "放映厅", "放映厅-电影"},
		{"电视剧", "放映厅", "放映厅-电视剧"},
		{"unknown", "unknown", ""},
	}
	for _, tc := range tests {
		p, full := zoneParentChild(tc.label)
		if p != tc.wantP || full != tc.wantFull {
			t.Errorf("zoneParentChild(%q) = (%q, %q), want (%q, %q)",
				tc.label, p, full, tc.wantP, tc.wantFull)
		}
	}
}
