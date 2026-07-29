package queue

import (
	"context"
	"errors"
	"os"
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

// ---------- integration tests (require RABBITMQ_URL) ----------

func rabbitmqURL(t *testing.T) string {
	t.Helper()
	url := os.Getenv("RABBITMQ_URL")
	if url == "" {
		t.Skip("set RABBITMQ_URL to run RabbitMQ integration tests")
	}
	return url
}

func TestDial_Success(t *testing.T) {
	url := rabbitmqURL(t)
	c, err := Dial(url)
	if err != nil {
		t.Fatalf("Dial failed: %v", err)
	}
	defer func() {
		if err := c.Close(); err != nil {
			t.Logf("Close: %v", err)
		}
	}()
	if c.conn == nil {
		t.Error("connection should not be nil")
	}
	if c.ch == nil {
		t.Error("channel should not be nil")
	}
}

func TestPublishTranscode_Integration(t *testing.T) {
	url := rabbitmqURL(t)
	c, err := Dial(url)
	if err != nil {
		t.Fatalf("Dial failed: %v", err)
	}
	defer func() { _ = c.Close() }()

	err = c.PublishTranscode(context.Background(), []byte(`{"test":true}`))
	if err != nil {
		t.Fatalf("PublishTranscode failed: %v", err)
	}
}

func TestConsumeTranscode_Integration(t *testing.T) {
	url := rabbitmqURL(t)
	c, err := Dial(url)
	if err != nil {
		t.Fatalf("Dial failed: %v", err)
	}
	defer func() { _ = c.Close() }()

	msgs, err := c.ConsumeTranscode("test-consume-integration")
	if err != nil {
		t.Fatalf("ConsumeTranscode failed: %v", err)
	}
	if msgs == nil {
		t.Fatal("message channel should not be nil")
	}
}

func TestNewConsumerChannel_Integration(t *testing.T) {
	url := rabbitmqURL(t)
	c, err := Dial(url)
	if err != nil {
		t.Fatalf("Dial failed: %v", err)
	}
	defer func() { _ = c.Close() }()

	ch, err := c.NewConsumerChannel()
	if err != nil {
		t.Fatalf("NewConsumerChannel failed: %v", err)
	}
	if ch == nil {
		t.Fatal("channel should not be nil")
	}
	_ = ch.Close()
}
