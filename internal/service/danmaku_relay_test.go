package service

import (
	"cakecake/internal/data"
	"cakecake/internal/ws"
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestDanmakuRelay_Publish(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})

	r := NewDanmakuRelay(rdb, ws.NewHub(), nil)
	ctx := context.Background()
	require.NoError(t, r.Publish(ctx, 42, map[string]interface{}{"msg": "hi"}))

	// Marshal error path.
	err = r.Publish(ctx, 42, make(chan int))
	require.Error(t, err)
}

func TestDanmakuRelay_RunSubscriber(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	hub := ws.NewHub()
	r := NewDanmakuRelay(rdb, hub, nil)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go r.RunSubscriber(ctx)
	time.Sleep(50 * time.Millisecond)
	require.NoError(t, r.Publish(ctx, 42, map[string]interface{}{"msg": "hi"}))
	// RunSubscriber stays alive; cancel stops it.
	cancel()
	time.Sleep(50 * time.Millisecond)

	_ = data.ChannelDanmakuFanout
}
