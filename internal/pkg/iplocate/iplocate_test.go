package iplocate

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFormatRegion(t *testing.T) {
	cases := []struct {
		raw  string
		want string
	}{
		{"中国|0|广东省|深圳市|电信", "广东"},
		{"中国|广东省|深圳市|电信", "广东"},
		{"中国|广东省|深圳市|南山区|电信", "广东"},
		{"中国|北京|北京市|朝阳区|联通", "北京"},
		{"中国|上海市|上海市|浦东新区|移动", "上海"},
		{"中国|0|内蒙古自治区|呼和浩特市|移动", "内蒙古"},
		{"内网IP", ""},
		{"美国|0|0|0|0", "美国"},
	}
	for _, tc := range cases {
		if got := formatRegion(tc.raw); got != tc.want {
			t.Fatalf("formatRegion(%q) = %q, want %q", tc.raw, got, tc.want)
		}
	}
}

func TestDisplayLabel(t *testing.T) {
	if got := DisplayLabel("中国|广东省|深圳市|电信"); got != "广东" {
		t.Fatalf("DisplayLabel(pipe) = %q, want 广东", got)
	}
	if got := DisplayLabel("广东"); got != "广东" {
		t.Fatalf("DisplayLabel(short) = %q, want 广东", got)
	}
}

func TestProvinceFromXDB(t *testing.T) {
	path := filepath.Join("..", "..", "..", "configs", "ip2region_v4.xdb")
	if _, err := os.Stat(path); err != nil {
		t.Skip("ip2region xdb not present:", path)
	}
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	// Shenzhen IP used in upstream ip2region tests.
	if got := s.Province("219.133.110.197"); got != "广东" {
		t.Fatalf("Province(219.133.110.197) = %q, want 广东", got)
	}
}
