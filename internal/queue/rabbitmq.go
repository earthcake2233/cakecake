package queue

import (
	"context"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
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

// amqpChannel is the subset of *amqp.Channel needed by Client.
type amqpChannel interface {
	PublishWithContext(ctx context.Context, exchange, key string, mandatory, immediate bool, msg amqp.Publishing) error
	Consume(queue, consumer string, autoAck, exclusive, noLocal, noWait bool, args amqp.Table) (<-chan amqp.Delivery, error)
	QueueDeclare(name string, durable, autoDelete, exclusive, noWait bool, args amqp.Table) (amqp.Queue, error)
	Close() error
}

// Compile-time check that *amqp.Channel satisfies amqpChannel.
var _ amqpChannel = (*amqp.Channel)(nil)

// Client wraps an AMQP channel for publishing and consuming.
type Client struct {
	conn *amqp.Connection
	ch   amqpChannel
}

// Dial connects to RabbitMQ and declares the transcode queue.
func Dial(url string) (*Client, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("amqp dial failed")
	}
	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("amqp channel: %w", err)
	}
	if _, err := ch.QueueDeclare(TranscodeQueue, true, false, false, false, nil); err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return nil, fmt.Errorf("queue declare: %w", err)
	}
	if _, err := ch.QueueDeclare(TranscodeDeadQueue, true, false, false, false, nil); err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return nil, fmt.Errorf("dead queue declare: %w", err)
	}
	return &Client{conn: conn, ch: ch}, nil
}

// Close releases resources.
func (c *Client) Close() error {
	if c.ch != nil {
		_ = c.ch.Close()
	}
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// PublishTranscode sends a persistent JSON body to the transcode queue.
func (c *Client) PublishTranscode(ctx context.Context, body []byte) error {
	if c.ch == nil {
		return fmt.Errorf("channel is nil")
	}
	return c.ch.PublishWithContext(ctx, "", TranscodeQueue, false, false, amqp.Publishing{
		DeliveryMode: amqp.Persistent,
		ContentType:  "application/json",
		Body:         body,
	})
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

// NewConsumerChannel opens a dedicated channel for consuming (separate from publish channel).
func (c *Client) NewConsumerChannel() (*amqp.Channel, error) {
	if c.conn == nil {
		return nil, fmt.Errorf("connection is nil")
	}
	return c.conn.Channel()
}
