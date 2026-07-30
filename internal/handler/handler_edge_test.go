package handler

import (
	"minibili/internal/model/dynamic"
	"minibili/internal/model/user"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"minibili/internal/pkg/sensitive"
	"minibili/internal/pkg/usercoin"
)

// ---------- coinLedgerReasonText ----------

func TestCoinLedgerReasonText(t *testing.T) {
	tests := []struct {
		name string
		row  *user.CoinLedger
		want string
	}{
		{name: "login reward", row: &user.CoinLedger{ReasonType: usercoin.ReasonLoginReward}, want: "登录奖励"},
		{name: "nickname change", row: &user.CoinLedger{ReasonType: usercoin.ReasonNicknameChange}, want: "修改昵称"},
		{name: "video tip with id", row: &user.CoinLedger{ReasonType: usercoin.ReasonVideoTip, VideoID: 42}, want: "给视频 BV42 打赏"},
		{name: "video tip no id", row: &user.CoinLedger{ReasonType: usercoin.ReasonVideoTip}, want: "给视频打赏"},
		{name: "video tip income with id", row: &user.CoinLedger{ReasonType: usercoin.ReasonVideoTipIncome, VideoID: 99}, want: "给视频 BV99 打赏"},
		{name: "video tip income no id", row: &user.CoinLedger{ReasonType: usercoin.ReasonVideoTipIncome}, want: "给视频打赏"},
		{name: "default", row: &user.CoinLedger{ReasonType: "unknown_type"}, want: "硬币变动"},
		{name: "empty", row: &user.CoinLedger{ReasonType: ""}, want: "硬币变动"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := coinLedgerReasonText(tc.row)
			if got != tc.want {
				t.Errorf("coinLedgerReasonText(%+v) = %q, want %q", tc.row, got, tc.want)
			}
		})
	}
}

// ---------- formatCoinLedgerItem ----------

func TestFormatCoinLedgerItem(t *testing.T) {
	now := time.Date(2025, 6, 15, 10, 30, 0, 0, time.UTC)
	tests := []struct {
		name string
		row  *user.CoinLedger
	}{
		{name: "positive delta", row: &user.CoinLedger{CreatedAt: now, DeltaTenths: 10, ReasonType: usercoin.ReasonLoginReward}},
		{name: "negative delta", row: &user.CoinLedger{CreatedAt: now, DeltaTenths: -60, ReasonType: usercoin.ReasonNicknameChange}},
		{name: "zero delta", row: &user.CoinLedger{CreatedAt: now, DeltaTenths: 0, ReasonType: ""}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := formatCoinLedgerItem(tc.row)
			if got["created_at"] != tc.row.CreatedAt.Format("2006-01-02 15:04:05") {
				t.Errorf("created_at = %q, want %q", got["created_at"], tc.row.CreatedAt.Format("2006-01-02 15:04:05"))
			}
			wantChange := float64(tc.row.DeltaTenths) / float64(usercoin.TenthsPerCoin)
			if got["change"] != wantChange {
				t.Errorf("change = %v, want %v", got["change"], wantChange)
			}
			if got["reason"] == "" || got["reason"] == nil {
				t.Error("reason should not be empty")
			}
		})
	}
}

// ---------- bannerSlideURL ----------

func TestBannerSlideURL(t *testing.T) {
	tests := []struct {
		name       string
		linkType   string
		linkTarget string
		want       string
	}{
		{"video valid", "video", "42", "/#/video/BV42"},
		{"video zero id", "video", "0", "/"},
		{"video negative id", "video", "-1", "/"},
		{"video non-numeric", "video", "abc", "/"},
		{"video trimmed spaces", "  video  ", "  42  ", "/#/video/BV42"},
		{"url valid", "url", "https://example.com", "https://example.com"},
		{"url empty", "url", "", "/"},
		{"url spaces", "url", "   ", "/"},
		{"url trimmed", "  url  ", "  https://ex.com  ", "https://ex.com"},
		{"none type", "none", "whatever", "/"},
		{"empty type", "", "target", "/"},
		{"unknown type", "article", "123", "/"},
		{"mixed case video", "VIDEO", "1", "/#/video/BV1"},
		{"mixed case url", "URL", "http://test.com", "http://test.com"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := bannerSlideURL(tc.linkType, tc.linkTarget)
			if got != tc.want {
				t.Errorf("bannerSlideURL(%q, %q) = %q, want %q", tc.linkType, tc.linkTarget, got, tc.want)
			}
		})
	}
}

// ---------- spacePrivacyFromUser ----------

func TestSpacePrivacyFromUser(t *testing.T) {
	tests := []struct {
		name string
		u    *user.User
		want spacePrivacyPayload
	}{
		{
			name: "all true",
			u:    &user.User{PrivacyPublicFavorites: true, PrivacyPublicRecentCoins: true, PrivacyPublicFollowing: true, PrivacyPublicFans: true, PrivacyPublicBirthday: true},
			want: spacePrivacyPayload{true, true, true, true, true},
		},
		{
			name: "all false",
			u:    &user.User{},
			want: spacePrivacyPayload{false, false, false, false, false},
		},
		{
			name: "mixed",
			u:    &user.User{PrivacyPublicFavorites: true, PrivacyPublicFollowing: true, PrivacyPublicBirthday: true},
			want: spacePrivacyPayload{true, false, true, false, true},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := spacePrivacyFromUser(tc.u)
			if got != tc.want {
				t.Errorf("spacePrivacyFromUser(%+v) = %+v, want %+v", tc.u, got, tc.want)
			}
		})
	}
}

// ---------- spaceViewerCanSee ----------

func TestSpaceViewerCanSee(t *testing.T) {
	tests := []struct {
		name     string
		ownerID  uint64
		viewerOK bool
		viewerID uint64
		allowed  bool
		want     bool
	}{
		{"owner views own", 1, true, 1, false, true},
		{"owner no viewer id", 1, true, 0, false, false},
		{"owner not logged in", 1, false, 0, true, true},
		{"other viewer allowed", 1, true, 2, true, true},
		{"other viewer not allowed", 1, true, 2, false, false},
		{"anon allowed", 1, false, 0, true, true},
		{"anon not allowed", 1, false, 0, false, false},
		{"viewer not ok owner", 1, false, 1, false, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := spaceViewerCanSee(tc.ownerID, tc.viewerOK, tc.viewerID, tc.allowed)
			if got != tc.want {
				t.Errorf("spaceViewerCanSee(%d, %v, %d, %v) = %v, want %v", tc.ownerID, tc.viewerOK, tc.viewerID, tc.allowed, got, tc.want)
			}
		})
	}
}

// ---------- initVideoZoneAllowed ----------

func TestInitVideoZoneAllowed(t *testing.T) {
	m := initVideoZoneAllowed()

	parentZones := []string{"动画", "番剧", "国创", "音乐", "舞蹈", "游戏", "科技", "生活", "鬼畜", "时尚", "广告", "娱乐", "影视", "放映厅"}
	for _, p := range parentZones {
		if _, ok := m[p]; !ok {
			t.Errorf("missing parent zone: %q", p)
		}
	}

	subZones := []string{"动画-MAD·AMV", "番剧-连载动画", "国创-国产动画", "生活-日常", "游戏-单机游戏", "科技-趣味科普人文"}
	for _, sz := range subZones {
		if _, ok := m[sz]; !ok {
			t.Errorf("missing sub-zone: %q", sz)
		}
	}

	for _, cat := range videoZoneCatalog {
		if _, ok := m[cat.parent]; !ok {
			t.Errorf("catalog parent %q not in allowed map", cat.parent)
		}
		for _, sub := range cat.subs {
			key := cat.parent + "-" + sub
			if _, ok := m[key]; !ok {
				t.Errorf("catalog sub %q not in allowed map", key)
			}
		}
	}

	if _, ok := m["广告-"]; ok {
		t.Error("广告 should not have a sub entry with empty name")
	}

	expectedCount := 14
	for _, cat := range videoZoneCatalog {
		expectedCount += len(cat.subs)
	}
	if len(m) != expectedCount {
		t.Errorf("allowed map size = %d, want %d", len(m), expectedCount)
	}
}

// ---------- rejectIfSensitive (nil Sens) ----------

func TestRejectIfSensitive_EmptyFilter(t *testing.T) {
	f := sensitive.NewFilter("", zap.NewNop())
	f.Reload()
	// Empty filter blocks ALL content by design
	api := &API{Dependencies: &Dependencies{Sens: f, Log: zap.NewNop()}}
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/", nil)
	got := api.rejectIfSensitive(c, "test content", 999)
	if !got {
		t.Error("expected true when Sens filter has empty word list (blocks all)")
	}
}
func TestRejectIfCommentSensitive_NilSens(t *testing.T) {
	api := &API{Dependencies: &Dependencies{Log: zap.NewNop()}}
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/", nil)
	got := api.rejectIfCommentSensitive(c, "comment content")
	if got {
		t.Error("expected false when Sens is nil")
	}
}

// ---------- HomeStats handler with nil DB/Hub ----------

func TestHomeStats_NilDB(t *testing.T) {
	api := &API{Dependencies: &Dependencies{Log: zap.NewNop()}}
	w := httptest.NewRecorder(); c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/home-stats", nil)

	api.HomeStats(c)
	require.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Code int `json:"code"`
		Data struct {
			WebOnline int   `json:"web_online"`
			AllCount  int64 `json:"all_count"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)
	require.Equal(t, 0, resp.Data.WebOnline)
	require.Equal(t, int64(0), resp.Data.AllCount)
}

// ---------- HotSearchList with nil SearchHot ----------

func TestHotSearchList_NilSearchHot(t *testing.T) {
	api := &API{Dependencies: &Dependencies{Log: zap.NewNop()}}
	w := httptest.NewRecorder(); c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/hot-search?limit=5", nil)
	api.HotSearchList(c)
	require.Equal(t, http.StatusOK, w.Code)
}

// ---------- GetMeSpacePrivacy with no auth ----------

func TestGetMeSpacePrivacy_NoUser(t *testing.T) {
	api := &API{Dependencies: &Dependencies{Log: zap.NewNop()}}
	w := httptest.NewRecorder(); c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/users/me/space-privacy", nil)
	api.GetMeSpacePrivacy(c)
	require.Equal(t, http.StatusUnauthorized, w.Code)
}

// ---------- queryIntDefault ----------

func TestQueryIntDefault(t *testing.T) {
	tests := []struct {
		name string
		s    string
		def  int
		want int
	}{
		{name: "empty string", s: "", def: 10, want: 10},
		{name: "spaces", s: "  ", def: 5, want: 5},
		{name: "valid number", s: "42", def: 1, want: 42},
		{name: "valid number with spaces", s: "  7  ", def: 1, want: 7},
		{name: "non-numeric", s: "abc", def: 99, want: 99},
		{name: "negative", s: "-5", def: 1, want: -5},
		{name: "zero", s: "0", def: 100, want: 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := queryIntDefault(tc.s, tc.def)
			if got != tc.want {
				t.Errorf("queryIntDefault(%q, %d) = %d, want %d", tc.s, tc.def, got, tc.want)
			}
		})
	}
}

// ---------- previewCommentContent ----------

func TestPreviewCommentContent(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		maxRunes int
		want     string
	}{
		{name: "empty string", s: "", maxRunes: 10, want: ""},
		{name: "spaces only", s: "  ", maxRunes: 10, want: ""},
		{name: "zero maxRunes", s: "hello", maxRunes: 0, want: ""},
		{name: "negative maxRunes", s: "hello", maxRunes: -1, want: ""},
		{name: "shorter than max", s: "hi", maxRunes: 10, want: "hi"},
		{name: "equal to max", s: "12345", maxRunes: 5, want: "12345"},
		{name: "longer than max", s: "hello world", maxRunes: 5, want: "hello\u2026"},
		{name: "unicode truncation", s: "你好世界", maxRunes: 2, want: "你好\u2026"},
		{name: "trimmed input", s: "  hello  ", maxRunes: 10, want: "hello"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := previewCommentContent(tc.s, tc.maxRunes)
			if got != tc.want {
				t.Errorf("previewCommentContent(%q, %d) = %q, want %q", tc.s, tc.maxRunes, got, tc.want)
			}
		})
	}
}

// ---------- dynamicDisplayTitle ----------

func TestDynamicDisplayTitle(t *testing.T) {
	tests := []struct {
		name string
		d    *dynamic.UserDynamic
		want string
	}{
		{name: "nil dynamic", d: nil, want: ""},
		{name: "has title", d: &dynamic.UserDynamic{Title: "My Title"}, want: "My Title"},
		{name: "title with spaces", d: &dynamic.UserDynamic{Title: "  Spaced Title  "}, want: "Spaced Title"},
		{name: "empty title, has content", d: &dynamic.UserDynamic{Title: "", Content: "Some content here"}, want: "Some content here"},
		{name: "long content truncated", d: &dynamic.UserDynamic{Title: "", Content: "This is a very long content that should be truncated after forty characters"}, want: "This is a very long content that should \u2026"},
		{name: "empty everything", d: &dynamic.UserDynamic{Title: "", Content: ""}, want: "图文动态"},
		{name: "spaces only in content", d: &dynamic.UserDynamic{Title: "", Content: "   "}, want: "图文动态"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := dynamicDisplayTitle(tc.d)
			if got != tc.want {
				t.Errorf("dynamicDisplayTitle(%+v) = %q, want %q", tc.d, got, tc.want)
			}
		})
	}
}

// ---------- dynamicCoverURL ----------

func TestDynamicCoverURL(t *testing.T) {
	tests := []struct {
		name string
		d    *dynamic.UserDynamic
		want string
	}{
		{name: "nil dynamic", d: nil, want: ""},
		{name: "empty images json", d: &dynamic.UserDynamic{ImagesJSON: ""}, want: ""},
		{name: "empty array", d: &dynamic.UserDynamic{ImagesJSON: "[]"}, want: ""},
		{name: "single image", d: &dynamic.UserDynamic{ImagesJSON: "[\"https://example.com/img.jpg\"]"}, want: "https://example.com/img.jpg"},
		{name: "multiple images returns first", d: &dynamic.UserDynamic{ImagesJSON: "[\"https://example.com/1.jpg\",\"https://example.com/2.jpg\"]"}, want: "https://example.com/1.jpg"},
		{name: "invalid json", d: &dynamic.UserDynamic{ImagesJSON: "not-json"}, want: ""},
		{name: "whitespace jpg", d: &dynamic.UserDynamic{ImagesJSON: "[\"  https://example.com/img.jpg  \"]"}, want: "https://example.com/img.jpg"},
		{name: "empty strings in array", d: &dynamic.UserDynamic{ImagesJSON: "[\"\",\"https://example.com/img.jpg\"]"}, want: "https://example.com/img.jpg"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := dynamicCoverURL(tc.d)
			if got != tc.want {
				t.Errorf("dynamicCoverURL(%+v) = %q, want %q", tc.d, got, tc.want)
			}
		})
	}
}

// ---------- normalizeGender ----------


// ---------- validProfileBirthday ----------

func TestValidProfileBirthday(t *testing.T) {
	tests := []struct {
		name string
		s    string
		want bool
	}{
		{name: "empty", s: "", want: true},
		{name: "valid date", s: "2000-01-01", want: true},
		{name: "valid trimmed", s: "  2000-06-15  ", want: true},
		{name: "min year", s: "1900-01-01", want: true},
		{name: "max year", s: "2100-12-31", want: true},
		{name: "invalid format", s: "01-01-2000", want: false},
		{name: "too early year", s: "1899-12-31", want: false},
		{name: "too late year", s: "2101-01-01", want: false},
		{name: "not a date", s: "abc", want: false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := validProfileBirthday(tc.s)
			if got != tc.want {
				t.Errorf("validProfileBirthday(%q) = %v, want %v", tc.s, got, tc.want)
			}
		})
	}
}

// ---------- creatorUpInclusiveDays ----------

func TestCreatorUpInclusiveDays(t *testing.T) {
	now := time.Now()
	yesterday := now.Add(-24 * time.Hour)
	twoDaysAgo := now.Add(-48 * time.Hour)
	future := now.Add(24 * time.Hour)

	tests := []struct {
		name  string
		first *time.Time
		want  int
	}{
		{name: "nil first", first: nil, want: 0},
		{name: "zero time", first: &time.Time{}, want: 0},
		{name: "today", first: &now, want: 1},
		{name: "yesterday", first: &yesterday, want: 2},
		{name: "two days ago", first: &twoDaysAgo, want: 3},
		{name: "future", first: &future, want: 1},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := creatorUpInclusiveDays(tc.first)
			if got != tc.want {
				t.Errorf("creatorUpInclusiveDays(%v) = %d, want %d", tc.first, got, tc.want)
			}
		})
	}
}

// ---------- bannerImageExt ----------

