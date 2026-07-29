package queue

import (
	"context"
	"errors"
		"testing"

	amqp "github.com/rabbitmq/amqp091-go"
)

// ---------- mock channel ----------

type mockChannel struct {
	t *testing.T

	closeErr        error
	publishErr      error
	queueDeclareErr error

	closed          bool
	publishedBody   []byte
	publishExchange string
	publishKey      string
	consumeCh       chan amqp.Delivery
	declaredQueue   string
}

func (m *mockChannel) Close() error {
	m.closed = true
	return m.closeErr
}

func (m *mockChannel) PublishWithContext(_ context.Context, exchange, key string, _ bool, _ bool, msg amqp.Publishing) error {
	m.publishedBody = msg.Body
	m.publishExchange = exchange
	m.publishKey = key
	return m.publishErr
}

func (m *mockChannel) Consume(queue, consumer string, autoAck, exclusive, noLocal, noWait bool, args amqp.Table) (<-chan amqp.Delivery, error) {
	if m.consumeCh == nil {
		m.consumeCh = make(chan amqp.Delivery)
	}
	return m.consumeCh, nil
}

func (m *mockChannel) QueueDeclare(name string, durable, autoDelete, exclusive, noWait bool, args amqp.Table) (amqp.Queue, error) {
	m.declaredQueue = name
	return amqp.Queue{Name: name}, m.queueDeclareErr
}

// ---------- unit tests ----------

func TestTranscodeQueueConstant(t *testing.T) {
	if TranscodeQueue != "mini_bili_transcode" {
		t.Errorf("TranscodeQueue = %q, want %q", TranscodeQueue, "mini_bili_transcode")
	}
}

func TestClientImplementsPublisherCompileCheck(t *testing.T) {
	var c *Client
	var _ TranscodePublisher = c
	_ = TranscodeQueue
}

func TestClientStructZeroValue(t *testing.T) {
	var c Client
	if c.conn != nil {
		t.Error("conn should be nil for zero value")
	}
	if c.ch != nil {
		t.Error("ch should be nil for zero value")
	}
}

func TestClientNilClose(t *testing.T) {
	c := &Client{}
	if err := c.Close(); err != nil {
		t.Logf("Close on nil client: %v", err)
	}
}

func TestClose_Success(t *testing.T) {
	mc := &mockChannel{t: t}
	c := &Client{conn: nil, ch: mc}
	err := c.Close()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !mc.closed {
		t.Error("channel should be closed")
	}
}

func TestClose_ChannelError(t *testing.T) {
	mc := &mockChannel{t: t, closeErr: errors.New("close failed")}
	c := &Client{conn: nil, ch: mc}
	err := c.Close()
	if err != nil {
		t.Logf("expected channel error swallowed: %v", err)
	}
	if !mc.closed {
		t.Error("channel.Close should still be called")
	}
}

func TestClose_NilChannel(t *testing.T) {
	c := &Client{conn: nil, ch: nil}
	if err := c.Close(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPublishTranscode_Success(t *testing.T) {
	mc := &mockChannel{t: t}
	c := &Client{ch: mc}
	body := []byte(`{"video_id":123}`)
	err := c.PublishTranscode(context.Background(), body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(mc.publishedBody) != string(body) {
		t.Errorf("body = %q, want %q", mc.publishedBody, body)
	}
	if mc.publishKey != TranscodeQueue {
		t.Errorf("key = %q, want %q", mc.publishKey, TranscodeQueue)
	}
}

func TestPublishTranscode_Error(t *testing.T) {
	mc := &mockChannel{t: t, publishErr: errors.New("publish failed")}
	c := &Client{ch: mc}
	err := c.PublishTranscode(context.Background(), []byte(`{}`))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestPublishTranscode_NilChannel(t *testing.T) {
	c := &Client{ch: nil}
	err := c.PublishTranscode(context.Background(), []byte(`{}`))
	if err == nil {
		t.Fatal("expected error for nil channel")
	}
}

func TestConsumeTranscode_NilChannel(t *testing.T) {
	c := &Client{ch: nil}
	_, err := c.ConsumeTranscode("test-consumer")
	if err == nil {
		t.Fatal("expected error for nil channel")
	}
}

func TestNewConsumerChannel_NilConn(t *testing.T) {
	c := &Client{conn: nil}
	_, err := c.NewConsumerChannel()
	if err == nil {
		t.Fatal("expected error for nil connection")
	}
}

func TestDial_Error(t *testing.T) {
	// Dial with bad URL should fail without needing RabbitMQ.
	_, err := Dial("")
	if err == nil {
		t.Fatal("expected Dial error with empty URL")
	}
}
