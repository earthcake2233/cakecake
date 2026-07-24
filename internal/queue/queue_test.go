package queue

import (
	"testing"
)

func TestTranscodeQueueConstant(t *testing.T) {
	if TranscodeQueue != "mini_bili_transcode" {
		t.Errorf("TranscodeQueue = %q, want %q", TranscodeQueue, "mini_bili_transcode")
	}
}

// Compile-time check: Client implements TranscodePublisher.
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

func TestPublishTranscode_skipped(t *testing.T) {
	t.Skip("requires RabbitMQ connection")
}

func TestConsumeTranscode_skipped(t *testing.T) {
	t.Skip("requires RabbitMQ connection")
}

func TestNewConsumerChannel_skipped(t *testing.T) {
	t.Skip("requires RabbitMQ connection")
}

func TestDial_skipped(t *testing.T) {
	t.Skip("requires RabbitMQ connection")
}
