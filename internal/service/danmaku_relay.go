package service

import (
	"context"
	"encoding/json"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"minibili/internal/data"
	"minibili/internal/ws"
)

// DanmakuRelay publishes danmaku-room WebSocket payloads via Redis Pub/Sub so
// multiple API replicas can fan out to their local Hub (SPEC NF-3).
type DanmakuRelay struct {
	Rdb *redis.Client
	Hub *ws.Hub
	Log *zap.Logger
}

// NewDanmakuRelay wires Redis to the in-process WebSocket hub.
func NewDanmakuRelay(rdb *redis.Client, hub *ws.Hub, log *zap.Logger) *DanmakuRelay {
	if log == nil {
		log = zap.NewNop()
	}
	return &DanmakuRelay{Rdb: rdb, Hub: hub, Log: log}
}

type danmakuFanoutEnvelope struct {
	VideoID uint64          `json:"video_id"`
	Body    json.RawMessage `json:"body"`
}

// Publish marshals payload to JSON and publishes one envelope to Redis.
func (r *DanmakuRelay) Publish(ctx context.Context, videoID uint64, payload interface{}) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	env := danmakuFanoutEnvelope{VideoID: videoID, Body: body}
	b, err := json.Marshal(env)
	if err != nil {
		return err
	}
	return r.Rdb.Publish(ctx, data.ChannelDanmakuFanout, b).Err()
}

// RunSubscriber blocks until ctx is cancelled; it must run in a goroutine per process.
func (r *DanmakuRelay) RunSubscriber(ctx context.Context) {
	sub := r.Rdb.Subscribe(ctx, data.ChannelDanmakuFanout)
	defer func() { _ = sub.Close() }()

	ch := sub.Channel()
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}
			if msg == nil || msg.Payload == "" {
				continue
			}
			var env danmakuFanoutEnvelope
			if err := json.Unmarshal([]byte(msg.Payload), &env); err != nil {
				r.Log.Warn("danmaku relay: skip bad envelope", zap.Error(err))
				continue
			}
			if env.VideoID == 0 || len(env.Body) == 0 {
				continue
			}
			r.Hub.BroadcastRaw(env.VideoID, env.Body)
		}
	}
}
