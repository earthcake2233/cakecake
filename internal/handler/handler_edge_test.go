package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"minibili/internal/model"
	"minibili/internal/pkg/sensitive"
	"minibili/internal/pkg/usercoin"
)

// ---------- coinLedgerReasonText ----------

func TestCoinLedgerReasonText(t *testing.T) {
	tests := []struct {
		name string
		row  *model.CoinLedger
		want string
	}{
		{name: "login reward", row: &model.CoinLedger{ReasonType: usercoin.ReasonLoginReward}, want: "登录奖励"},
		{name: "nickname change", row: &model.CoinLedger{ReasonType: usercoin.ReasonNicknameChange}, want: "修改昵称"},
		{name: "video tip with id", row: &model.CoinLedger{ReasonType: usercoin.ReasonVideoTip, VideoID: 42}, want: "给视频 BV42 打赏"},
		{name: "video tip no id", row: &model.CoinLedger{ReasonType: usercoin.ReasonVideoTip}, want: "给视频打赏"},
		{name: "video tip income with id", row: &model.CoinLedger{ReasonType: usercoin.ReasonVideoTipIncome, VideoID: 99}, want: "给视频 BV99 打赏"},
		{name: "video tip income no id", row: &model.CoinLedger{ReasonType: usercoin.ReasonVideoTipIncome}, want: "给视频打赏"},
		{name: "default", row: &model.CoinLedger{ReasonType: "unknown_type"}, want: "硬币变动"},
		{name: "empty", row: &model.CoinLedger{ReasonType: ""}, want: "硬币变动"},
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
		row  *model.CoinLedger
	}{
		{name: "positive delta", row: &model.CoinLedger{CreatedAt: now, DeltaTenths: 10, ReasonType: usercoin.ReasonLoginReward}},
		{name: "negative delta", row: &model.CoinLedger{CreatedAt: now, DeltaTenths: -60, ReasonType: usercoin.ReasonNicknameChange}},
		{name: "zero delta", row: &model.CoinLedger{CreatedAt: now, DeltaTenths: 0, ReasonType: ""}},
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
		u    *model.User
		want spacePrivacyPayload
	}{
		{
			name: "all true",
			u:    &model.User{PrivacyPublicFavorites: true, PrivacyPublicRecentCoins: true, PrivacyPublicFollowing: true, PrivacyPublicFans: true, PrivacyPublicBirthday: true},
			want: spacePrivacyPayload{true, true, true, true, true},
		},
		{
			name: "all false",
			u:    &model.User{},
			want: spacePrivacyPayload{false, false, false, false, false},
		},
		{
			name: "mixed",
			u:    &model.User{PrivacyPublicFavorites: true, PrivacyPublicFollowing: true, PrivacyPublicBirthday: true},
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
