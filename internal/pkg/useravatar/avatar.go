package useravatar

import (
	"fmt"
	"strings"

	"minibili/internal/model"
)

// PublicURL appends ?v=updated_at so browsers refetch after OSS overwrite at a fixed key.
func PublicURL(u *model.User) string {
	if u == nil || model.IsUserAnonymized(u) {
		return ""
	}
	raw := strings.TrimSpace(u.AvatarURL)
	if raw == "" {
		return ""
	}
	if u.UpdatedAt.IsZero() {
		return raw
	}
	sep := "?"
	if strings.Contains(raw, "?") {
		sep = "&"
	}
	return fmt.Sprintf("%s%sv=%d", raw, sep, u.UpdatedAt.Unix())
}
