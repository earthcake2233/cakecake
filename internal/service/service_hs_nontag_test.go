package service

import (
	"context"
	"testing"
	"minibili/internal/model"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

)

func TestHS_TopWithScores(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	rec := &SearchHotRecorder{Rdb: rdb}
	ctx := context.Background()
	rows, err := rec.TopWithScores(ctx, 10)
	require.NoError(t, err)
	require.Equal(t, 0, len(rows))
	_ = rec.Record(ctx, 1, "", "golang")
	_ = rec.Record(ctx, 2, "", "rust")
	_ = rec.Record(ctx, 3, "", "rust")
	rows, err = rec.TopWithScores(ctx, 10)
	require.NoError(t, err)
	require.Equal(t, 2, len(rows))
	require.Equal(t, "rust", rows[0].Keyword)
	rows, err = (*SearchHotRecorder)(nil).TopWithScores(ctx, 10)
	require.NoError(t, err)
	require.Nil(t, rows)
	rows, err = rec.TopWithScores(ctx, 0)
	require.NoError(t, err)
	require.Nil(t, rows)
}

func TestHS_BoostKeyword(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	rec := &SearchHotRecorder{Rdb: rdb}
	ctx := context.Background()
	require.NoError(t, rec.BoostKeyword(ctx, "feature-x", 10))
	require.NoError(t, rec.BoostKeyword(ctx, "feature-x", 5))
	require.NoError(t, (*SearchHotRecorder)(nil).BoostKeyword(ctx, "x", 1))
}

func TestHS_RemoveKeyword(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	rec := &SearchHotRecorder{Rdb: rdb}
	ctx := context.Background()
	_ = rec.Record(ctx, 1, "", "remove-me")
	_ = rec.Record(ctx, 2, "", "keep-me")
	require.NoError(t, rec.RemoveKeyword(ctx, "remove-me"))
	require.NoError(t, rec.RemoveKeyword(ctx, "nonexistent"))
	require.NoError(t, (*SearchHotRecorder)(nil).RemoveKeyword(ctx, "x"))
}

func TestHS_ListHotSearchMergedLegacy(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	rec := &SearchHotRecorder{Rdb: rdb}
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.HotSearchOp{}))
	items, err := listHotSearchMergedLegacy(context.Background(), nil, rec, 10)
	require.NoError(t, err)
	require.NotNil(t, items)
	now := time.Now()
	end := now.Add(24 * time.Hour)
	op := model.HotSearchOp{Keyword: "news", OpType: "pin", PinRank: 1, Enabled: true, StartAt: &now, EndAt: &end}
	require.NoError(t, db.Create(&op).Error)
	items, err = listHotSearchMergedLegacy(context.Background(), db, rec, 10)
	require.NoError(t, err)
	require.NotNil(t, items)
}

func TestHS_BuildMergePools(t *testing.T) {
	pools := buildHotSearchMergePools(context.Background(), nil, nil, 10)
	require.NotNil(t, pools)
}

// resolveHotSearchEntry is tested indirectly via integration tests

func TestHS_MergeFromLayout(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	rec := &SearchHotRecorder{Rdb: rdb}
	items, has := mergeHotSearchFromLayout(context.Background(), nil, rec, 10)
	require.False(t, has)
	require.Nil(t, items)
}

func TestHS_EnsureLayoutFromMerged(t *testing.T) {
	err := EnsureHotSearchLayoutFromMerged(context.Background(), nil, nil, 10)
	require.NoError(t, err)
}

func TestHS_ReloadProfiles_NilRC(t *testing.T) {
	svc := &AgentService{RC: nil, DB: nil}
	svc.ReloadProfiles()
}
