package handler

import (
	"cakecake/internal/model/admin"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"cakecake/internal/config"
)

// ---------- adminHotSearchLimit ----------

func TestAdminHotSearchLimit(t *testing.T) {
	tests := []struct {
		name  string
		query string
		def   int
		max   int
		want  int
	}{
		{"no query", "", 10, 20, 10},
		{"within bounds", "5", 10, 20, 5},
		{"at max", "20", 10, 20, 20},
		{"above max clipped", "30", 10, 20, 20},
		{"invalid query returns def", "abc", 10, 20, 10},
		{"zero query returns def", "0", 10, 20, 10},
		{"negative query returns def", "-1", 10, 20, 10},
		{"def above max", "0", 25, 20, 20},
		{"def default when query invalid", "xyz", 8, 15, 8},
		{"large max", "15", 5, 100, 15},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(w)
			req := httptest.NewRequest(http.MethodGet, "/?limit="+tc.query, nil)
			ctx.Request = req
			got := adminHotSearchLimit(ctx, tc.def, tc.max)
			if got != tc.want {
				t.Errorf("adminHotSearchLimit(query=%q, def=%d, max=%d) = %d, want %d",
					tc.query, tc.def, tc.max, got, tc.want)
			}
		})
	}
}

// ---------- hotSearchDisplayTitle ----------

func TestHotSearchDisplayTitleHandler(t *testing.T) {
	tests := []struct {
		name string
		op   *admin.HotSearchOp
		want string
	}{
		{"nil op", nil, ""},
		{"display title set", &admin.HotSearchOp{Keyword: "kw", DisplayTitle: "Display"}, "Display"},
		{"no display title", &admin.HotSearchOp{Keyword: "keyword"}, "keyword"},
		{"empty keyword and display", &admin.HotSearchOp{}, ""},
		{"both spaces", &admin.HotSearchOp{Keyword: "  ", DisplayTitle: "  "}, ""},
		{"trimmed display", &admin.HotSearchOp{Keyword: "kw", DisplayTitle: "  Title  "}, "Title"},
		{"trimmed keyword fallback", &admin.HotSearchOp{Keyword: "  kw2  "}, "kw2"},
		{"only display spaces", &admin.HotSearchOp{Keyword: "real", DisplayTitle: "  "}, "real"},
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

// ---------- adminAgentMeta ----------

func TestAdminAgentMeta(t *testing.T) {
	t.Run("nil cfg", func(t *testing.T) {
		api := &API{Dependencies: &Dependencies{Cfg: nil}}
		m := api.adminAgentMeta()
		if m.DeepseekConfigured != false {
			t.Errorf("nil cfg: deepseek_configured should be false, got %v", m.DeepseekConfigured)
		}
		if m.MaxProfiles <= 0 {
			t.Error("max_profiles should be present")
		}
	})

	t.Run("no api key", func(t *testing.T) {
		api := &API{Dependencies: &Dependencies{Cfg: &config.C{DeepSeekAPIKey: ""}}}
		m := api.adminAgentMeta()
		if m.DeepseekConfigured != false {
			t.Error("empty key: deepseek_configured should be false")
		}
	})

	t.Run("api key with spaces", func(t *testing.T) {
		api := &API{Dependencies: &Dependencies{Cfg: &config.C{DeepSeekAPIKey: "  "}}}
		m := api.adminAgentMeta()
		if m.DeepseekConfigured != false {
			t.Error("whitespace key: deepseek_configured should be false")
		}
	})

	t.Run("configured", func(t *testing.T) {
		api := &API{Dependencies: &Dependencies{Cfg: &config.C{DeepSeekAPIKey: "sk-abc123"}}}
		m := api.adminAgentMeta()
		if m.DeepseekConfigured != true {
			t.Error("valid key: deepseek_configured should be true")
		}
	})
}
