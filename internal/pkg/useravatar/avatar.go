package useravatar

import (
	"minibili/internal/model/user"
	"fmt"
	"strings"

)

// PublicURL appends ?v=updated_at so browsers refetch after OSS overwrite at a fixed key.
func PublicURL(u *user.User) string {
	if u == nil || user.IsUserAnonymized(u) {
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
