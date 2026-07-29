//go:build integration

package queue

import (
	"context"
	"os"
	"testing"
)

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
