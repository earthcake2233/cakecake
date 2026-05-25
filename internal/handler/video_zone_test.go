package handler

import "testing"

func TestNormalizeVideoZone(t *testing.T) {
	cases := map[string]string{
		"动画":        "动画",
		"生活-日常":     "生活-日常",
		"生活 → 日常":   "生活-日常",
		"":          "",
		"  游戏  ":    "游戏",
		"科技-数码":     "科技-数码",
		"知识":        "",
	}
	for in, want := range cases {
		if got := normalizeVideoZone(in); got != want {
			t.Fatalf("normalizeVideoZone(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestVideoZoneCategoryLabel(t *testing.T) {
	if got := videoZoneCategoryLabel("生活-日常"); got != "生活 > 日常" {
		t.Fatalf("got %q", got)
	}
	if got := videoZoneCategoryLabel("动画"); got != "动画" {
		t.Fatalf("got %q", got)
	}
}
