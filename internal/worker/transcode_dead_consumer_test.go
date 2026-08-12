package worker

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type fakeCloseable struct {
	closed bool
}

func (f *fakeCloseable) Close() error {
	f.closed = true
	return nil
}

type fakeDeadSubscriber struct {
	calls int
	errs  []error
	ch    *fakeCloseable
	msgs  chan amqp.Delivery
}

func (f *fakeDeadSubscriber) NewTranscodeDeadConsumer(_ string) (interface{ Close() error }, <-chan amqp.Delivery, error) {
	f.calls++
	if len(f.errs) > 0 {
		err := f.errs[0]
		f.errs = f.errs[1:]
		return nil, nil, err
	}
	return f.ch, f.msgs, nil
}

func TestHandleTranscodeDeadLetter_Acks(t *testing.T) {
	ack := &fakeAck{}
	body, _ := json.Marshal(TranscodeJob{VideoID: 42, RetryCount: 3})
	handleTranscodeDeadLetter(amqp.Delivery{Body: body, Acknowledger: ack}, zap.NewNop())
	require.Equal(t, 1, ack.acked)
}

func TestConsumeTranscodeDead_AcksAndCloses(t *testing.T) {
	ack := &fakeAck{}
	body, _ := json.Marshal(TranscodeJob{VideoID: 7, RetryCount: 2})
	msgs := make(chan amqp.Delivery, 1)
	msgs <- amqp.Delivery{Body: body, Acknowledger: ack}
	close(msgs)

	closer := &fakeCloseable{}
	consumeTranscodeDead(context.Background(), closer, msgs, zap.NewNop())
	require.Equal(t, 1, ack.acked)
	require.True(t, closer.closed)
}

func TestStartTranscodeDeadConsumer_ReconnectsUntilCancel(t *testing.T) {
	old := transcodeDeadRetryDelay
	transcodeDeadRetryDelay = time.Millisecond
	defer func() { transcodeDeadRetryDelay = old }()

	sub := &fakeDeadSubscriber{
		errs: []error{errTestBrokerDown, errTestBrokerDown},
		ch:   &fakeCloseable{},
		msgs: make(chan amqp.Delivery),
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		StartTranscodeDeadConsumer(ctx, nil, nil, sub, zap.NewNop())
	}()

	// Let the loop hit the subscribe errors and reach the stable msgs channel.
	deadline := time.Now().Add(2 * time.Second)
	for sub.calls < 3 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	cancel()
	<-done
	require.GreaterOrEqual(t, sub.calls, 3)
}

var errTestBrokerDown = &testBrokerError{}

type testBrokerError struct{}

func (*testBrokerError) Error() string { return "broker down" }
