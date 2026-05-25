package bvid

import "strconv"

// Encode formats a video id as BV{id} for display and URLs (Mini-Bili global convention).
func Encode(id uint64) string {
	if id == 0 {
		return ""
	}
	return "BV" + strconv.FormatUint(id, 10)
}
