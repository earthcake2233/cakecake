package handler

import (
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"

	"minibili/internal/config"
	"minibili/internal/model"
)

func TestNormalizeDanmakuFontSize(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"", "md"},
		{"md", "md"},
		{"sm", "sm"},
		{"small", "sm"},
		{"lg", "lg"},
		{"large", "lg"},
		{"  SM  ", "sm"},
		{"  MD  ", "md"},
		{"  LG  ", "lg"},
		{"bogus", "md"},
	}
	for _, tc := range tests {
		got := normalizeDanmakuFontSize(tc.input)
		if got != tc.want {
			t.Errorf("normalizeDanmakuFontSize(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestDanmakuFontSizeField(t *testing.T) {
	tests := []struct {
		dm   model.Danmaku
		want string
	}{
		{model.Danmaku{FontSize: "sm"}, "sm"},
		{model.Danmaku{FontSize: "lg"}, "lg"},
		{model.Danmaku{FontSize: ""}, "md"},
		{model.Danmaku{FontSize: "  "}, "md"},
		{model.Danmaku{}, "md"},
	}
	for _, tc := range tests {
		got := danmakuFontSizeField(tc.dm)
		if got != tc.want {
			t.Errorf("danmakuFontSizeField(%+v) = %q, want %q", tc.dm, got, tc.want)
		}
	}
}

func TestValidateVideoDraftContentEdge(t *testing.T) {
	desc2001 := string(make([]rune, 2001))
	if validateVideoDraftContent("title", desc2001, true) {
		t.Error("2001-char desc should fail")
	}
	desc2000 := string(make([]rune, 2000))
	if !validateVideoDraftContent("title", desc2000, true) {
		t.Error("2000-char desc should pass")
	}
	title81 := string(make([]rune, 81))
	if validateVideoDraftContent(title81, "", true) {
		t.Error("81-char title should fail")
	}
	title80 := string(make([]rune, 80))
	if !validateVideoDraftContent(title80, "", true) {
		t.Error("80-char title should pass")
	}
	if !validateVideoDraftContent("x", "", false) {
		t.Error("1-char title should pass")
	}
}

func TestValidateMetadataOnlyDraft(t *testing.T) {
	if !validateMetadataOnlyDraft("Title", "") {
		t.Error("title only should pass")
	}
	if !validateMetadataOnlyDraft("Title", "Desc") {
		t.Error("title + desc should pass")
	}
	if validateMetadataOnlyDraft("", "") {
		t.Error("empty should fail")
	}
	if validateMetadataOnlyDraft("", "Desc") {
		t.Error("desc without title should fail")
	}
	title81 := string(make([]rune, 81))
	if validateMetadataOnlyDraft(title81, "") {
		t.Error("81-char title should fail")
	}
	desc2001 := string(make([]rune, 2001))
	if validateMetadataOnlyDraft("title", desc2001) {
		t.Error("2001-char desc should fail")
	}
}

func TestVideoDraftDir(t *testing.T) {
	got := videoDraftDir("/tmp/uploads")
	want := filepath.FromSlash("/tmp/uploads/drafts")
	if got != want {
		t.Errorf("videoDraftDir = %q, want %q", got, want)
	}
}

func TestVideoDraftRawPath(t *testing.T) {
	got := videoDraftRawPath("/drafts", 42, ".mp4")
	want := filepath.FromSlash("/drafts/42.mp4")
	if got != want {
		t.Errorf("videoDraftRawPath = %q, want %q", got, want)
	}
	got = videoDraftRawPath("/drafts", 42, "")
	want = filepath.FromSlash("/drafts/42.bin")
	if got != want {
		t.Errorf("videoDraftRawPath empty ext = %q, want %q", got, want)
	}
	got = videoDraftRawPath("/drafts", 42, "avi")
	want = filepath.FromSlash("/drafts/42.avi")
	if got != want {
		t.Errorf("videoDraftRawPath = %q, want %q", got, want)
	}
}

func TestVideoDraftCoverPath(t *testing.T) {
	got := videoDraftCoverPath("/drafts", 42, ".png")
	want := filepath.FromSlash("/drafts/42_cover.png")
	if got != want {
		t.Errorf("videoDraftCoverPath = %q, want %q", got, want)
	}
	got = videoDraftCoverPath("/drafts", 42, ".jpeg")
	want = filepath.FromSlash("/drafts/42_cover.jpg")
	if got != want {
		t.Errorf("videoDraftCoverPath jpeg = %q, want %q", got, want)
	}
	got = videoDraftCoverPath("/drafts", 42, "")
	want = filepath.FromSlash("/drafts/42_cover.jpg")
	if got != want {
		t.Errorf("videoDraftCoverPath empty = %q, want %q", got, want)
	}
}

func TestNormalizeTagStrings(t *testing.T) {
	tests := []struct {
		name string
		arr  []string
		want []string
	}{
		{"empty", []string{}, nil},
		{"with spaces", []string{"  tag1  ", "tag2"}, []string{"tag1", "tag2"}},
		{"dedup", []string{"Tag", "tag", "TAG"}, []string{"Tag"}},
		{"limit", []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k"},
			[]string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j"}},
		{"trim long", []string{string(make([]rune, 40))}, []string{string(make([]rune, 32))}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := normalizeTagStrings(tc.arr)
			if len(got) != len(tc.want) {
				t.Fatalf("len=%d, want %d: %v", len(got), len(tc.want), got)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Errorf("got[%d] = %q, want %q", i, got[i], tc.want[i])
				}
			}
		})
	}
}

func TestParseTagsPostForm(t *testing.T) {
	tests := []struct {
		input   string
		want    string
		wantErr bool
	}{
		{"", "[]", false},
		{"  ", "[]", false},
		{`["a", "b"]`, `["a","b"]`, false},
		{`["  spaced  "]`, `["spaced"]`, false},
		{"not json", "", true},
	}
	for _, tc := range tests {
		got, err := parseTagsPostForm(tc.input)
		if tc.wantErr {
			if err == nil {
				t.Errorf("parseTagsPostForm(%q) expected error", tc.input)
			}
			continue
		}
		if err != nil {
			t.Errorf("parseTagsPostForm(%q) unexpected error: %v", tc.input, err)
			continue
		}
		if got != tc.want {
			t.Errorf("parseTagsPostForm(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestVideoTagsForResponse(t *testing.T) {
	tests := []struct {
		tagsJSON string
		want     []string
	}{
		{"", []string{}},
		{"[]", []string{}},
		{`["a", "b"]`, []string{"a", "b"}},
		{`["A", "a"]`, []string{"A"}},
		{"invalid", []string{}},
	}
	for _, tc := range tests {
		got := videoTagsForResponse(tc.tagsJSON)
		if len(got) != len(tc.want) {
			t.Errorf("videoTagsForResponse(%q) = %v, want %v", tc.tagsJSON, got, tc.want)
		}
	}
}

func TestValidateArticleContent(t *testing.T) {
	if !validateArticleContent("Title", "Body", true) {
		t.Error("valid publish should pass")
	}
	if validateArticleContent("", "Body", true) {
		t.Error("publish without title should fail")
	}
	if validateArticleContent("Title", "", true) {
		t.Error("publish without body should fail")
	}
	if validateArticleContent("", "", true) {
		t.Error("publish empty should fail")
	}
	if !validateArticleContent("Title", "", false) {
		t.Error("draft with title should pass")
	}
	if !validateArticleContent("", "Body", false) {
		t.Error("draft with body should pass")
	}
	if validateArticleContent("", "", false) {
		t.Error("draft empty should fail")
	}
}

func TestUploaderNameForAPI(t *testing.T) {
	tests := []struct {
		name string
		u    *model.User
		want string
	}{
		{"nil user", nil, ""},
		{"nickname set", &model.User{Nickname: "Alice", Username: "alice"}, "Alice"},
		{"no nickname", &model.User{Nickname: "", Username: "bob"}, "bob"},
		{"nickname spaces", &model.User{Nickname: "  ", Username: "charlie"}, "charlie"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := uploaderNameForAPI(tc.u)
			if got != tc.want {
				t.Errorf("uploaderNameForAPI(%+v) = %q, want %q", tc.u, got, tc.want)
			}
		})
	}
}

func TestNormalizeGender(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"", "secret"},
		{"male", "male"},
		{"female", "female"},
		{"secret", "secret"},
		{"other", "secret"},
		{"unknown", "secret"},
		{"  male  ", "male"},
	}
	for _, tc := range tests {
		got := normalizeGender(tc.input)
		if got != tc.want {
			t.Errorf("normalizeGender(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestValidateFollowGroupName(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{"", false},
		{"  ", false},
		{"Friends", true},
		{string(make([]rune, 17)), false},
		{string(make([]rune, 16)), true},
	}
	for _, tc := range tests {
		got := validateFollowGroupName(tc.name)
		if got != tc.want {
			t.Errorf("validateFollowGroupName(%q) = %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestDanmakuTypeLabel(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"scroll", "普通"},
		{"top", "顶部"},
		{"bottom", "底部"},
		{"", "普通"},
		{"unknown", "普通"},
	}
	for _, tc := range tests {
		got := danmakuTypeLabel(tc.input)
		if got != tc.want {
			t.Errorf("danmakuTypeLabel(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestFormatDanmakuPlayTime(t *testing.T) {
	tests := []struct {
		sec  float64
		want string
	}{
		{0, "00:00"},
		{30.5, "00:30"},
		{60, "01:00"},
		{3661, "61:01"},
		{-1, "00:00"},
	}
	for _, tc := range tests {
		got := formatDanmakuPlayTime(tc.sec)
		if got != tc.want {
			t.Errorf("formatDanmakuPlayTime(%v) = %q, want %q", tc.sec, got, tc.want)
		}
	}
}

func TestBannerOSSObjectKeysExtended(t *testing.T) {
	cfg := &config.C{
		OSSBucket:          "bucket",
		OSSEndpoint:        "https://oss-cn-beijing.aliyuncs.com",
		OSSPublicURLPrefix: "https://bucket.oss-cn-beijing.aliyuncs.com",
	}
	b := model.HomeBanner{
		ID:       3,
		ImageURL: "https://bucket.oss-cn-beijing.aliyuncs.com/home-banners/abc.jpg",
	}
	keys := bannerOSSObjectKeys(cfg, b)
	if len(keys) < 2 {
		t.Fatalf("expected at least 2 keys, got %v", keys)
	}
}

func TestVideoOSSObjectKeysExtended(t *testing.T) {
	cfg := &config.C{
		OSSPublicURLPrefix: "https://bucket.oss.aliyuncs.com",
	}
	v := model.Video{
		ID:       7,
		VideoURL: "https://bucket.oss.aliyuncs.com/videos/7.mp4",
		CoverURL: "https://bucket.oss.aliyuncs.com/covers/7.png",
	}
	keys := videoOSSObjectKeys(cfg, v)
	if len(keys) < 3 {
		t.Fatalf("expected at least 3 keys, got %v", keys)
	}
}

func TestNormalizeSearchHistoryKeywords(t *testing.T) {
	tests := []struct {
		input []string
		want  []string
	}{
		{nil, nil},
		{[]string{}, []string{}},
		{[]string{"  hello  ", "WORLD"}, []string{"hello", "WORLD"}},
		{[]string{"", "  ", "test"}, []string{"test"}},
	}
	for _, tc := range tests {
		got := normalizeSearchHistoryKeywords(tc.input)
		if len(got) != len(tc.want) {
			t.Fatalf("len=%d, want %d: %v", len(got), len(tc.want), got)
		}
		for i := range got {
			if got[i] != tc.want[i] {
				t.Errorf("got[%d] = %q, want %q", i, got[i], tc.want[i])
			}
		}
	}
}

func TestAppendVideoZoneFields(t *testing.T) {
	m := gin.H{}
	appendVideoZoneFields(m, "生活-日常")
	if m["zone"] != "生活-日常" {
		t.Errorf("zone = %q", m["zone"])
	}
	if m["zone_parent"] != "生活" {
		t.Errorf("zone_parent = %q", m["zone_parent"])
	}
	if m["zone_child"] != "日常" {
		t.Errorf("zone_child = %q", m["zone_child"])
	}
	if m["category"] != "生活 > 日常" {
		t.Errorf("category = %q", m["category"])
	}
}

func TestHandlerFormatUint(t *testing.T) {
	tests := []struct {
		n    uint64
		want string
	}{
		{0, "0"},
		{1, "1"},
		{123, "123"},
		{18446744073709551615, "18446744073709551615"},
	}
	for _, tc := range tests {
		got := formatUint(tc.n)
		if got != tc.want {
			t.Errorf("formatUint(%d) = %q, want %q", tc.n, got, tc.want)
		}
	}
}

func TestValidProfileNickname(t *testing.T) {
	if !validProfileNickname("Alice") {
		t.Error("short nickname should be valid")
	}
	if validProfileNickname(string(make([]rune, 31))) {
		t.Error("31-char nickname should be invalid")
	}
	if !validProfileNickname(string(make([]rune, 30))) {
		t.Error("30-char nickname should be valid")
	}
}

func TestValidProfileSign(t *testing.T) {
	if !validProfileSign("hello") {
		t.Error("short sign should be valid")
	}
	if validProfileSign(string(make([]rune, 501))) {
		t.Error("501-char sign should be invalid")
	}
	if !validProfileSign(string(make([]rune, 500))) {
		t.Error("500-char sign should be valid")
	}
}

func TestValidSpaceAnnouncement(t *testing.T) {
	if !validSpaceAnnouncement("hello") {
		t.Error("short announcement should be valid")
	}
	if validSpaceAnnouncement(string(make([]rune, 151))) {
		t.Error("151-char announcement should be invalid")
	}
	if !validSpaceAnnouncement(string(make([]rune, 150))) {
		t.Error("150-char announcement should be valid")
	}
}
