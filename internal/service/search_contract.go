package service

import "cakecake/internal/search"

// Search contract types re-exported from the search package so the handler
// layer can depend on service (and later a gRPC contract) instead of the
// search infrastructure package directly.
type (
	// AllResult is the aggregate search result across all content types.
	AllResult = search.AllResult
	// SearchResultBuckets groups results by content type.
	SearchResultBuckets = search.SearchResultBuckets
	// TopTlist is a top-N result list for one content type.
	TopTlist = search.TopTlist
	// VideoHit is a single video search hit.
	VideoHit = search.VideoHit
	// ArticleHit is a single article search hit.
	ArticleHit = search.ArticleHit
	// UserHit is a single user search hit.
	UserHit = search.UserHit
	// SearchParams carries keyword, paging and filter options.
	SearchParams = search.SearchParams
	// VideoFilter carries video-specific filter options.
	VideoFilter = search.VideoFilter
)

// ValidateKeyword wraps the shared keyword validation rule.
func ValidateKeyword(k string) error {
	return search.ValidateKeyword(k)
}

// ParseVideoFilter wraps the shared video filter parsing rule.
func ParseVideoFilter(order, duration, zone string) VideoFilter {
	return search.ParseVideoFilter(order, duration, zone)
}
