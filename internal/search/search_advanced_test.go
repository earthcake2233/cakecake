package search

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"cakecake/internal/config"
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
		_, _ = w.Write([]byte(`{"name":"es-mock","version":{"number":"8.0.0"}}`))
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
			_, _ = w.Write([]byte(`{}`))
			return
		}
		if r.Method == "GET" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{}`))
			return
		}
		if r.Method == "PUT" {
			createdIndices = append(createdIndices, r.URL.Path)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"acknowledged":true}`))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	})
	c := newMockESClient(t, srv.URL)
	err := c.EnsureIndices(context.TODO())
	require.NoError(t, err)
	require.Len(t, createdIndices, 3)
}

func TestEnsureIndices_AlreadyExist(t *testing.T) {
	var putCalls int
	srv := newMockESServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "HEAD" || r.Method == "GET" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{}`))
			return
		}
		putCalls++
		_, _ = w.Write([]byte(`{}`))
	})
	c := newMockESClient(t, srv.URL)
	err := c.EnsureIndices(context.TODO())
	require.NoError(t, err)
	assert.Equal(t, 0, putCalls)
}

func TestSearchAll_Basic(t *testing.T) {
	var lastBody string
	var lastPath string
	srv := newMockESServer(t, func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "_search") {
			raw, _ := io.ReadAll(r.Body)
			lastBody = string(raw)
			lastPath = r.URL.Path
			resp := map[string]interface{}{
				"hits": map[string]interface{}{
					"hits": []interface{}{
						map[string]interface{}{
							"_source": map[string]interface{}{
								"id":            float64(1),
								"user_id":       float64(42),
								"title":         "Test Video",
								"description":   "<em>golang</em> rocks",
								"uploader":      "tester",
								"cover_url":     "https://cdn.example.com/cover.jpg",
								"play_count":    float64(100),
								"danmaku_count": float64(7),
								"duration_sec":  float64(61),
								"zone":          "Tech-Code",
								"created_at":    "2026-01-02T03:04:05+08:00",
							},
							"highlight": map[string]interface{}{
								"description": []string{"<em>golang</em> rocks"},
							},
						},
					},
					"total": map[string]interface{}{
						"value": float64(1),
					},
				},
			}
			_ = json.NewEncoder(w).Encode(resp)
			return
		}
		_, _ = w.Write([]byte(`{}`))
	})
	c := newMockESClient(t, srv.URL)
	result, err := c.SearchAll(context.TODO(), SearchParams{Keyword: "test", Page: 1, PageSize: 10})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.Result.Video, 1)
	hit := result.Result.Video[0]
	require.Equal(t, uint64(1), hit.Aid)
	require.Equal(t, "Test Video", hit.Title)
	require.Equal(t, "tester", hit.Author)
	require.Equal(t, uint64(42), hit.Mid)
	require.Equal(t, "golang rocks", hit.Description)
	require.Equal(t, "01:01", hit.Duration)
	require.Equal(t, uint64(100), hit.Play)
	require.Equal(t, "TechCode", hit.TypeName)
	require.Equal(t, 1, result.TopTlist.Video)
	require.Contains(t, lastPath, "/_search")
	require.Contains(t, lastBody, `"test"`)
}

func TestSearchAll_EmptyKeyword(t *testing.T) {
	srv := newMockESServer(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{}`))
	})
	c := newMockESClient(t, srv.URL)
	result, err := c.SearchAll(context.TODO(), SearchParams{Keyword: ""})
	require.NoError(t, err)
	require.NotNil(t, result)
}

func TestDeleteVideo_WithMock(t *testing.T) {
	var deletedID string
	srv := newMockESServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "DELETE" && strings.Contains(r.URL.Path, "_doc") {
			deletedID = path.Base(r.URL.Path)
			_, _ = w.Write([]byte(`{"result":"deleted"}`))
			return
		}
		_, _ = w.Write([]byte(`{}`))
	})
	c := newMockESClient(t, srv.URL)
	err := c.DeleteVideo(context.TODO(), 42)
	require.NoError(t, err)
	assert.Equal(t, "42", deletedID)
}

func TestDeleteArticle_WithMock(t *testing.T) {
	var deletedID string
	srv := newMockESServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "DELETE" && strings.Contains(r.URL.Path, "_doc") {
			deletedID = path.Base(r.URL.Path)
			_, _ = w.Write([]byte(`{"result":"deleted"}`))
			return
		}
		_, _ = w.Write([]byte(`{}`))
	})
	c := newMockESClient(t, srv.URL)
	err := c.DeleteArticle(context.TODO(), 456)
	require.NoError(t, err)
	assert.Equal(t, "456", deletedID)
}

func TestOperations_WithNilClient(t *testing.T) {
	var c *Client
	require.NoError(t, c.EnsureIndices(context.TODO()))
	_, err2 := c.SearchAll(context.TODO(), SearchParams{Keyword: "test"})
	require.Error(t, err2)
	require.NoError(t, c.DeleteVideo(context.TODO(), 1))
	require.NoError(t, c.DeleteArticle(context.TODO(), 1))
}
