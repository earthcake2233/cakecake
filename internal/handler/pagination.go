package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

// maxPageSize caps the page_size query parameter across handlers.
const maxPageSize = 50

// parsePagination parses the common page/page_size query parameters.
// Missing or invalid values fall back to defaultSize; page is clamped to >= 1
// and page_size to [1, maxPageSize].
func parsePagination(c *gin.Context, defaultSize int) (page, pageSize int) {
	page = queryIntDefault(c.Query("page"), 1)
	if page < 1 {
		page = 1
	}
	pageSize = queryIntDefault(c.Query("page_size"), defaultSize)
	if pageSize < 1 {
		pageSize = defaultSize
	}
	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}
	return page, pageSize
}

// parseLimit parses the limit query parameter.
// Invalid or out-of-range values fall back to defaultLimit; pass maxLimit <= 0
// to disable the upper bound.
func parseLimit(c *gin.Context, defaultLimit, maxLimit int) int {
	limit := defaultLimit
	if s := c.Query("limit"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 && (maxLimit <= 0 || n <= maxLimit) {
			limit = n
		}
	}
	return limit
}
