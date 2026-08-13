package queue

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

// TranscodePublisher publishes transcode jobs (implemented by *Client).
type TranscodePublisher interface {
	PublishTranscode(ctx context.Context, body []byte) error
}

// Compile-time check.
var _ TranscodePublisher = (*Client)(nil)

// TranscodeQueue is the durable queue name for video transcoding jobs.
const TranscodeQueue = "mini_bili_transcode"

// TranscodeDeadQueue receives jobs whose retries were exhausted, so failed
// transcodes are observable and compensable instead of silently dropped.
const TranscodeDeadQueue = "mini_bili_transcode_dead"

// Retry queues carry jobs that failed with a retryable error. Each queue has
// a fixed per-message TTL; when a message expires it is dead-lettered back to
// TranscodeQueue, so backoff is owned by RabbitMQ instead of a worker sleep.
const (
	TranscodeRetryQueue30s = "mini_bili_transcode_retry_30s"
	TranscodeRetryQueue60s = "mini_bili_transcode_retry_60s"
	TranscodeRetryQueue90s = "mini_bili_transcode_retry_90s"
)

// RetryQueueForAttempt returns the retry queue for a 1-based retry attempt
// (attempt 1 = 30s, 2 = 60s, 3+ = 90s).
func RetryQueueForAttempt(attempt int) string {
	switch attempt {
	case 1:
		return TranscodeRetryQueue30s
	case 2:
		return TranscodeRetryQueue60s
	default:
		return TranscodeRetryQueue90s
	}
}

// retryQueues is the ordered set of delayed queues declared at Dial time.
var retryQueues = []struct {
	name string
	ttl  time.Duration
}{
	{TranscodeRetryQueue30s, 30 * time.Second},
	{TranscodeRetryQueue60s, 60 * time.Second},
	{TranscodeRetryQueue90s, 90 * time.Second},
}

// publishConfirmTimeout bounds how long PublishConfirmed waits for a broker
// confirmation before failing the call.
const publishConfirmTimeout = 5 * time.Second

// rabbitmqReconnectDelay is the backoff between connection rebuild attempts;
// variable for tests.
var rabbitmqReconnectDelay = 3 * time.Second

// amqpChannel is the subset of *amqp.Channel needed by Client.
type amqpChannel interface {
	PublishWithContext(ctx context.Context, exchange, key string, mandatory, immediate bool, msg amqp.Publishing) error
	Confirm(noWait bool) error
	GetNextPublishSeqNo() uint64
	NotifyPublish(confirm chan amqp.Confirmation) chan amqp.Confirmation
	NotifyReturn(returns chan amqp.Return) chan amqp.Return
	Consume(queue, consumer string, autoAck, exclusive, noLocal, noWait bool, args amqp.Table) (<-chan amqp.Delivery, error)
	QueueDeclare(name string, durable, autoDelete, exclusive, noWait bool, args amqp.Table) (amqp.Queue, error)
	Close() error
}

// Compile-time check that *amqp.Channel satisfies amqpChannel.
var _ amqpChannel = (*amqp.Channel)(nil)

// Client wraps an AMQP channel for publishing and consuming.
type Client struct {
	url  string
	conn *amqp.Connection
	ch   amqpChannel

	mu       sync.Mutex
	confirms <-chan amqp.Confirmation
	returns  <-chan amqp.Return

	closed        bool
	reconnectWait time.Duration
	dialFn        func(url string) (*amqp.Connection, error)

	ctx    context.Context
	cancel context.CancelFunc

	log atomic.Value // *zap.Logger
}

// TranscodeConsumer is the minimal channel surface the main transcode
// consumer loop needs, kept small so the reconnect loop is unit-testable.
type TranscodeConsumer interface {
	Close() error
	Qos(prefetchCount, prefetchSize int, global bool) error
}

// Dial connects to RabbitMQ and declares the transcode queue.
func Dial(url string) (*Client, error) {
	return dialWithFn(url, amqp.Dial)
}

func dialWithFn(url string, dialFn func(string) (*amqp.Connection, error)) (*Client, error) {
	conn, err := dialFn(url)
	if err != nil {
		return nil, fmt.Errorf("amqp dial failed")
	}
	ctx, cancel := context.WithCancel(context.Background())
	c := &Client{
		url:           url,
		conn:          conn,
		reconnectWait: rabbitmqReconnectDelay,
		dialFn:        dialFn,
		ctx:           ctx,
		cancel:        cancel,
	}
	c.log.Store(zap.NewNop())
	if err := c.initChannel(); err != nil {
		cancel()
		_ = conn.Close()
		return nil, err
	}
	go c.watchConnection()
	return c, nil
}

// SetLogger replaces the reconnect logger (project-wide zap instance).
func (c *Client) SetLogger(lg *zap.Logger) {
	if lg != nil {
		c.log.Store(lg)
	}
}

func (c *Client) logger() *zap.Logger {
	if v := c.log.Load(); v != nil {
		return v.(*zap.Logger)
	}
	return zap.NewNop()
}

// initChannel (re)builds the publish channel, declares the queues and enables
// publisher confirms. Callers must not hold c.mu.
func (c *Client) initChannel() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.initChannelLocked()
}

func (c *Client) initChannelLocked() error {
	ch, err := c.conn.Channel()
	if err != nil {
		return fmt.Errorf("amqp channel: %w", err)
	}
	if _, err := ch.QueueDeclare(TranscodeQueue, true, false, false, false, nil); err != nil {
		_ = ch.Close()
		return fmt.Errorf("queue declare: %w", err)
	}
	if _, err := ch.QueueDeclare(TranscodeDeadQueue, true, false, false, false, nil); err != nil {
		_ = ch.Close()
		return fmt.Errorf("dead queue declare: %w", err)
	}
	for _, rq := range retryQueues {
		args := amqp.Table{
			"x-message-ttl":             int64(rq.ttl.Milliseconds()),
			"x-dead-letter-exchange":    "",
			"x-dead-letter-routing-key": TranscodeQueue,
		}
		if _, err := ch.QueueDeclare(rq.name, true, false, false, false, args); err != nil {
			_ = ch.Close()
			return fmt.Errorf("retry queue %s declare: %w", rq.name, err)
		}
	}
	confirms := ch.NotifyPublish(make(chan amqp.Confirmation, 32))
	returns := ch.NotifyReturn(make(chan amqp.Return, 32))
	if err := ch.Confirm(false); err != nil {
		_ = ch.Close()
		return fmt.Errorf("publisher confirm: %w", err)
	}
	if c.ch != nil {
		_ = c.ch.Close()
	}
	c.ch = ch
	c.confirms = confirms
	c.returns = returns
	return nil
}

// watchConnection waits for the underlying connection to die and rebuilds it
// (channel included) so a broker restart self-heals without a process restart.
func (c *Client) watchConnection() {
	for {
		notify := c.conn.NotifyClose(make(chan *amqp.Error, 1))
		select {
		case <-c.ctx.Done():
			return
		case err, ok := <-notify:
			c.mu.Lock()
			closed := c.closed
			c.mu.Unlock()
			if closed {
				return
			}
			if ok && err != nil {
				c.logger().Warn("rabbitmq connection lost; reconnecting", zap.Error(err))
			}
			if !c.reconnect() {
				return
			}
		}
	}
}

// reconnect dials a fresh connection and re-initializes the publish channel.
// It returns false when the client was closed while reconnecting.
func (c *Client) reconnect() bool {
	for {
		select {
		case <-c.ctx.Done():
			return false
		default:
		}
		conn, err := c.dialFn(c.url)
		if err != nil {
			if !c.waitReconnect() {
				return false
			}
			continue
		}
		c.mu.Lock()
		if c.closed {
			c.mu.Unlock()
			_ = conn.Close()
			return false
		}
		oldConn, oldCh := c.conn, c.ch
		c.conn = conn
		c.ch = nil
		c.confirms = nil
		c.returns = nil
		if err := c.initChannelLocked(); err != nil {
			c.mu.Unlock()
			_ = conn.Close()
			// The old connection/channel are still owned by this client; close
			// them explicitly so a live-but-misconfigured broker cannot leak
			// the previous connection (dead-connection reconnect is a no-op).
			if oldCh != nil {
				_ = oldCh.Close()
			}
			if oldConn != nil {
				_ = oldConn.Close()
			}
			if !c.waitReconnect() {
				return false
			}
			continue
		}
		c.mu.Unlock()
		if oldCh != nil {
			_ = oldCh.Close()
		}
		if oldConn != nil {
			_ = oldConn.Close()
		}
		c.logger().Info("rabbitmq reconnected")
		return true
	}
}

func (c *Client) waitReconnect() bool {
	t := time.NewTimer(c.reconnectWait)
	defer t.Stop()
	select {
	case <-c.ctx.Done():
		return false
	case <-t.C:
		return true
	}
}

// Close releases resources.
func (c *Client) Close() error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil
	}
	c.closed = true
	ch := c.ch
	conn := c.conn
	c.mu.Unlock()
	if c.cancel != nil {
		c.cancel()
	}
	if ch != nil {
		_ = ch.Close()
	}
	if conn != nil {
		return conn.Close()
	}
	return nil
}

// PublishTranscode sends a persistent JSON body to the transcode queue and
// waits for a broker confirmation, so enqueue callers know the job actually
// reached RabbitMQ.
func (c *Client) PublishTranscode(ctx context.Context, body []byte) error {
	return c.PublishConfirmed(ctx, "", TranscodeQueue, true, amqp.Publishing{
		DeliveryMode: amqp.Persistent,
		ContentType:  "application/json",
		Body:         body,
	})
}

// PublishConfirmed publishes a message and blocks until RabbitMQ confirms it.
// With mandatory=true, an unroutable message triggers a basic.return and is
// reported as an error. Confirmations are matched by delivery tag, so a late
// ack from an earlier timed-out publish is ignored instead of being
// misattributed to the next call.
func (c *Client) PublishConfirmed(ctx context.Context, exchange, key string, mandatory bool, msg amqp.Publishing) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return fmt.Errorf("client closed")
	}
	if c.ch == nil || c.confirms == nil {
		return fmt.Errorf("publisher confirm not enabled")
	}
	seq := c.ch.GetNextPublishSeqNo()
	if err := c.ch.PublishWithContext(ctx, exchange, key, mandatory, false, msg); err != nil {
		return err
	}
	timeout := time.NewTimer(publishConfirmTimeout)
	defer timeout.Stop()
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("publish confirm: %w", ctx.Err())
		case ret, ok := <-c.returns:
			if ok {
				return fmt.Errorf("publish returned by broker: %s (code %d)", ret.ReplyText, ret.ReplyCode)
			}
			return fmt.Errorf("publish return channel closed")
		case conf, ok := <-c.confirms:
			if !ok {
				return fmt.Errorf("publish confirm channel closed")
			}
			if conf.DeliveryTag < seq {
				// Late confirmation from an earlier timed-out publish;
				// ignore it and keep waiting for ours.
				continue
			}
			if conf.DeliveryTag > seq {
				return fmt.Errorf("publish confirm tag mismatch: got %d, want %d", conf.DeliveryTag, seq)
			}
			if !conf.Ack {
				return fmt.Errorf("publish nacked by broker")
			}
			// RabbitMQ sends basic.return before basic.ack for unroutable
			// messages; drain a return that is already buffered.
			select {
			case ret, ok := <-c.returns:
				if ok {
					return fmt.Errorf("publish returned by broker: %s (code %d)", ret.ReplyText, ret.ReplyCode)
				}
			default:
			}
			return nil
		case <-timeout.C:
			return fmt.Errorf("publish confirm timeout after %s", publishConfirmTimeout)
		}
	}
}

// connSnapshot returns the current connection under lock, or nil when closed.
// Channels derived from a stale snapshot may fail; consumers retry on error.
func (c *Client) connSnapshot() *amqp.Connection {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return nil
	}
	return c.conn
}

// ConsumeTranscode registers a consumer (manual ack).
func (c *Client) ConsumeTranscode(consumerTag string) (<-chan amqp.Delivery, error) {
	if c.ch == nil {
		return nil, fmt.Errorf("channel is nil")
	}
	return c.ch.Consume(TranscodeQueue, consumerTag, false, false, false, false, nil)
}

// ConsumeTranscodeDead registers a consumer for the dead-letter queue.
func (c *Client) ConsumeTranscodeDead(consumerTag string) (<-chan amqp.Delivery, error) {
	if c.ch == nil {
		return nil, fmt.Errorf("channel is nil")
	}
	return c.ch.Consume(TranscodeDeadQueue, consumerTag, false, false, false, false, nil)
}

// NewTranscodeDeadConsumer opens a dedicated channel and subscribes it to the
// dead-letter queue, returning both for the consumer loop. The channel is
// returned as a minimal closer so tests can substitute a fake.
func (c *Client) NewTranscodeDeadConsumer(consumerTag string) (interface{ Close() error }, <-chan amqp.Delivery, error) {
	conn := c.connSnapshot()
	if conn == nil {
		return nil, nil, fmt.Errorf("connection is nil")
	}
	ch, err := conn.Channel()
	if err != nil {
		return nil, nil, err
	}
	msgs, err := ch.Consume(TranscodeDeadQueue, consumerTag, false, false, false, false, nil)
	if err != nil {
		_ = ch.Close()
		return nil, nil, err
	}
	return ch, msgs, nil
}

// NewConsumerChannel opens a dedicated channel for consuming (separate from publish channel).
func (c *Client) NewConsumerChannel() (*amqp.Channel, error) {
	conn := c.connSnapshot()
	if conn == nil {
		return nil, fmt.Errorf("connection is nil")
	}
	return conn.Channel()
}

// NewTranscodeConsumer opens a dedicated channel, applies QoS=1 and subscribes
// to the main transcode queue. The caller owns closing the returned consumer.
func (c *Client) NewTranscodeConsumer(consumerTag string) (TranscodeConsumer, <-chan amqp.Delivery, error) {
	conn := c.connSnapshot()
	if conn == nil {
		return nil, nil, fmt.Errorf("connection is nil")
	}
	ch, err := conn.Channel()
	if err != nil {
		return nil, nil, err
	}
	if err := ch.Qos(1, 0, false); err != nil {
		_ = ch.Close()
		return nil, nil, err
	}
	msgs, err := ch.Consume(TranscodeQueue, consumerTag, false, false, false, false, nil)
	if err != nil {
		_ = ch.Close()
		return nil, nil, err
	}
	return ch, msgs, nil
}
