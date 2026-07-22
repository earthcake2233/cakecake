package search

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"minibili/internal/config"
)

func newMockESServer(t *testing.T, handler func(w http.ResponseWriter, r *http.Request)) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Elastic-Product", "Elasticsearch")
		w.Header().Set("Content-Type", "application/json")
		handler(w, r)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func newMockESClient(t *testing.T, url string) *Client {
	t.Helper()
	cfg := &config.C{ElasticsearchURL: url}
	c, err := Dial(cfg)
	require.NoError(t, err)
	require.NotNil(t, c)
	return c
}
func TestDial_EmptyURL(t *testing.T) {
	c, err := Dial(&config.C{})
	require.NoError(t, err)
	require.Nil(t, c)
}
func TestDial_WithMockServer(t *testing.T) {
	srv := newMockESServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"name":"es-mock","version":{"number":"8.0.0"}}`))
	})
	c, err := Dial(&config.C{ElasticsearchURL: srv.URL})
	require.NoError(t, err)
	require.NotNil(t, c)
}
func TestEnsureIndices_Create(t *testing.T) {
	var createdIndices []string
	srv := newMockESServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "HEAD" {
			if r.URL.Path == "/" {
				w.WriteHeader(http.StatusOK)
			} else {
				w.WriteHeader(http.StatusNotFound)
			}
			w.Write([]byte(`{}`))
			return
		}
		if r.Method == "GET" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{}`))
			return
		}
		if r.Method == "PUT" {
			createdIndices = append(createdIndices, r.URL.Path)
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"acknowledged":true}`))
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	})
	c := newMockESClient(t, srv.URL)
	err := c.EnsureIndices(nil)
	require.NoError(t, err)
	require.Len(t, createdIndices, 3)
}

func TestEnsureIndices_AlreadyExist(t *testing.T) {
	var putCalls int
	srv := newMockESServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "HEAD" || r.Method == "GET" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{}`))
			return
		}
		putCalls++
		w.Write([]byte(`{}`))
	})
	c := newMockESClient(t, srv.URL)
	err := c.EnsureIndices(nil)
	require.NoError(t, err)
	assert.Equal(t, 0, putCalls)
}

func TestSearchAll_Basic(t *testing.T) {
	srv := newMockESServer(t, func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "_search") {
			resp := map[string]interface{}{
				"hits": map[string]interface{}{
					"hits": []interface{}{
						map[string]interface{}{
							"_source": map[string]interface{}{
								"video_id":  float64(1),
								"title":     "Test Video",
								"user_name": "tester",
							},
						},
					},
					"total": map[string]interface{}{
						"value": float64(1),
					},
				},
			}
			json.NewEncoder(w).Encode(resp)
			return
		}
		w.Write([]byte(`{}`))
	})
	c := newMockESClient(t, srv.URL)
	result, err := c.SearchAll(nil, SearchParams{Keyword: "test", Page: 1, PageSize: 10})
	require.NoError(t, err)
	require.NotNil(t, result)
}

func TestSearchAll_EmptyKeyword(t *testing.T) {
	srv := newMockESServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{}`))
	})
	c := newMockESClient(t, srv.URL)
	result, err := c.SearchAll(nil, SearchParams{Keyword: ""})
	require.NoError(t, err)
	require.NotNil(t, result)
}

func TestDeleteVideo_WithMock(t *testing.T) {
	var deletedID string
	srv := newMockESServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "DELETE" && strings.Contains(r.URL.Path, "_doc") {
			deletedID = path.Base(r.URL.Path)
			w.Write([]byte(`{"result":"deleted"}`))
			return
		}
		w.Write([]byte(`{}`))
	})
	c := newMockESClient(t, srv.URL)
	err := c.DeleteVideo(nil, 42)
	require.NoError(t, err)
	assert.Equal(t, "42", deletedID)
}

func TestDeleteArticle_WithMock(t *testing.T) {
	var deletedID string
	srv := newMockESServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "DELETE" && strings.Contains(r.URL.Path, "_doc") {
			deletedID = path.Base(r.URL.Path)
			w.Write([]byte(`{"result":"deleted"}`))
			return
		}
		w.Write([]byte(`{}`))
	})
	c := newMockESClient(t, srv.URL)
	err := c.DeleteArticle(nil, 456)
	require.NoError(t, err)
	assert.Equal(t, "456", deletedID)
}

func TestOperations_WithNilClient(t *testing.T) {
	var c *Client
	require.NoError(t, c.EnsureIndices(nil))
	_, err2 := c.SearchAll(nil, SearchParams{Keyword: "test"})
	require.Error(t, err2)
	require.NoError(t, c.DeleteVideo(nil, 1))
	require.NoError(t, c.DeleteArticle(nil, 1))
}
