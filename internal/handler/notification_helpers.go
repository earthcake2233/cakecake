package handler

// mergeUniqueDisplayNames deduplicates sender display names preserving order.
func mergeUniqueDisplayNames(names []string) []string {
	seen := make(map[string]bool)
	var out []string
	for _, n := range names {
		if !seen[n] {
			seen[n] = true
			out = append(out, n)
		}
	}
	return out
}

// isReplyInboxType returns true if the notification type is reply-related.
func isReplyInboxType(notifType string) bool {
	switch notifType {
	case "reply_received", "article_reply_received", "dynamic_reply_received":
		return true
	default:
		return false
	}
}

// notifUint64 converts a value to uint64 for notification processing.
func notifUint64(v interface{}) uint64 {
	switch val := v.(type) {
	case float64:
		return uint64(val)
	case uint64:
		return val
	default:
		return 0
	}
}
