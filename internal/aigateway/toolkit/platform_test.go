package toolkit

import (
	"cakecake/internal/model/comment"
	"cakecake/internal/model/danmaku"
	"cakecake/internal/model/user"
	"cakecake/internal/model/video"
	"cakecake/internal/pkg/sensitive"
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func platformTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&video.Video{}, &user.User{}, &comment.Comment{}, &danmaku.Danmaku{},
	))
	return db
}

func newPlatformExecutor(t *testing.T) *PlatformExecutor {
	t.Helper()
	return &PlatformExecutor{DB: platformTestDB(t)}
}

func seedPlatformData(t *testing.T, db *gorm.DB) {
	t.Helper()
	require.NoError(t, db.Create(&user.User{ID: 1, Username: "alice", Nickname: "Alice"}).Error)
	require.NoError(t, db.Create(&video.Video{
		ID: 10, UserID: 1, Title: "golang tutorial", Status: video.StatusPublished, PlayCount: 5, DurationSec: 60,
	}).Error)
	require.NoError(t, db.Create(&video.Video{ID: 11, UserID: 1, Title: "draft", Status: video.StatusDraft}).Error)
	require.NoError(t, db.Create(&comment.Comment{ID: 5, VideoID: 10, UserID: 1, Content: "nice", Approved: true}).Error)
	require.NoError(t, db.Create(&danmaku.Danmaku{ID: 3, VideoID: 10, UserID: 1, Content: "hello"}).Error)
}

func TestPlatformExecute_UnknownTool(t *testing.T) {
	p := newPlatformExecutor(t)
	_, err := p.Execute(context.Background(), "nope", json.RawMessage(`{}`))
	require.Error(t, err)
}

func TestPlatformExecute_SensitiveArgs(t *testing.T) {
	p := newPlatformExecutor(t)
	f, err := os.CreateTemp("", "sens-*.txt")
	require.NoError(t, err)
	t.Cleanup(func() { os.Remove(f.Name()) })
	_, err = f.WriteString("badword\n")
	require.NoError(t, err)
	require.NoError(t, f.Close())
	p.Sens = sensitive.NewFilter(f.Name(), zap.NewNop())
	require.NoError(t, p.Sens.Reload())

	_, err = p.Execute(context.Background(), ToolSearchVideos, json.RawMessage(`{"keyword":"badword"}`))
	require.Error(t, err)
}

func TestPlatformSearchVideos(t *testing.T) {
	p := newPlatformExecutor(t)
	seedPlatformData(t, p.DB)
	ctx := context.Background()

	// Empty keyword.
	out, err := p.searchVideos(ctx, json.RawMessage(`{"keyword":""}`))
	require.NoError(t, err)
	require.Equal(t, `[]`, out)

	// Normal search (DB fallback).
	out, err = p.searchVideos(ctx, json.RawMessage(`{"keyword":"golang"}`))
	require.NoError(t, err)
	require.Contains(t, out, "golang tutorial")
	require.Contains(t, out, "Alice")

	// Invalid args.
	_, err = p.searchVideos(ctx, json.RawMessage(`not-json`))
	require.Error(t, err)

	// Execute dispatch.
	out, err = p.Execute(ctx, ToolSearchVideos, json.RawMessage(`{"keyword":"golang"}`))
	require.NoError(t, err)
	require.Contains(t, out, "golang tutorial")
}

func TestPlatformGetVideoDetail(t *testing.T) {
	p := newPlatformExecutor(t)
	seedPlatformData(t, p.DB)
	ctx := context.Background()

	out, err := p.getVideoDetail(ctx, json.RawMessage(`{"video_id":10}`))
	require.NoError(t, err)
	require.Contains(t, out, "golang tutorial")

	// Missing video id.
	_, err = p.getVideoDetail(ctx, json.RawMessage(`{"video_id":0}`))
	require.Error(t, err)

	// Not found returns an error JSON, no Go error.
	out, err = p.getVideoDetail(ctx, json.RawMessage(`{"video_id":999}`))
	require.NoError(t, err)
	require.Contains(t, out, "video not found")

	// Dispatch via Execute.
	out, err = p.Execute(ctx, ToolGetVideoDetail, json.RawMessage(`{"video_id":10}`))
	require.NoError(t, err)
	require.Contains(t, out, "golang tutorial")
}

func TestPlatformGetTrending(t *testing.T) {
	p := newPlatformExecutor(t)
	seedPlatformData(t, p.DB)
	ctx := context.Background()

	out, err := p.getTrending(ctx, json.RawMessage(`{}`))
	require.NoError(t, err)
	require.Contains(t, out, "golang tutorial")
	out, err = p.Execute(ctx, ToolGetTrending, json.RawMessage(`{}`))
	require.NoError(t, err)
	require.Contains(t, out, "golang tutorial")
	_, err = p.getTrending(ctx, json.RawMessage(`bad`))
	require.Error(t, err)
}

func TestPlatformGetVideoComments(t *testing.T) {
	p := newPlatformExecutor(t)
	seedPlatformData(t, p.DB)
	ctx := context.Background()

	out, err := p.getVideoComments(ctx, json.RawMessage(`{"video_id":10}`))
	require.NoError(t, err)
	require.Contains(t, out, "nice")

	out, err = p.getVideoComments(ctx, json.RawMessage(`{"video_id":999}`))
	require.NoError(t, err)
	require.Contains(t, out, `"items":[]`)

	_, err = p.getVideoComments(ctx, json.RawMessage(`{"video_id":0}`))
	require.Error(t, err)
	out, err = p.Execute(ctx, ToolGetVideoComments, json.RawMessage(`{"video_id":10}`))
	require.NoError(t, err)
	require.Contains(t, out, "nice")
}

func TestPlatformGetVideoDanmaku(t *testing.T) {
	p := newPlatformExecutor(t)
	seedPlatformData(t, p.DB)
	ctx := context.Background()

	out, err := p.getVideoDanmaku(ctx, json.RawMessage(`{"video_id":10}`))
	require.NoError(t, err)
	require.Contains(t, out, "hello")

	out, err = p.getVideoDanmaku(ctx, json.RawMessage(`{"video_id":999}`))
	require.NoError(t, err)
	require.Contains(t, out, `"items":[]`)

	_, err = p.getVideoDanmaku(ctx, json.RawMessage(`{"video_id":0}`))
	require.Error(t, err)
}

func TestTruncateStr(t *testing.T) {
	require.Equal(t, "abc", truncateStr("abc", 5))
	require.Equal(t, "abcde...", truncateStr("abcdef", 5))
	require.Equal(t, "", truncateStr("", 5))
}

func TestPlatformExecute_InvalidArgs(t *testing.T) {
	p := newPlatformExecutor(t)
	_, err := p.Execute(context.Background(), ToolSearchVideos, json.RawMessage(`bad`))
	require.Error(t, err)
	require.True(t, strings.Contains(err.Error(), "invalid args"))
}
