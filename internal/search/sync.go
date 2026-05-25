package search

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"gorm.io/gorm"

	"minibili/internal/model"
)

type videoDoc struct {
	ID            uint64    `json:"id"`
	UserID        uint64    `json:"user_id"`
	Title         string    `json:"title"`
	Description   string    `json:"description"`
	Tags          string    `json:"tags,omitempty"`
	Uploader      string    `json:"uploader"`
	CoverURL      string    `json:"cover_url"`
	PlayCount     uint64    `json:"play_count"`
	DanmakuCount  uint64    `json:"danmaku_count"`
	FavCount      uint64    `json:"fav_count"`
	DurationSec   float64   `json:"duration_sec"`
	Zone          string    `json:"zone,omitempty"`
	ZoneParent    string    `json:"zone_parent,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	Status        string    `json:"status"`
}

type articleDoc struct {
	ID            uint64     `json:"id"`
	UserID        uint64     `json:"user_id"`
	Title         string     `json:"title"`
	Body          string     `json:"body"`
	Excerpt       string     `json:"excerpt"`
	Author        string     `json:"author"`
	AuthorAvatar  string     `json:"author_avatar"`
	CoverURL      string     `json:"cover_url"`
	ViewCount     uint64     `json:"view_count"`
	CommentCount  uint64     `json:"comment_count"`
	FavCount      uint64     `json:"fav_count"`
	Tags          string     `json:"tags,omitempty"`
	Category      string     `json:"category"`
	PublishedAt   *time.Time `json:"published_at,omitempty"`
	Status        string     `json:"status"`
}

type userDoc struct {
	ID         uint64 `json:"id"`
	Nickname   string `json:"nickname"`
	Username   string `json:"username"`
	CakeID     string `json:"cake_id"`
	Sign       string `json:"sign"`
	AvatarURL  string `json:"avatar_url"`
	Status     string `json:"status"`
}

func firstTagLabel(tagsJSON string) string {
	tagsJSON = strings.TrimSpace(tagsJSON)
	if tagsJSON == "" || tagsJSON == "[]" {
		return "专栏"
	}
	var arr []string
	if err := json.Unmarshal([]byte(tagsJSON), &arr); err != nil {
		return "专栏"
	}
	for _, t := range arr {
		t = strings.TrimSpace(t)
		if t != "" {
			return t
		}
	}
	return "专栏"
}

func articleExcerpt(bodyMD string, maxRunes int) string {
	s := strings.TrimSpace(bodyMD)
	if s == "" {
		return ""
	}
	var b strings.Builder
	for _, r := range s {
		switch r {
		case '#', '*', '`', '>', '[', ']', '(', ')':
			continue
		default:
			b.WriteRune(r)
		}
	}
	s = strings.Join(strings.Fields(b.String()), " ")
	if utf8.RuneCountInString(s) > maxRunes {
		rs := []rune(s)
		return string(rs[:maxRunes]) + "…"
	}
	return s
}

func tagsPlain(tagsJSON string) string {
	tagsJSON = strings.TrimSpace(tagsJSON)
	if tagsJSON == "" || tagsJSON == "[]" {
		return ""
	}
	var arr []string
	if err := json.Unmarshal([]byte(tagsJSON), &arr); err != nil {
		return ""
	}
	return strings.Join(arr, " ")
}

func (c *Client) indexDoc(ctx context.Context, index, docID string, body any) error {
	if !c.Enabled() {
		return nil
	}
	b, err := json.Marshal(body)
	if err != nil {
		return err
	}
	res, err := c.es.Index(
		index,
		bytes.NewReader(b),
		c.es.Index.WithContext(ctx),
		c.es.Index.WithDocumentID(docID),
		c.es.Index.WithRefresh("false"),
	)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.IsError() {
		msg, _ := io.ReadAll(res.Body)
		return fmt.Errorf("index %s/%s: %s %s", index, docID, res.Status(), strings.TrimSpace(string(msg)))
	}
	return nil
}

func (c *Client) deleteDoc(ctx context.Context, index, docID string) error {
	if !c.Enabled() {
		return nil
	}
	res, err := c.es.Delete(
		index,
		docID,
		c.es.Delete.WithContext(ctx),
	)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.IsError() && res.StatusCode != 404 {
		return fmt.Errorf("delete %s/%s: %s", index, docID, res.Status())
	}
	return nil
}

// IndexVideoFromDB upserts one published video document.
func (c *Client) IndexVideoFromDB(ctx context.Context, db *gorm.DB, videoID uint64) error {
	if !c.Enabled() {
		return nil
	}
	var v model.Video
	if err := db.First(&v, videoID).Error; err != nil {
		return err
	}
	if v.Status != "published" {
		return c.deleteDoc(ctx, IndexVideos, strconv.FormatUint(videoID, 10))
	}
	var u model.User
	_ = db.First(&u, v.UserID).Error
	zone := normalizeVideoZoneForSearch(v.Zone)
	parent, _ := splitVideoZoneForSearch(zone)
	doc := videoDoc{
		ID:           v.ID,
		UserID:       v.UserID,
		Title:        v.Title,
		Description:  v.Description,
		Tags:         tagsPlain(v.TagsJSON),
		Uploader:     model.DisplayUsername(&u),
		CoverURL:     v.CoverURL,
		PlayCount:    v.PlayCount,
		DanmakuCount: v.DanmakuCount,
		FavCount:     v.FavCount,
		DurationSec:  v.DurationSec,
		Zone:         zone,
		ZoneParent:   parent,
		CreatedAt:    v.CreatedAt,
		Status:       v.Status,
	}
	return c.indexDoc(ctx, IndexVideos, strconv.FormatUint(videoID, 10), doc)
}

// DeleteVideo removes a video document from the search index.
func (c *Client) DeleteVideo(ctx context.Context, videoID uint64) error {
	return c.deleteDoc(ctx, IndexVideos, strconv.FormatUint(videoID, 10))
}

// IndexArticleFromDB upserts one published article document.
func (c *Client) IndexArticleFromDB(ctx context.Context, db *gorm.DB, articleID uint64) error {
	if !c.Enabled() {
		return nil
	}
	var a model.Article
	if err := db.First(&a, articleID).Error; err != nil {
		return err
	}
	if a.Status != "published" {
		return c.deleteDoc(ctx, IndexArticles, strconv.FormatUint(articleID, 10))
	}
	body := strings.TrimSpace(a.BodyMD)
	if len(body) > 8000 {
		body = body[:8000]
	}
	var u model.User
	_ = db.First(&u, a.UserID).Error
	doc := articleDoc{
		ID:           a.ID,
		UserID:       a.UserID,
		Title:        a.Title,
		Body:         body,
		Excerpt:      articleExcerpt(a.BodyMD, 120),
		Author:       model.DisplayUsername(&u),
		AuthorAvatar: strings.TrimSpace(u.AvatarURL),
		CoverURL:     a.CoverURL,
		ViewCount:    a.ViewCount,
		CommentCount: a.CommentCount,
		FavCount:     a.FavCount,
		Tags:         tagsPlain(a.TagsJSON),
		Category:     firstTagLabel(a.TagsJSON),
		PublishedAt:  a.PublishedAt,
		Status:       a.Status,
	}
	return c.indexDoc(ctx, IndexArticles, strconv.FormatUint(articleID, 10), doc)
}

// DeleteArticle removes an article document from the search index.
func (c *Client) DeleteArticle(ctx context.Context, articleID uint64) error {
	return c.deleteDoc(ctx, IndexArticles, strconv.FormatUint(articleID, 10))
}

// IndexUserFromDB upserts one active user document (skips anonymized accounts).
func (c *Client) IndexUserFromDB(ctx context.Context, db *gorm.DB, userID uint64) error {
	if !c.Enabled() {
		return nil
	}
	var u model.User
	if err := db.First(&u, userID).Error; err != nil {
		return err
	}
	if model.IsUserAnonymized(&u) {
		return c.deleteDoc(ctx, IndexUsers, strconv.FormatUint(userID, 10))
	}
	doc := userDoc{
		ID:        u.ID,
		Nickname:  strings.TrimSpace(u.Nickname),
		Username:  strings.TrimSpace(u.Username),
		CakeID:    strings.TrimSpace(u.CakeID),
		Sign:      strings.TrimSpace(u.Sign),
		AvatarURL: strings.TrimSpace(u.AvatarURL),
		Status:    "active",
	}
	if doc.Nickname == "" {
		doc.Nickname = doc.Username
	}
	return c.indexDoc(ctx, IndexUsers, strconv.FormatUint(userID, 10), doc)
}

// ReindexAll rebuilds all published videos, articles, and active users (startup / manual).
func (c *Client) ReindexAll(ctx context.Context, db *gorm.DB) error {
	if !c.Enabled() {
		return nil
	}
	var vids []uint64
	if err := db.Model(&model.Video{}).Where("status = ?", "published").Pluck("id", &vids).Error; err != nil {
		return err
	}
	for _, id := range vids {
		if err := c.IndexVideoFromDB(ctx, db, id); err != nil {
			return fmt.Errorf("video %d: %w", id, err)
		}
	}
	var aids []uint64
	if err := db.Model(&model.Article{}).Where("status = ?", "published").Pluck("id", &aids).Error; err != nil {
		return err
	}
	for _, id := range aids {
		if err := c.IndexArticleFromDB(ctx, db, id); err != nil {
			return fmt.Errorf("article %d: %w", id, err)
		}
	}
	var uids []uint64
	if err := db.Model(&model.User{}).Where("anonymized_at IS NULL").Pluck("id", &uids).Error; err != nil {
		return err
	}
	for _, id := range uids {
		if err := c.IndexUserFromDB(ctx, db, id); err != nil {
			return fmt.Errorf("user %d: %w", id, err)
		}
	}
	return nil
}
