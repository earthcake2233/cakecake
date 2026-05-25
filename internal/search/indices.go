package search

import (
	"context"
	"fmt"
	"strings"
)

const (
	IndexVideos   = "minibili_videos"
	IndexArticles = "minibili_articles"
	IndexUsers    = "minibili_users"
)

// EnsureIndices creates search indices if they do not exist.
func (c *Client) EnsureIndices(ctx context.Context) error {
	if !c.Enabled() {
		return nil
	}
	for name, body := range map[string]string{
		IndexVideos:   videoIndexMapping,
		IndexArticles: articleIndexMapping,
		IndexUsers:    userIndexMapping,
	} {
		res, err := c.es.Indices.Exists([]string{name}, c.es.Indices.Exists.WithContext(ctx))
		if err != nil {
			return err
		}
		exists := res.StatusCode == 200
		res.Body.Close()
		if exists {
			continue
		}
		cr, err := c.es.Indices.Create(
			name,
			c.es.Indices.Create.WithContext(ctx),
			c.es.Indices.Create.WithBody(strings.NewReader(body)),
		)
		if err != nil {
			return err
		}
		if cr.IsError() {
			defer cr.Body.Close()
			return fmt.Errorf("create index %s: %s", name, cr.Status())
		}
		cr.Body.Close()
	}
	return nil
}

// fmt import for indices create error - need to add fmt to indices.go