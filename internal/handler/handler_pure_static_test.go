package handler

import (
	"cakecake/internal/model/article"
	"cakecake/internal/model/dm"
	"cakecake/internal/model/dynamic"
	"testing"
	"time"

	"cakecake/internal/config"
)

func TestSearchCacheKey_Unit(t *testing.T) {
	got := searchCacheKey("kw", "video", "time", 1, 20)
	if got == "" {
		t.Error("empty key")
	}
}

func TestManuscriptVideoStatusToDB_Unit(t *testing.T) {
	if got := manuscriptVideoStatusToDB("draft"); got != "draft" {
		t.Errorf("draft: %q", got)
	}
	if got := manuscriptVideoStatusToDB("published"); got != "published" {
		t.Errorf("pub: %q", got)
	}
	if got := manuscriptVideoStatusToDB("rejected"); got != "failed" {
		t.Errorf("rejected: %q", got)
	}
	if got := manuscriptVideoStatusToDB("unknown"); got != "" {
		t.Errorf("unknown: %q", got)
	}
}

func TestManuscriptVideoStatusFilter_Unit(t *testing.T) {
	s, m := manuscriptVideoStatusFilter("")
	if s != "" || len(m) != 0 {
		t.Errorf("empty: %q %v", s, m)
	}
	s, m = manuscriptVideoStatusFilter("draft")
	if s != "draft" {
		t.Errorf("draft: %q", s)
	}
}

func TestArticleListItem_Unit(t *testing.T) {
	now := time.Now()
	art := article.Article{ID: 1, Title: "T", PublishedAt: &now}
	item := articleListItem(art, "Author", articleEngagement{FavoritedByMe: true})
	if item.AuthorName != "Author" {
		t.Errorf("got %v", item.AuthorName)
	}
	if item.FavoritedByMe != true {
		t.Errorf("favorited_by_me=%v", item.FavoritedByMe)
	}
}

func TestArticleListItem_NilPubAt(t *testing.T) {
	art := article.Article{Title: "Draft"}
	item := articleListItem(art, "", articleEngagement{})
	if item.PublishedAt != "" {
		t.Errorf("got %v", item.PublishedAt)
	}
}

func TestUserDynamicPayload_EmptyImages(t *testing.T) {
	d := &dynamic.UserDynamic{}
	p := userDynamicPayload(d, false)
	if len(p.Images) != 0 {
		t.Error("expected empty")
	}
}

func TestUserDynamicPayload_WithData(t *testing.T) {
	d := &dynamic.UserDynamic{
		ImagesJSON: "[\"img.jpg\"]",
		LikeCount:  10,
	}
	p := userDynamicPayload(d, true)
	if p.LikedByMe != true {
		t.Errorf("got %v", p.LikedByMe)
	}
}

func TestArticleStatusAfterSubmit_All(t *testing.T) {
	api := &API{Dependencies: &Dependencies{Cfg: &config.C{}}}
	if got := api.articleStatusAfterSubmit(false); got != "draft" {
		t.Errorf("draft: %s", got)
	}
}

func TestDmPairIDs_All(t *testing.T) {
	lo, hi := dmPairIDs(5, 10)
	if lo != 5 || hi != 10 {
		t.Errorf("(5,10)=>(%d,%d)", lo, hi)
	}
	lo, hi = dmPairIDs(10, 5)
	if lo != 5 || hi != 10 {
		t.Errorf("(10,5)=>(%d,%d)", lo, hi)
	}
}

func TestDmPeerID_All(t *testing.T) {
	conv := &dm.DmConversation{UserLow: 1, UserHigh: 2}
	if got := dmPeerID(conv, 1); got != 2 {
		t.Errorf("1=>%d", got)
	}
	if got := dmPeerID(conv, 2); got != 1 {
		t.Errorf("2=>%d", got)
	}
}

func TestDmPinnedAtAfter_All(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name string
		a, b *time.Time
		want bool
	}{
		{"bothnil", nil, nil, false},
		{"aset", &now, nil, true},
		{"bset", nil, &now, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := dmPinnedAtAfter(tc.a, tc.b)
			if got != tc.want {
				t.Errorf("got %v", got)
			}
		})
	}
}

func TestDmTrimPreview_All(t *testing.T) {
	if got := dmTrimPreview("hello"); got != "hello" {
		t.Errorf("got %q", got)
	}
	long := string(make([]rune, 100))
	got := dmTrimPreview(long)
	if len([]rune(got)) != 81 {
		t.Errorf("len=%d", len([]rune(got)))
	}
}

func TestNormalizeSearchHistoryKeywords_All(t *testing.T) {
	if got := normalizeSearchHistoryKeywords(nil); len(got) != 0 {
		t.Errorf("nil:%d", len(got))
	}
	if got := normalizeSearchHistoryKeywords([]string{}); len(got) != 0 {
		t.Errorf("empty:%d", len(got))
	}
	if got := normalizeSearchHistoryKeywords([]string{"a", "a", "b"}); len(got) != 2 {
		t.Errorf("dedup:%d", len(got))
	}
}

func TestParseFolderIsPublicForm_All(t *testing.T) {
	if got := parseFolderIsPublicForm("true"); got != true {
		t.Error("true")
	}
	if got := parseFolderIsPublicForm("false"); got != false {
		t.Error("false")
	}
	if got := parseFolderIsPublicForm("1"); got != true {
		t.Error("1")
	}
}
