package search

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/elastic/elastic-transport-go/v8/elastictransport"
	"github.com/elastic/go-elasticsearch/v8"

	"minibili/internal/config"
)

// openSearchCompatInterceptor adapts go-elasticsearch/v8 for OpenSearch / Bonsai:
// - strips ES8 "compatible-with=8" media types (OpenSearch returns 406)
// - adds X-Elastic-Product on responses for the client's product check
func openSearchCompatInterceptor() elastictransport.InterceptorFunc {
	return func(next elastictransport.RoundTripFunc) elastictransport.RoundTripFunc {
		return func(req *http.Request) (*http.Response, error) {
			stripElasticsearchCompatHeaders(req.Header)
			res, err := next(req)
			if err != nil || res == nil {
				return res, err
			}
			if res.Header.Get("X-Elastic-Product") == "" {
				res.Header.Set("X-Elastic-Product", "Elasticsearch")
			}
			return res, nil
		}
	}
}

func stripElasticsearchCompatHeaders(h http.Header) {
	for _, key := range []string{"Content-Type", "Accept"} {
		v := h.Get(key)
		if v == "" {
			continue
		}
		if strings.Contains(v, "compatible-with=8") || strings.Contains(v, "elasticsearch+json") {
			h.Set(key, "application/json")
		}
	}
}

// Client wraps the Elasticsearch HTTP API for Mini-Bili search indices.
type Client struct {
	es *elasticsearch.Client
}

// Dial connects when ELASTICSEARCH_URL is set; returns (nil, nil) when URL is empty.
func Dial(cfg *config.C) (*Client, error) {
	url := strings.TrimSpace(cfg.ElasticsearchURL)
	if url == "" {
		return nil, nil
	}
	es, err := elasticsearch.NewClient(elasticsearch.Config{
		Addresses: []string{url},
		Username:  cfg.ElasticsearchUsername,
		Password:  cfg.ElasticsearchPassword,
		Interceptors: []elastictransport.InterceptorFunc{
			openSearchCompatInterceptor(),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("elasticsearch new client: %w", err)
	}
	c := &Client{es: es}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := c.ping(ctx); err != nil {
		return nil, err
	}
	return c, nil
}

func (c *Client) ping(ctx context.Context) error {
	res, err := c.es.Ping(c.es.Ping.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("elasticsearch ping: %w", err)
	}
	defer res.Body.Close()
	if res.IsError() {
		return fmt.Errorf("elasticsearch ping: %s", res.Status())
	}
	return nil
}

// Enabled reports whether the client is connected.
func (c *Client) Enabled() bool {
	return c != nil && c.es != nil
}

// Close is a no-op (HTTP client needs no explicit close).
func (c *Client) Close() error { return nil }
