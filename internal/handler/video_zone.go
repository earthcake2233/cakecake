package handler

import "strings"

// videoZoneAllowed matches the top bar menuLeft / constants/videoZones.js.
var videoZoneAllowed = initVideoZoneAllowed()

func normalizeVideoZone(raw string) string {
	z := strings.TrimSpace(raw)
	if z == "" {
		return ""
	}
	z = strings.ReplaceAll(z, " → ", "-")
	z = strings.ReplaceAll(z, "→", "-")
	z = strings.ReplaceAll(z, "—", "-")
	if _, ok := videoZoneAllowed[z]; ok {
		return z
	}
	return ""
}

func splitVideoZone(zone string) (parent, child string) {
	z := normalizeVideoZone(zone)
	if z == "" {
		return "", ""
	}
	if i := strings.Index(z, "-"); i > 0 {
		return strings.TrimSpace(z[:i]), strings.TrimSpace(z[i+1:])
	}
	return z, ""
}

func videoZoneCategoryLabel(zone string) string {
	parent, child := splitVideoZone(zone)
	if parent == "" {
		return ""
	}
	if child != "" {
		return parent + " > " + child
	}
	return parent
}

// videoZoneFields is embedded in video response DTOs to expose zone metadata.
type videoZoneFields struct {
	Zone       string `json:"zone"`
	ZoneParent string `json:"zone_parent"`
	ZoneChild  string `json:"zone_child"`
	Category   string `json:"category"`
}

func appendVideoZoneFields(m *videoZoneFields, zone string) {
	z := normalizeVideoZone(zone)
	parent, child := splitVideoZone(z)
	m.Zone = z
	m.ZoneParent = parent
	m.ZoneChild = child
	m.Category = videoZoneCategoryLabel(z)
}
