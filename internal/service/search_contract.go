package service

import "cakecake/internal/search"

// Search contract types re-exported from the search package so the handler
// layer can depend on service (and later a gRPC contract) instead of the
// search infrastructure package directly.
type (
	AllResult           = search.AllResult
	SearchResultBuckets = search.SearchResultBuckets
	TopTlist            = search.TopTlist
	VideoHit            = search.VideoHit
	ArticleHit          = search.ArticleHit
	UserHit             = search.UserHit
	SearchParams        = search.SearchParams
	VideoFilter         = search.VideoFilter
)

// ValidateKeyword wraps the shared keyword validation rule.
func ValidateKeyword(k string) error {
	return search.ValidateKeyword(k)
}

// ParseVideoFilter wraps the shared video filter parsing rule.
func ParseVideoFilter(order, duration, zone string) VideoFilter {
	return search.ParseVideoFilter(order, duration, zone)
}
