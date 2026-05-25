package handler

import (
	"strings"

	"github.com/gin-gonic/gin"
)

// videoZoneAllowed matches顶栏 menuLeft / constants/videoZones.js.
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

func appendVideoZoneFields(m gin.H, zone string) {
	z := normalizeVideoZone(zone)
	parent, child := splitVideoZone(z)
	m["zone"] = z
	m["zone_parent"] = parent
	m["zone_child"] = child
	m["category"] = videoZoneCategoryLabel(z)
}
