package danmaku

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"cakecake/internal/ws"
)

// ---------- DanmakuRelay ----------

func TestNewDanmakuRelay(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	hub := ws.NewHub()
	log := zap.NewNop()

	relay := NewDanmakuRelay(rdb, hub, log)
	require.NotNil(t, relay)
	require.Equal(t, rdb, relay.Rdb)
	require.Equal(t, hub, relay.Hub)
	require.Equal(t, log, relay.Log)
}

func TestNewDanmakuRelay_NilLog(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	hub := ws.NewHub()

	relay := NewDanmakuRelay(rdb, hub, nil)
	require.NotNil(t, relay)
	require.NotNil(t, relay.Log) // should be zap.NewNop()
}

func TestDanmakuRelay_Publish_Simple(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	hub := ws.NewHub()
	relay := NewDanmakuRelay(rdb, hub, zap.NewNop())

	ctx := context.Background()
	err = relay.Publish(ctx, uint64(100), map[string]interface{}{"text": "hello"})
	require.NoError(t, err)
}
func TestDanmakuRelay_Publish_Error(t *testing.T) {
	// Use a disconnected client to simulate publish failure
	rdb := redis.NewClient(&redis.Options{Addr: "127.0.0.1:1"})
	hub := ws.NewHub()
	relay := NewDanmakuRelay(rdb, hub, zap.NewNop())

	ctx := context.Background()
	err := relay.Publish(ctx, uint64(1), "test")
	require.Error(t, err)
}
