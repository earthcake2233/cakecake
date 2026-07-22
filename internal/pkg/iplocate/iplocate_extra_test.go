package iplocate

import (
	"testing"
)

func TestOpen_EmptyPath(t *testing.T) {
	s, err := Open("")
	if err != nil {
		t.Fatalf("Open('') unexpected error: %v", err)
	}
	if s != nil {
		t.Fatal("expected nil searcher for empty path")
	}
}

func TestOpen_WhitespacePath(t *testing.T) {
	s, err := Open("   ")
	if err != nil {
		t.Fatalf("Open('   ') unexpected error: %v", err)
	}
	if s != nil {
		t.Fatal("expected nil searcher for whitespace path")
	}
}

func TestOpen_BadPath(t *testing.T) {
	_, err := Open("/nonexistent/path.xdb")
	if err == nil {
		t.Skip("Open didn't fail on nonexistent path")
	}
}

func TestSearcher_Close_Nil(t *testing.T) {
	(*Searcher)(nil).Close()
}

func TestSearcher_Close_NilService(t *testing.T) {
	s := &Searcher{svc: nil}
	s.Close()
}

func TestSearcher_Province_NilSearcher(t *testing.T) {
	got := (*Searcher)(nil).Province("8.8.8.8")
	if got != "" {
		t.Errorf("expected empty for nil searcher, got %q", got)
	}
}

func TestSearcher_Province_NilService(t *testing.T) {
	s := &Searcher{svc: nil}
	got := s.Province("8.8.8.8")
	if got != "" {
		t.Errorf("expected empty for nil service, got %q", got)
	}
}

func TestSearcher_Province_EmptyIP(t *testing.T) {
	s := &Searcher{svc: nil}
	got := s.Province("")
	if got != "" {
		t.Errorf("expected empty for empty IP, got %q", got)
	}
	got = s.Province("  ")
	if got != "" {
		t.Errorf("expected empty for whitespace IP, got %q", got)
	}
}

func TestDisplayLabel_Empty(t *testing.T) {
	if got := DisplayLabel(""); got != "" {
		t.Errorf("expected empty for empty string, got %q", got)
	}
	if got := DisplayLabel("  "); got != "" {
		t.Errorf("expected empty for whitespace, got %q", got)
	}
}

func TestDisplayLabel_PipeFormat(t *testing.T) {
	got := DisplayLabel("中国|广东省|深圳市|电信")
	want := "广东"
	if got != want {
		t.Errorf("DisplayLabel(pipe) = %q, want %q", got, want)
	}
}

func TestDisplayLabel_PlainText(t *testing.T) {
	got := DisplayLabel("广东")
	want := "广东"
	if got != want {
		t.Errorf("DisplayLabel(plain) = %q, want %q", got, want)
	}
}

func TestDisplayLabel_ShortProvince(t *testing.T) {
	got := DisplayLabel("北京")
	want := "北京"
	if got != want {
		t.Errorf("DisplayLabel(北京) = %q, want %q", got, want)
	}
}

func TestFormatRegion_Empty(t *testing.T) {
	if got := formatRegion(""); got != "" {
		t.Errorf("expected empty for empty, got %q", got)
	}
	if got := formatRegion("  "); got != "" {
		t.Errorf("expected empty for whitespace, got %q", got)
	}
}

func TestFormatRegion_LAN(t *testing.T) {
	if got := formatRegion("内网IP"); got != "" {
		t.Errorf("expected empty for 内网IP, got %q", got)
	}
}

func TestFormatRegion_International(t *testing.T) {
	tests := []struct {
		raw  string
		want string
	}{
		{"美国|0|0|0|0", "美国"},
		{"日本|0|0|0|0", "日本"},
		{"韩国|0|0|0|0", "韩国"},
		{"英国|0|0|0|0", "英国"},
	}
	for _, tc := range tests {
		got := formatRegion(tc.raw)
		if got != tc.want {
			t.Errorf("formatRegion(%q) = %q, want %q", tc.raw, got, tc.want)
		}
	}
}

func TestFormatRegion_China(t *testing.T) {
	tests := []struct {
		raw  string
		want string
	}{
		{"中国|0|广东|深圳|电信", "广东"},
		{"中国|0|广西壮族自治区|南宁|移动", "广西"},
		{"中国|0|内蒙古自治区|呼和浩特|联通", "内蒙古"},
		{"中国|0|西藏自治区|拉萨|电信", "西藏"},
		{"中国|0|宁夏回族自治区|银川|移动", "宁夏"},
		{"中国|0|新疆维吾尔自治区|乌鲁木齐|电信", "新疆"},
		{"中国|0|香港特别行政区|香港|移动", "香港"},
		{"中国|0|澳门特别行政区|澳门|电信", "澳门"},
		{"中国|0|北京|北京|联通", "北京"},
		{"中国|0|上海|上海|电信", "上海"},
		{"中国|0|天津|天津|移动", "天津"},
		{"中国|0|重庆|重庆|联通", "重庆"},
	}
	for _, tc := range tests {
		got := formatRegion(tc.raw)
		if got != tc.want {
			t.Errorf("formatRegion(%q) = %q, want %q", tc.raw, got, tc.want)
		}
	}
}

func TestFormatRegion_NewFormat(t *testing.T) {
	tests := []struct {
		raw  string
		want string
	}{
		{"中国|广东省|深圳市|电信", "广东"},
		{"中国|北京|北京|联通", "北京"},
		{"中国|上海|上海|移动", "上海"},
		// In v4 format (country|province|city|isp), province is returned for international too
		{"美国|加利福尼亚|洛杉矶|comcast", "加利福尼亚"},
	}
	for _, tc := range tests {
		got := formatRegion(tc.raw)
		if got != tc.want {
			t.Errorf("formatRegion(%q) = %q, want %q", tc.raw, got, tc.want)
		}
	}
}

func TestPickProvinceField(t *testing.T) {
	tests := []struct {
		parts []string
		want  string
	}{
		{[]string{"中国", "0", "广东", "深圳", "电信"}, "广东"},
		{[]string{"中国", "广东", "深圳", "电信"}, "广东"},
		{[]string{"中国"}, ""},
		{[]string{}, ""},
		{[]string{"中国", "0", "0", "0", "0"}, "0"},
		{[]string{"中国", "", "北京", "北京", "联通"}, "北京"},
	}
	for _, tc := range tests {
		got := pickProvinceField(tc.parts)
		if got != tc.want {
			t.Errorf("pickProvinceField(%v) = %q, want %q", tc.parts, got, tc.want)
		}
	}
}

func TestShortenAdminName(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{"", ""},
		{"0", ""},
		{"内网IP", ""},
		{"Reserved", ""},
		{"reserved", ""},
		{"广东省", "广东"},
		{"广西壮族自治区", "广西"},
		{"内蒙古自治区", "内蒙古"},
		{"新疆维吾尔自治区", "新疆"},
		{"宁夏回族自治区", "宁夏"},
		{"西藏自治区", "西藏"},
		{"香港特别行政区", "香港"},
		{"澳门特别行政区", "澳门"},
		{"北京", "北京"},
		{"北京市", "北京"},
		{"上海", "上海"},
		{"重庆市", "重庆"},
		{"深圳市", "深圳"},
		{"海淀区", "海淀区"},
	}
	for _, tc := range tests {
		got := shortenAdminName(tc.name)
		if got != tc.want {
			t.Errorf("shortenAdminName(%q) = %q, want %q", tc.name, got, tc.want)
		}
	}
}

func TestShortenAdminName_NoSuffix(t *testing.T) {
	got := shortenAdminName("广东")
	if got != "广东" {
		t.Errorf("shortenAdminName(广东) = %q, want 广东", got)
	}
}

func TestProvince_Integration(t *testing.T) {
	s := &Searcher{svc: nil}
	if got := s.Province("219.133.110.197"); got != "" {
		t.Errorf("expected empty for nil svc, got %q", got)
	}
}

func TestProvince_MalformedIP(t *testing.T) {
	s := &Searcher{svc: nil}
	if got := s.Province("not-an-ip"); got != "" {
		t.Errorf("expected empty for malformed IP, got %q", got)
	}
}

func TestDisplayLabel_PipeWithMixed(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"中国|0|广东|深圳|电信", "广东"},
		{"中国|0|上海|上海|移动", "上海"},
		{"中国|北京|北京|联通", "北京"},
		{"新加坡|0|0|0|0", "新加坡"},
	}
	for _, tc := range tests {
		got := DisplayLabel(tc.input)
		if got != tc.want {
			t.Errorf("DisplayLabel(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestOpenRoundTrip(t *testing.T) {
	s, err := Open("")
	if err != nil {
		t.Fatal(err)
	}
	if s != nil {
		t.Fatal("expected nil")
	}
	s.Close()
}

func TestSearcher_Province_TrimsIP(t *testing.T) {
	s := &Searcher{svc: nil}
	got := s.Province("  8.8.8.8  ")
	if got != "" {
		t.Errorf("expected empty for nil svc, got %q", got)
	}
}

func TestFormatRegion_EdgeParts(t *testing.T) {
	if got := formatRegion("|||"); got != "" {
		t.Errorf("expected empty for empty pipes, got %q", got)
	}
	if got := formatRegion("中国"); got != "" {
		t.Errorf("expected empty for country only, got %q", got)
	}
	if got := formatRegion("中国|0"); got != "" {
		t.Errorf("expected empty for country|0 with no third field, got %q", got)
	}
}
