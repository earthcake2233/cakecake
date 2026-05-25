package search

import "strings"

// Mirrors handler video zone normalization for ES indexing.
func normalizeVideoZoneForSearch(raw string) string {
	z := strings.TrimSpace(raw)
	if z == "" {
		return ""
	}
	z = strings.ReplaceAll(z, " → ", "-")
	z = strings.ReplaceAll(z, "→", "-")
	z = strings.ReplaceAll(z, "—", "-")
	return z
}

func splitVideoZoneForSearch(zone string) (parent, child string) {
	z := normalizeVideoZoneForSearch(zone)
	if z == "" {
		return "", ""
	}
	if i := strings.Index(z, "-"); i > 0 {
		return strings.TrimSpace(z[:i]), strings.TrimSpace(z[i+1:])
	}
	return z, ""
}

// videoSearchTypeName is the compact zone tag on search list rows (e.g. 动画综合).
func videoSearchTypeName(zone string) string {
	parent, child := splitVideoZoneForSearch(zone)
	if parent == "" {
		return ""
	}
	if child != "" {
		return parent + child
	}
	return parent
}
