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

// Client wraps an AMQP channel for publishing and consuming.
type Client struct {
	conn *amqp.Connection
	ch   *amqp.Channel
}

// Dial connects to RabbitMQ and declares the transcode queue.
func Dial(url string) (*Client, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("amqp dial: %w", err)
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
	return c.ch.PublishWithContext(ctx, "", TranscodeQueue, false, false, amqp.Publishing{
		DeliveryMode: amqp.Persistent,
		ContentType:  "application/json",
		Body:         body,
	})
}

// ConsumeTranscode registers a consumer (manual ack).
func (c *Client) ConsumeTranscode(consumerTag string) (<-chan amqp.Delivery, error) {
	return c.ch.Consume(TranscodeQueue, consumerTag, false, false, false, false, nil)
}

// NewConsumerChannel opens a dedicated channel for consuming (separate from publish channel).
func (c *Client) NewConsumerChannel() (*amqp.Channel, error) {
	return c.conn.Channel()
}
