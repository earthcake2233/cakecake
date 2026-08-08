package agent

import (
	"context"
	"encoding/json"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"cakecake/internal/data"
	"cakecake/internal/ws"
)

// AgentRelay is the cross-instance transport for agent WebSocket events and
// generation control commands:
//
//   - events (agent_delta, tool frames, suggestions, dm_message, ...) are
//     published to ChannelAgentEvent and every replica fans them out to its
//     local ChatHub for the target user;
//   - control commands (pause / resume / supersede) are published to
//     ChannelAgentControl and only the replica that owns the user's running
//     generation applies them (see AgentService.handleControl).
//
// With no relay wired (unit tests / single-process fallback) the service
// pushes events straight to its local ChatHub.
type AgentRelay struct {
	Rdb *redis.Client
	Hub *ws.ChatHub
	Log *zap.Logger
	Svc *AgentService
}

// NewAgentRelay wires Redis to the in-process ChatHub and the owning
// AgentService (used for control dispatch).
func NewAgentRelay(rdb *redis.Client, hub *ws.ChatHub, log *zap.Logger, svc *AgentService) *AgentRelay {
	if log == nil {
		log = zap.NewNop()
	}
	return &AgentRelay{Rdb: rdb, Hub: hub, Log: log, Svc: svc}
}

type agentEnvelope struct {
	UID     uint64          `json:"uid"`
	Payload json.RawMessage `json:"payload"`
}

// PublishEvent publishes one per-user agent event to Redis.
func (r *AgentRelay) PublishEvent(ctx context.Context, uid uint64, payload interface{}) error {
	if r == nil || r.Rdb == nil || uid == 0 {
		return nil
	}
	return r.publish(ctx, data.ChannelAgentEvent, uid, payload)
}

// PublishControl publishes one cross-instance control command to Redis.
func (r *AgentRelay) PublishControl(ctx context.Context, uid uint64, payload interface{}) error {
	if r == nil || r.Rdb == nil || uid == 0 {
		return nil
	}
	return r.publish(ctx, data.ChannelAgentControl, uid, payload)
}

func (r *AgentRelay) publish(ctx context.Context, channel string, uid uint64, payload interface{}) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	env := agentEnvelope{UID: uid, Payload: body}
	b, err := json.Marshal(env)
	if err != nil {
		return err
	}
	return r.Rdb.Publish(ctx, channel, b).Err()
}

// RunSubscriber blocks until ctx is cancelled; it must run in a goroutine per
// process. Events are fanned out to the local ChatHub, control commands are
// dispatched to the owning AgentService.
func (r *AgentRelay) RunSubscriber(ctx context.Context) {
	if r == nil || r.Rdb == nil {
		return
	}
	sub := r.Rdb.Subscribe(ctx, data.ChannelAgentEvent, data.ChannelAgentControl)
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
			var env agentEnvelope
			if err := json.Unmarshal([]byte(msg.Payload), &env); err != nil {
				r.Log.Warn("agent relay: skip bad envelope", zap.Error(err))
				continue
			}
			if env.UID == 0 || len(env.Payload) == 0 {
				continue
			}
			switch msg.Channel {
			case data.ChannelAgentEvent:
				if r.Hub != nil {
					r.Hub.PushRaw(env.UID, env.Payload)
				}
			case data.ChannelAgentControl:
				var ctrl map[string]interface{}
				if err := json.Unmarshal(env.Payload, &ctrl); err != nil {
					r.Log.Warn("agent relay: skip bad control", zap.Error(err))
					continue
				}
				if r.Svc != nil {
					r.Svc.handleControl(env.UID, ctrl)
				}
			}
		}
	}
}
