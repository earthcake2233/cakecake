//go:build integration

package handler

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAdminHotSearch_Ops(t *testing.T) {
	_, r, jm := newTestAPI(t)
	access, _, _, err := jm.IssueAdminPair(1)
	require.NoError(t, err)

	// Dashboard.
	w := doReq(r, "GET", "/api/v1/admin/hot-search/dashboard", access, "", nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	// Quick op.
	w = doJSON(r, "POST", "/api/v1/admin/hot-search/quick-op", access, map[string]interface{}{
		"keyword": "golang", "op_type": "pin", "display_title": "Go", "badge": "热", "pin_rank": 1,
	})
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	// Invalid op type.
	w = doJSON(r, "POST", "/api/v1/admin/hot-search/quick-op", access, map[string]interface{}{
		"keyword": "x", "op_type": "bogus",
	})
	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())

	// Reorder.
	w = doJSON(r, "POST", "/api/v1/admin/hot-search/reorder", access, map[string]interface{}{
		"items": []map[string]interface{}{{"keyword": "golang", "title": "Go", "op_id": 1}},
	})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	// Empty items.
	w = doJSON(r, "POST", "/api/v1/admin/hot-search/reorder", access, map[string]interface{}{"items": []interface{}{}})
	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())

	// Redis remove/boost.
	w = doJSON(r, "POST", "/api/v1/admin/hot-search/redis/boost", access, map[string]interface{}{"keyword": "golang", "delta": 3})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	w = doJSON(r, "POST", "/api/v1/admin/hot-search/redis/remove", access, map[string]interface{}{"keyword": "golang"})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	// Empty keyword.
	w = doJSON(r, "POST", "/api/v1/admin/hot-search/redis/remove", access, map[string]interface{}{"keyword": ""})
	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())

	// Reset display order.
	w = doJSON(r, "POST", "/api/v1/admin/hot-search/display-order/reset", access, nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
}
