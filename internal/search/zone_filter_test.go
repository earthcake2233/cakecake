package search

import "testing"

func TestAppendVideoFilters_zoneParent(t *testing.T) {
	filters := appendVideoFilters([]any{
		map[string]any{"term": map[string]any{"status": "published"}},
	}, VideoFilter{Zone: "动画"})
	if len(filters) != 2 {
		t.Fatalf("filters len = %d, want 2", len(filters))
	}
	outer, ok := filters[1].(map[string]any)["bool"].(map[string]any)
	if !ok {
		t.Fatalf("expected bool zone filter, got %#v", filters[1])
	}
	should, _ := outer["should"].([]any)
	if len(should) < 2 {
		t.Fatalf("should clauses = %d", len(should))
	}
}

func TestAppendVideoFilters_zoneFull(t *testing.T) {
	filters := appendVideoFilters(nil, VideoFilter{Zone: "纪录片"})
	if len(filters) != 1 {
		t.Fatalf("filters len = %d", len(filters))
	}
}

func TestVideoSearchTypeName(t *testing.T) {
	if got := videoSearchTypeName("动画-MAD·AMV"); got != "动画MAD·AMV" {
		t.Fatalf("got %q", got)
	}
	if got := videoSearchTypeName("生活"); got != "生活" {
		t.Fatalf("got %q", got)
	}
}

func TestZoneParentChild(t *testing.T) {
	p, f := zoneParentChild("纪录片")
	if p != "放映厅" || f != "放映厅-纪录片" {
		t.Fatalf("纪录片 => %q %q", p, f)
	}
	p, f = zoneParentChild("番剧")
	if p != "番剧" || f != "" {
		t.Fatalf("番剧 => %q %q", p, f)
	}
}
