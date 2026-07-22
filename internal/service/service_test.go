package service

import (
	"strings"
	"testing"
	"time"

	"minibili/internal/model"
)

func TestNormalizeSearchKeyword(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"", ""},
		{"   ", ""},
		{"Hello World", "helloworld"},
		{"  Spring Anime  ", "springanime"},
		{"Go Programming", "goprogramming"},
		{"ABC DEF", "abcdef"},
		{"  a  b  c  ", "abc"},
	}
	for _, tc := range tests {
		got := NormalizeSearchKeyword(tc.input)
		if got != tc.want {
			t.Errorf("NormalizeSearchKeyword(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestEscapeHTML(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"hello", "hello"},
		{"a & b", "a &amp; b"},
		{"<tag>", "&lt;tag&gt;"},
		{`"quote"`, "&quot;quote&quot;"},
		{"& < > \"", "&amp; &lt; &gt; &quot;"},
		{"", ""},
	}
	for _, tc := range tests {
		got := escapeHTML(tc.input)
		if got != tc.want {
			t.Errorf("escapeHTML(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestValidateSuggestTerm(t *testing.T) {
	tests := []struct {
		term string
		want bool
	}{
		{"hello", true},
		{"", true},
		{"   ", true},
		{strings.Repeat("a", 50), true},
		{strings.Repeat("a", 51), false},
	}
	for _, tc := range tests {
		got := ValidateSuggestTerm(tc.term)
		if got != tc.want {
			t.Errorf("ValidateSuggestTerm(%q) = %v, want %v", tc.term, got, tc.want)
		}
	}
}

func TestHighlightSuggestKeyword(t *testing.T) {
	tests := []struct {
		display string
		term    string
		want    string
	}{
		{"", "test", ""},
		{"  ", "test", ""},
		{"hello", "", "hello"},
		{"hello world", "hello", `<em class="suggest_high_light">hello</em> world`},
		{"Hello World", "world", `Hello <em class="suggest_high_light">World</em>`},
		{"Go Programming", "go", `<em class="suggest_high_light">Go</em> Programming`},
		{"abc", "xyz", "abc"},
		{"a & b", "a", `<em class="suggest_high_light">a</em> &amp; b`},
	}
	for _, tc := range tests {
		got := HighlightSuggestKeyword(tc.display, tc.term)
		if got != tc.want {
			t.Errorf("HighlightSuggestKeyword(%q, %q) = %q, want %q", tc.display, tc.term, got, tc.want)
		}
	}
}

func TestKeywordMatchesSuggest(t *testing.T) {
	tests := []struct {
		termNorm string
		kwNorm   string
		display  string
		want     bool
	}{
		{"", "anything", "Anything", true},
		{"hello", "helloworld", "", true},
		{"hello", "myhello", "My Hello", true},
		{"hello", "world", "World", false},
		{"abc", "abcdef", "ABCDEF", true},
		{"go", "goprogramming", "Go Programming", true},
	}
	for _, tc := range tests {
		got := keywordMatchesSuggest(tc.termNorm, tc.kwNorm, tc.display)
		if got != tc.want {
			t.Errorf("keywordMatchesSuggest(%q, %q, %q) = %v, want %v",
				tc.termNorm, tc.kwNorm, tc.display, got, tc.want)
		}
	}
}

func TestDedupIdentity(t *testing.T) {
	tests := []struct {
		userID    uint64
		clientKey string
		want      string
	}{
		{0, "", "anon"},
		{0, "1.2.3.4", "ip:1.2.3.4"},
		{42, "", "u:42"},
		{42, "1.2.3.4", "u:42"},
		{100, "some-key", "u:100"},
	}
	for _, tc := range tests {
		got := dedupIdentity(tc.userID, tc.clientKey)
		if got != tc.want {
			t.Errorf("dedupIdentity(%d, %q) = %q, want %q", tc.userID, tc.clientKey, got, tc.want)
		}
	}
}

func TestHotSearchOpActive(t *testing.T) {
	now := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name string
		start *time.Time
		end   *time.Time
		want  bool
	}{
		{"no bounds", nil, nil, true},
		{"in future", ptrTime(now.Add(time.Hour)), nil, false},
		{"started but no end", ptrTime(now.Add(-time.Hour)), nil, true},
		{"expired", nil, ptrTime(now.Add(-time.Hour)), false},
		{"within range", ptrTime(now.Add(-time.Hour)), ptrTime(now.Add(time.Hour)), true},
		{"ends exactly now", nil, ptrTime(now), true},
		{"starts exactly now", ptrTime(now), nil, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := hotSearchOpActive(now, tc.start, tc.end)
			if got != tc.want {
				t.Errorf("hotSearchOpActive(%v, %v, %v) = %v, want %v", now, tc.start, tc.end, got, tc.want)
			}
		})
	}
}

func ptrTime(t time.Time) *time.Time {
	return &t
}

func TestHotSearchDisplayTitle(t *testing.T) {
	tests := []struct {
		name string
		op   *model.HotSearchOp
		want string
	}{
		{"display title set", &model.HotSearchOp{Keyword: "kw", DisplayTitle: "Display"}, "Display"},
		{"no display title", &model.HotSearchOp{Keyword: "keyword"}, "keyword"},
		{"empty", &model.HotSearchOp{}, ""},
		{"only spaces", &model.HotSearchOp{Keyword: "  ", DisplayTitle: "  "}, ""},
		{"trimmed display", &model.HotSearchOp{Keyword: "kw", DisplayTitle: "  Title  "}, "Title"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := hotSearchDisplayTitle(tc.op)
			if got != tc.want {
				t.Errorf("hotSearchDisplayTitle(%+v) = %q, want %q", tc.op, got, tc.want)
			}
		})
	}
}

func TestQuotaKey(t *testing.T) {
	s := &AgentService{Cfg: nil}
	key := s.quotaKey(123)
	if !strings.Contains(key, "mb:agent:quota:123:") {
		t.Errorf("quotaKey(123) = %q, missing expected pattern", key)
	}
	today := time.Now().Format("20060102")
	if !strings.Contains(key, today) {
		t.Errorf("quotaKey should contain today's date %s, got %q", today, key)
	}
}

func TestValidateSuggestTermUnicode(t *testing.T) {
	term := "你好世界"
	if !ValidateSuggestTerm(term) {
		t.Errorf("ValidateSuggestTerm(%q) should be true", term)
	}
	long := strings.Repeat("你", 51)
	if ValidateSuggestTerm(long) {
		t.Errorf("ValidateSuggestTerm(%q) should be false", long)
	}
}

func TestNormalizeSearchKeywordUnicode(t *testing.T) {
	input := "  你好 世界  "
	got := NormalizeSearchKeyword(input)
	if got != "你好世界" {
		t.Errorf("NormalizeSearchKeyword(%q) = %q, want %q", input, got, "你好世界")
	}
}

func TestHighlightSuggestKeywordMultipleMatches(t *testing.T) {
	got := HighlightSuggestKeyword("hello hello", "hello")
	first := `<em class="suggest_high_light">hello</em> hello`
	if got != first {
		t.Errorf("expected %q, got %q", first, got)
	}
}

func TestHighlightSuggestKeywordHTMLEscaping(t *testing.T) {
	got := HighlightSuggestKeyword("<test> & more", "test")
	want := `&lt;<em class="suggest_high_light">test</em>&gt; &amp; more`
	if got != want {
		t.Errorf("expected %q, got %q", want, got)
	}
}

func TestEscapeHTMLAllEntities(t *testing.T) {
	input := "&<>\""
	want := "&amp;&lt;&gt;&quot;"
	got := escapeHTML(input)
	if got != want {
		t.Errorf("escapeHTML(%q) = %q, want %q", input, got, want)
	}
}

func TestKeywordMatchesSuggestEdge(t *testing.T) {
	if !keywordMatchesSuggest("", "anything", "anything") {
		t.Error("empty termNorm should match everything")
	}
	if keywordMatchesSuggest("xyz", "abc", "ABC") {
		t.Error("should not match when termNorm not in keyword or display")
	}
}

func TestHotSearchOpActiveNilBoth(t *testing.T) {
	now := time.Now()
	if !hotSearchOpActive(now, nil, nil) {
		t.Error("nil start and nil end should be active")
	}
}

func TestDedupIdentityEdge(t *testing.T) {
	got := dedupIdentity(0, "")
	if got != "anon" {
		t.Errorf("expected 'anon', got %q", got)
	}
}
