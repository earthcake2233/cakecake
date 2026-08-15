//go:build integration

package queue

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
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

func TestPublishConfirmed_Integration(t *testing.T) {
	url := rabbitmqURL(t)
	c, err := Dial(url)
	if err != nil {
		t.Fatalf("Dial failed: %v", err)
	}
	defer func() { _ = c.Close() }()

	// Publish to the dead queue: durable, no TTL, so the confirmed message
	// stays put and proves the confirm round-trip against a real broker.
	err = c.PublishConfirmed(context.Background(), "", TranscodeDeadQueue, true, amqp.Publishing{
		DeliveryMode: amqp.Persistent,
		ContentType:  "application/json",
		Body:         []byte(`{"integration":true}`),
	})
	if err != nil {
		t.Fatalf("PublishConfirmed failed: %v", err)
	}
}

func TestPublishConfirmed_Concurrent_Integration(t *testing.T) {
	url := rabbitmqURL(t)
	c, err := Dial(url)
	if err != nil {
		t.Fatalf("Dial failed: %v", err)
	}
	defer func() { _ = c.Close() }()

	const n = 8
	var wg sync.WaitGroup
	errs := make([]error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			errs[i] = c.PublishConfirmed(context.Background(), "", TranscodeQueue, true, amqp.Publishing{
				DeliveryMode: amqp.Persistent,
				Body:         []byte(fmt.Sprintf(`{"concurrent":%d}`, i)),
			})
		}(i)
	}
	wg.Wait()
	for i, err := range errs {
		if err != nil {
			t.Fatalf("concurrent publish %d failed: %v", i, err)
		}
	}
}

func TestRetryQueueDeclaration_Integration(t *testing.T) {
	url := rabbitmqURL(t)
	c, err := Dial(url)
	if err != nil {
		t.Fatalf("Dial failed: %v", err)
	}
	defer func() { _ = c.Close() }()

	for _, attempt := range []int{1, 2, 3} {
		queueName := RetryQueueForAttempt(attempt)
		err = c.PublishConfirmed(context.Background(), "", queueName, true, amqp.Publishing{
			DeliveryMode: amqp.Persistent,
			Body:         []byte(`{"retry_queue_test":true}`),
		})
		if err != nil {
			t.Fatalf("publish to retry queue %s failed: %v", queueName, err)
		}
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

func TestNewTranscodeConsumer_Integration(t *testing.T) {
	url := rabbitmqURL(t)
	c, err := Dial(url)
	if err != nil {
		t.Fatalf("Dial failed: %v", err)
	}
	defer func() { _ = c.Close() }()

	ch, msgs, err := c.NewTranscodeConsumer("test-reconnect-integration")
	if err != nil {
		t.Fatalf("NewTranscodeConsumer failed: %v", err)
	}
	defer func() { _ = ch.Close() }()
	if msgs == nil {
		t.Fatal("message channel should not be nil")
	}
}

func TestReconnect_Integration(t *testing.T) {
	url := rabbitmqURL(t)
	old := rabbitmqReconnectDelay
	rabbitmqReconnectDelay = 100 * time.Millisecond
	defer func() { rabbitmqReconnectDelay = old }()

	c, err := Dial(url)
	if err != nil {
		t.Fatalf("Dial failed: %v", err)
	}
	defer func() { _ = c.Close() }()

	oldConn := c.conn
	if err := oldConn.Close(); err != nil {
		t.Fatalf("close connection to trigger reconnect: %v", err)
	}

	deadline := time.Now().Add(5 * time.Second)
	for {
		c.mu.Lock()
		curConn, curCh := c.conn, c.ch
		c.mu.Unlock()
		if curConn != nil && curConn != oldConn && curCh != nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("connection did not self-heal after broker connection loss")
		}
		time.Sleep(20 * time.Millisecond)
	}

	if err := c.PublishTranscode(context.Background(), []byte(`{"reconnect_test":true}`)); err != nil {
		t.Fatalf("publish after reconnect failed: %v", err)
	}
}

func TestConcurrentConsumers_Integration(t *testing.T) {
	url := rabbitmqURL(t)
	c, err := Dial(url)
	if err != nil {
		t.Fatalf("Dial failed: %v", err)
	}
	defer func() { _ = c.Close() }()

	const n = 3
	for i := 0; i < n; i++ {
		if err := c.PublishTranscode(context.Background(), []byte(fmt.Sprintf(`{"seq":%d}`, i))); err != nil {
			t.Fatalf("publish %d failed: %v", i, err)
		}
	}

	received := make(chan string, n)
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			ch, msgs, err := c.NewTranscodeConsumer(fmt.Sprintf("concurrent-%d", i))
			if err != nil {
				t.Errorf("consumer %d subscribe failed: %v", i, err)
				return
			}
			defer func() { _ = ch.Close() }()
			select {
			case d, ok := <-msgs:
				if !ok {
					t.Errorf("consumer %d channel closed", i)
					return
				}
				_ = d.Ack(false)
				received <- string(d.Body)
			case <-time.After(5 * time.Second):
				t.Errorf("consumer %d did not receive a message", i)
			}
		}(i)
	}
	wg.Wait()
	close(received)

	seen := map[string]bool{}
	for b := range received {
		seen[b] = true
	}
	if len(seen) != n {
		t.Errorf("expected %d distinct messages across consumers, got %d", n, len(seen))
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
