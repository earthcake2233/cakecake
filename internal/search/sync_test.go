package search

import (
	"cakecake/internal/model/article"
	"cakecake/internal/model/user"
	"cakecake/internal/model/video"
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func syncTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&video.Video{}, &article.Article{}, &user.User{}))
	return db
}

func TestIndexVideoFromDB_Published(t *testing.T) {
	db := syncTestDB(t)
	require.NoError(t, db.Create(&user.User{ID: 1, Username: "alice", Nickname: "Alice"}).Error)
	require.NoError(t, db.Create(&video.Video{
		ID: 10, UserID: 1, Title: "v", Status: video.StatusPublished, Zone: "动画",
	}).Error)

	var indexed bool
	srv := newMockESServer(t, func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "_doc/10") && r.Method == "PUT" {
			indexed = true
			_, _ = w.Write([]byte(`{"result":"created"}`))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	})
	c := newMockESClient(t, srv.URL)
	require.NoError(t, c.IndexVideoFromDB(context.Background(), db, 10))
	require.True(t, indexed)
}

func TestIndexVideoFromDB_NotPublished(t *testing.T) {
	db := syncTestDB(t)
	require.NoError(t, db.Create(&video.Video{ID: 10, UserID: 1, Title: "v", Status: video.StatusDraft}).Error)
	var deleted bool
	srv := newMockESServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "DELETE" {
			deleted = true
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{}`))
			return
		}
		_, _ = w.Write([]byte(`{}`))
	})
	c := newMockESClient(t, srv.URL)
	require.NoError(t, c.IndexVideoFromDB(context.Background(), db, 10))
	require.True(t, deleted)
}

func TestIndexArticleAndUser(t *testing.T) {
	db := syncTestDB(t)
	now := time.Now()
	require.NoError(t, db.Create(&article.Article{ID: 20, UserID: 1, Title: "a", BodyMD: "body", Status: article.StatusPublished, PublishedAt: &now}).Error)
	require.NoError(t, db.Create(&user.User{ID: 1, Username: "alice", Nickname: "Alice"}).Error)

	var docs []string
	srv := newMockESServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "PUT" {
			docs = append(docs, r.URL.Path)
			_, _ = w.Write([]byte(`{"result":"created"}`))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	})
	c := newMockESClient(t, srv.URL)
	require.NoError(t, c.IndexArticleFromDB(context.Background(), db, 20))
	require.NoError(t, c.IndexUserFromDB(context.Background(), db, 1))
	require.Len(t, docs, 2)
}

func TestIndexUser_Anonymized(t *testing.T) {
	db := syncTestDB(t)
	now := time.Now()
	require.NoError(t, db.Create(&user.User{ID: 1, Username: "gone", AnonymizedAt: &now}).Error)
	var deleted bool
	srv := newMockESServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "DELETE" {
			deleted = true
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{}`))
			return
		}
		_, _ = w.Write([]byte(`{}`))
	})
	c := newMockESClient(t, srv.URL)
	require.NoError(t, c.IndexUserFromDB(context.Background(), db, 1))
	require.True(t, deleted)
}

func TestDeleteDocs(t *testing.T) {
	srv := newMockESServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "DELETE" {
			w.WriteHeader(http.StatusNotFound)
		} else {
			w.WriteHeader(http.StatusOK)
		}
		_, _ = w.Write([]byte(`{}`))
	})
	c := newMockESClient(t, srv.URL)
	require.NoError(t, c.DeleteVideo(context.Background(), 10))
	require.NoError(t, c.DeleteArticle(context.Background(), 20))
}

func TestReindexAll(t *testing.T) {
	db := syncTestDB(t)
	require.NoError(t, db.Create(&video.Video{ID: 10, UserID: 1, Title: "v", Status: video.StatusPublished}).Error)
	require.NoError(t, db.Create(&video.Video{ID: 11, UserID: 1, Title: "draft", Status: video.StatusDraft}).Error)
	require.NoError(t, db.Create(&article.Article{ID: 20, UserID: 1, Title: "a", BodyMD: "b", Status: article.StatusPublished}).Error)
	require.NoError(t, db.Create(&user.User{ID: 1, Username: "alice"}).Error)

	var putCount int
	srv := newMockESServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "PUT" {
			putCount++
			_, _ = w.Write([]byte(`{"result":"created"}`))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	})
	c := newMockESClient(t, srv.URL)
	require.NoError(t, c.ReindexAll(context.Background(), db))
	require.Equal(t, 3, putCount) // 1 video + 1 article + 1 user
}

func TestIndexHelpers_Disabled(t *testing.T) {
	c := &Client{} // disabled
	ctx := context.Background()
	require.NoError(t, c.IndexVideoFromDB(ctx, nil, 1))
	require.NoError(t, c.IndexArticleFromDB(ctx, nil, 1))
	require.NoError(t, c.IndexUserFromDB(ctx, nil, 1))
	require.NoError(t, c.ReindexAll(ctx, nil))
	require.NoError(t, c.DeleteVideo(ctx, 1))
	require.NoError(t, c.DeleteArticle(ctx, 1))
}

func TestIndexDoc_Error(t *testing.T) {
	srv := newMockESServer(t, func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "_doc") {
			w.WriteHeader(http.StatusBadGateway)
		} else {
			w.WriteHeader(http.StatusOK)
		}
		_, _ = w.Write([]byte(`{"error":"boom"}`))
	})
	c := newMockESClient(t, srv.URL)
	err := c.indexDoc(context.Background(), IndexVideos, "10", map[string]interface{}{"a": 1})
	require.Error(t, err)
	require.Contains(t, err.Error(), "502")
}

func TestSearchArticles_Basic(t *testing.T) {
	srv := newMockESServer(t, func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "_search") {
			resp := map[string]interface{}{
				"hits": map[string]interface{}{
					"hits": []interface{}{
						map[string]interface{}{
							"_source": map[string]interface{}{
								"id":      float64(1),
								"title":   "Article",
								"author":  "alice",
								"excerpt": "excerpt",
							},
						},
					},
					"total": map[string]interface{}{"value": float64(1)},
				},
			}
			_ = json.NewEncoder(w).Encode(resp)
			return
		}
		_, _ = w.Write([]byte(`{}`))
	})
	c := newMockESClient(t, srv.URL)
	payload, err := c.SearchArticles(context.Background(), SearchParams{Keyword: "go", Page: 1, PageSize: 10, Sort: "pubdate"})
	require.NoError(t, err)
	require.NotNil(t, payload)
	require.Len(t, payload.Result, 1)
}

func TestArticleSortClause(t *testing.T) {
	require.NotEmpty(t, articleSortClause("pubdate"))
	require.NotEmpty(t, articleSortClause("click"))
	require.NotEmpty(t, articleSortClause("like"))
	require.NotEmpty(t, articleSortClause("reply"))
	require.NotEmpty(t, articleSortClause("default"))
	require.NotEmpty(t, articleSortClause(""))
}
