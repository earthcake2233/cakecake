package iplocate

import (
	"strings"

	"github.com/lionsoul2014/ip2region/binding/golang/service"
)

// Searcher resolves IPs to a short province/region label for display.
type Searcher struct {
	svc *service.Ip2Region
}

// Open loads an ip2region IPv4 xdb file. Empty path disables lookup.
func Open(v4Path string) (*Searcher, error) {
	v4Path = strings.TrimSpace(v4Path)
	if v4Path == "" {
		return nil, nil
	}
	svc, err := service.NewIp2RegionWithPath(v4Path, "")
	if err != nil {
		return nil, err
	}
	return &Searcher{svc: svc}, nil
}

// Close releases resources.
func (s *Searcher) Close() {
	if s == nil || s.svc == nil {
		return
	}
	s.svc.Close()
}

// Province returns a Bilibili-style location label, e.g. "广东", "北京".
func (s *Searcher) Province(ip string) string {
	if s == nil || s.svc == nil {
		return ""
	}
	ip = strings.TrimSpace(ip)
	if ip == "" {
		return ""
	}
	raw, err := s.svc.Search(ip)
	if err != nil || strings.TrimSpace(raw) == "" {
		return ""
	}
	return formatRegion(raw)
}

// DisplayLabel normalizes a stored or raw ip2region string for API display (province only).
func DisplayLabel(stored string) string {
	stored = strings.TrimSpace(stored)
	if stored == "" {
		return ""
	}
	if strings.Contains(stored, "|") {
		return formatRegion(stored)
	}
	return stored
}

func formatRegion(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if raw == "内网IP" {
		return ""
	}
	parts := strings.Split(raw, "|")
	country := ""
	if len(parts) > 0 {
		country = strings.TrimSpace(parts[0])
	}
	province := pickProvinceField(parts)
	if province == "" || province == "0" {
		if country != "" && country != "中国" {
			return country
		}
		return ""
	}
	return shortenAdminName(province)
}

func pickProvinceField(parts []string) string {
	if len(parts) < 2 {
		return ""
	}
	area := strings.TrimSpace(parts[1])
	// Legacy ip2region: country|area|province|city|isp — area is "0".
	if area == "0" || area == "" {
		if len(parts) >= 3 {
			return strings.TrimSpace(parts[2])
		}
		return ""
	}
	// ip2region v4: country|province|city|isp — province is at index 1.
	return area
}

func shortenAdminName(name string) string {
	name = strings.TrimSpace(name)
	switch name {
	case "", "0", "内网IP", "Reserved", "reserved":
		return ""
	}
	for _, suf := range []string{
		"壮族自治区", "回族自治区", "维吾尔自治区", "特别行政区", "自治区", "省", "市",
	} {
		if strings.HasSuffix(name, suf) {
			name = strings.TrimSuffix(name, suf)
			break
		}
	}
	return strings.TrimSpace(name)
}
